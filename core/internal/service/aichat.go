// AI 对话（M31 v5）：工具治理循环——模块化工具集（load_tools 按需展开）+
// 权限档位门禁（M60 四档：read_only / confirm / auto_edit / full）+ 操作审计 + SSE 流式。
// langchaingo 每轮 GenerateContent 单独传 WithTools，轮间切换工具列表合法；
// ask 通过 pendingAsks channel 挂起等待前端确认（POST /ai/ask/:id），fail-closed。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
)

// M60：长任务（多步安装/迁移/排查）经常超出小轮次上限后被静默掐断，放宽并配合耗尽提示。
const maxToolRounds = 30

// errAIPermDenied：RBAC 工具权限拒绝（M60）——调用方按「无权限」步骤呈现，区别于普通失败。
var errAIPermDenied = errors.New("权限不足")

// toolOutputTrunc 卡片「原始输出」上限（完整结果仍在模型上下文，此处仅供用户展开查看）。
const toolOutputTrunc = 2000

// 权限档位（M60 对齐 ZCode 权限级别）：
// read_only 只读（写/danger 工具注入期剔除）；confirm 变更前确认（write+danger 全部弹确认）；
// auto_edit 自动编辑（write 自动执行，danger 仍需确认）；full 完全访问（全部自动执行）。
const (
	aiModeReadOnly = "read_only"
	aiModeConfirm  = "confirm"
	aiModeAutoEdit = "auto_edit"
	aiModeFull     = "full"
)

// normalizeAIMode 归一权限档位，兼容旧值（standard→confirm、auto→full），未知回落 confirm。
func normalizeAIMode(mode string) string {
	switch mode {
	case aiModeReadOnly, aiModeConfirm, aiModeAutoEdit, aiModeFull:
		return mode
	case "standard":
		return aiModeConfirm
	case "auto":
		return aiModeFull
	default:
		return aiModeConfirm
	}
}

const aiSystemPrompt = "你是 YPanel 服务器管理面板的运维助手。用简洁中文回答服务器运维问题，回答使用 Markdown 格式。" +
	"面板能力按模块组织：需要某模块的工具时先调 load_tools 加载（模块目录见 load_tools 描述；用户当前页面相关模块通常已预注入，可直接调用）。" +
	"查询类工具直接调用；写操作与危险操作（删除/停启/清理等）是否弹确认框取决于用户选择的权限档位——被拒绝或超时未确认的操作不得擅自重试，应说明情况并询问用户意愿。" +
	"执行任何变更前，先用一句话向用户说明你要做什么。" +
	"涉及外部软件文档、报错资料、最佳实践时可用 web_search 联网搜索、fetch_url 抓取网页正文；系统提示中的长期记忆包含本环境服务器特性与历史踩坑，回答前优先对照。" +
	"用户表达的是意图而非具体操作时，主动串联工具完成端到端交付而不是只给建议，常见工作流：" +
	"①想要某类应用（博客/图床/网盘等）：store_search_apps 检索 → store_app_detail 看参数 → 安装前先 list_database_instances 和 list_installed_apps 检查是否已有可复用的数据库/依赖服务（有则告知用户并复用，不重复安装）；存在多个同类实例时，列出候选（名称/类型/端口/备注）交由用户选择目标实例，确认后用 reveal_db_credentials 取连接信息并按需 create_database/create_db_user，再填入应用参数→ store_install_app 安装（确认后执行）→ get_task 跟踪安装 → list_compose_projects 验证运行，需要域名时 site_create 反代应用端口 + issue_cert 签发证书；" +
	"②把 git 仓库跑起来：src_build_preview 检测语言栈与构建参数 → 与用户确认服务/端口 → src_build_create 创建构建任务 → get_task 跟踪构建进度 → list_compose_projects/compose_logs 验证跑通；" +
	"③服务挂了/访问不了：get_overview → journal_logs(系统服务)或 container_logs(容器) → check_port 看监听 → http_probe 探测连通 → dns_resolve 验证解析，定位根因后给修复方案；" +
	"④数据库类：list_database_instances → list_databases/list_tables/list_db_columns → query_database 只读验证 → 变更走 row_insert/row_update/row_delete（需确认）。"

// buildLLM 按供应商配置构造 langchaingo 模型。
func buildLLM(p *model.AIProvider) (llms.Model, error) {
	opts := []openai.Option{
		openai.WithToken(p.APIKey),
		openai.WithModel(p.Model),
	}
	if p.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(strings.TrimRight(p.BaseURL, "/")))
	}
	return openai.New(opts...)
}

// defsToLLMTools 工具声明转 langchaingo Tool 列表（带 JSON Schema 参数）。
func defsToLLMTools(defs []aiToolDef) []llms.Tool {
	out := make([]llms.Tool, 0, len(defs))
	for i := range defs {
		out = append(out, llms.Tool{Type: "function", Function: &llms.FunctionDefinition{
			Name:        defs[i].Name,
			Description: defs[i].Desc,
			Parameters:  defs[i].Parameters,
		}})
	}
	return out
}

// RunAgentChat 工具治理循环 + 事件流输出（M62：与 HTTP 连接解耦——emit/ping 由
// 调用方注入（chatRun 总线或旧式 SSE 写出），ctx 为任务自身生命周期）。
// 事件：data: {"scene":{...}} / {"step":{...}} / {"step_result":{...}} / {"ask":{...}} /
// {"ask_result":{...}} / {"reasoning":"..."} / {"content":"..."} / {"knowledge":[...]} / [DONE]
func (s *AIService) RunAgentChat(
	ctx context.Context,
	emit func(any),
	ping func(),
	provider *model.AIProvider,
	history []ChatMessage,
	scenePath string,
	focus string,
	mode string,
) error {
	// 权限档位（M60）：read_only 只读工具集；confirm 变更前确认；auto_edit 自动编辑；full 完全访问
	mode = normalizeAIMode(mode)
	llm, err := buildLLM(provider)
	if err != nil {
		return err
	}

	// 场景感知
	if scene := s.SceneContext(ctx, scenePath, focus); scene != nil {
		emit(map[string]any{"scene": scene})
	}

	// system：提示词 + 技能注入 + 长期记忆召回 + 知识库检索 + 场景摘要
	sys := aiSystemPrompt
	if sk := s.skills.EnabledBodies(ctx); len(sk) > 0 {
		sys += "\n\n可用技能（用户提问匹配技能用途时，按技能正文执行）：\n" + strings.Join(sk, "\n---\n")
	}
	userText := scenePath
	for _, m := range history {
		userText += " " + m.Content
	}
	if mems := s.RecallMemories(userText); len(mems) > 0 {
		var parts []string
		for _, m := range mems {
			parts = append(parts, "- "+m.Content)
		}
		sys += "\n\n长期记忆（过往沉淀的经验/偏好，回答时参考）：\n" + strings.Join(parts, "\n")
	}
	kbHits := s.searchKnowledge(userText)
	if len(kbHits) > 0 {
		var parts []string
		cits := make([]map[string]string, 0, len(kbHits))
		for _, item := range kbHits {
			parts = append(parts, "["+item.Title+"]"+item.Body)
			cits = append(cits, map[string]string{"title": item.Title, "body": item.Body})
		}
		sys += "\n\n相关知识库条目：\n" + strings.Join(parts, "\n---\n")
		// 引用来源事件：前端在回答下方展示来源（点击可展开正文）
		emit(map[string]any{"knowledge": cits})
	}
	sys += "\n\n" + s.buildSceneSummary(ctx, scenePath, focus)

	// 消息链（标准 function calling 多轮格式）
	msgs := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, sys)}
	for _, m := range history {
		if m.Role == "user" {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeHuman, m.Content))
		} else {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeAI, m.Content))
		}
	}

	// 会话工具集（常驻 + 场景预注入；expanded 随循环演进）
	expanded := map[string]bool{}
	defs := s.toolDefsFor(ctx, scenePath, expanded, mode)

	// M42：同参数熔断计数（tool|归一化参数 → 连续失败次数）与成功调用数（后台学习判定）
	fuse := map[string]int{}
	toolOK := 0

	// 标准多轮 function calling 循环：
	// 每轮 GenerateContent → 若返回 tool_calls → 治理后执行 → 追加 AI+Tool 消息 → 重新生成。
	// 每轮重建 toolList（load_tools 可能已展开新模块）。
	// roundsExhausted：跑满轮次上限仍未产出最终回答（M60 之前是静默退出，长任务像「AI 自己断联」）
	roundsExhausted := true
	for round := 0; round < maxToolRounds; round++ {
		// M42：上下文压缩（超阈值时裁剪历史工具结果，保留最近 4 条完整）
		msgs = compressContext(msgs)
		resp, err := llm.GenerateContent(ctx, msgs,
			llms.WithTools(defsToLLMTools(defs)),
			// langchaingo v0.1.15 流式回调形态（openaiclient/chat.go）：
			// - reasoning_content 只经 StreamingReasoningFunc 透出，普通 StreamingFunc 永远收不到；
			// - content delta 是原始文本字节（非 JSON），tool_calls delta 是累积 JSON 数组（元素含 function 键）。
			llms.WithStreamingReasoningFunc(func(_ context.Context, reasoning, chunk []byte) error {
				if len(reasoning) > 0 {
					emit(map[string]string{"reasoning": string(reasoning)})
				}
				if len(chunk) == 0 {
					return nil
				}
				if chunk[0] == '[' {
					var probe []struct {
						Function *json.RawMessage `json:"function"`
					}
					if json.Unmarshal(chunk, &probe) == nil && len(probe) > 0 && probe[0].Function != nil {
						return nil
					}
				}
				emit(map[string]string{"content": string(chunk)})
				return nil
			}))
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			roundsExhausted = false
			break
		}
		choice := resp.Choices[0]
		// 无 tool_calls = 最终文本回答（已流式透出）
		if len(choice.ToolCalls) == 0 {
			roundsExhausted = false
			break
		}
		// 有 tool_calls：执行工具并追加 AI + Tool 消息对，继续循环
		aiParts := []llms.ContentPart{}
		if choice.Content != "" {
			aiParts = append(aiParts, llms.TextContent{Text: choice.Content})
		}
		for _, tc := range choice.ToolCalls {
			aiParts = append(aiParts, tc)
		}
		msgs = append(msgs, llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: aiParts})

		loadCalled := false
		for _, tc := range choice.ToolCalls {
			emit(map[string]any{"step": map[string]string{
				"type": "tool", "name": tc.FunctionCall.Name,
				"detail": fmt.Sprintf("执行 %s(%s)", tc.FunctionCall.Name, tc.FunctionCall.Arguments),
			}})
			def, known := findToolDef(defs, tc.FunctionCall.Name)
			var result string
			var terr error
			// M42：同参数熔断——同工具+同参数连续失败 2 次后直接短路，不再执行
			fuseKey := tc.FunctionCall.Name + "|" + strings.TrimSpace(tc.FunctionCall.Arguments)
			if fuse[fuseKey] >= 2 {
				result = "已熔断：该工具以完全相同的参数连续失败 2 次，已停止重试。请调整参数、补充前置步骤或改用其他方法。"
				emit(map[string]any{"step_result": map[string]string{
					"name": tc.FunctionCall.Name, "status": "熔断", "detail": "同参数连续失败，已短路", "output": "",
				}})
				msgs = append(msgs, llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{llms.ToolCallResponse{
						ToolCallID: tc.ID, Name: tc.FunctionCall.Name, Content: result,
					}},
				})
				continue
			}
			switch {
			case known && def.Name == "load_tools":
				result, terr = def.Fn(ctx, tc.FunctionCall.Arguments)
				loadCalled = true
			case known && def.Name == "ask_user":
				result, terr = s.runAskUser(ctx, emit, ping, tc.FunctionCall.Arguments)
			case known:
				result, terr = s.governToolCall(ctx, def, tc.FunctionCall.Arguments, emit, ping, mode)
			default:
				result, terr = "", fmt.Errorf("未找到工具 %s（可能未通过 load_tools 加载对应模块）", tc.FunctionCall.Name)
			}
			if terr != nil {
				// M60：RBAC 权限拒绝单列「无权限」（区别于普通失败，用户可行动：去角色管理开权限）
				if errors.Is(terr, errAIPermDenied) {
					result = terr.Error()
				} else {
					result = "工具执行失败: " + terr.Error()
				}
				fuse[fuseKey]++
			} else {
				toolOK++
				delete(fuse, fuseKey)
			}
			// 步骤结果事件：前端流程卡片据它把该卡片置为完成/失败并展示结果摘要
			status := "完成"
			if terr != nil {
				status = "失败"
				if errors.Is(terr, errAIPermDenied) {
					status = "无权限"
				}
			}
			// 卡片主行：可读摘要（列表渲染/键值平铺）；展开项：原始输出（截断），均已脱敏
			summary := truncText(strings.ReplaceAll(strings.TrimSpace(maskSensitive(toolSummary(def, result))), "\n", " "), 420)
			output := truncText(maskSensitive(result), toolOutputTrunc)
			emit(map[string]any{"step_result": map[string]string{
				"name": tc.FunctionCall.Name, "status": status, "detail": summary, "output": output,
			}})
			msgs = append(msgs, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: tc.ID,
					Name:       tc.FunctionCall.Name,
					Content:    result,
				}},
			})
		}
		// load_tools 展开了新模块 → 下一轮注入对应工具集
		if loadCalled {
			defs = s.toolDefsFor(ctx, scenePath, expanded, mode)
		}
	}
	// M60：轮次跑满仍未收尾 → 明确告知用户进度与继续方式（此前静默退出，像「AI 断联」）
	if roundsExhausted {
		emit(map[string]string{"content": fmt.Sprintf(
			"\n\n> ⚠️ 本次任务已达单次工具调用轮次上限（%d 轮），已暂停。上面的进度是当前已完成的部分；回复「继续」我会接着完成剩余步骤。",
			maxToolRounds,
		)})
	}
	emit(map[string]string{"done": "1"})
	return nil
}

// compressContext 上下文压缩（M42）：文本体量超阈值时，把保留窗口（最近 keep 条）之外的
// 工具结果替换为「[已压缩] + 首 200 字符」。纯确定性裁剪，不调第二模型。
func compressContext(msgs []llms.MessageContent, keep ...int) []llms.MessageContent {
	const budget = 120_000
	const keepN = 4
	total := 0
	for i := range msgs {
		for _, part := range msgs[i].Parts {
			if tc, ok := part.(llms.ToolCallResponse); ok {
				total += len(tc.Content)
			} else if tc, ok := part.(llms.TextContent); ok {
				total += len(tc.Text)
			}
		}
	}
	if total <= budget {
		return msgs
	}
	// 从头压缩 Tool 结果（保留最近 keepN 条不动）
	toolIdx := []int{}
	for i := range msgs {
		if msgs[i].Role == llms.ChatMessageTypeTool {
			toolIdx = append(toolIdx, i)
		}
	}
	cut := len(toolIdx) - keepN
	if cut <= 0 {
		return msgs
	}
	for _, i := range toolIdx[:cut] {
		for pi, part := range msgs[i].Parts {
			if tc, ok := part.(llms.ToolCallResponse); ok && len(tc.Content) > 240 {
				tc.Content = "[已压缩] " + truncText(tc.Content, 200)
				msgs[i].Parts[pi] = tc
			}
		}
	}
	return msgs
}

// maskSensitive 工具结果摘要脱敏：JSON 中密码类键值替换为 ******（模型上下文保留真实值，界面步骤卡片与审计不留明文）。
func maskSensitive(text string) string {
	trimmed := strings.TrimSpace(text)
	var probe map[string]any
	if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
		return text
	}
	maskSensitiveMap(probe)
	b, err := json.Marshal(probe)
	if err != nil {
		return text
	}
	return string(b)
}

func maskSensitiveMap(m map[string]any) {
	for k, v := range m {
		switch sub := v.(type) {
		case string:
			if sensitiveKey(k) {
				m[k] = "******"
			}
		case map[string]any:
			maskSensitiveMap(sub)
		case []any:
			for _, item := range sub {
				if sm, ok := item.(map[string]any); ok {
					maskSensitiveMap(sm)
				}
			}
		}
	}
}

func sensitiveKey(k string) bool {
	kl := strings.ToLower(k)
	return strings.Contains(kl, "password") || strings.Contains(kl, "passwd") ||
		strings.Contains(kl, "secret") || strings.Contains(kl, "token") || strings.Contains(kl, "pwd")
}

// runAskUser 交互提问：emit ask_user 事件挂起等待用户回答（无超时，随会话中断取消）。
func (s *AIService) runAskUser(ctx context.Context, emit func(any), ping func(), args string) (string, error) {
	var payload struct {
		Questions []map[string]any `json:"questions"`
	}
	if err := json.Unmarshal([]byte(unwrapArgs(args)), &payload); err != nil || len(payload.Questions) == 0 {
		return "", fmt.Errorf("缺少 questions 参数（1-4 个问题）")
	}
	if len(payload.Questions) > 4 {
		payload.Questions = payload.Questions[:4]
	}
	id := randAskID()
	req := &aiQuestionRequest{ch: make(chan string, 1)}
	s.pendingQuestions.Store(id, req)
	defer s.pendingQuestions.Delete(id)
	slog.Info("ai ask-user waiting", "id", id)
	emit(map[string]any{"ask_user": map[string]any{
		"id": id, "questions": payload.Questions,
	}})
	hb := time.NewTicker(15 * time.Second)
	defer hb.Stop()
	for {
		select {
		case answer := <-req.ch:
			slog.Info("ai ask-user answered", "id", id, "answer", truncText(answer, 200))
			return answer, nil
		case <-ctx.Done():
			slog.Info("ai ask-user cancelled", "id", id)
			emit(map[string]any{"ask_user_result": map[string]any{"id": id, "cancelled": true}})
			return "用户没有回答（会话已中断）。请基于已有信息继续，或稍后重新询问。", nil
		case <-hb.C:
			ping()
		}
	}
}

// governToolCall 权限门禁统一出口（M60 四档）：
// RBAC 预检（无权限 fail-fast）；write/danger 按 confirm/auto_edit/full 档位决定是否 ask 用户
// （一次性审批，fail-closed）；工具执行期 15s 心跳保活；结果与 ask 结论全部写操作审计。
func (s *AIService) governToolCall(
	ctx context.Context,
	def aiToolDef,
	args string,
	emit func(any),
	ping func(),
	mode string,
) (string, error) {
	// M48：危险操作锁定——write/danger 级工具一律拒绝（锁定期内 AI 不得执行恢复/回滚/重装/清理类操作）
	if def.Risk != aiRiskRead && s.settings != nil {
		var sec SecuritySettingsService
		sec.settings = s.settings
		if err := sec.CheckDangerLock("POST", "/ai/tool/"+def.Name); err != nil {
			emit(map[string]any{"step_result": map[string]string{
				"name": def.Name, "status": "锁定", "detail": "危险操作锁定已开启", "output": "",
			}})
			return "", err
		}
	}
	// M60：RBAC 工具权限预检——无权限显性标「无权限」而不是混进普通失败
	// （工具注入期已预过滤无权限工具，此处兜底节点范围/权限点校验；权限档位不影响 RBAC 授权）
	if err := rbac.CheckTool(ctx, def.Module, def.Risk, args); err != nil {
		s.auditOperation(def, args, "", err.Error(), false, "denied")
		return "", fmt.Errorf("%w：%v。请联系管理员在「角色管理」开通对应权限；聊天权限档位不影响此限制", errAIPermDenied, err)
	}
	// 权限档位（M60）：full 全自动；auto_edit 写自动、危险仍确认；confirm（默认）写+危险都确认
	needAsk := def.Risk == aiRiskDanger || def.Risk == aiRiskWrite
	if mode == aiModeFull {
		needAsk = false
	} else if mode == aiModeAutoEdit {
		needAsk = def.Risk == aiRiskDanger
	}
	action := "executed"
	if needAsk {
		id := randAskID()
		req := &aiAskRequest{ch: make(chan bool, 1)}
		s.pendingAsks.Store(id, req)
		emit(map[string]any{"ask": map[string]any{
			"id": id, "tool": def.Name, "module": def.Module, "risk": def.Risk,
			"args": truncText(args, 2000),
		}})
		slog.Info("ai ask emitted", "tool", def.Name, "id", id, "mode", mode)
		approved := false
		timedOut := false
		timer := time.NewTimer(aiAskTimeout)
		heartbeat := time.NewTicker(15 * time.Second)
		defer timer.Stop()
		defer heartbeat.Stop()
	wait:
		for {
			select {
			case v := <-req.ch:
				approved = v
				break wait
			case <-timer.C:
				timedOut = true
				break wait
			case <-heartbeat.C:
				ping()
			case <-ctx.Done():
				break wait
			}
		}
		s.pendingAsks.Delete(id)
		switch {
		case timedOut:
			emit(map[string]any{"ask_result": map[string]any{"id": id, "approved": false, "reason": "timeout"}})
			s.auditOperation(def, args, "", "确认超时（120 秒未响应），操作已取消", false, "timeout")
			return "操作未执行：用户确认超时。请向用户说明情况并询问是否重试，不要擅自重试。", nil
		case !approved:
			emit(map[string]any{"ask_result": map[string]any{"id": id, "approved": false, "reason": "denied"}})
			s.auditOperation(def, args, "", "用户拒绝了该操作", false, "denied")
			return "操作未执行：用户拒绝了该操作。请说明后续方案并询问用户意愿。", nil
		default:
			emit(map[string]any{"ask_result": map[string]any{"id": id, "approved": true, "reason": "approved"}})
			action = "approved"
		}
	}
	// M60：工具执行期心跳——长命令/安装类工具执行期间 SSE 可能因空闲被中间层掐断，
	// 后台协程 15s 一帧 ping；退出前 stop+join，保证不在 handler 返回后写连接。
	stopPing := make(chan struct{})
	pingDone := make(chan struct{})
	go func() {
		defer close(pingDone)
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-t.C:
				ping()
			}
		}
	}()
	result, err := def.Fn(ctx, args)
	close(stopPing)
	<-pingDone
	if err != nil {
		s.auditOperation(def, args, "", err.Error(), false, action)
	} else {
		s.auditOperation(def, args, result, "", true, action)
	}
	return result, err
}
