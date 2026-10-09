package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// DashboardAPI 首页聚合接口（面板级计数 + local docker 计数 + 到期证书 + 最近通知）。
type DashboardAPI struct {
	Svc *service.DashboardService
}

// Dashboard GET /api/v1/system/dashboard
func (a *DashboardAPI) Dashboard(c *gin.Context) {
	out, err := a.Svc.Get(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
