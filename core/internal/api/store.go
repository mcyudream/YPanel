package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

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

// Installed GET /api/v1/store/installed
func (a *StoreAPI) Installed(c *gin.Context) {
	respOK(c, a.Store.Installed())
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

// Uninstall DELETE /api/v1/store/install/:project
func (a *StoreAPI) Uninstall(c *gin.Context) {
	out, err := a.Store.Uninstall(c.Request.Context(), c.Param("project"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
