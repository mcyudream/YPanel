// SystemToolAPI 系统工具箱接口（M40）。
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SystemToolAPI 系统管理动作。
type SystemToolAPI struct {
	Tools *service.SystemToolService
	Sec   *service.SecuritySettingsService
}

// SwapStatus GET /api/v1/system/swap
func (a *SystemToolAPI) SwapStatus(c *gin.Context) {
	out, err := a.Tools.SwapStatus(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SwapApply POST /api/v1/system/swap {sizeGB}
func (a *SystemToolAPI) SwapApply(c *gin.Context) {
	req, ok := bind[struct {
		SizeGB int `json:"sizeGB"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Tools.SwapApply(c.Request.Context(), req.SizeGB)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// BBRStatus GET /api/v1/system/bbr
func (a *SystemToolAPI) BBRStatus(c *gin.Context) {
	out, err := a.Tools.BBRStatus(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// BBRApply POST /api/v1/system/bbr {enable}
func (a *SystemToolAPI) BBRApply(c *gin.Context) {
	req, ok := bind[struct {
		Enable bool `json:"enable"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Tools.BBRApply(c.Request.Context(), req.Enable)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Clean POST /api/v1/system/clean
func (a *SystemToolAPI) Clean(c *gin.Context) {
	out, err := a.Tools.SystemClean(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"output": out})
}

// GetPasswordPolicy GET /api/v1/security/password-policy
func (a *SystemToolAPI) GetPasswordPolicy(c *gin.Context) {
	respOK(c, gin.H{"minPasswordLen": a.Sec.Get(c.Request.Context()).MinPasswordLen})
}

// PutPasswordPolicy PUT /api/v1/security/password-policy {minPasswordLen}
func (a *SystemToolAPI) PutPasswordPolicy(c *gin.Context) {
	req, ok := bind[struct {
		MinPasswordLen int `json:"minPasswordLen"`
	}](c)
	if !ok {
		return
	}
	if err := a.Sec.SetMinPasswordLen(req.MinPasswordLen); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
