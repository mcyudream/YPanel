package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// NodeAPI 节点管理接口。
type NodeAPI struct {
	Nodes *service.NodeService
}

// List GET /api/v1/nodes
func (a *NodeAPI) List(c *gin.Context) {
	respOK(c, a.Nodes.ListNodes())
}

// PairingCode POST /api/v1/nodes/pairing-code（admin）
func (a *NodeAPI) PairingCode(c *gin.Context) {
	code, err := a.Nodes.GeneratePairingCode()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"code": code, "expireIn": 600})
}

// Delete DELETE /api/v1/nodes/:id（admin）
func (a *NodeAPI) Delete(c *gin.Context) {
	if err := a.Nodes.DeleteNode(c.Param("id")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Pair POST /api/v1/pair（公开：配对码交换，供 agent 调用）
func (a *NodeAPI) Pair(c *gin.Context) {
	req, ok := bind[struct {
		Code     string `json:"code" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Addr     string `json:"addr" binding:"required"`
		Hostname string `json:"hostname"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
	}](c)
	if !ok {
		return
	}
	token, err := a.Nodes.Pair(req.Code, req.Name, req.Addr, req.Hostname, req.OS, req.Arch, req.Version)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"token": token, "heartbeatInterval": 30})
}

// Heartbeat POST /api/v1/pair/heartbeat?name=（公开路由，节点 Bearer PSK 自证）
func (a *NodeAPI) Heartbeat(c *gin.Context) {
	name := c.Query("name")
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if name == "" || token == "" {
		respErr(c, errs.ErrBadRequest)
		return
	}
	if err := a.Nodes.Heartbeat(name, token); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
