// M49 系统安全工具 API：FTP 管理 / SSH 管理 / Fail2ban SSH 暴力破解防护。
package api

import (
	"fmt"
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// FtpAPI FTP（vsftpd）管理。
type FtpAPI struct {
	Ftp *service.FtpService
}

// Status GET /api/v1/ftp/status
func (a *FtpAPI) Status(c *gin.Context) {
	out, err := a.Ftp.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Install POST /api/v1/ftp/install
func (a *FtpAPI) Install(c *gin.Context) {
	out, err := a.Ftp.Install(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Power POST /api/v1/ftp/power {action}
func (a *FtpAPI) Power(c *gin.Context) {
	req, ok := bind[struct {
		Action string `json:"action" binding:"required,oneof=start stop restart"`
	}](c)
	if !ok {
		return
	}
	if err := a.Ftp.Power(c.Request.Context(), req.Action); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetPort POST /api/v1/ftp/port {port,pasvMin,pasvMax}
func (a *FtpAPI) SetPort(c *gin.Context) {
	req, ok := bind[struct {
		Port    int `json:"port" binding:"required,min=1024,max=65535"`
		PasvMin int `json:"pasvMin"`
		PasvMax int `json:"pasvMax"`
	}](c)
	if !ok {
		return
	}
	pasvMin, pasvMax := req.PasvMin, req.PasvMax
	if pasvMin == 0 && pasvMax == 0 {
		pasvMin, pasvMax = 40000, 40100
	}
	if err := a.Ftp.SetPort(c.Request.Context(), req.Port, pasvMin, pasvMax); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SshAPI SSH 安全管理（配置/密钥）。
type SshAPI struct {
	Ssh *service.SshGuardService
	F2B *service.Fail2banService
}

// GetConfig GET /api/v1/ssh/config
func (a *SshAPI) GetConfig(c *gin.Context) {
	out, err := a.Ssh.GetConfig(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SetConfig PUT /api/v1/ssh/config
func (a *SshAPI) SetConfig(c *gin.Context) {
	req, ok := bind[struct {
		Port            *int    `json:"port"`
		PasswordAuth    *bool   `json:"passwordAuth"`
		PubkeyAuth      *bool   `json:"pubkeyAuth"`
		PermitRootLogin *string `json:"permitRootLogin"`
	}](c)
	if !ok {
		return
	}
	if err := a.Ssh.SetConfig(c.Request.Context(), req.Port, req.PasswordAuth, req.PubkeyAuth, req.PermitRootLogin); err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Ssh.GetConfig(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Keys GET /api/v1/ssh/keys
func (a *SshAPI) Keys(c *gin.Context) {
	out, err := a.Ssh.ListKeys(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// FailedAttempts GET /api/v1/ssh/attempts?limit=（SSH 失败登录聚合，标记已封禁）
func (a *SshAPI) FailedAttempts(c *gin.Context) {
	n := 100
	_, _ = fmt.Sscanf(c.DefaultQuery("limit", "100"), "%d", &n)
	out, err := a.Ssh.FailedAttemptsMarked(c.Request.Context(), a.F2B, n)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
