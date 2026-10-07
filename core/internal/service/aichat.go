// AI 对话（B18 v3）：工具循环（上下文重注入模式）+ 场景感知 + 知识库/技能注入 + SSE。
// 每轮独立生成，工具结果并入 system 重新生成；无 tool_calls 的纯文本即最终回答。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/ypanel/core/internal/model"
)

const maxToolRounds = 6

const aiSystemPrompt = "你是 YPanel 服务器管理面板的运维助手。用简洁中文回答服务器运维问题。" +
	"优先调用提供的工具获取面板实时数据再回答；容器启停等破坏性操作前先向用户确认。"

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

// StreamAgentChat 工具循环 + SSE 输出。
// 事件：data: {"scene":{...}} / {"step":{...}} / {"content":"..."} / [DONE]
func (s *AIService) StreamAgentChat(
	ctx context.Context,
	w http.ResponseWriter,
	provider *model.AIProvider,
	history []ChatMessage,
	scenePath string,
) error {
	llm, err := buildLLM(provider)
	if err != nil {
		return err
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return errWrapAI("当前连接不支持流式")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	emitJSON := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}
	emitContent := func(text string) {
		emitJSON(map[string]string{"content": text})
	}

	// 场景感知
	if scene := s.SceneContext(ctx, scenePath); scene != nil {
		emitJSON(map[string]any{"scene": scene})
	}

	// system：提示词 + 技能注入 + 知识库检索 + 场景摘要
	sys := aiSystemPrompt
	if sk := s.skills.EnabledBodies(ctx); len(sk) > 0 {
		sys += "\n\n可用技能（用户提问匹配技能用途时，按技能正文执行）：\n" + strings.Join(sk, "\n---\n")
	}
	if ks := s.searchKnowledge(strings.Join(func() []string {
		msgs := []string{scenePath}
		for _, m := range history {
			msgs = append(msgs, m.Content)
		}
		return msgs
	}(), " ")); len(ks) > 0 {
		var parts []string
		for _, item := range ks {
			parts = append(parts, "["+item.Title+"]"+item.Body)
		}
		sys += "\n\n相关知识库条目：\n" + strings.Join(parts, "\n---\n")
	}
	sys += "\n\n" + s.buildSceneSummary(ctx, scenePath)

	// 基础消息（system + 历史）
	baseMsgs := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, sys)}
	for _, m := range history {
		if m.Role == "user" {
			baseMsgs = append(baseMsgs, llms.TextParts(llms.ChatMessageTypeHuman, m.Content))
		} else {
			baseMsgs = append(baseMsgs, llms.TextParts(llms.ChatMessageTypeAI, m.Content))
		}
	}

	// 工具定义（面板工具 + MCP 工具）
	toolDefs := s.toolsFor(ctx)
	for _, mt := range s.enabledMCPTools(ctx) {
		mt := mt
		toolDefs = append(toolDefs, &aiTool{
			name:        mt.Name,
			description: mt.Descr + "（来自 MCP 服务器 " + mt.Server + "）",
			fn: func(_ context.Context, input string) (string, error) {
				return s.callMCPTool(ctx, mt.Name, input)
			},
		})
	}
	toolList := make([]llms.Tool, len(toolDefs))
	for i, t := range toolDefs {
		toolList[i] = llms.Tool{Type: "function", Function: &llms.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
		}}
	}

	// 工具循环：每轮独立生成，工具结果摘要并入 system 重新生成
	var toolResults []string
	for round := 0; round < maxToolRounds; round++ {
		roundSys := sys
		if round > 0 && len(toolResults) > 0 {
			roundSys += "\n\n[工具执行结果]\n" + strings.Join(toolResults, "\n")
		}
		roundMsgs := append([]llms.MessageContent{}, baseMsgs...)
		roundMsgs[0] = llms.TextParts(llms.ChatMessageTypeSystem, roundSys)

		resp, err := llm.GenerateContent(ctx, roundMsgs,
			llms.WithTools(toolList))
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			break
		}
		choice := resp.Choices[0]
		// 纯文本回答 = 最终结果
		if len(choice.ToolCalls) == 0 {
			if choice.Content != "" {
				emitContent(choice.Content)
			}
			break
		}
		// 有 tool_calls：执行工具，结果并入下轮上下文
		for _, tc := range choice.ToolCalls {
			emitJSON(map[string]any{"step": map[string]string{
				"type": "tool", "name": tc.FunctionCall.Name,
				"detail": fmt.Sprintf("执行 %s(%s)", tc.FunctionCall.Name, tc.FunctionCall.Arguments),
			}})
			result := "未找到工具实现"
			var terr error
			for _, t := range toolDefs {
				if t.Name() == tc.FunctionCall.Name {
					result, terr = t.Call(ctx, tc.FunctionCall.Arguments)
					if terr != nil {
						result = "工具执行失败: " + terr.Error()
					}
					break
				}
			}
			toolResults = append(toolResults,
				fmt.Sprintf("%s(%s) = %s", tc.FunctionCall.Name, tc.FunctionCall.Arguments, result))
		}
	}
	emitJSON(map[string]string{"done": "1"})
	flusher.Flush()
	return nil
}
