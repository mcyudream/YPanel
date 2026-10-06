package api

import (
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
