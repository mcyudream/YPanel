package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// DiskGuardAPI 磁盘空间保护接口（admin）。
type DiskGuardAPI struct {
	G *service.DiskGuardService
}

// Status GET /api/v1/diskguard/status → 配置 + 事件 + 各节点实时磁盘
func (a *DiskGuardAPI) Status(c *gin.Context) {
	respOK(c, a.G.Status(c.Request.Context()))
}

// UpdateConfig PUT /api/v1/diskguard/config {enabled, thresholdGB, exclude}
func (a *DiskGuardAPI) UpdateConfig(c *gin.Context) {
	req, ok := bind[struct {
		Enabled     bool     `json:"enabled"`
		ThresholdGB int      `json:"thresholdGB" binding:"min=1,max=500"`
		Exclude     []string `json:"exclude"`
	}](c)
	if !ok {
		return
	}
	cfg, err := a.G.UpdateConfig(req.Enabled, req.ThresholdGB, req.Exclude)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, cfg)
}

// Restore POST /api/v1/diskguard/restore {eventId}（0/缺省 = 恢复全部未恢复事件）
func (a *DiskGuardAPI) Restore(c *gin.Context) {
	req, ok := bind[struct {
		EventID uint `json:"eventId"`
	}](c)
	if !ok {
		return
	}
	started, failed, err := a.G.Restore(c.Request.Context(), req.EventID)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"started": started, "failed": failed})
}
