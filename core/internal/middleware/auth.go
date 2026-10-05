// Package middleware gin 中间件：会话鉴权、CORS、访问日志。
package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// CtxKeys gin context 键。
const (
	CtxUID      = "uid"
	CtxUsername = "username"
	CtxRole     = "role"
)

// Auth 会话鉴权。
// token 来源：Authorization: Bearer（首选）；兼容 ?token=（WS/下载等无法带 header 的场景）。
func Auth(auth *service.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			abort(c, errs.ErrUnauthorized)
			return
		}
		claims, err := auth.ParseToken(token)
		if err != nil {
			abort(c, errs.ErrUnauthorized)
			return
		}
		c.Set(CtxUID, claims.UID)
		c.Set(CtxUsername, claims.Username)
		if u, err := auth.ByID(claims.UID); err == nil {
			c.Set(CtxRole, u.Role)
		}
		c.Next()
	}
}

// Admin 仅管理员。
func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(CtxRole) != "admin" {
			abort(c, errs.ErrForbidden)
			return
		}
		c.Next()
	}
}

func bearerToken(h string) string {
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return h[len(prefix):]
	}
	return ""
}

func abort(c *gin.Context, err *errs.Error) {
	c.AbortWithStatusJSON(200, gin.H{"code": err.Code, "message": err.Message})
}

// CORS 开发期跨域（vite dev 端口）；生产同源部署时形同虚设。
func CORS() gin.HandlerFunc {
	allow := map[string]bool{
		"http://localhost:5173": true, "http://127.0.0.1:5173": true,
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allow[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AccessLog 访问日志（跳过 /health）。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		slog.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
			"cost", time.Since(start).Round(time.Millisecond).String(),
		)
	}
}
