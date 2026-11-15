package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"github.com/ypanel/core/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIProtocolAdapters(t *testing.T) {
	for _, protocol := range []string{"openai", "anthropic", "responses"} {
		t.Run(protocol, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				var body map[string]any
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Error(err)
				}
				if !strings.Contains(string(raw), "image") {
					t.Error("multimodal image parts missing")
				}
				if body["model"] != "mock-model" {
					t.Error("model override ignored")
				}
				if _, exists := body["temperature"]; exists {
					t.Error("blank sampling setting must use provider default")
				}
				if protocol == "anthropic" {
					if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "test-only" || r.Header.Get("Authorization") != "" {
						t.Error("wrong Anthropic endpoint/auth")
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"id":"m","type":"message","role":"assistant","model":"mock-model","content":[{"type":"text","text":"native anthropic"}],"stop_reason":"end_turn","usage":{"input_tokens":12,"output_tokens":3}}`)
				} else if protocol == "openai" {
					if body["max_tokens"] != float64(128) || body["max_completion_tokens"] != nil {
						t.Error("compatible token field was not normalized")
					}
					if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-only" {
						t.Error("wrong OpenAI endpoint/auth")
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"native openai"},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`)
				} else {
					if r.URL.Path != "/responses" || body["store"] != false {
						t.Error("invalid Responses request")
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"id":"r","status":"completed","output":[{"type":"reasoning","id":"rs","encrypted_content":"opaque-signed-data","summary":[]},{"type":"function_call","call_id":"call-1","name":"inspect","arguments":"{}"}],"usage":{"input_tokens":12,"output_tokens":3}}`)
				}
			}))
			defer server.Close()
			llm, err := buildLLM(&model.AIProvider{APIType: protocol, BaseURL: server.URL, APIKey: "test-only", Model: "mock-model", AllowPrivate: true})
			if err != nil {
				t.Fatal(err)
			}
			result, err := llm.GenerateContent(context.Background(), []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, "policy"), llms.MessageContent{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart("hi"), llms.BinaryPart("image/png", []byte{1, 2, 3})}}}, llms.WithMaxTokens(128))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Choices) != 1 {
				t.Fatal("missing native answer")
			}
			if protocol == "responses" {
				choice := result.Choices[0]
				if len(choice.ToolCalls) != 1 || choice.ToolCalls[0].ID != "call-1" {
					t.Fatal("tool call id lost")
				}
				messages := []llms.MessageContent{{Role: llms.ChatMessageTypeAI, Parts: choice.Parts}, {Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: "call-1", Content: "ok"}}}}
				input, _, err := responsesInput(messages)
				raw, _ := json.Marshal(input)
				if err != nil || !strings.Contains(string(raw), "opaque-signed-data") || !strings.Contains(string(raw), "function_call_output") {
					t.Fatalf("opaque output replay lost: %s %v", raw, err)
				}
			} else if !strings.Contains(result.Choices[0].Content, "native "+protocol) {
				t.Fatal("wrong protocol content")
			}
		})
	}
}

func TestAIAutoCompactPreservesHistory(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"s","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"目标、约束、完成与待办"}]}]}`)
	}))
	defer server.Close()
	p := model.AIProvider{Name: "summary", APIType: "responses", Model: "mock", BaseURL: server.URL, AllowPrivate: true}
	s.SaveProvider(&p)
	history := []ChatMessage{}
	for i := 0; i < 20; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		history = append(history, ChatMessage{Role: role, Content: strings.Repeat("constraint ", 100)})
	}
	id, _ := s.SaveConversation(ctx, 0, "auto", history)
	runCtx := context.WithValue(ctx, aiRunContextKey{}, aiRunIdentity{UserID: 1, ConversationID: id, RunID: "summary"})
	cfg := defaultAISettings()
	cfg.ContextLimit = 2048
	cfg.MaxOutputTokens = 128
	cfg.RetainTurns = 2
	var compacted bool
	msgs, usage, err := s.prepareChatContext(runCtx, &p, cfg, history, "policy", nil, func(ev any) {
		raw, _ := json.Marshal(ev)
		if strings.Contains(string(raw), `"compacting":true`) {
			compacted = true
		}
	})
	if err != nil || !compacted || usage.Summary == "" || usage.SummaryThrough != 16 {
		t.Fatalf("auto compaction missing %+v %v", usage, err)
	}
	if err := fitAIToolResults(msgs, cfg.ContextLimit-cfg.MaxOutputTokens); err != nil {
		t.Fatal(err)
	}
	row, _ := s.ownedConversation(ctx, id)
	full, _ := s.conversationMessages(ctx, row)
	if len(full) != 20 || full[0].Content != history[0].Content || row.SummaryThrough != 16 {
		t.Fatal("auto summary modified originals")
	}
}

func TestAIResponsesStreamAndSummary(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, ev := range []string{`{"type":"response.output_text.delta","delta":"stream answer"}`, `{"type":"response.completed","response":{"id":"r","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"stream answer"}]}],"usage":{"input_tokens":20,"output_tokens":4}}}`} {
				fmt.Fprintf(w, "data: %s\n\n", ev)
			}
		} else {
			if body["tools"] != nil {
				t.Error("summary included tools")
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"summary","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"目标与约束，完成和待办"}]}],"usage":{"input_tokens":20,"output_tokens":4}}`)
		}
	}))
	defer server.Close()
	p := model.AIProvider{Name: "responses", APIType: "responses", BaseURL: server.URL, APIKey: "test", Model: "mock", AllowPrivate: true}
	if err := s.SaveProvider(&p); err != nil {
		t.Fatal(err)
	}
	llm, _ := buildLLM(&p)
	var streamed strings.Builder
	result, err := llm.GenerateContent(ctx, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "hi")}, llms.WithStreamingFunc(func(ctx context.Context, b []byte) error { streamed.Write(b); return nil }))
	if err != nil || streamed.String() != "stream answer" || result.Choices[0].GenerationInfo["InputTokens"] != 20 {
		t.Fatalf("bad stream: %s %v", streamed.String(), err)
	}
	history := []ChatMessage{}
	for i := 0; i < 20; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		history = append(history, ChatMessage{Role: role, Content: fmt.Sprintf("original-%d", i)})
	}
	id, _ := s.SaveConversation(ctx, 0, "summary", history)
	cfg := defaultAISettings()
	cfg.ProviderID = p.ID
	cfg.RetainTurns = 2
	if err := s.UpdateConversation(ctx, id, nil, nil, nil, &cfg); err != nil {
		t.Fatal(err)
	}
	if err := s.CompactConversation(ctx, id); err != nil {
		t.Fatal(err)
	}
	row, _ := s.ownedConversation(ctx, id)
	full, _ := s.conversationMessages(ctx, row)
	if row.Summary == "" || row.SummaryThrough != 16 || len(full) != 20 || full[0].Content != "original-0" {
		t.Fatal("summary corrupted canonical history")
	}
}
