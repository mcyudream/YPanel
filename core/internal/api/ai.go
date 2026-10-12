// AIAPI AI 助手接口（B18）：供应商 CRUD + SSE 流式对话。
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// callerCtx 下游工具链路的授权 ctx：auth 中间件已在请求 ctx 注入完整 Caller
// （权限集 + 节点范围 + 数据范围，超管 AllNodes=true），此处只透传；
// 曾在此重建 Caller 丢失节点范围导致超管被「无节点权限」误拒（M33 修复）。
func callerCtx(c *gin.Context) context.Context {
	if caller, ok := rbac.CallerFrom(c.Request.Context()); ok {
		_ = caller
		return c.Request.Context()
	}
	// 兜底（无 auth 中间件的链路）：仅按权限集构建，节点范围闭合拒绝
	set, _ := c.Get(middleware.CtxPerms)
	permSet, _ := set.(map[string]struct{})
	if permSet == nil {
		permSet = map[string]struct{}{}
	}
	return rbac.WithCaller(c.Request.Context(), rbac.Caller{UserID: c.GetUint(middleware.CtxUID), PermSet: permSet})
}

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
		APIKey: req.APIKey, Model: req.Model, Models: req.Models, IsDefault: req.IsDefault,
	}
	if err := a.AI.SaveProvider(&row); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.AI.ListProviders())
}

// ProviderModels GET /api/v1/ai/providers/:id/models（经供应商 API 拉取可用模型）
func (a *AIAPI) ProviderModels(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("供应商 ID 不合法"))
		return
	}
	models, err := a.AI.ProviderModels(uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, models)
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

// ChatStart POST /api/v1/ai/chat/start（M62）：创建后台对话任务，立即返回会话 ID。
// 任务在后端 goroutine 运行（30min 上限），不随 HTTP 连接断开而终止；
// 前端经 GET /ai/chat/stream 订阅事件流（换窗口/强刷新可重连介入）。
func (a *AIAPI) ChatStart(c *gin.Context) {
	req, ok := bind[struct {
		ConversationID uint                  `json:"conversationId"`
		ProviderID     uint                  `json:"providerId"`
		Model          string                `json:"model"`
		Mode           string                `json:"mode"`
		Messages       []service.ChatMessage `json:"messages"`
		Scene          string                `json:"scene"`
		Focus          string                `json:"focus"`
		MaxRounds      int                   `json:"maxRounds"`
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
	// 同一供应商多模型：请求可指定模型覆盖供应商默认值
	if req.Model != "" {
		cp := *provider
		cp.Model = req.Model
		provider = &cp
	}
	convoID, err := a.AI.StartChatRun(callerCtx(c), req.ConversationID, provider, req.Messages, req.Scene, req.Focus, req.Mode, req.MaxRounds)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"conversationId": convoID})
}

// ChatStream GET /api/v1/ai/chat/stream?conversationId=N&offset=K（M62）：
// 订阅任务事件流——回放 events[K:] 后实时推送；响应头 X-Ai-Run 报告任务状态
// （none=无运行体，前端回退静态历史）。断开仅取消订阅，不影响任务本身。
func (a *AIAPI) ChatStream(c *gin.Context) {
	id64, _ := strconv.ParseUint(c.Query("conversationId"), 10, 64)
	offset, _ := strconv.Atoi(c.Query("offset"))
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		respErr(c, errBadRequest("当前连接不支持流式"))
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	write := func(b []byte) {
		_, _ = c.Writer.Write(b)
		flusher.Flush()
	}
	runFrame := func(v any) {
		b, _ := json.Marshal(v)
		write(append(append([]byte("data: "), b...), '\n', '\n'))
	}

	convoID := uint(id64)
	replay, sub, doneCh, status, has := a.AI.SubscribeChatRun(convoID, offset)
	if !has {
		c.Header("X-Ai-Run", "none")
		runFrame(map[string]string{"done": "1"})
		return
	}
	c.Header("X-Ai-Run", status)
	for _, b := range replay {
		write(append(append([]byte("data: "), b...), '\n', '\n'))
	}
	if status != service.AIRunStatusRunning {
		return
	}
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case b := <-sub:
			if isPingFrame(b) {
				write(b)
				continue
			}
			write(append(append([]byte("data: "), b...), '\n', '\n'))
		case <-heartbeat.C:
			write([]byte(": ping\n\n"))
		case <-doneCh:
			// 任务收尾：剩余帧已按序进缓冲，排空后结束订阅
			for {
				select {
				case b := <-sub:
					if isPingFrame(b) {
						write(b)
						continue
					}
					write(append(append([]byte("data: "), b...), '\n', '\n'))
					continue
				default:
				}
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

// isPingFrame 心跳注释帧原样透传（不入客户端 offset 计数）。
func isPingFrame(b []byte) bool {
	return len(b) > 0 && b[0] == ':'
}

// ChatStop POST /api/v1/ai/chat/stop {conversationId}（M62）：终止后台任务
// （fail-closed 语义保留：未批准的操作视为拒绝），半截内容落库。
func (a *AIAPI) ChatStop(c *gin.Context) {
	req, ok := bind[struct {
		ConversationID uint `json:"conversationId"`
	}](c)
	if !ok {
		return
	}
	if !a.AI.StopChatRun(req.ConversationID) {
		respErr(c, errBadRequest("该会话没有进行中的任务"))
		return
	}
	respOK(c, struct{}{})
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

// ResolveAsk POST /api/v1/ai/ask/:id {approve}（危险操作确认回执）
func (a *AIAPI) ResolveAsk(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respErr(c, errBadRequest("ask ID 不合法"))
		return
	}
	req, ok := bind[struct {
		Approve bool `json:"approve"`
	}](c)
	if !ok {
		return
	}
	if !a.AI.ResolveAsk(id, req.Approve) {
		respErr(c, errBadRequest("确认请求不存在或已超时"))
		return
	}
	respOK(c, gin.H{"resolved": true})
}

// ResolveAskQuestion POST /api/v1/ai/ask-question/:id（交互提问回执）
func (a *AIAPI) ResolveAskQuestion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respErr(c, errBadRequest("提问 ID 不合法"))
		return
	}
	// 前端提交的是答案对象数组（每题 header/selected/text），序列化后作为工具结果回给模型
	req, ok := bind[struct {
		Answers []map[string]any `json:"answers"`
	}](c)
	if !ok {
		return
	}
	b, err := json.Marshal(req.Answers)
	if err != nil {
		respErr(c, errBadRequest("答案格式不合法"))
		return
	}
	if !a.AI.ResolveQuestion(id, string(b)) {
		respErr(c, errBadRequest("提问不存在或已取消"))
		return
	}
	respOK(c, gin.H{"resolved": true})
}

// GetMaxToolRounds GET /api/v1/ai/tools/max-rounds
func (a *AIAPI) GetMaxToolRounds(c *gin.Context) {
	respOK(c, gin.H{"rounds": a.AI.MaxToolRoundsForUI()})
}

// SetMaxToolRounds POST /api/v1/ai/tools/max-rounds {rounds}
func (a *AIAPI) SetMaxToolRounds(c *gin.Context) {
	req, ok := bind[struct {
		Rounds int `json:"rounds"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.SetMaxToolRounds(req.Rounds); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"rounds": a.AI.MaxToolRoundsForUI()})
}

// ListOperationLogs GET /api/v1/ai/oplogs（AI 操作审计，分页）
func (a *AIAPI) ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	respOK(c, a.AI.ListOperationLogs(page, size))
}

// GetAskMode GET /api/v1/ai/tools/ask-mode
func (a *AIAPI) GetAskMode(c *gin.Context) {
	respOK(c, gin.H{"mode": a.AI.AskMode()})
}

// SetAskMode POST /api/v1/ai/tools/ask-mode {mode: strict|danger_only}
func (a *AIAPI) SetAskMode(c *gin.Context) {
	req, ok := bind[struct {
		Mode string `json:"mode"`
	}](c)
	if !ok {
		return
	}
	if err := a.AI.SetAskMode(req.Mode); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"mode": a.AI.AskMode()})
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
