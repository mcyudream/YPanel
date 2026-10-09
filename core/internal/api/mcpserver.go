// MCPServerAPI MCP 开放端点（M41）：token 鉴权 + 开关校验后委托 mcp-go Streamable HTTP。
package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/service"
)

// MCPServerAPI MCP 开放。
type MCPServerAPI struct {
	MCP  *service.MCPService
	Auth *service.Auth
	Ops  *service.MCPOperationService
	RBAC *rbac.Service
}

// Handler ANY /api/v1/mcp（Bearer token 或 ?token= 鉴权；mcp.enabled 关闭时 404）
func (a *MCPServerAPI) Handler(c *gin.Context) {
	if !a.MCP.Enabled() {
		c.String(http.StatusNotFound, "404")
		return
	}
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	claims, err := a.Auth.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	// M54：MCP 不经业务路由中间件，在此按调用者权限/节点范围闭合（解析失败按空集拒绝）
	permSet := map[string]struct{}{}
	allNodes := true
	nodeSet := map[string]struct{}{}
	if u, err := a.Auth.ByID(claims.UID); err == nil && a.RBAC != nil {
		scope := a.RBAC.ScopeForUser(u)
		permSet = scope.Perms
		allNodes = scope.AllNodes
		nodeSet = scope.Nodes
	}
	ctx := rbac.WithCaller(c.Request.Context(), rbac.Caller{UserID: claims.UID, PermSet: permSet, AllNodes: allNodes, NodeSet: nodeSet})
	a.MCP.HTTPHandler().ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}

// Status GET /api/v1/mcp/status（管理视图：开关/白名单/默认集）
func (a *MCPServerAPI) Status(c *gin.Context) {
	respOK(c, gin.H{
		"enabled":      a.MCP.Enabled(),
		"endpoint":     "/api/v1/mcp",
		"tools":        a.MCP.ToolNames(),
		"allowlistRaw": a.MCP.AllowlistRaw(),
	})
}

// SetEnabled PUT /api/v1/mcp/status {enabled, tools?}（admin）
func (a *MCPServerAPI) SetEnabled(c *gin.Context) {
	req, ok := bind[struct {
		Enabled *bool   `json:"enabled"`
		Tools   *string `json:"tools"`
	}](c)
	if !ok {
		return
	}
	if req.Enabled != nil {
		if err := a.MCP.SetEnabled(*req.Enabled); err != nil {
			respErr(c, err)
			return
		}
	}
	if req.Tools != nil {
		if err := a.MCP.SetAllowlist(*req.Tools); err != nil {
			respErr(c, err)
			return
		}
	}
	respOK(c, struct{}{})
}

// ---- M45：MCP 写操作审批 ----

// OpList GET /api/v1/mcp/operations?state=（审批列表）
func (a *MCPServerAPI) OpList(c *gin.Context) {
	respOK(c, a.Ops.List(c.Query("state")))
}

// OpApprove PUT /api/v1/mcp/operations/:id {approve}（面板登录态审批）
func (a *MCPServerAPI) OpApprove(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Approve bool `json:"approve"`
	}](c)
	if !ok {
		return
	}
	if err := a.Ops.Approve(id, c.GetString("username"), req.Approve); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
