package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/errs"
)

// CtxPerms 当前用户权限集（map[string]struct{}，Auth 中间件注入，调用方只读）。
const CtxPerms = "perms"

// CtxPermDenied 本次请求被权限闸拒绝（Audit 中间件据此把写操作记为失败）。
const CtxPermDenied = "permDenied"

// RequirePerm 权限点闸门：keys 任一命中即放行（OR 语义）。
func RequirePerm(keys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		set, _ := c.Get(CtxPerms)
		permSet, _ := set.(map[string]struct{})
		for _, k := range keys {
			if rbac.Match(permSet, k) {
				c.Next()
				return
			}
		}
		c.Set(CtxPermDenied, true)
		abort(c, errs.ErrForbidden)
	}
}

// Perm 单权限点闸门的便捷形态（路由标注用）。
func Perm(key string) gin.HandlerFunc { return RequirePerm(key) }
