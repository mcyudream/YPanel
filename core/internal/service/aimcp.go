// MCPManager（B18+）：MCP 服务器配置与工具桥接。
// 配置存 settings KV（ai.mcp_servers JSON 数组）；stdio 型在 core 进程内 spawn
// MCP server 子进程（npx/uvx 二进制等），streamable-http 型直连 URL。
// 工具桥接：MCP tools/list 的工具包装为 langchaingo 工具加入对话循环。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/errs"
)

// MCPServerConf 单个 MCP 服务器配置。
type MCPServerConf struct {
	Name         string            `json:"name"`
	AllowPrivate bool              `json:"allowPrivate"`
	Transport    string            `json:"transport"` // stdio / streamable-http
	Command      string            `json:"command,omitempty"`
	Args         []string          `json:"args,omitempty"`
	URL          string            `json:"url,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Enabled      bool              `json:"enabled"`
}

// mcpConfKey settings 键。
const mcpConfKey = "ai.mcp_servers"

// ListMCPServers 配置列表。
func (s *AIService) ListMCPServers() []MCPServerConf {
	raw := s.settings.Get(mcpConfKey, "[]")
	var out []MCPServerConf
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []MCPServerConf{}
	}
	return out
}

// SaveMCPServers 保存配置列表。
func (s *AIService) SaveMCPServers(list []MCPServerConf) error {
	if len(list) > 30 {
		return errWrapAI("MCP 服务过多")
	}
	names := map[string]bool{}
	for _, c := range list {
		if c.Name == "" || len(c.Name) > 80 || names[c.Name] {
			return errWrapAI("MCP 名称为空、过长或重复")
		}
		names[c.Name] = true
		if c.Transport != "stdio" && c.Transport != "streamable-http" {
			return errWrapAI("不支持的 MCP 传输方式")
		}
		if c.Transport == "stdio" && strings.TrimSpace(c.Command) == "" {
			return errWrapAI("stdio MCP 需要命令")
		}
		if c.Transport == "streamable-http" {
			if _, err := newAIHTTP(&model.AIProvider{BaseURL: c.URL, AllowPrivate: c.AllowPrivate}); err != nil {
				return err
			}
		}
	}

	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	if err := s.settings.Set(mcpConfKey, string(b)); err != nil {
		return err
	}
	s.CloseAllMCP()
	return nil
}

// MCPSession 一个已连接的 MCP 服务器会话。
type MCPSession struct {
	Client *client.Client
	Tools  []mcp.Tool
}

// mcpSessions 活跃会话缓存（按服务器名）。
// 活跃连接由 AIService 管理，不在多个实例间共享。

func mcpSessionKey(name string) string { return "mcp:" + name }

// ConnectMCP 连接（或复用）指定 MCP 服务器，返回会话。
func (s *AIService) ConnectMCP(ctx context.Context, conf MCPServerConf) (*MCPSession, error) {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if v, ok := s.mcpSessions.Load(mcpSessionKey(conf.Name)); ok {
		return v.(*MCPSession), nil
	}
	var c *client.Client
	var err error
	switch conf.Transport {
	case "stdio":
		if conf.Command == "" {
			return nil, errs.Wrap(errs.ErrBadRequest, "stdio 型 MCP 需要 command")
		}
		env := make([]string, 0, len(conf.Env))
		for k, v := range conf.Env {
			env = append(env, k+"="+v)
		}
		c, err = client.NewStdioMCPClient(conf.Command, env, conf.Args...)
	case "streamable-http", "sse", "http":
		if conf.URL == "" {
			return nil, errs.Wrap(errs.ErrBadRequest, "http 型 MCP 需要 url")
		}
		httpClient, herr := newAIHTTP(&model.AIProvider{BaseURL: conf.URL, AllowPrivate: conf.AllowPrivate})
		if herr != nil {
			return nil, herr
		}
		c, err = client.NewStreamableHttpClient(conf.URL, transport.WithHTTPBasicClient(httpClient.client))
	default:
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的传输类型: "+conf.Transport)
	}
	if err != nil {
		return nil, err
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = "2025-06-18"
	initReq.Params.ClientInfo = mcp.Implementation{Name: "YPanel", Version: "1.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		_ = c.Close()
		return nil, errs.Wrap(errs.ErrAgentUnreach, "MCP 初始化失败: "+err.Error())
	}
	toolsReq := mcp.ListToolsRequest{}
	lt, err := c.ListTools(ctx, toolsReq)
	if err != nil {
		_ = c.Close()
		return nil, errs.Wrap(errs.ErrAgentUnreach, "MCP 工具列表失败: "+err.Error())
	}
	all := append([]mcp.Tool{}, lt.Tools...)
	seen := map[mcp.Cursor]bool{}
	for lt.NextCursor != "" {
		if seen[lt.NextCursor] || len(all) > 500 {
			_ = c.Close()
			return nil, errWrapAI("MCP 工具分页循环或工具数量超过 500")
		}
		seen[lt.NextCursor] = true
		toolsReq.Params.Cursor = lt.NextCursor
		lt, err = c.ListTools(ctx, toolsReq)
		if err != nil {
			_ = c.Close()
			return nil, err
		}
		all = append(all, lt.Tools...)
	}
	if len(all) > 500 {
		_ = c.Close()
		return nil, errWrapAI("单个 MCP 服务最多 500 个工具")
	}
	sess := &MCPSession{Client: c, Tools: all}
	s.mcpSessions.Store(mcpSessionKey(conf.Name), sess)
	return sess, nil
}

// CloseMCP 断开指定服务器。
func (s *AIService) CloseMCP(name string) {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	if v, ok := s.mcpSessions.Load(mcpSessionKey(name)); ok {
		_ = v.(*MCPSession).Client.Close()
		s.mcpSessions.Delete(mcpSessionKey(name))
	}
}

// CloseAllMCP 关闭全部 MCP 会话（进程退出时调用）。
func (s *AIService) CloseAllMCP() {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	s.mcpSessions.Range(func(_, v any) bool {
		_ = v.(*MCPSession).Client.Close()
		return true
	})
	s.mcpSessions.Clear()
}

// TestMCP 连接并列出工具（配置测试）。
func (s *AIService) TestMCP(ctx context.Context, conf MCPServerConf) (map[string]any, error) {
	conf.Name = "test-" + randAskID()
	defer s.CloseMCP(conf.Name)
	sess, err := s.ConnectMCP(ctx, conf)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(sess.Tools))
	for _, t := range sess.Tools {
		names = append(names, t.Name)
	}
	return map[string]any{"connected": true, "tools": names}, nil
}

// mcpToolDef MCP 工具定义（含来源服务器）。
type mcpToolDef struct {
	Server string
	Conf   MCPServerConf
	Name   string
	Descr  string
	Schema json.RawMessage
}

// enabledMCPTools 收集全部启用服务器的工具定义。
func (s *AIService) enabledMCPTools(ctx context.Context) []mcpToolDef {
	var out []mcpToolDef
	for _, conf := range s.ListMCPServers() {
		if !conf.Enabled {
			continue
		}
		sess, err := s.ConnectMCP(ctx, conf)
		if err != nil {
			slog.Warn("ai mcp discovery failed", "server", conf.Name, "err", maskSensitive(err.Error()))
			continue
		}
		for _, t := range sess.Tools {
			schema, _ := json.Marshal(t.InputSchema)
			out = append(out, mcpToolDef{Server: conf.Name, Name: mcpSafeToolName(conf.Name, t.Name), Descr: t.Description, Schema: schema})
		}
	}
	return out
}

// callMCPTool 调用指定 MCP 工具（name = mcp_<server>_<tool>）。
func (s *AIService) callMCPTool(ctx context.Context, fullName, argsJSON string) (string, error) {
	for _, conf := range s.ListMCPServers() {
		if !conf.Enabled {
			continue
		}
		sess, err := s.ConnectMCP(ctx, conf)
		if err != nil {
			continue
		}
		toolName := ""
		for _, tool := range sess.Tools {
			if mcpSafeToolName(conf.Name, tool.Name) == fullName {
				toolName = tool.Name
				break
			}
		}
		if toolName == "" {
			continue
		}
		req := mcp.CallToolRequest{}
		req.Params.Name = toolName
		req.Params.Arguments = json.RawMessage(argsJSON)
		res, err := sess.Client.CallTool(ctx, req)
		if err != nil {
			return "", err
		}
		var parts []string
		for _, c := range res.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				parts = append(parts, tc.Text)
			}
		}
		if res.IsError {
			return "", fmt.Errorf("MCP 工具返回错误：%s", maskSensitive(strings.Join(parts, "\n")))
		}
		return strings.Join(parts, "\n"), nil
	}
	return "", fmt.Errorf("MCP 工具未找到: %s", fullName)
}

var _ = strings.TrimSpace

func (s *AIService) aiMCPTools(ctx context.Context) []aiToolDef {
	out := []aiToolDef{}
	for _, tool := range s.enabledMCPTools(ctx) {
		t := tool
		var schema map[string]any
		if json.Unmarshal(t.Schema, &schema) != nil {
			continue
		}
		out = append(out, aiToolDef{Name: t.Name, Module: "mcp", Risk: aiRiskDanger, Desc: "外部 MCP 工具，来源 " + t.Server + "。未知副作用，执行前需要确认。 " + t.Descr, Parameters: schema, Fn: func(c context.Context, args string) (string, error) {
			if err := rbac.CheckTool(c, "mcp", aiRiskDanger, args); err != nil {
				return "", err
			}
			return s.callMCPTool(c, t.Name, args)
		}})
	}
	return out
}
