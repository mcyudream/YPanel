// AI 对话（B18 v2 终版）：native function calling 工具循环 + 场景感知 + 知识库注入 + SSE。
// 不使用 ReAct 文本协议（GLM 等模型遵循差，会污染正文）；工具调用走模型原生 tool_calls。
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
// 事件：data: {"scene":{...}} / {"tool":{...}} / {"content":"..."} / [DONE]
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

	// system：提示词 + 知识库检索 + 场景摘要
	sys := aiSystemPrompt
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

	// 组装消息
	msgs := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, sys)}
	for _, m := range history {
		if m.Role == "user" {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeHuman, m.Content))
		} else {
			msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeAI, m.Content))
		}
	}

	// 原生 function calling 工具循环
	toolDefs := s.toolsFor(ctx)
	toolList := make([]llms.Tool, len(toolDefs))
	for i, t := range toolDefs {
		toolList[i] = llms.Tool{Type: "function", Function: &llms.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
		}}
	}

	for round := 0; round < maxToolRounds; round++ {
		resp, err := llm.GenerateContent(ctx, msgs,
			llms.WithTools(toolList),
			llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				// 仅最终轮文本从这里出（工具调用轮的 delta 是 tool_call 片段，非文本）
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
		if len(choice.ToolCalls) == 0 {
			break
		}
		// 执行工具并把结果回填
		parts := []llms.ContentPart{}
		if choice.Content != "" {
			parts = append(parts, llms.TextContent{Text: choice.Content})
		}
		for _, tc := range choice.ToolCalls {
			parts = append(parts, tc)
		}
		msgs = append(msgs, llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: parts})
		for _, tc := range choice.ToolCalls {
			emitJSON(map[string]any{"tool": map[string]string{"name": tc.FunctionCall.Name, "input": tc.FunctionCall.Arguments}})
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
				Role:  llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{Name: tc.FunctionCall.Name, Content: result}},
			})
		}
	}
	emitJSON(map[string]string{"done": "1"})
	flusher.Flush()
	return nil
}
