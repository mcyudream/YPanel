package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// MarketStoreAPI 应用商店（1Panel 默认源）接口。
type MarketStoreAPI struct {
	Store *service.MarketStoreService
}

// List GET /api/v1/store/apps?search=&tag=
func (a *MarketStoreAPI) List(c *gin.Context) {
	out, err := a.Store.List(c.Query("search"), c.Query("tag"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Get GET /api/v1/store/apps/:key
func (a *MarketStoreAPI) Get(c *gin.Context) {
	app, versions, err := a.Store.Get(c.Param("key"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"app": app, "versions": versions})
}

// Sync POST /api/v1/store/sync?force=
func (a *MarketStoreAPI) Sync(c *gin.Context) {
	out, err := a.Store.Sync(c.Request.Context(), c.Query("force") == "1")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Installed GET /api/v1/store/installed
func (a *MarketStoreAPI) Installed(c *gin.Context) {
	respOK(c, a.Store.Installed())
}

// Install POST /api/v1/store/install
func (a *MarketStoreAPI) Install(c *gin.Context) {
	req, ok := bind[struct {
		Key     string            `json:"key" binding:"required"`
		Version string            `json:"version"`
		Name    string            `json:"name" binding:"required"`
		Params  map[string]string `json:"params"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Store.Install(c.Request.Context(), req.Key, req.Version, req.Name, req.Params)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Uninstall DELETE /api/v1/store/install/:project
func (a *MarketStoreAPI) Uninstall(c *gin.Context) {
	if err := a.Store.Uninstall(c.Request.Context(), c.Param("project")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
