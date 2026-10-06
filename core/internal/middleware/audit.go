package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
)

// Audit 写操作审计：POST/PUT/DELETE 请求完成后落 AuditLog。
func Audit(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		switch c.Request.Method {
		case "POST", "PUT", "DELETE":
		default:
			return
		}
		// 登录/心跳等高频或敏感端点不重复记录（登录已有 LoginLog）
		path := c.Request.URL.Path
		if path == "/api/v1/auth/login" || path == "/api/v1/pair" || path == "/api/v1/pair/heartbeat" {
			return
		}
		row := model.AuditLog{
			Username:  c.GetString(CtxUsername),
			Method:    c.Request.Method,
			Path:      path,
			IP:        c.ClientIP(),
			Success:   c.Writer.Status() < 400,
			CreatedAt: time.Now(),
		}
		_ = db.Create(&row).Error
	}
}
