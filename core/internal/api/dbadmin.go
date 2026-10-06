package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// DBAdminAPI db-admin 插件接口（只读浏览与查询）。
type DBAdminAPI struct {
	Admin *service.DBAdminService
}

// Instances GET /plugin/db-admin/instances
func (a *DBAdminAPI) Instances(c *gin.Context) {
	out, err := a.Admin.Instances(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Databases GET /plugin/db-admin/:id/databases
func (a *DBAdminAPI) Databases(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Databases(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Tables GET /plugin/db-admin/:id/tables?db=
func (a *DBAdminAPI) Tables(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Admin.Tables(c.Request.Context(), id, c.Query("db"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Query POST /plugin/db-admin/:id/query {db, sql}
func (a *DBAdminAPI) Query(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB  string `json:"db" binding:"required"`
		SQL string `json:"sql" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Admin.Query(c.Request.Context(), id, req.DB, req.SQL)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
