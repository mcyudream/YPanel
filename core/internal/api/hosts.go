package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
)

// HostsAPI hosts 可视化编辑（M28）。
type HostsAPI struct {
	HS *service.HostsService
}

// ListRecords GET /api/v1/hosts/records
func (a *HostsAPI) ListRecords(c *gin.Context) {
	rows, err := a.HS.ListRecords()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, rows)
}

// SaveRecord POST /api/v1/hosts/records（id=0 新建 / 非 0 更新）。
func (a *HostsAPI) SaveRecord(c *gin.Context) {
	req, ok := bind[struct {
		ID        uint   `json:"id"`
		IP        string `json:"ip" binding:"required"`
		Hostnames string `json:"hostnames" binding:"required"`
		Comment   string `json:"comment"`
		Enabled   *bool  `json:"enabled"`
		Sort      int    `json:"sort"`
	}](c)
	if !ok {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.HostRecord{
		ID: req.ID, IP: req.IP, Hostnames: req.Hostnames,
		Comment: req.Comment, Enabled: enabled, Sort: req.Sort,
	}
	saved, err := a.HS.SaveRecord(c.Request.Context(), row)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, saved)
}

// DeleteRecord DELETE /api/v1/hosts/records/:id
func (a *HostsAPI) DeleteRecord(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.HS.DeleteRecord(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// SetRecordEnabled POST /api/v1/hosts/records/:id/enable | disable
func (a *HostsAPI) SetRecordEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		if err := a.HS.SetRecordEnabled(c.Request.Context(), id, enabled); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, gin.H{})
	}
}

// Status GET /api/v1/hosts/status
func (a *HostsAPI) Status(c *gin.Context) {
	res, err := a.HS.StatusAll(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, res)
}

// Apply POST /api/v1/hosts/apply（{nodeIds:[]} 应用/纳管）。
func (a *HostsAPI) Apply(c *gin.Context) {
	req, ok := bind[struct {
		NodeIDs []string `json:"nodeIds" binding:"required"`
	}](c)
	if !ok {
		return
	}
	res, err := a.HS.Apply(c.Request.Context(), req.NodeIDs)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"results": res})
}

// Remove POST /api/v1/hosts/remove（解除托管并还原托管块）。
func (a *HostsAPI) Remove(c *gin.Context) {
	req, ok := bind[struct {
		NodeID string `json:"nodeId" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.HS.RemoveNode(c.Request.Context(), req.NodeID); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}
