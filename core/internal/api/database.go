package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// DatabaseAPI 数据库实例接口。
type DatabaseAPI struct {
	DBS *service.DatabaseService
}

// List GET /api/v1/database/instances
func (a *DatabaseAPI) List(c *gin.Context) {
	out, err := a.DBS.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/database/instances
func (a *DatabaseAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type" binding:"required,oneof=mysql postgres redis mongo"`
		Port     int    `json:"port"`
		Password string `json:"password"`
	}](c)
	if !ok {
		return
	}
	inst, err := a.DBS.CreateInstance(c.Request.Context(), req.Name, req.Type, req.Port, req.Password)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": inst.ID, "name": inst.Name, "port": inst.Port})
}

// CreateExternal POST /api/v1/database/instances/external（接管已有实例）
func (a *DatabaseAPI) CreateExternal(c *gin.Context) {
	req, ok := bind[struct {
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type" binding:"required,oneof=mysql postgres redis mongo"`
		Host     string `json:"host"`
		Port     int    `json:"port" binding:"required"`
		User     string `json:"user"`
		Password string `json:"password" binding:"required"`
		Remark   string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	inst, err := a.DBS.AddExternalInstance(c.Request.Context(), req.Name, req.Type, req.Host, req.Port, req.User, req.Password, req.Remark)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": inst.ID, "name": inst.Name, "host": inst.Host, "port": inst.Port})
}

// Delete DELETE /api/v1/database/instances/:id?purge=
func (a *DatabaseAPI) Delete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DBS.DeleteInstance(c.Request.Context(), id, c.DefaultQuery("purge", "false") == "true"); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// StartStop POST /api/v1/database/instances/:id/start | /stop
func (a *DatabaseAPI) StartStop(up bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		if err := a.DBS.StartStop(c.Request.Context(), id, up); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}

// Reveal GET /api/v1/database/instances/:id/reveal（含明文密码，admin 专用）
func (a *DatabaseAPI) Reveal(c *gin.Context) {
	if c.GetString(middleware.CtxRole) != "admin" {
		respErr(c, errs.ErrForbidden)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.Reveal(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Databases GET /api/v1/database/instances/:id/databases
func (a *DatabaseAPI) Databases(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.Databases(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateDatabase POST /api/v1/database/instances/:id/databases
func (a *DatabaseAPI) CreateDatabase(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Name    string `json:"name" binding:"required"`
		Charset string `json:"charset"`
	}](c)
	if !ok {
		return
	}
	if err := a.DBS.CreateDatabase(c.Request.Context(), id, req.Name, req.Charset); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DropDatabase DELETE /api/v1/database/instances/:id/databases/:name
func (a *DatabaseAPI) DropDatabase(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DBS.DropDatabase(c.Request.Context(), id, c.Param("name")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Users GET /api/v1/database/instances/:id/users
func (a *DatabaseAPI) Users(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.Users(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateUser POST /api/v1/database/instances/:id/users
func (a *DatabaseAPI) CreateUser(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Name     string `json:"name" binding:"required"`
		Host     string `json:"host"`
		Password string `json:"password" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.DBS.CreateUser(c.Request.Context(), id, req.Name, req.Host, req.Password); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DropUser DELETE /api/v1/database/instances/:id/users/:name?host=
func (a *DatabaseAPI) DropUser(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DBS.DropUser(c.Request.Context(), id, c.Param("name"), c.Query("host")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ChangeUserPassword PUT /api/v1/database/instances/:id/users/:name/password?host=
func (a *DatabaseAPI) ChangeUserPassword(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Password string `json:"password" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.DBS.ChangeUserPassword(c.Request.Context(), id, c.Param("name"), c.Query("host"), req.Password); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Backups GET /api/v1/database/instances/:id/backups
// RemoteAccess POST /api/v1/database/instances/:id/remote {enable}（B3）
func (a *DatabaseAPI) RemoteAccess(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("实例 ID 不合法"))
		return
	}
	req, ok := bind[struct {
		Enable bool `json:"enable"`
	}](c)
	if !ok {
		return
	}
	out, err := a.DBS.RemoteAccess(c.Request.Context(), uint(id), req.Enable)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

func (a *DatabaseAPI) Backups(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.Backups(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateBackup POST /api/v1/database/instances/:id/backups
func (a *DatabaseAPI) CreateBackup(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.CreateBackup(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DeleteBackup DELETE /api/v1/database/instances/:id/backups?file=
func (a *DatabaseAPI) DeleteBackup(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DBS.DeleteBackup(c.Request.Context(), id, c.Query("file")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// RestoreBackup POST /api/v1/database/instances/:id/backups/restore?file=
func (a *DatabaseAPI) RestoreBackup(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DBS.Restore(c.Request.Context(), id, c.Query("file")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
