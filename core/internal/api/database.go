package api

import (

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// DatabaseAPI 数据库实例接口。
type DatabaseAPI struct {
	DBS *service.DatabaseService
}

// SetOwner PUT /api/v1/database/instances/:id/owner {ownerId}（M54-P3：仅 all 数据范围可操作；0=公共）
func (a *DatabaseAPI) SetOwner(c *gin.Context) {
	if !requireAllScope(c) {
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	ownerID, ok := parseOwnerBody(c)
	if !ok {
		respErr(c, errBadRequest("ownerId 不合法"))
		return
	}
	if err := a.DBS.SetInstanceOwner(id, ownerID); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
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
	// M54-P3：assigned 创建 → 属主自己；all → 公共
	if uid := stampOwnerForCreate(c); uid != 0 {
		_ = a.DBS.SetInstanceOwner(inst.ID, uid)
		inst.OwnerID = uid
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
	if uid := stampOwnerForCreate(c); uid != 0 {
		_ = a.DBS.SetInstanceOwner(inst.ID, uid)
		inst.OwnerID = uid
	}
	respOK(c, gin.H{"id": inst.ID, "name": inst.Name, "host": inst.Host, "port": inst.Port})
}

// Adopt POST /api/v1/database/instances/adopt（接管商店已装的数据库应用）
func (a *DatabaseAPI) Adopt(c *gin.Context) {
	req, ok := bind[struct {
		Project string `json:"project" binding:"required"`
		NodeID  string `json:"nodeId"`
	}](c)
	if !ok {
		return
	}
	inst, err := a.DBS.Adopt(c.Request.Context(), req.Project, req.NodeID)
	if err != nil {
		respErr(c, err)
		return
	}
	if uid := stampOwnerForCreate(c); uid != 0 {
		_ = a.DBS.SetInstanceOwner(inst.ID, uid)
		inst.OwnerID = uid
	}
	respOK(c, gin.H{"id": inst.ID, "name": inst.Name, "type": inst.Type, "port": inst.Port})
}

// Delete DELETE /api/v1/database/instances/:id?purge=
func (a *DatabaseAPI) Delete(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.DBS.DeleteInstance(c.Request.Context(), id, c.DefaultQuery("data", "false") == "true", c.DefaultQuery("backups", "false") == "true"); err != nil {
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
	if !isSuperCaller(c) { // M54：按通配权限判断（等价旧 admin）
		respErr(c, errs.ErrForbidden)
		return
	}
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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

// RemoteAccessStatus GET /api/v1/database/instances/:id/remote
func (a *DatabaseAPI) RemoteAccessStatus(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	enabled, err := a.DBS.RemoteAccessStatus(c.Request.Context(), uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"enabled": enabled})
}

// RemoteAccess POST /api/v1/database/instances/:id/remote {enable}（B3）
func (a *DatabaseAPI) RemoteAccess(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
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

// BackupImport POST /api/v1/database/instances/:id/backups/import {filename, content(base64)}
func (a *DatabaseAPI) BackupImport(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Filename string `json:"filename" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.DBS.ImportBackup(c.Request.Context(), id, c.GetString(middleware.CtxUsername), req.Filename, req.Content)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

func (a *DatabaseAPI) Backups(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
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

// ExecEnvCheck GET /api/v1/database/execenv/check —— 外接实例执行环境（Docker+镜像缓存）检测。
func (a *DatabaseAPI) ExecEnvCheck(c *gin.Context) {
	out, err := a.DBS.ExecEnvCheck(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ExecEnvPull POST /api/v1/database/execenv/pull {type} —— 预拉取指定类型的执行镜像。
func (a *DatabaseAPI) ExecEnvPull(c *gin.Context) {
	var req struct {
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == "" {
		respErr(c, errs.Wrap(errs.ErrBadRequest, "请指定实例类型"))
		return
	}
	out, err := a.DBS.ExecEnvPull(c.Request.Context(), req.Type)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateBackup POST /api/v1/database/instances/:id/backups（body 可选 {storageAccountId,keep}）
func (a *DatabaseAPI) CreateBackup(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	var opts service.BackupUploadOpts
	if c.Request.ContentLength > 0 {
		var req struct {
			StorageAccountId uint `json:"storageAccountId"`
			Keep             int  `json:"keep"`
		}
		if berr := c.ShouldBindJSON(&req); berr == nil {
			opts = service.BackupUploadOpts{StorageAccountID: req.StorageAccountId, Keep: req.Keep}
		}
	}
	out, err := a.DBS.CreateBackup(c.Request.Context(), id, opts)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DeleteBackup DELETE /api/v1/database/instances/:id/backups?file=
func (a *DatabaseAPI) DeleteBackup(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
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
	id, err := ownedInstanceID(c, a.DBS)
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

// ---- M39：MySQL 管理深化 ----

// GrantMatrix GET /api/v1/database/instances/:id/privileges?db=
func (a *DatabaseAPI) GrantMatrix(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.GrantMatrix(c.Request.Context(), id, c.Query("db"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SetPrivileges PUT /api/v1/database/instances/:id/privileges {db,user,host,privs,grant}
func (a *DatabaseAPI) SetPrivileges(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		DB    string   `json:"db" binding:"required"`
		User  string   `json:"user" binding:"required"`
		Host  string   `json:"host"`
		Privs []string `json:"privs" binding:"required,min=1"`
		Grant bool     `json:"grant"`
	}](c)
	if !ok {
		return
	}
	if err := a.DBS.GrantPrivs(c.Request.Context(), id, req.DB, req.User, req.Host, req.Privs, req.Grant); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Variables GET /api/v1/database/instances/:id/variables?filter=
func (a *DatabaseAPI) Variables(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.Variables(c.Request.Context(), id, c.Query("filter"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SetVariable PUT /api/v1/database/instances/:id/variables {name,value}
func (a *DatabaseAPI) SetVariable(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Name  string `json:"name" binding:"required"`
		Value string `json:"value" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.DBS.SetVariable(c.Request.Context(), id, req.Name, req.Value); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DBStatus GET /api/v1/database/instances/:id/status
func (a *DatabaseAPI) DBStatus(c *gin.Context) {
	id, err := ownedInstanceID(c, a.DBS)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.DBS.StatusStats(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
