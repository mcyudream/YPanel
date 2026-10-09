package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
)

// StoreAPI 统一应用商店接口（多源）。
type StoreAPI struct {
	Store *service.StoreService
}

// ListSources GET /api/v1/store/sources
func (a *StoreAPI) ListSources(c *gin.Context) {
	respOK(c, a.Store.Sources())
}

// CreateSource POST /api/v1/store/sources
func (a *StoreAPI) CreateSource(c *gin.Context) {
	req, ok := bind[service.StoreSourceInput](c)
	if !ok {
		return
	}
	row, err := a.Store.CreateSource(*req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// UpdateSource PUT /api/v1/store/sources/:id
func (a *StoreAPI) UpdateSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("参数不合法"))
		return
	}
	req, ok := bind[service.StoreSourceInput](c)
	if !ok {
		return
	}
	row, err := a.Store.UpdateSource(uint(id), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// SetSourceEnabled POST /api/v1/store/sources/:id/enable|disable
func (a *StoreAPI) SetSourceEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			respErr(c, errBadRequest("参数不合法"))
			return
		}
		if err := a.Store.SetSourceEnabled(uint(id), enabled); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}

// DeleteSource DELETE /api/v1/store/sources/:id
func (a *StoreAPI) DeleteSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("参数不合法"))
		return
	}
	if err := a.Store.DeleteSource(uint(id)); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SyncSource POST /api/v1/store/sources/:id/sync?force=
func (a *StoreAPI) SyncSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("参数不合法"))
		return
	}
	out, err := a.Store.SyncSource(c.Request.Context(), uint(id), c.Query("force") == "1")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Sync POST /api/v1/store/sync?force= （全部启用源）
func (a *StoreAPI) Sync(c *gin.Context) {
	out, err := a.Store.SyncAll(c.Request.Context(), c.Query("force") == "1")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Apps GET /api/v1/store/apps?search=&tag=&sourceId=&kind=&status=&orderBy=&order=&page=&pageSize=
func (a *StoreAPI) Apps(c *gin.Context) {
	q := service.StoreListQuery{
		Search:  c.Query("search"),
		Tag:     c.Query("tag"),
		Status:  c.DefaultQuery("status", "all"),
		OrderBy: c.Query("orderBy"),
		Order:   c.Query("order"),
		Kind:    c.Query("kind"),
	}
	if v, err := strconv.ParseUint(c.Query("sourceId"), 10, 64); err == nil {
		q.SourceID = uint(v)
	}
	if v, err := strconv.Atoi(c.Query("page")); err == nil {
		q.Page = v
	}
	if v, err := strconv.Atoi(c.Query("pageSize")); err == nil {
		q.PageSize = v
	}
	out, err := a.Store.Apps(q)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Tags GET /api/v1/store/tags
func (a *StoreAPI) Tags(c *gin.Context) {
	respOK(c, a.Store.Tags())
}

// Get GET /api/v1/store/apps/:sourceId/:key
func (a *StoreAPI) Get(c *gin.Context) {
	sourceID, err := strconv.ParseUint(c.Param("sourceId"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("参数不合法"))
		return
	}
	out, err := a.Store.App(uint(sourceID), c.Param("key"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Icon GET /api/v1/store/apps/:sourceId/:key/icon
func (a *StoreAPI) Icon(c *gin.Context) {
	sourceID, err := strconv.ParseUint(c.Param("sourceId"), 10, 64)
	if err != nil {
		c.Status(404)
		return
	}
	data, ct, err := a.Store.AppIcon(uint(sourceID), c.Param("key"))
	if err != nil {
		c.Status(404)
		return
	}
	c.Data(200, ct, data)
}

// Installed GET /api/v1/store/installed（详情聚合：状态/端口/图标/可升级/参数）
func (a *StoreAPI) Installed(c *gin.Context) {
	out, err := a.Store.InstalledDetailed(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// InstalledAction POST /api/v1/store/installed/:project/:action（start|stop|restart|rebuild）
func (a *StoreAPI) InstalledAction(c *gin.Context) {
	if err := a.Store.InstalledAction(c.Request.Context(), c.Param("project"), c.Param("action")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// InstallEnv GET /api/v1/store/installed/:project/env（admin，含密码明文）
func (a *StoreAPI) InstallEnv(c *gin.Context) {
	if c.GetString(middleware.CtxRole) != "admin" {
		respErr(c, errBadRequest("仅管理员可查看安装参数"))
		return
	}
	out, err := a.Store.InstallEnv(c.Request.Context(), c.Param("project"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SaveInstallEnv PUT /api/v1/store/installed/:project/env（admin，保存并重建容器生效）
func (a *StoreAPI) SaveInstallEnv(c *gin.Context) {
	if c.GetString(middleware.CtxRole) != "admin" {
		respErr(c, errBadRequest("仅管理员可修改安装参数"))
		return
	}
	req, ok := bind[struct {
		Content string `json:"content" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Store.SaveInstallEnv(c.Request.Context(), c.Param("project"), req.Content); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Install POST /api/v1/store/install
func (a *StoreAPI) Install(c *gin.Context) {
	req, ok := bind[service.StoreInstallInput](c)
	if !ok {
		return
	}
	out, err := a.Store.Install(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Uninstall DELETE /api/v1/store/install/:project（任务化；?purgeData=&rmi=&cascadeDB=）
func (a *StoreAPI) Uninstall(c *gin.Context) {
	opts := service.StoreUninstallOptions{
		PurgeData:   c.DefaultQuery("purgeData", "false") == "true",
		RemoveImage: c.DefaultQuery("rmi", "false") == "true",
		CascadeDB:   c.DefaultQuery("cascadeDB", "false") == "true",
	}
	out, err := a.Store.Uninstall(c.Request.Context(), c.Param("project"), opts)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ScanLocalStore POST /api/v1/store/local/scan（商店收尾：本地包扫描入库）
func (a *StoreAPI) ScanLocalStore(c *gin.Context) {
	out, err := a.Store.ScanLocalStore(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CheckUpgrades GET /api/v1/store/upgrades/check（已装应用升级检查）
func (a *StoreAPI) CheckUpgrades(c *gin.Context) {
	out, err := a.Store.CheckUpgrades(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpgradeApp POST /api/v1/store/installed/:project/upgrade（升级到源内最新版）
func (a *StoreAPI) UpgradeApp(c *gin.Context) {
	out, err := a.Store.UpgradeApp(c.Request.Context(), c.Param("project"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
