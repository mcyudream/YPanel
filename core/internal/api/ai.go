// AIAPI AI 助手接口（B18）：供应商 CRUD + SSE 流式对话。
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
)

// AIAPI AI 助手。
type AIAPI struct {
	AI    *service.AIService
	Nodes *service.NodeService
}

func aiIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0
	}
	return uint(id)
}

// ListProviders GET /api/v1/ai/providers
func (a *AIAPI) ListProviders(c *gin.Context) {
	respOK(c, a.AI.ListProviders())
}

// Presets GET /api/v1/ai/presets
func (a *AIAPI) Presets(c *gin.Context) {
	respOK(c, a.AI.Presets())
}

// SaveProvider POST /api/v1/ai/providers
func (a *AIAPI) SaveProvider(c *gin.Context) {
	req, ok := bind[service.AIProvider](c)
	if !ok {
		return
	}
	row := model.AIProvider{
		ID: req.ID, Name: req.Name, APIType: req.APIType, BaseURL: req.BaseURL,
		APIKey: req.APIKey, Model: req.Model, IsDefault: req.IsDefault,
	}
	if err := a.AI.SaveProvider(&row); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.AI.ListProviders())
}

// DeleteProvider DELETE /api/v1/ai/providers/:id
func (a *AIAPI) DeleteProvider(c *gin.Context) {
	if id := aiIDParam(c); id == 0 {
		respErr(c, errBadRequest("供应商 ID 不合法"))
		return
	} else if err := a.AI.DeleteProvider(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Chat POST /api/v1/ai/chat {providerId?, messages:[{role,content}]}（SSE 流式）
func (a *AIAPI) Chat(c *gin.Context) {
	req, ok := bind[struct {
		ProviderID uint                  `json:"providerId"`
		Messages   []service.ChatMessage `json:"messages"`
	}](c)
	if !ok {
		return
	}
	var provider *model.AIProvider
	var err error
	if req.ProviderID > 0 {
		provider, err = a.AI.ProviderByID(req.ProviderID)
	} else {
		provider, err = a.AI.DefaultProvider()
	}
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.AI.StreamAgentChat(c.Request.Context(), c.Writer, provider, req.Messages, c.Query("scene")); err != nil {
		c.SSEvent("error", gin.H{"message": err.Error()})
	}
}

// WorkspaceList GET /api/v1/ai/workspace（B18：工作空间文件列表）
func (a *AIAPI) WorkspaceList(c *gin.Context) {
	node, _ := a.AI.NodeLocal()
	ac := agentclient.New(fmt.Sprint(node["baseURL"]), fmt.Sprint(node["token"]))
	out, err := agentclient.GetJSON[json.RawMessage](ac, c.Request.Context(),
		"/agent/v1/files/list?path="+escape("/opt/ypanel/ai-workspace"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// WorkspaceRun POST /api/v1/ai/workspace/run {command}（B18：沙箱内执行）
func (a *AIAPI) WorkspaceRun(c *gin.Context) {
	req, ok := bind[struct {
		Command string `json:"command" binding:"required"`
	}](c)
	if !ok {
		return
	}
	node, _ := a.AI.NodeLocal()
	ac := agentclient.New(fmt.Sprint(node["baseURL"]), fmt.Sprint(node["token"]))
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, c.Request.Context(), http.MethodPost, "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("mkdir -p /opt/ypanel/ai-workspace && cd /opt/ypanel/ai-workspace && %s", req.Command), TimeoutSecs: 120})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"output": out.Output, "exitCode": out.ExitCode})
}

// ListKnowledge GET /api/v1/ai/knowledge
func (a *AIAPI) ListKnowledge(c *gin.Context) {
	respOK(c, a.AI.ListKnowledge())
}

// SaveKnowledge POST /api/v1/ai/knowledge
func (a *AIAPI) SaveKnowledge(c *gin.Context) {
	req, ok := bind[model.AIKnowledge](c)
	if !ok {
		return
	}
	if err := a.AI.SaveKnowledge(req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.AI.ListKnowledge())
}

// DeleteKnowledge DELETE /api/v1/ai/knowledge/:id
func (a *AIAPI) DeleteKnowledge(c *gin.Context) {
	id := aiIDParam(c)
	if id == 0 {
		respErr(c, errBadRequest("知识条目 ID 不合法"))
		return
	}
	if err := a.AI.DeleteKnowledge(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.AI.ListKnowledge())
}
