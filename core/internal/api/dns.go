package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
)

// DnsAPI 内网 DNS（dnsmasq 自管部署，M27）。
type DnsAPI struct {
	DNS *service.DnsService
	NF  *service.NatForwardService // 复用网卡/IP 探测
}

// Overview GET /api/v1/dns/overview?nodeId=
func (a *DnsAPI) Overview(c *gin.Context) {
	ov, err := a.DNS.Overview(c.Request.Context(), c.Query("nodeId"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, ov)
}

// ListRecords GET /api/v1/dns/records
func (a *DnsAPI) ListRecords(c *gin.Context) {
	rows, err := a.DNS.ListRecords()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, rows)
}

// SaveRecord POST /api/v1/dns/records（id=0 新建 / 非 0 更新）。
func (a *DnsAPI) SaveRecord(c *gin.Context) {
	req, ok := bind[struct {
		ID      uint   `json:"id"`
		Type    string `json:"type" binding:"required"`
		Domain  string `json:"domain" binding:"required"`
		Target  string `json:"target" binding:"required"`
		Enabled *bool  `json:"enabled"`
		Comment string `json:"comment"`
		Sort    int    `json:"sort"`
	}](c)
	if !ok {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.DnsRecord{
		ID: req.ID, Type: req.Type, Domain: req.Domain, Target: req.Target,
		Enabled: enabled, Comment: req.Comment, Sort: req.Sort,
	}
	saved, err := a.DNS.SaveRecord(c.Request.Context(), row)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, saved)
}

// DeleteRecord DELETE /api/v1/dns/records/:id
func (a *DnsAPI) DeleteRecord(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DNS.DeleteRecord(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// SetRecordEnabled POST /api/v1/dns/records/:id/enable | disable
func (a *DnsAPI) SetRecordEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		if err := a.DNS.SetRecordEnabled(c.Request.Context(), id, enabled); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, gin.H{})
	}
}

// GetSettings GET /api/v1/dns/settings
func (a *DnsAPI) GetSettings(c *gin.Context) {
	respOK(c, a.DNS.GetConfig())
}

// SaveSettings PUT /api/v1/dns/settings
func (a *DnsAPI) SaveSettings(c *gin.Context) {
	req, ok := bind[service.DnsConfig](c)
	if !ok {
		return
	}
	if err := a.DNS.SaveConfig(c.Request.Context(), *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, a.DNS.GetConfig())
}

// Deploy POST /api/v1/dns/deploy
func (a *DnsAPI) Deploy(c *gin.Context) {
	req, ok := bind[struct {
		NodeID string `json:"nodeId" binding:"required"`
	}](c)
	if !ok {
		return
	}
	warnings, err := a.DNS.Deploy(c.Request.Context(), req.NodeID)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"warnings": warnings})
}

// Undeploy POST /api/v1/dns/undeploy
func (a *DnsAPI) Undeploy(c *gin.Context) {
	req, ok := bind[struct {
		NodeID      string `json:"nodeId" binding:"required"`
		RemoveFiles bool   `json:"removeFiles"`
	}](c)
	if !ok {
		return
	}
	if err := a.DNS.Undeploy(c.Request.Context(), req.NodeID, req.RemoveFiles); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// Apply POST /api/v1/dns/apply（手动重载）。
func (a *DnsAPI) Apply(c *gin.Context) {
	req, ok := bind[struct {
		NodeID string `json:"nodeId" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.DNS.ApplyNode(c.Request.Context(), req.NodeID); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// CheckResolve POST /api/v1/dns/check
func (a *DnsAPI) CheckResolve(c *gin.Context) {
	req, ok := bind[struct {
		NodeID string `json:"nodeId" binding:"required"`
		Domain string `json:"domain" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.DNS.CheckResolve(c.Request.Context(), req.NodeID, req.Domain)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"output": out})
}

// Interfaces GET /api/v1/dns/interfaces?nodeId=&family=（复用 NAT 探测）。
func (a *DnsAPI) Interfaces(c *gin.Context) {
	family := 0
	if v := c.Query("family"); v != "" {
		family, _ = strconv.Atoi(v)
	}
	ifaces, err := a.NF.GetInterfaces(c.Request.Context(), c.Query("nodeId"), family)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, ifaces)
}
