// AI 对话（B18 v4）：标准 function calling 工具循环（多轮 tool 消息对）+ SSE。
// GLM/DeepSeek/OpenAI 兼容层均原生支持 tool_calls + tool_call_id 格式；
// reasoning_content（思考过程）经 llms.WithStreamingReasoningFunc 透出（deepseek-flash 等思考型模型）。
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
	"优先调用提供的工具获取面板实时数据再回答；容器启停等破坏性操作前先向用户确认。" +
	"回答使用 Markdown 格式。"

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

// StreamAgentChat 标准 function calling 工具循环 + SSE 流式输出。
// 事件：data: {"scene":{...}} / {"step":{...}} / {"reasoning":"..."} / {"content":"..."} / [DONE]
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

	// 消息链（标准 function calling 多轮格式）
	msgs := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, sys)}
	for _, m := range history {
		if m.Role == "user" {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeHuman, m.Content))
		} else {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeAI, m.Content))
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

	// 标准多轮 function calling 循环：
	// 每轮 GenerateContent → 如果返回 tool_calls → 执行 → 追加 AI+Tool 消息 → 重新生成
	for round := 0; round < maxToolRounds; round++ {
		resp, err := llm.GenerateContent(ctx, msgs,
			llms.WithTools(toolList),
			// langchaingo v0.1.15 流式回调形态（openaiclient/chat.go）：
			// - reasoning_content 只经 StreamingReasoningFunc 透出，普通 StreamingFunc 永远收不到；
			// - content delta 是原始文本字节（非 JSON），tool_calls delta 是累积 JSON 数组（元素含 function 键）。
			llms.WithStreamingReasoningFunc(func(_ context.Context, reasoning, chunk []byte) error {
				if len(reasoning) > 0 {
					emitJSON(map[string]string{"reasoning": string(reasoning)})
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
				emitContent(string(chunk))
				return nil
			}))
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			break
		}
		choice := resp.Choices[0]
		// 无 tool_calls = 最终文本回答（已流式透出）
		if len(choice.ToolCalls) == 0 {
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
			msgs = append(msgs, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: tc.ID,
					Name:       tc.FunctionCall.Name,
					Content:    result,
				}},
			})
		}
	}
	emitJSON(map[string]string{"done": "1"})
	flusher.Flush()
	return nil
}
