package api

import (
	"fmt"
	"strings"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// ComposeAPI compose 项目接口（代理 agent）。
type ComposeAPI struct {
	Nodes *service.NodeService
}

func (c *ComposeAPI) client(ctx *gin.Context) *agentclient.Client {
	node, _ := c.Nodes.ByID(ctx.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
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

// ServiceAction POST /api/v1/compose/service-action {project, service, action}
func (c *ComposeAPI) ServiceAction(ctx *gin.Context) {
	req, ok := bind[struct {
		Project string `json:"project" binding:"required"`
		Service string `json:"service" binding:"required"`
		Action  string `json:"action" binding:"required,oneof=start stop restart pull"`
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
