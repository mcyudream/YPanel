// MCP 对外开放（M41）：面板只读能力以 MCP server（Streamable HTTP）暴露。
// 鉴权 = 面板用户 token（Bearer）；工具 = AI 注册表 read 级子集（白名单可控）；
// 每次调用落 AIOperationLog（module=mcp）。OAuth/写操作审批列后续。
package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
)

// mcpDefaultTools 默认暴露的只读工具白名单（空 mcp.tools 设置时生效）。
var mcpDefaultTools = map[string]bool{
	"get_overview": true, "list_nodes": true,
	"list_containers": true, "container_logs": true, "list_images": true,
	"list_sites": true, "list_certs": true,
	"list_cron_jobs": true, "list_database_instances": true, "query_database": true,
	"read_file": true, "list_processes": true, "list_services": true,
	"list_firewall_rules": true, "list_nat_rules": true,
}

// MCPSettingKey 全局开关（"1" 开）。
const MCPSettingKey = "mcp.enabled"

// MCPToolsSettingKey 工具白名单（逗号分隔；空=默认只读集）。
const MCPToolsSettingKey = "mcp.tools"

// MCPService MCP server 服务。
type MCPService struct {
	db       *gorm.DB
	ai       *AIService
	settings *SettingService
	nodes    *NodeService
	bk       *PanelBackupService
	ops      *MCPOperationService
	server   *server.MCPServer
	http     *server.StreamableHTTPServer
	toolsMu  sync.Once
}

// NewMCPService 创建（注册只读工具集）。
func NewMCPService(db *gorm.DB, ai *AIService, settings *SettingService) *MCPService {
	s := &MCPService{db: db, ai: ai, settings: settings}
	s.server = server.NewMCPServer("ypanel", "1.0.0")
	s.http = server.NewStreamableHTTPServer(s.server, server.WithStateLess(true)) // 无状态：token 客户端免 session 握手
	return s
}

// RegisterWriteDeps 装配写工具依赖并注册（M45；main 调用）。
func (s *MCPService) RegisterWriteDeps(nodes *NodeService, bk *PanelBackupService, ops *MCPOperationService) {
	s.nodes, s.bk, s.ops = nodes, bk, ops
	s.RegisterWriteTools(ops)
}

// Enabled 全局开关。
func (s *MCPService) Enabled() bool {
	return s.settings.Get(MCPSettingKey, "0") == "1"
}

// allowlist 白名单集合。
func (s *MCPService) allowlist() map[string]bool {
	raw := s.settings.Get(MCPToolsSettingKey, "")
	if strings.TrimSpace(raw) == "" {
		return mcpDefaultTools
	}
	out := map[string]bool{}
	for _, n := range strings.Split(raw, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out[n] = true
		}
	}
	return out
}

// HTTPHandler 返回挂载到路由的 http.Handler（调用方自行鉴权 + 开关校验）。
// 首次访问时从 AI 注册表惰性注册 read 工具（白名单内）。
func (s *MCPService) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.toolsMu.Do(func() { s.registerTools(context.Background()) }) // 不可绑请求 ctx：注册是进程级单次，随首请求取消会污染后续所有调用
		s.http.ServeHTTP(w, r)
	})
}

// registerTools 把 AI 注册表中白名单内的 read 工具注册为 MCP tool。
func (s *MCPService) registerTools(ctx context.Context) {
	allow := s.allowlist()
	for _, def := range s.ai.aiToolsAll(ctx) {
		if def.Risk != aiRiskRead || !allow[def.Name] {
			continue
		}
		schema, _ := json.Marshal(def.Parameters)
		if len(def.Parameters) == 0 {
			schema = []byte(`{"type":"object","properties":{}}`)
		}
		t := mcp.NewToolWithRawSchema(def.Name, def.Desc, schema)
		name := def.Name
		fn := def.Fn
		s.server.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := "{}"
			if len(req.GetArguments()) > 0 {
				if b, err := json.Marshal(req.GetArguments()); err == nil {
					args = string(b)
				}
			}
			out, err := fn(ctx, args)
			s.audit(name, args, err == nil)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(out), nil
		})
	}
}

// audit MCP 工具调用留痕。
func (s *MCPService) audit(tool, args string, success bool) {
	row := model.AIOperationLog{
		Tool: tool, Module: "mcp", Risk: aiRiskRead, Action: "executed",
		Args: truncStr(args, 2000), Success: success,
	}
	_ = s.db.Create(&row).Error
}

// ToolNames 当前注册的 MCP 工具名。
func (s *MCPService) ToolNames() []string {
	out := []string{}
	for name := range s.server.ListTools() {
		out = append(out, name)
	}
	return out
}

// AllowlistRaw 白名单原始串。
func (s *MCPService) AllowlistRaw() string {
	return s.settings.Get(MCPToolsSettingKey, "")
}

// SetEnabled 设置全局开关。
func (s *MCPService) SetEnabled(v bool) error {
	flag := "0"
	if v {
		flag = "1"
	}
	return s.settings.Set(MCPSettingKey, flag)
}

// SetAllowlist 设置工具白名单（空=默认只读集）。
func (s *MCPService) SetAllowlist(raw string) error {
	return s.settings.Set(MCPToolsSettingKey, strings.TrimSpace(raw))
}
