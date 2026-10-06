package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
)

// NatForwardAPI NAT 端口转发接口（M24）。
type NatForwardAPI struct {
	NF *service.NatForwardService
}

// List GET /api/v1/nat/forwards?nodeId=
func (a *NatForwardAPI) List(c *gin.Context) {
	rows, err := a.NF.List(c.Query("nodeId"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, rows)
}

// Save POST /api/v1/nat/forwards（id=0 新建 / 非 0 更新）。
func (a *NatForwardAPI) Save(c *gin.Context) {
	req, ok := bind[struct {
		ID            uint   `json:"id"`
		NodeID        string `json:"nodeId" binding:"required"`
		Name          string `json:"name" binding:"required"`
		Protocol      string `json:"protocol" binding:"required"`
		IPFamily      int    `json:"ipFamily"`
		ListenPort    int    `json:"listenPort"`
		ListenPortEnd int    `json:"listenPortEnd"`
		TargetIP      string `json:"targetIp" binding:"required"`
		TargetPort    int    `json:"targetPort"`
		TargetPortEnd int    `json:"targetPortEnd"`
		Iface         string `json:"iface"`
		Enabled       *bool  `json:"enabled"`
		Sort          int    `json:"sort"`
	}](c)
	if !ok {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.NatForwardRule{
		ID: req.ID, NodeID: req.NodeID, Name: req.Name, Protocol: req.Protocol,
		IPFamily: req.IPFamily, ListenPort: req.ListenPort, ListenPortEnd: req.ListenPortEnd,
		TargetIP: req.TargetIP, TargetPort: req.TargetPort, TargetPortEnd: req.TargetPortEnd,
		Iface: req.Iface, Enabled: enabled, Sort: req.Sort,
	}
	saved, warnings, err := a.NF.Save(c.Request.Context(), row)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"rule": saved, "warnings": warnings})
}

// Delete DELETE /api/v1/nat/forwards/:id
func (a *NatForwardAPI) Delete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	warnings, err := a.NF.Delete(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"warnings": warnings})
}

// SetEnabled POST /api/v1/nat/forwards/:id/enable | disable
func (a *NatForwardAPI) SetEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		warnings, err := a.NF.SetEnabled(c.Request.Context(), id, enabled)
		if err != nil {
			respErr(c, err)
			return
		}
		respOK(c, gin.H{"warnings": warnings})
	}
}

// Interfaces GET /api/v1/nat/interfaces?nodeId=&family=
func (a *NatForwardAPI) Interfaces(c *gin.Context) {
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

// CheckPort POST /api/v1/nat/check-port（表单即时校验，命中占用返回明细不报错）。
func (a *NatForwardAPI) CheckPort(c *gin.Context) {
	req, ok := bind[struct {
		NodeID        string `json:"nodeId" binding:"required"`
		Protocol      string `json:"protocol" binding:"required"`
		ListenPort    int    `json:"listenPort"`
		ListenPortEnd int    `json:"listenPortEnd"`
	}](c)
	if !ok {
		return
	}
	occupied, err := a.NF.CheckPort(c.Request.Context(), req.NodeID, req.Protocol, req.ListenPort, req.ListenPortEnd)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"occupied": occupied})
}

// Apply POST /api/v1/nat/apply（手动整链重放）。
func (a *NatForwardAPI) Apply(c *gin.Context) {
	req, ok := bind[struct {
		NodeID string `json:"nodeId" binding:"required"`
	}](c)
	if !ok {
		return
	}
	warnings, err := a.NF.ApplyNode(c.Request.Context(), req.NodeID)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"warnings": warnings})
}
