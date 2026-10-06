package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SiteAPI 站点接口。
type SiteAPI struct {
	Sites *service.SiteService
}

// Status GET /api/v1/nginx/status
func (a *SiteAPI) Status(c *gin.Context) {
	out, err := a.Sites.Status(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Install POST /api/v1/nginx/install
func (a *SiteAPI) Install(c *gin.Context) {
	if err := a.Sites.Install(c.Request.Context()); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// List GET /api/v1/sites
func (a *SiteAPI) List(c *gin.Context) {
	out, err := a.Sites.List(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/sites
func (a *SiteAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required,oneof=static proxy"`
		Domain    string `json:"domain" binding:"required"`
		Port      int    `json:"port"`
		ProxyPass string `json:"proxyPass"`
	}](c)
	if !ok {
		return
	}
	site, err := a.Sites.Create(c.Request.Context(), req.Name, req.Type, req.Domain, req.Port, req.ProxyPass)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, site)
}

// Delete DELETE /api/v1/sites/:id?purge=
func (a *SiteAPI) Delete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Sites.Delete(c.Request.Context(), id, c.DefaultQuery("purge", "false") == "true"); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// SetEnabled POST /api/v1/sites/:id/enable | disable
func (a *SiteAPI) SetEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			respErr(c, err)
			return
		}
		if err := a.Sites.SetEnabled(c.Request.Context(), id, enabled); err != nil {
			respErr(c, err)
			return
		}
		respOK(c, struct{}{})
	}
}

// Config GET /api/v1/sites/:id/config
func (a *SiteAPI) Config(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	content, err := a.Sites.Config(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"content": content})
}

// UpdateConfig PUT /api/v1/sites/:id/config
func (a *SiteAPI) UpdateConfig(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Content string `json:"content" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Sites.UpdateConfig(c.Request.Context(), id, req.Content); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Scan GET /api/v1/sites/scan（站点识别）
func (a *SiteAPI) Scan(c *gin.Context) {
	out, err := a.Sites.ScanSites(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Adopt POST /api/v1/sites/adopt（接管发现的站点）
func (a *SiteAPI) Adopt(c *gin.Context) {
	req, ok := bind[struct {
		File      string `json:"file" binding:"required"`
		Domain    string `json:"domain" binding:"required"`
		Type      string `json:"type" binding:"required,oneof=static proxy"`
		ProxyPass string `json:"proxyPass"`
	}](c)
	if !ok {
		return
	}
	site, err := a.Sites.Adopt(c.Request.Context(), req.File, req.Domain, req.Type, req.ProxyPass)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, site)
}

// IssueSelfSigned POST /api/v1/sites/:id/cert/selfsigned
func (a *SiteAPI) IssueSelfSigned(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Sites.IssueSelfSigned(c.Request.Context(), id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
