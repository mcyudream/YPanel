package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// ProbeAPI 监控探针接口（M29）。
type ProbeAPI struct {
	Probes *service.ProbeService
}

// List GET /api/v1/probes
func (a *ProbeAPI) List(c *gin.Context) {
	out, err := a.Probes.List()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/probes（默认停用）
func (a *ProbeAPI) Create(c *gin.Context) {
	req, ok := bind[service.ProbeInput](c)
	if !ok {
		return
	}
	row, err := a.Probes.Create(*req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Update PUT /api/v1/probes/:id
func (a *ProbeAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("参数不合法"))
		return
	}
	req, ok := bind[service.ProbeInput](c)
	if !ok {
		return
	}
	row, err := a.Probes.Update(uint(id), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Delete DELETE /api/v1/probes/:id
func (a *ProbeAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("参数不合法"))
		return
	}
	if err := a.Probes.Delete(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetEnabled POST /api/v1/probes/:id/enable|disable
func (a *ProbeAPI) SetEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			respErr(c, errBadRequest("参数不合法"))
			return
		}
		if err := a.Probes.SetEnabled(uint(id), enabled); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}

// Test POST /api/v1/probes/test（试跑一次，不改状态不发通知）
func (a *ProbeAPI) Test(c *gin.Context) {
	req, ok := bind[service.ProbeInput](c)
	if !ok {
		return
	}
	out, err := a.Probes.Test(*req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
