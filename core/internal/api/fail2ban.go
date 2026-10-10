package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// Fail2banAPI 入侵防护接口。
type Fail2banAPI struct {
	F2B *service.Fail2banService
}

// Status GET /api/v1/fail2ban/status
func (a *Fail2banAPI) Status(c *gin.Context) {
	f2b, err := a.F2B.WithNode(c.Query("nodeId"))
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := f2b.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Install POST /api/v1/fail2ban/install（一键安装：apt/dnf/yum 多通道 + enable --now）。
func (a *Fail2banAPI) Install(c *gin.Context) {
	f2b, err := a.F2B.WithNode(c.Query("nodeId"))
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := f2b.Install(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Unban POST /api/v1/fail2ban/unban {jail, ip}
func (a *Fail2banAPI) Unban(c *gin.Context) {
	f2b, err := a.F2B.WithNode(c.Query("nodeId"))
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Jail string `json:"jail" binding:"required"`
		IP   string `json:"ip" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := service.ValidateJail(req.Jail); err != nil {
		respErr(c, err)
		return
	}
	if err := f2b.Unban(c.Request.Context(), req.Jail, req.IP); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Ban POST /api/v1/fail2ban/ban {jail, ip}
func (a *Fail2banAPI) Ban(c *gin.Context) {
	f2b, err := a.F2B.WithNode(c.Query("nodeId"))
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Jail string `json:"jail" binding:"required"`
		IP   string `json:"ip" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := service.ValidateJail(req.Jail); err != nil {
		respErr(c, err)
		return
	}
	if err := f2b.Ban(c.Request.Context(), req.Jail, req.IP); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
