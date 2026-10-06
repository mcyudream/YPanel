package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SecurityAPI 安全基线设置接口（admin）。
type SecurityAPI struct {
	Sec  *service.SecuritySettingsService
	Auth *service.Auth
}

// Get GET /api/v1/security/settings
func (a *SecurityAPI) Get(c *gin.Context) {
	respOK(c, a.Sec.Get(c.Request.Context()))
}

// Update PUT /api/v1/security/settings {safeEntry, allowedIps, sessionHours}
func (a *SecurityAPI) Update(c *gin.Context) {
	req, ok := bind[struct {
		SafeEntry    string `json:"safeEntry"`
		AllowedIPs   string `json:"allowedIps"`
		SessionHours int    `json:"sessionHours" binding:"min=1,max=720"`
	}](c)
	if !ok {
		return
	}
	if err := a.Sec.Update(c.Request.Context(), req.SafeEntry, req.AllowedIPs, req.SessionHours, clientIP(c)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.Sec.Get(c.Request.Context()))
}
