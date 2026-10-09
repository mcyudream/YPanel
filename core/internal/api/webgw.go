// M51 内网浏览器：代理会话管理 API。
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// WebGwAPI 内网浏览器代理会话（gw 网关端口的服务端配对）。
type WebGwAPI struct {
	GW *service.WebGwService
}

// Create POST /api/v1/webgw/session {url} → {sid, target}；同时签发网关端口鉴权 Cookie。
// Cookie 值为随机不透明令牌（非 JWT），Path=/ —— 根路径逃逸请求（目标应用的 /assets 等绝对路径）
// 带不上 Path=/s 的 Cookie，统一靠该令牌过网关门禁。
func (a *WebGwAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		URL string `json:"url" binding:"required"`
	}](c)
	if !ok {
		return
	}
	uid, _ := c.Get(middleware.CtxUID)
	var uidV uint
	if v, is := uid.(uint); is {
		uidV = v
	}
	ses, err := a.GW.Create(req.URL, uidV)
	if err != nil {
		respErr(c, err)
		return
	}
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(service.WebGwCookie, a.GW.EnsureToken(uidV), 8*24*3600, "/", "", secure, true)
	respOK(c, gin.H{"sid": ses.Sid, "target": ses.Target, "expiresAt": ses.ExpiresAt})
}

// Get GET /api/v1/webgw/session/:sid → 窗口重挂载时的复用校验。
func (a *WebGwAPI) Get(c *gin.Context) {
	ses := a.GW.Get(c.Param("sid"))
	if ses == nil {
		respErr(c, errs.ErrNotFound)
		return
	}
	respOK(c, gin.H{"sid": ses.Sid, "target": ses.Target, "expiresAt": ses.ExpiresAt})
}

// Delete DELETE /api/v1/webgw/session/:sid（关窗 best effort 销毁）。
func (a *WebGwAPI) Delete(c *gin.Context) {
	a.GW.Delete(c.Param("sid"))
	respOK(c, gin.H{"ok": true})
}

// NodeTargets GET /api/v1/webgw/targets?node=（M50：节点可浏览目标发现 + 可达性预检）
func (a *WebGwAPI) NodeTargets(c *gin.Context) {
	out, err := a.GW.NodeTargets(c.Request.Context(), c.DefaultQuery("node", "local"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
