package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// FirewallAPI 防火墙接口。
type FirewallAPI struct {
	FW *service.FirewallService
}

// Status GET /api/v1/firewall/status
func (a *FirewallAPI) Status(c *gin.Context) {
	out, err := a.FW.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Allow POST /api/v1/firewall/allow
func (a *FirewallAPI) Allow(c *gin.Context) {
	req, ok := bind[struct {
		Port  string `json:"port" binding:"required"`
		Proto string `json:"proto"`
	}](c)
	if !ok {
		return
	}
	if err := a.FW.Allow(c.Request.Context(), req.Port, req.Proto); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DeleteRule DELETE /api/v1/firewall/rules/:number
func (a *FirewallAPI) DeleteRule(c *gin.Context) {
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.FW.DeleteRule(c.Request.Context(), n); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetEnabled POST /api/v1/firewall/enable | disable
func (a *FirewallAPI) SetEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := a.FW.SetEnabled(c.Request.Context(), enabled); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}
