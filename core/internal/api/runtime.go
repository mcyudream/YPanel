package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// idQuery 解析 query 参数 id。
func idQuery(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errBadRequest("ID 不合法")
	}
	return uint(id), nil
}

// RuntimeAPI 运行环境接口。
type RuntimeAPI struct {
	RT *service.RuntimeService
}

// List GET /api/v1/runtimes
func (a *RuntimeAPI) List(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	out, err := rt.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Detail GET /api/v1/runtimes/:id
func (a *RuntimeAPI) Detail(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	row, err := rt.Detail(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Create POST /api/v1/runtimes（任务化：模板渲染 → build → up）
func (a *RuntimeAPI) Create(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[service.RuntimeCreateInput](c)
	if !ok {
		return
	}
	out, err := rt.Create(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// AttachExternal POST /api/v1/runtimes/external（接管本机 php-fpm）
func (a *RuntimeAPI) AttachExternal(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		Name     string `json:"name" binding:"required"`
		Version  string `json:"version"`
		FCGIAddr string `json:"fcgiAddr" binding:"required"`
		Remark   string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	row, err := rt.AttachExternal(req.Name, req.Version, req.FCGIAddr, req.Remark)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// Delete DELETE /api/v1/runtimes/:id
func (a *RuntimeAPI) Delete(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := rt.Delete(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Operate POST /api/v1/runtimes/:id/start | stop | restart
func (a *RuntimeAPI) Operate(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		if err := a.RT.Operate(c.Request.Context(), id, action); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}

// PHPExtensionCatalog GET /api/v1/runtimes/php/catalog（创建向导用）
func (a *RuntimeAPI) PHPExtensionCatalog(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	respOK(c, rt.PHPExtensionCatalog())
}

// PHPExtensions GET /api/v1/runtimes/php/extensions?id=
func (a *RuntimeAPI) PHPExtensions(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.PHPExtensions(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// PHPExtensionInstall POST /api/v1/runtimes/php/extensions/install {id, name}
func (a *RuntimeAPI) PHPExtensionInstall(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		ID   uint   `json:"id" binding:"required"`
		Name string `json:"name" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := rt.InstallPHPExtension(c.Request.Context(), req.ID, req.Name)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// PHPExtensionUninstall POST /api/v1/runtimes/php/extensions/uninstall {id, name}
func (a *RuntimeAPI) PHPExtensionUninstall(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		ID   uint   `json:"id" binding:"required"`
		Name string `json:"name" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := rt.UninstallPHPExtension(c.Request.Context(), req.ID, req.Name)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetPHPConfig GET /api/v1/runtimes/php/config?id=
func (a *RuntimeAPI) GetPHPConfig(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.GetPHPConfig(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdatePHPConfig POST /api/v1/runtimes/php/config
func (a *RuntimeAPI) UpdatePHPConfig(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.PHPConfigUpdate](c)
	if !ok {
		return
	}
	if err := rt.UpdatePHPConfig(c.Request.Context(), id, *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// GetFPMConfig GET /api/v1/runtimes/php/fpm-config?id=
func (a *RuntimeAPI) GetFPMConfig(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.GetFPMConfig(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateFPMConfig POST /api/v1/runtimes/php/fpm-config {id, params}
func (a *RuntimeAPI) UpdateFPMConfig(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		ID     uint              `json:"id" binding:"required"`
		Params map[string]string `json:"params"`
	}](c)
	if !ok {
		return
	}
	if err := rt.UpdateFPMConfig(c.Request.Context(), req.ID, req.Params); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// FPMStatus GET /api/v1/runtimes/php/fpm-status?id=
func (a *RuntimeAPI) FPMStatus(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.FPMStatus(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// NodeModules GET /api/v1/runtimes/node/modules?id=
func (a *RuntimeAPI) NodeModules(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.NodeModules(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// OperateNodeModule POST /api/v1/runtimes/node/modules/operate {id, operate, module, pkgManager}
func (a *RuntimeAPI) OperateNodeModule(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		ID         uint   `json:"id" binding:"required"`
		Operate    string `json:"operate" binding:"required"`
		Module     string `json:"module"`
		PkgManager string `json:"pkgManager"`
	}](c)
	if !ok {
		return
	}
	out, err := rt.OperateNodeModule(c.Request.Context(), req.ID, req.Operate, req.Module, req.PkgManager)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Rebuild POST /api/v1/runtimes/:id/rebuild（用当前面板模板重建镜像，保留用户配置）
func (a *RuntimeAPI) Rebuild(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.Rebuild(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SupervisorList GET /api/v1/runtimes/php/supervisor?id=
func (a *RuntimeAPI) SupervisorList(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.SupervisorList(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SupervisorUpsert POST /api/v1/runtimes/php/supervisor {id, name, command, autoStart, autoRestart}
func (a *RuntimeAPI) SupervisorUpsert(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.SupervisorUpsertInput](c)
	if !ok {
		return
	}
	if err := rt.SupervisorUpsert(c.Request.Context(), id, *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SupervisorOperate POST /api/v1/runtimes/php/supervisor/operate {id, name, action}
func (a *RuntimeAPI) SupervisorOperate(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		ID     uint   `json:"id" binding:"required"`
		Name   string `json:"name" binding:"required"`
		Action string `json:"action" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := rt.SupervisorOperate(c.Request.Context(), req.ID, req.Name, req.Action); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SupervisorDelete DELETE /api/v1/runtimes/php/supervisor?id=&name=
func (a *RuntimeAPI) SupervisorDelete(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := rt.SupervisorDelete(c.Request.Context(), id, c.Query("name")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SupervisorLog GET /api/v1/runtimes/php/supervisor/log?id=&name=
func (a *RuntimeAPI) SupervisorLog(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.SupervisorLog(c.Request.Context(), id, c.Query("name"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SlowLog GET /api/v1/runtimes/php/slow-log?id=
func (a *RuntimeAPI) SlowLog(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idQuery(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.SlowLog(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SlowLogClear POST /api/v1/runtimes/php/slow-log/clear {id}
func (a *RuntimeAPI) SlowLogClear(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	req, ok := bind[struct {
		ID uint `json:"id" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := rt.SlowLogClear(c.Request.Context(), req.ID); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// BackupList GET /api/v1/runtimes/:id/backups
func (a *RuntimeAPI) BackupList(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.BackupList(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// BackupCreate POST /api/v1/runtimes/:id/backups
func (a *RuntimeAPI) BackupCreate(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := rt.BackupCreate(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// BackupRestore POST /api/v1/runtimes/:id/backups/restore {file}
func (a *RuntimeAPI) BackupRestore(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		File string `json:"file" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := rt.BackupRestore(c.Request.Context(), id, req.File)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// BackupDelete DELETE /api/v1/runtimes/:id/backups?file=
func (a *RuntimeAPI) BackupDelete(c *gin.Context) {
	rt, nerr := a.RT.WithNode(c.Query("node"))
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := rt.BackupDelete(c.Request.Context(), id, c.Query("file")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
