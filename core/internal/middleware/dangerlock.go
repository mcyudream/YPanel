// DangerLock 危险操作锁定中间件（M48）：锁定开启时拒绝高危端点。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// DangerLock 危险操作锁定（挂 Auth 之后：仅对登录请求生效）。
func DangerLock(sec *service.SecuritySettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := sec.CheckDangerLock(c.Request.Method, c.FullPath()); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, errs.RespErr(errs.From(err)))
			return
		}
		c.Next()
	}
}
