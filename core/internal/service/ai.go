// AIService AI 运维助手（B18）：多供应商（内置预设 + 自定义）、三种 API 端点类型
// （openai chat completions / anthropic messages / openai response）、SSE 流式透传。
package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
)

// AIService AI 助手。
type AIService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewAIService 创建。
func NewAIService(db *gorm.DB, nodes *NodeService) *AIService {
	return &AIService{db: db, nodes: nodes}
}

// AIProvider 供应商配置。
type AIProvider struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	APIType   string `json:"apiType"` // openai / anthropic / response
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	IsDefault bool   `json:"isDefault"`
}

func (s *AIService) toOut(p *model.AIProvider, withKey bool) AIProvider {
	out := AIProvider{ID: p.ID, Name: p.Name, APIType: p.APIType, BaseURL: p.BaseURL, Model: p.Model, IsDefault: p.IsDefault}
	if withKey {
		out.APIKey = p.APIKey
	}
	return out
}

// ListProviders 供应商列表（不回显 key）。
func (s *AIService) ListProviders() []AIProvider {
	rows := []model.AIProvider{}
	_ = s.db.Order("is_default desc, id").Find(&rows).Error
	out := []AIProvider{}
	for i := range rows {
		out = append(out, s.toOut(&rows[i], false))
	}
	return out
}

// Presets 内置供应商预设（前端新建时快速填充）。
func (s *AIService) Presets() []AIProvider {
	return []AIProvider{
		{ID: 0, Name: "智谱 GLM", APIType: "openai", BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4", Model: "glm-4.6"},
		{ID: 0, Name: "OpenAI", APIType: "openai", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
		{ID: 0, Name: "Anthropic", APIType: "anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-sonnet-4-20250514"},
		{ID: 0, Name: "DeepSeek", APIType: "openai", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
		{ID: 0, Name: "Ollama 本地", APIType: "openai", BaseURL: "http://127.0.0.1:11434/v1", Model: "qwen2.5:7b"},
	}
}

// SaveProvider 新建/更新（isDefault 时清除其他默认）。
func (s *AIService) SaveProvider(p *model.AIProvider) error {
	if p.APIType != "openai" && p.APIType != "anthropic" && p.APIType != "response" {
		return errs.Wrap(errs.ErrBadRequest, "API 类型仅支持 openai/anthropic/response")
	}
	if p.IsDefault {
		s.db.Model(&model.AIProvider{}).Where("is_default = ?", true).Update("is_default", false)
	}
	if p.ID == 0 {
		return s.db.Create(p).Error
	}
	return s.db.Model(p).Updates(map[string]any{
		"name": p.Name, "api_type": p.APIType, "base_url": p.BaseURL,
		"api_key": p.APIKey, "model": p.Model, "is_default": p.IsDefault,
	}).Error
}

// DeleteProvider 删除。
func (s *AIService) DeleteProvider(id uint) error {
	return s.db.Delete(&model.AIProvider{}, id).Error
}

// AIProviderModel 供 api 层类型引用。
type AIProviderModel = model.AIProvider

// ProviderByID 单条查询。
func (s *AIService) ProviderByID(id uint) (*model.AIProvider, error) {
	var row model.AIProvider
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "AI 供应商不存在")
	}
	return &row, nil
}

// DefaultProvider 取默认供应商（无默认取第一个）。
func (s *AIService) DefaultProvider() (*model.AIProvider, error) {
	return s.defaultProvider()
}

// defaultProvider 取默认供应商（无默认取第一个）。
func (s *AIService) defaultProvider() (*model.AIProvider, error) {
	var rows []model.AIProvider
	if err := s.db.Order("is_default desc, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "未配置 AI 供应商")
	}
	return &rows[0], nil
}

// panelContext 面板实时状态摘要（注入 system）。
func (s *AIService) panelContext(ctx context.Context) string {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return ""
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ctx, "/agent/v1/sysinfo/overview")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("当前主机：%s %s，CPU %.1f%%（%d核），内存 %.1f%%（%.1f/%.1fGB），负载 %.2f。",
		ov.Hostname, ov.Platform, ov.CPU.UsagePercent, ov.CPU.LogicalCount,
		ov.Memory.UsagePercent, float64(ov.Memory.Used)/1e9, float64(ov.Memory.Total)/1e9, ov.Load.Load1)
}

const aiSystemPrompt = "你是 YPanel 服务器管理面板的运维助手，用简洁中文回答服务器运维问题，可基于提供的实时主机状态给建议。"

// ChatMessage 对话消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamChat SSE 流式对话：upstream 解析 → 统一 data: {"content":"..."} 事件写出。
func (s *AIService) StreamChat(ctx context.Context, w http.ResponseWriter, provider *model.AIProvider, messages []ChatMessage) error {
	sys := ChatMessage{Role: "system", Content: aiSystemPrompt + " " + s.panelContext(ctx)}
	all := append([]ChatMessage{sys}, messages...)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return errs.Wrap(errs.ErrBadRequest, "当前连接不支持流式")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	req, err := s.buildUpstreamRequest(ctx, provider, all)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return errs.Wrap(errs.ErrAgentUnreach, "AI 服务不可达: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("AI 上游 %d: %s", resp.StatusCode, strings.TrimSpace(string(b))))
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var content string
		switch provider.APIType {
		case "anthropic":
			var ev struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if json.Unmarshal([]byte(payload), &ev) == nil && ev.Type == "content_block_delta" && ev.Delta.Type == "text_delta" {
				content = ev.Delta.Text
			}
		default: // openai chat completions / response 文本增量
			var ev struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Delta string `json:"delta"`
			}
			if json.Unmarshal([]byte(payload), &ev) == nil {
				if len(ev.Choices) > 0 {
					content = ev.Choices[0].Delta.Content
				} else if ev.Delta != "" {
					content = ev.Delta
				}
			}
		}
		if content != "" {
			b, _ := json.Marshal(map[string]string{"content": content})
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
	return nil
}

func (s *AIService) buildUpstreamRequest(ctx context.Context, p *model.AIProvider, messages []ChatMessage) (*http.Request, error) {
	var body map[string]any
	switch p.APIType {
	case "anthropic":
		body = map[string]any{"model": p.Model, "max_tokens": 4096, "stream": true, "messages": messages}
	default:
		body = map[string]any{"model": p.Model, "messages": messages, "stream": true}
	}
	b, _ := json.Marshal(body)
	base := strings.TrimRight(p.BaseURL, "/")
	var url string
	var req *http.Request
	var err error
	switch p.APIType {
	case "anthropic":
		url = base + "/v1/messages"
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err == nil {
			req.Header.Set("x-api-key", p.APIKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	case "response":
		body = map[string]any{"model": p.Model, "input": messages, "stream": true}
		b, _ = json.Marshal(body)
		url = base + "/responses"
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
	default:
		url = base + "/chat/completions"
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
	}
	if req != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, err
}
