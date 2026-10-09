package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// ComposeAPI compose 项目接口（代理 agent）。
type ComposeAPI struct {
	Nodes *service.NodeService
	Src2  *service.Src2ComposeService
	Creds *service.GitCredService
}

func (c *ComposeAPI) client(ctx *gin.Context) *agentclient.Client {
	node, _ := c.Nodes.ByID(ctx.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

func uintAt(s string) uint {
	n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return uint(n)
}

// List GET /api/v1/compose/projects
func (c *ComposeAPI) List(ctx *gin.Context) {
	out, err := agentclient.GetJSON[[]dto.ComposeProject](c.client(ctx), ctx.Request.Context(), "/agent/v1/compose/projects")
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}

// Topology GET /api/v1/compose/topology?name=&dir=（M26 P1：项目服务拓扑）
func (c *ComposeAPI) Topology(ctx *gin.Context) {
	q := "?name=" + escape(ctx.Query("name")) + "&dir=" + escape(ctx.Query("dir"))
	out, err := agentclient.GetJSON[dto.ComposeTopology](c.client(ctx), ctx.Request.Context(), "/agent/v1/compose/topology"+q)
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}

// Config GET /api/v1/compose/config?name=&dir=
func (c *ComposeAPI) Config(ctx *gin.Context) {
	q := "?name=" + escape(ctx.Query("name")) + "&dir=" + escape(ctx.Query("dir"))
	out, err := agentclient.GetJSON[dto.ComposeConfigResp](c.client(ctx), ctx.Request.Context(), "/agent/v1/compose/config"+q)
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}

// Write POST /api/v1/compose/config
func (c *ComposeAPI) Write(ctx *gin.Context) {
	req, ok := bind[dto.ComposeWriteReq](ctx)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.ComposeWriteReq, struct{}](c.client(ctx), ctx.Request.Context(), http.MethodPost, "/agent/v1/compose/config", req); err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, struct{}{})
}

// Action POST /api/v1/compose/up|down
func (c *ComposeAPI) Action(action string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := bind[dto.ComposeActionReq](ctx)
		if !ok {
			return
		}
		out, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](c.client(ctx), ctx.Request.Context(), http.MethodPost, "/agent/v1/compose/"+action, req)
		if err != nil {
			respErr(ctx, err)
			return
		}
		respOK(ctx, out)
	}
}

// Logs GET /api/v1/compose/logs?name=&dir=&tail=&follow=&service=
func (c *ComposeAPI) Logs(ctx *gin.Context) {
	q := "?name=" + escape(ctx.Query("name")) + "&dir=" + escape(ctx.Query("dir")) +
		"&tail=" + escape(ctx.DefaultQuery("tail", "500")) +
		"&follow=" + escape(ctx.DefaultQuery("follow", "0")) +
		"&service=" + escape(ctx.Query("service"))
	req, err := c.client(ctx).NewRequest(ctx.Request.Context(), http.MethodGet, "/agent/v1/compose/logs"+q, nil)
	if err != nil {
		respErr(ctx, err)
		return
	}
	resp, err := c.client(ctx).HTTP.Do(req)
	if err != nil {
		respErr(ctx, errAgentUnreach(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		respErr(ctx, errAgentUnreach(nil))
		return
	}
	ctx.Header("Content-Type", "text/plain; charset=utf-8")
	ctx.Status(http.StatusOK)
	flusher, _ := ctx.Writer.(http.Flusher)
	buf := make([]byte, 8192)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := ctx.Writer.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			return
		}
	}
}

// ScanCompose GET /api/v1/compose/scan?dir=（B12：外部项目发现）
func (c *ComposeAPI) ScanCompose(ctx *gin.Context) {
	q := "?dir=" + escape(ctx.Query("dir"))
	out, err := agentclient.GetJSON[[]map[string]any](c.client(ctx), ctx.Request.Context(), "/agent/v1/compose/scan"+q)
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}

// AdoptCompose POST /api/v1/compose/adopt {dir}（B12：复制外部项目入托管）
func (c *ComposeAPI) AdoptCompose(ctx *gin.Context) {
	req, ok := bind[struct {
		Dir string `json:"dir" binding:"required"`
	}](ctx)
	if !ok {
		return
	}
	name := req.Dir[strings.LastIndex(req.Dir, "/")+1:]
	if name == "" {
		respErr(ctx, errBadRequest("目录不合法"))
		return
	}
	target := "/opt/ypanel/compose/" + name
	if _, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](c.client(ctx), ctx.Request.Context(),
		"POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("mkdir -p %s && cp %s/docker-compose.* %s/ 2>/dev/null; cp %s/compose.* %s/ 2>/dev/null; ls %s/docker-compose.* %s/compose.* 2>/dev/null | head -1", target, req.Dir, target, req.Dir, target, target, target), TimeoutSecs: 30}); err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, gin.H{"adopted": name, "dir": target})
}

// ProjectDelete DELETE /api/v1/compose/projects/:name（down + 移除编排目录，含数据）
func (c *ComposeAPI) ProjectDelete(ctx *gin.Context) {
	if _, err := agentclient.DoJSON[struct{}, struct{}](c.client(ctx), ctx.Request.Context(), http.MethodDelete,
		"/agent/v1/compose/projects/"+escape(ctx.Param("name")), nil); err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, struct{}{})
}

// ServiceAction POST /api/v1/compose/service-action {project, service, action}
func (c *ComposeAPI) ServiceAction(ctx *gin.Context) {
	req, ok := bind[struct {
		Project string `json:"project" binding:"required"`
		Service string `json:"service" binding:"required"`
		Action  string `json:"action" binding:"required,oneof=start stop restart pull up"`
	}](ctx)
	if !ok {
		return
	}
	q := "?project=" + escape(req.Project) + "&service=" + escape(req.Service) + "&action=" + escape(req.Action)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](c.client(ctx), ctx.Request.Context(), http.MethodPost,
		"/agent/v1/compose/service-action"+q, &dto.ExecReq{Command: req.Project, TimeoutSecs: 300})
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, gin.H{"output": out.Output})
}

// Src2Templates GET /api/v1/compose/src2compose/templates（支持的语言栈）
func (c *ComposeAPI) Src2Templates(ctx *gin.Context) {
	respOK(ctx, service.SrcLangs)
}

// Src2PreviewStream GET /api/v1/compose/src2compose/preview/stream（SSE：分步进度 + 最终候选）
func (c *ComposeAPI) Src2PreviewStream(ctx *gin.Context) {
	req := &dto.Src2ComposePreviewReq{
		GitURL:       ctx.Query("gitUrl"),
		Branch:       ctx.Query("branch"),
		CredentialID: uintAt(ctx.DefaultQuery("credentialId", "0")),
	}
	w := ctx.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		respErr(ctx, errBadRequest("当前连接不支持流式响应"))
		return
	}
	send := func(v gin.H) {
		b, _ := json.Marshal(v)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}
	det, sugg, err := c.Src2.PreviewStream(ctx.Request.Context(), *req, func(text string) {
		send(gin.H{"type": "log", "text": text})
	})
	if err != nil {
		send(gin.H{"type": "error", "message": err.Error()})
		return
	}
	send(gin.H{"type": "done", "commit": det.Commit, "suggestions": sugg})
}

// Src2Create POST /api/v1/compose/src2compose（创建构建任务）
func (c *ComposeAPI) Src2Create(ctx *gin.Context) {
	req, ok := bind[dto.Src2ComposeBuildReq](ctx)
	if !ok {
		return
	}
	out, err := c.Src2.Create(ctx.Request.Context(), *req)
	if err != nil {
		respErr(ctx, err)
		return
	}
	respOK(ctx, out)
}
