package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/ypanel/core/internal/model"
)

type aiHTTPDoer struct {
	client         *http.Client
	origin         *url.URL
	allowPrivate   bool
	model          string
	protocol       string
	temperature    *float64
	maxTokensField string
}

type aiCheckedTransport struct {
	owner *aiHTTPDoer
	next  http.RoundTripper
}

func (t aiCheckedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.owner.validate(req); err != nil {
		return nil, err
	}
	if err := t.owner.normalizeModelRequest(req); err != nil {
		return nil, err
	}
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	body := &aiTimedBody{ReadCloser: resp.Body, remaining: 32 << 20}
	body.timer = time.AfterFunc(120*time.Second, func() { _ = resp.Body.Close() })
	resp.Body = body
	return resp, nil
}

// SDK defaults cannot express an omitted temperature for every compatible model.
// Normalize only model requests; MCP and model-list HTTP bodies are untouched.
func (d *aiHTTPDoer) normalizeModelRequest(req *http.Request) error {
	if d.model == "" || req.Method != "POST" || req.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, (128<<20)+1))
	_ = req.Body.Close()
	if err != nil {
		return err
	}
	if len(raw) > 128<<20 {
		return errWrapAI("模型请求超过 128MB，请减少图片数量")
	}
	var payload map[string]any
	if err = json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	delete(payload, "temperature")
	if d.temperature != nil {
		payload["temperature"] = *d.temperature
	}
	if d.protocol == "openai" || d.protocol == "" {
		field := d.maxTokensField
		if field == "" {
			field = "max_tokens"
			if strings.EqualFold(d.origin.Hostname(), "api.openai.com") {
				field = "max_completion_tokens"
			}
		}
		value := payload["max_completion_tokens"]
		if value == nil {
			value = payload["max_tokens"]
		}
		delete(payload, "max_tokens")
		delete(payload, "max_completion_tokens")
		if value != nil {
			payload[field] = value
		}
	}
	raw, err = json.Marshal(payload)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	req.ContentLength = int64(len(raw))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(raw)), nil }
	return nil
}

func aiPublicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return false
	}
	for _, cidr := range []string{"100.64.0.0/10", "198.18.0.0/15", "192.0.0.0/24", "192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "64:ff9b::/96"} {
		if netip.MustParsePrefix(cidr).Contains(ip) {
			return false
		}
	}
	return true
}
func aiEndpoint(p *model.AIProvider) string {
	if p.BaseURL != "" {
		return strings.TrimRight(p.BaseURL, "/")
	}
	if p.APIType == "anthropic" {
		return "https://api.anthropic.com"
	}
	return "https://api.openai.com/v1"
}
func newAIHTTP(p *model.AIProvider) (*aiHTTPDoer, error) {
	origin, err := url.Parse(aiEndpoint(p))
	if err != nil || origin.Hostname() == "" || origin.User != nil || origin.Fragment != "" || origin.RawQuery != "" || (origin.Scheme != "https" && origin.Scheme != "http") {
		return nil, errWrapAI("AI 端点需为有效的 HTTP(S) 地址，不能含凭据、查询串或片段")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 45 * time.Second
	transport.TLSHandshakeTimeout = 15 * time.Second
	transport.IdleConnTimeout = 60 * time.Second
	transport.MaxIdleConnsPerHost = 4
	dialer := net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		// 环境代理由部署者配置，目标 URL 仍在 Do 中逐跳复核；直连时固定本次解析的 IP。
		if !strings.EqualFold(host, origin.Hostname()) {
			return dialer.DialContext(ctx, network, address)
		}
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !p.AllowPrivate && !aiPublicIP(ip) {
				return nil, errWrapAI("端点解析到非公网地址，请由管理员显式启用该端点的私网访问")
			}
		}
		var last error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		return nil, last
	}
	doer := &aiHTTPDoer{origin: origin, allowPrivate: p.AllowPrivate}
	doer.client = &http.Client{Transport: aiCheckedTransport{owner: doer, next: transport}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errWrapAI("AI 端点重定向过多")
		}
		return doer.validate(req)
	}}
	return doer, nil
}
func (d *aiHTTPDoer) validate(req *http.Request) error {
	if req.URL.User != nil || req.URL.Scheme != d.origin.Scheme || !strings.EqualFold(req.URL.Host, d.origin.Host) {
		return errWrapAI("拒绝 AI 凭据跨端点重定向")
	}
	if d.allowPrivate {
		return nil
	}
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", req.URL.Hostname())
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return errWrapAI("AI 端点未解析到地址")
	}
	for _, ip := range ips {
		if !aiPublicIP(ip) {
			return errWrapAI("AI 端点不允许访问非公网地址；请由管理员显式配置私网许可")
		}
	}
	return nil
}
func (d *aiHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return d.client.Do(req)
}

type aiTimedBody struct {
	io.ReadCloser
	timer     *time.Timer
	remaining int
}

func (r *aiTimedBody) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, fmt.Errorf("AI 响应超过 32MB 上限")
	}
	n, err := r.ReadCloser.Read(p)
	r.remaining -= n
	r.timer.Reset(120 * time.Second)
	return n, err
}
func (r *aiTimedBody) Close() error { r.timer.Stop(); return r.ReadCloser.Close() }

func buildLLM(p *model.AIProvider, settings ...AIChatSettings) (llms.Model, error) {
	if strings.TrimSpace(p.Model) == "" {
		return nil, errWrapAI("尚未配置模型")
	}
	client, err := newAIHTTP(p)
	if err != nil {
		return nil, err
	}
	client.model = p.Model
	client.protocol = p.APIType
	client.maxTokensField = modelCapabilities(p).MaxTokensField
	if len(settings) > 0 {
		client.temperature = settings[0].Temperature
	}
	switch p.APIType {
	case "anthropic":
		endpoint := aiEndpoint(p)
		if !strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/v1"
		}
		return anthropic.New(anthropic.WithToken(p.APIKey), anthropic.WithModel(p.Model), anthropic.WithBaseURL(endpoint), anthropic.WithHTTPClient(client))
	case "response", "responses":
		return &aiResponsesModel{provider: *p, http: client}, nil
	case "openai", "":
		return openai.New(openai.WithToken(p.APIKey), openai.WithModel(p.Model), openai.WithBaseURL(aiEndpoint(p)), openai.WithHTTPClient(client))
	default:
		return nil, errWrapAI("不支持的 AI 协议类型")
	}
}

// Responses 适配复用项目 llms.Model 接口，保留完整 output Items 供工具轮次原样回传。
type aiResponsesModel struct {
	provider model.AIProvider
	http     *aiHTTPDoer
}

func (m *aiResponsesModel) Call(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, m, prompt, opts...)
}

const responsesItemPrefix = "responses-items:"

func responsesInput(messages []llms.MessageContent) ([]any, string, error) {
	input := []any{}
	system := []string{}
	for _, msg := range messages {
		text := []any{}
		flush := func() {
			if len(text) > 0 {
				role := "user"
				if msg.Role == llms.ChatMessageTypeAI {
					role = "assistant"
				}
				input = append(input, map[string]any{"role": role, "content": text})
				text = nil
			}
		}
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llms.TextContent:
				if msg.Role == llms.ChatMessageTypeSystem {
					system = append(system, p.Text)
					continue
				}
				typ := "input_text"
				if msg.Role == llms.ChatMessageTypeAI {
					typ = "output_text"
				}
				text = append(text, map[string]any{"type": typ, "text": p.Text})
			case llms.BinaryContent:
				text = append(text, map[string]any{"type": "input_image", "image_url": p.String()})
			case llms.ImageURLContent:
				text = append(text, map[string]any{"type": "input_image", "image_url": p.URL})
			case llms.ToolCall:
				flush()
				if p.FunctionCall != nil {
					input = append(input, map[string]any{"type": "function_call", "call_id": p.ID, "name": p.FunctionCall.Name, "arguments": p.FunctionCall.Arguments})
				}
			case llms.ToolCallResponse:
				flush()
				input = append(input, map[string]any{"type": "function_call_output", "call_id": p.ToolCallID, "output": p.Content})
			case llms.ThinkingContent:
				if strings.HasPrefix(p.Signature, responsesItemPrefix) {
					flush()
					raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(p.Signature, responsesItemPrefix))
					if err != nil {
						return nil, "", err
					}
					var items []any
					if err := json.Unmarshal(raw, &items); err != nil {
						return nil, "", err
					}
					input = append(input, items...)
				}
			}
		}
		flush()
	}
	return input, strings.Join(system, "\n\n"), nil
}

type aiResponseItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
type aiResponseResult struct {
	Status string          `json:"status"`
	Output json.RawMessage `json:"output"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		Input  int `json:"input_tokens"`
		Output int `json:"output_tokens"`
	} `json:"usage"`
}

func responseChoice(result aiResponseResult) (*llms.ContentResponse, error) {
	if result.Error != nil {
		return nil, errWrapAI(result.Error.Message)
	}
	var items []aiResponseItem
	if err := json.Unmarshal(result.Output, &items); err != nil {
		return nil, err
	}
	choice := &llms.ContentChoice{StopReason: result.Status, GenerationInfo: map[string]any{"InputTokens": result.Usage.Input, "OutputTokens": result.Usage.Output}}
	for _, item := range items {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" {
					choice.Content += part.Text
				}
			}
		case "function_call":
			choice.ToolCalls = append(choice.ToolCalls, llms.ToolCall{ID: item.CallID, Type: "function", FunctionCall: &llms.FunctionCall{Name: item.Name, Arguments: item.Arguments}})
		}
	}
	if result.Status == "incomplete" {
		choice.StopReason = "length"
	}
	choice.Parts = []llms.ContentPart{llms.ThinkingContent{Thinking: choice.Content, Signature: responsesItemPrefix + base64.StdEncoding.EncodeToString(result.Output)}}
	return &llms.ContentResponse{Choices: []*llms.ContentChoice{choice}}, nil
}
func (m *aiResponsesModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	cfg := llms.CallOptions{}
	for _, option := range options {
		option(&cfg)
	}
	input, instructions, err := responsesInput(messages)
	if err != nil {
		return nil, err
	}
	stream := cfg.StreamingFunc != nil || cfg.StreamingReasoningFunc != nil
	payload := map[string]any{"model": m.provider.Model, "input": input, "instructions": instructions, "store": false, "stream": stream, "include": []string{"reasoning.encrypted_content"}}
	if cfg.MaxTokens > 0 {
		payload["max_output_tokens"] = cfg.MaxTokens
	}
	if cfg.Temperature != 0 {
		payload["temperature"] = cfg.Temperature
	}
	if len(cfg.Tools) > 0 {
		tools := []any{}
		for _, tool := range cfg.Tools {
			if tool.Function != nil {
				f := tool.Function
				tools = append(tools, map[string]any{"type": "function", "name": f.Name, "description": f.Description, "parameters": f.Parameters, "strict": false})
			}
		}
		payload["tools"] = tools
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, aiEndpoint(&m.provider)+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.provider.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, errWrapAI(fmt.Sprintf("模型 HTTP %d：%s", resp.StatusCode, maskSensitive(string(raw))))
	}
	emit := func(reasoning, text string) error {
		if cfg.StreamingReasoningFunc != nil {
			return cfg.StreamingReasoningFunc(ctx, []byte(reasoning), []byte(text))
		}
		if cfg.StreamingFunc != nil && text != "" {
			return cfg.StreamingFunc(ctx, []byte(text))
		}
		return nil
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
		var result aiResponseResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		out, err := responseChoice(result)
		if err == nil {
			err = emit("", out.Choices[0].Content)
		}
		return out, err
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	var data strings.Builder
	var final *llms.ContentResponse
	handle := func() error {
		if data.Len() == 0 {
			return nil
		}
		raw := data.String()
		data.Reset()
		var ev struct {
			Type     string           `json:"type"`
			Delta    string           `json:"delta"`
			Message  string           `json:"message"`
			Response aiResponseResult `json:"response"`
		}
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			return err
		}
		switch ev.Type {
		case "response.output_text.delta":
			return emit("", ev.Delta)
		case "response.reasoning_summary_text.delta":
			return emit(ev.Delta, "")
		case "response.completed", "response.incomplete":
			out, err := responseChoice(ev.Response)
			final = out
			return err
		case "response.failed":
			if ev.Response.Error != nil {
				return errWrapAI(ev.Response.Error.Message)
			}
			return errWrapAI("Responses 生成失败")
		case "error":
			return errWrapAI(ev.Message)
		}
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := handle(); err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := handle(); err != nil {
		return nil, err
	}
	if final == nil {
		return nil, errWrapAI("Responses 流未正常结束，已保留生成内容")
	}
	return final, nil
}
func mcpSafeToolName(server, tool string) string {
	sum := sha256.Sum256([]byte(server + "\x00" + tool))
	return "mcp_" + hex.EncodeToString(sum[:12])
}
