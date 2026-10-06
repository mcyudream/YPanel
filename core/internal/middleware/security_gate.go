// SecurityGate 安全入口 + IP 白名单（面板全局门禁，login/health 例外）。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SecurityGate 面板门禁中间件。
func SecurityGate(sec *service.SecuritySettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// health 与前端静态资源永不拦截
		if path == "/health" || (!strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/plugin/")) {
			c.Next()
			return
		}
		// 登录接口走安全入口检查（启用时需带 ?entry=<段> 或 X-Safe-Entry 头）
		if path == "/api/v1/auth/login" {
			entry := sec.SafeEntry()
			if entry != "" {
				e := c.Query("entry")
				if e != entry && c.GetHeader("X-Safe-Entry") != entry {
					c.AbortWithStatus(http.StatusNotFound)
					return
				}
			}
			if !sec.IsIPAllowed(c.ClientIP()) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}
		// 其余 API：IP 白名单（已认证会话同样受限）
		if !sec.IsIPAllowed(c.ClientIP()) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
