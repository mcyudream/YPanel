// AIAPI AI 助手接口（B18）：供应商 CRUD + SSE 流式对话。
package api

import (
	"encoding/json"
	"fmt"
	"io"
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

// ListTools GET /api/v1/ai/tools（系统工具清单 + 开关状态）
func (a *AIAPI) ListTools(c *gin.Context) {
	respOK(c, a.AI.ListTools())
}

// SetToolFlag POST /api/v1/ai/tools/flag {name, enabled}
func (a *AIAPI) SetToolFlag(c *gin.Context) {
	req, ok := bind[struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.SetToolFlag(req.Name, req.Enabled); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.AI.ListTools())
}

// UploadSkillZip POST /api/v1/ai/skills/upload（multipart file）
func (a *AIAPI) UploadSkillZip(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		respErr(c, errBadRequest("缺少 file 字段"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		respErr(c, err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		respErr(c, err)
		return
	}
	name, desc, err := a.AI.UploadSkillZip(c.Request.Context(), data)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"name": name, "description": desc})
}

// ListKnowledgeDocs GET /api/v1/ai/knowledge/docs
func (a *AIAPI) ListKnowledgeDocs(c *gin.Context) {
	respOK(c, a.AI.ListKnowledgeDocs())
}

// SaveKnowledgeDoc POST /api/v1/ai/knowledge/doc {title?, filename, content}
func (a *AIAPI) SaveKnowledgeDoc(c *gin.Context) {
	req, ok := bind[struct {
		Title    string `json:"title"`
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}](c)
	if !ok {
		return
	}
	doc, err := a.AI.SaveKnowledgeDoc(req.Title, req.Filename, req.Content)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, doc)
}

// GetKnowledgeDoc GET /api/v1/ai/knowledge/doc/:id（全文）
func (a *AIAPI) GetKnowledgeDoc(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("文档 ID 不合法"))
		return
	}
	out, err := a.AI.GetKnowledgeDoc(uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DeleteKnowledgeDoc DELETE /api/v1/ai/knowledge/doc/:id
func (a *AIAPI) DeleteKnowledgeDoc(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("文档 ID 不合法"))
		return
	}
	if err := a.AI.DeleteKnowledgeDoc(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
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

// ListConversations GET /api/v1/ai/conversations
func (a *AIAPI) ListConversations(c *gin.Context) {
	respOK(c, a.AI.ListConversations())
}

// GetConversation GET /api/v1/ai/conversations/:id
func (a *AIAPI) GetConversation(c *gin.Context) {
	if id := aiIDParam(c); id == 0 {
		respErr(c, errBadRequest("会话 ID 不合法"))
		return
	} else if out, err := a.AI.GetConversation(id); err != nil {
		respErr(c, err)
		return
	} else {
		respOK(c, out)
	}
}

// SaveConversation POST /api/v1/ai/conversations {id?, title, messages}
func (a *AIAPI) SaveConversation(c *gin.Context) {
	req, ok := bind[struct {
		ID       uint                  `json:"id"`
		Title    string                `json:"title"`
		Messages []service.ChatMessage `json:"messages"`
	}](c)
	if !ok {
		return
	}
	id, err := a.AI.SaveConversation(c.Request.Context(), req.ID, req.Title, req.Messages)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": id})
}

// DeleteConversation DELETE /api/v1/ai/conversations/:id
func (a *AIAPI) DeleteConversation(c *gin.Context) {
	if id := aiIDParam(c); id == 0 {
		respErr(c, errBadRequest("会话 ID 不合法"))
		return
	} else if err := a.AI.DeleteConversation(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ListMemories GET /api/v1/ai/memories
func (a *AIAPI) ListMemories(c *gin.Context) {
	respOK(c, a.AI.RecallMemories(""))
}

// ClearMemories DELETE /api/v1/ai/memories（清空全部记忆）
func (a *AIAPI) ClearMemories(c *gin.Context) {
	if err := a.AI.DeleteAllMemories(); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ---- Skills（技能包） ----

// ListSkills GET /api/v1/ai/skills
func (a *AIAPI) ListSkills(c *gin.Context) {
	out, err := a.AI.ListSkills(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SaveSkill POST /api/v1/ai/skills {name, description, body}
func (a *AIAPI) SaveSkill(c *gin.Context) {
	req, ok := bind[struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Body        string `json:"body" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.SaveSkill(c.Request.Context(), req.Name, req.Description, req.Body); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetSkillEnabled POST /api/v1/ai/skills/:name/enable {enabled}
func (a *AIAPI) SetSkillEnabled(c *gin.Context) {
	name := c.Param("name")
	req, ok := bind[struct {
		Enabled bool `json:"enabled"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.SetSkillEnabled(name, req.Enabled); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// RemoveSkill DELETE /api/v1/ai/skills/:name
func (a *AIAPI) RemoveSkill(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		respErr(c, errBadRequest("技能名必填"))
		return
	}
	if err := a.AI.RemoveSkill(c.Request.Context(), name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ---- MCP（服务器配置与测试） ----

// ListMCPServers GET /api/v1/ai/mcp/servers
func (a *AIAPI) ListMCPServers(c *gin.Context) {
	respOK(c, a.AI.ListMCPServers())
}

// SaveMCPServers PUT /api/v1/ai/mcp/servers（整表保存）
func (a *AIAPI) SaveMCPServers(c *gin.Context) {
	req, ok := bind[struct {
		Servers []service.MCPServerConf `json:"servers"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.SaveMCPServers(req.Servers); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, req.Servers)
}

// TestMCP POST /api/v1/ai/mcp/test（连接并列出工具）
func (a *AIAPI) TestMCP(c *gin.Context) {
	req, ok := bind[service.MCPServerConf](c)
	if !ok {
		return
	}
	out, err := a.AI.TestMCP(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CloseMCP DELETE /api/v1/ai/mcp/:name（断开会话）
func (a *AIAPI) CloseMCP(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		respErr(c, errBadRequest("名称必填"))
		return
	}
	a.AI.CloseMCP(name)
	respOK(c, struct{}{})
}
