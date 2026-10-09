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

// CheckOnline GET /api/v1/system/update/check（GitHub/Gitee 双源并行探测）
func (a *SelfUpdateAPI) CheckOnline(c *gin.Context) {
	out, err := a.SU.CheckOnline(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Upgrade POST /api/v1/system/update/upgrade {source: github|gitee}（下载→校验→替换→重启）
func (a *SelfUpdateAPI) Upgrade(c *gin.Context) {
	req, ok := bind[struct {
		Source string `json:"source" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.SU.Upgrade(c.Request.Context(), req.Source)
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
