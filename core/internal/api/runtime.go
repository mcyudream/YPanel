package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// RuntimeAPI PHP 运行环境接口。
type RuntimeAPI struct {
	RT *service.RuntimeService
}

// List GET /api/v1/runtimes
func (a *RuntimeAPI) List(c *gin.Context) {
	out, err := a.RT.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/runtimes
func (a *RuntimeAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Name    string `json:"name" binding:"required"`
		Version string `json:"version" binding:"required"`
	}](c)
	if !ok {
		return
	}
	row, err := a.RT.Create(c.Request.Context(), req.Name, req.Version)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Delete DELETE /api/v1/runtimes/:id
func (a *RuntimeAPI) Delete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.RT.Delete(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetEnabled POST /api/v1/runtimes/:id/start | stop
func (a *RuntimeAPI) SetEnabled(up bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		if err := a.RT.SetEnabled(c.Request.Context(), id, up); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}
