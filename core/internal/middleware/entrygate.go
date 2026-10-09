// EntryGate 安全入口路径门禁（M49 严格模式）：入口开启后，除资源/API 外
// 仅 <入口> 路径可加载面板；根路径与其他任何路径一律 404 伪装（无论有无 cookie）。
// 访问 <入口> 时种 30 天 HttpOnly cookie，供登录 API 的 SecurityGate 检查使用。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// EntryCookieName 入口凭证 cookie（SecurityGate 登录检查同源）。
const EntryCookieName = "yp_entry_ok"

// EntryGate 安全入口路径门禁（须挂中间件链最前）。
func EntryGate(sec *service.SecuritySettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		entry := strings.Trim(sec.SafeEntry(), "/ ")
		// 未启用安全入口：全放行
		if entry == "" {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		// 健康检查 / API / 静态资源：放行（API 受 JWT + 危险锁定管控；资源不含业务信息）
		if path == "/health" ||
			strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/plugin/") ||
			strings.HasPrefix(path, "/assets/") ||
			strings.HasPrefix(path, "/snap/") ||
			strings.HasPrefix(path, "/browser_upgrade") ||
			path == "/favicon.svg" || path == "/favicon.ico" {
			c.Next()
			return
		}
		// 入口路径：命中即种 30 天 cookie，随后由 NoRoute 回退 index.html
		if strings.TrimSuffix(path, "/") == "/"+entry {
			c.SetCookie(EntryCookieName, "1", 30*24*3600, "/", "", false, true)
			c.Next()
			return
		}
		// 已授权浏览器：入口凭证 cookie 或登录会话 cookie 任一存在即放行（刷新/直达面板页）
		if ck, err := c.Cookie(EntryCookieName); err == nil && ck == "1" {
			c.Next()
			return
		}
		if sk, err := c.Cookie("yp_session"); err == nil && sk == "1" {
			c.Next()
			return
		}
		// 严格模式：其余一律 404 伪装（含根路径与未授权浏览器的访问）
		c.String(http.StatusNotFound, "404 page not found")
		c.Abort()
	}
}
