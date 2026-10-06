// MCPManager（B18+）：MCP 服务器配置与工具桥接。
// 配置存 settings KV（ai.mcp_servers JSON 数组）；stdio 型在 core 进程内 spawn
// MCP server 子进程（npx/uvx 二进制等），streamable-http 型直连 URL。
// 工具桥接：MCP tools/list 的工具包装为 langchaingo 工具加入对话循环。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ypanel/shared/errs"
)

// MCPServerConf 单个 MCP 服务器配置。
type MCPServerConf struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"` // stdio / streamable-http
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Enabled   bool              `json:"enabled"`
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
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return s.settings.Set(mcpConfKey, string(b))
}

// MCPSession 一个已连接的 MCP 服务器会话。
type MCPSession struct {
	Client *client.Client
	Tools  []mcp.Tool
}

// mcpSessions 活跃会话缓存（按服务器名）。
var mcpSessions sync.Map

func mcpSessionKey(name string) string { return "mcp:" + name }

// ConnectMCP 连接（或复用）指定 MCP 服务器，返回会话。
func (s *AIService) ConnectMCP(ctx context.Context, conf MCPServerConf) (*MCPSession, error) {
	if v, ok := mcpSessions.Load(mcpSessionKey(conf.Name)); ok {
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
		c, err = client.NewStreamableHttpClient(conf.URL)
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
	sess := &MCPSession{Client: c, Tools: lt.Tools}
	mcpSessions.Store(mcpSessionKey(conf.Name), sess)
	return sess, nil
}

// CloseMCP 断开指定服务器。
func (s *AIService) CloseMCP(name string) {
	if v, ok := mcpSessions.Load(mcpSessionKey(name)); ok {
		_ = v.(*MCPSession).Client.Close()
		mcpSessions.Delete(mcpSessionKey(name))
	}
}

// CloseAllMCP 关闭全部 MCP 会话（进程退出时调用）。
func (s *AIService) CloseAllMCP() {
	mcpSessions.Range(func(_, v any) bool {
		_ = v.(*MCPSession).Client.Close()
		return true
	})
	mcpSessions = sync.Map{}
}

// TestMCP 连接并列出工具（配置测试）。
func (s *AIService) TestMCP(ctx context.Context, conf MCPServerConf) (map[string]any, error) {
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
	Server  string
	Conf    MCPServerConf
	Name    string
	Descr   string
	Schema  json.RawMessage
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
			continue
		}
		for _, t := range sess.Tools {
			schema, _ := json.Marshal(t.InputSchema)
			out = append(out, mcpToolDef{Server: conf.Name, Name: "mcp_" + conf.Name + "_" + t.Name, Descr: t.Description, Schema: schema})
		}
	}
	return out
}

// callMCPTool 调用指定 MCP 工具（name = mcp_<server>_<tool>）。
func (s *AIService) callMCPTool(ctx context.Context, fullName, argsJSON string) (string, error) {
	for _, conf := range s.ListMCPServers() {
		prefix := "mcp_" + conf.Name + "_"
		if !strings.HasPrefix(fullName, prefix) {
			continue
		}
		toolName := strings.TrimPrefix(fullName, prefix)
		sess, err := s.ConnectMCP(ctx, conf)
		if err != nil {
			return "", err
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
		return strings.Join(parts, "\n"), nil
	}
	return "", fmt.Errorf("MCP 工具未找到: %s", fullName)
}

var _ = strings.TrimSpace
