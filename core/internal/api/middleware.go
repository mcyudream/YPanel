// MiddlewareAPI 可复用中间件实例接口（M68）。
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// MiddlewareAPI 中间件实例接口。
type MiddlewareAPI struct {
	MW *service.MiddlewareService
}

// List GET /middleware/instances
func (a *MiddlewareAPI) List(c *gin.Context) {
	out, err := a.MW.List(c.Request.Context(), c.DefaultQuery("node", "local"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Types GET /middleware/types
func (a *MiddlewareAPI) Types(c *gin.Context) {
	respOK(c, a.MW.Types())
}

// Create POST /middleware/instances {name,type,host,port?,user?,password?,remark?}
func (a *MiddlewareAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type" binding:"required"`
		Host     string `json:"host" binding:"required"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Remark   string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	inst, err := a.MW.Create(c.Request.Context(), req.Name, req.Type, req.Host, req.Port, req.User, req.Password, req.Remark)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": inst.ID, "name": inst.Name})
}

// Update PUT /middleware/instances/:id {host?,port?,user?,password?,remark?}
func (a *MiddlewareAPI) Update(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Remark   string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	inst, err := a.MW.Update(c.Request.Context(), id, req.Host, req.Port, req.User, req.Password, req.Remark)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": inst.ID})
}

// Delete DELETE /middleware/instances/:id
func (a *MiddlewareAPI) Delete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.MW.Delete(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{})
}

// Test POST /middleware/instances/:id/test
func (a *MiddlewareAPI) Test(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.MW.Test(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"ok": true})
}

// Reveal GET /middleware/instances/:id/reveal（凭据查看）
func (a *MiddlewareAPI) Reveal(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.MW.Reveal(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// VersionRefresh POST /middleware/instances/:id/version {nodeId?}
func (a *MiddlewareAPI) VersionRefresh(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	nodeId := c.Query("node")
	v, err := a.MW.RefreshVersion(c.Request.Context(), id, nodeId)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"version": v})
}

// Scan GET /middleware/scan?node=
func (a *MiddlewareAPI) Scan(c *gin.Context) {
	out, err := a.MW.Scan(c.Request.Context(), c.DefaultQuery("node", "local"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Adopt POST /middleware/adopt {node?, containerName, name?}
func (a *MiddlewareAPI) Adopt(c *gin.Context) {
	req, ok := bind[struct {
		Node          string `json:"node"`
		ContainerName string `json:"containerName" binding:"required"`
		Name          string `json:"name"`
	}](c)
	if !ok {
		return
	}
	if node := c.Query("node"); node != "" {
		if req.Node != "" && req.Node != node {
			respErr(c, errs.Wrap(errs.ErrBadRequest, "请求中的节点不一致"))
			return
		}
		req.Node = node
	}
	inst, err := a.MW.Adopt(c.Request.Context(), defaultNode(req.Node), req.ContainerName, req.Name)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": inst.ID, "name": inst.Name, "type": inst.Type})
}

// ReuseOptions GET /middleware/reuse?type=&prefix=（商店安装向导复用候选：实例清单 + 版本校验，不回凭据）
func (a *MiddlewareAPI) ReuseOptions(c *gin.Context) {
	typ := c.Query("type")
	if typ == "" {
		respErr(c, errs.Wrap(errs.ErrBadRequest, "type 必填"))
		return
	}
	prefix := c.Query("prefix")
	out, err := a.MW.ReuseOptions(c.Request.Context(), typ, prefix)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

func defaultNode(n string) string {
	if n == "" {
		return "local"
	}
	return n
}
