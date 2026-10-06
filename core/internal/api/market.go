package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// MarketAPI 应用市场接口。
type MarketAPI struct {
	Market *service.MarketService
}

// List GET /api/v1/market/apps
func (a *MarketAPI) List(c *gin.Context) {
	out, err := a.Market.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Installed GET /api/v1/market/installed
func (a *MarketAPI) Installed(c *gin.Context) {
	out, err := a.Market.Installed(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Install POST /api/v1/market/install
func (a *MarketAPI) Install(c *gin.Context) {
	req, ok := bind[struct {
		AppID  string            `json:"appId" binding:"required"`
		Params map[string]string `json:"params"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Market.Install(c.Request.Context(), req.AppID, req.Params)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
