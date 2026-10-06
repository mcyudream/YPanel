package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SelfUpdateAPI 面板自更新接口（admin）。
type SelfUpdateAPI struct {
	SU *service.SelfUpdateService
}

// Status GET /api/v1/system/update/status
func (a *SelfUpdateAPI) Status(c *gin.Context) {
	out, err := a.SU.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Apply POST /api/v1/system/update/apply
func (a *SelfUpdateAPI) Apply(c *gin.Context) {
	req, ok := bind[struct {
		File string `json:"file" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.SU.Apply(c.Request.Context(), req.File)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
