package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
)

// RevisionAPI 受管配置版本接口（M23）。
type RevisionAPI struct {
	Rev *service.RevisionService
}

// List GET /api/v1/config-revisions?node=&path=
func (r *RevisionAPI) List(c *gin.Context) {
	out, err := r.Rev.List(c.Request.Context(), c.DefaultQuery("node", "local"), c.Query("path"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Get GET /api/v1/config-revisions/:id
func (r *RevisionAPI) Get(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := r.Rev.Get(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Restore POST /api/v1/config-revisions/:id/restore
func (r *RevisionAPI) Restore(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	rev, err := r.Rev.Restore(c.Request.Context(), id, c.GetString(middleware.CtxUsername))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, rev)
}
