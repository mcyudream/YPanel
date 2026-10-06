package api

import (
	"strconv"
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SiteAPI 站点接口。
type SiteAPI struct {
	Sites *service.SiteService
	Acme  *service.AcmeService
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

// IssueACME POST /api/v1/sites/:id/cert/acme {domain?}（B1：DNS API 挑战签发）
func (a *SiteAPI) IssueACME(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("站点 ID 不合法"))
		return
	}
	req, ok := bind[struct {
		Domain string `json:"domain"`
	}](c)
	if !ok {
		return
	}
	if a.Acme == nil {
		respErr(c, errBadRequest("ACME 服务未启用"))
		return
	}
	out, err := a.Acme.IssueACME(c.Request.Context(), uint(id), req.Domain)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetSite GET /api/v1/sites/:id/detail// GetSite GET /api/v1/sites/:id/detail（F8 单条端点）
func (a *SiteAPI) GetSite(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respErr(c, errBadRequest("站点 ID 不合法"))
		return
	}
	site, err := a.Sites.GetByIDF(uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, site)
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
		Name         string                  `json:"name" binding:"required"`
		Type         string                  `json:"type" binding:"required,oneof=static proxy php"`
		Domain       string                  `json:"domain" binding:"required"`
		ExtraDomains []string                `json:"extraDomains"`
		Port         int                     `json:"port"`
		ProxyRules   []service.ProxyRule     `json:"proxyRules"`
		ProxyPass    string                  `json:"proxyPass"`
		IndexFiles   string                  `json:"indexFiles"`
		RuntimeID    uint                    `json:"runtimeId"`
	}](c)
	if !ok {
		return
	}
	if len(req.ProxyRules) == 0 && req.ProxyPass != "" {
		req.ProxyRules = []service.ProxyRule{{Prefix: "/", Target: req.ProxyPass}}
	}
	site, err := a.Sites.Create(c.Request.Context(), service.SiteCreateInput{
		Name: req.Name, Type: req.Type, Domain: req.Domain, ExtraDomains: req.ExtraDomains,
		Port: req.Port, ProxyPass: req.ProxyPass, ProxyRules: req.ProxyRules, IndexFiles: req.IndexFiles,
		RuntimeID: req.RuntimeID,
	})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, site)
}

// SiteLogs GET /api/v1/sites/:id/logs?type=access|error&tail=N（M13 站点日志）
func (a *SiteAPI) SiteLogs(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.SiteLogs(c.Request.Context(), id, c.DefaultQuery("type", "access"), c.DefaultQuery("tail", "200"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"content": out})
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

// RewriteTemplates GET /api/v1/sites/rewrite-templates
func (a *SiteAPI) RewriteTemplates(c *gin.Context) {
	respOK(c, service.ListRewriteTemplates())
}

// GetExt GET /api/v1/sites/:id/ext
func (a *SiteAPI) GetExt(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.GetExt(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateExt PUT /api/v1/sites/:id/ext
func (a *SiteAPI) UpdateExt(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.SiteExtConfig](c)
	if !ok {
		return
	}
	if err := a.Sites.UpdateExt(c.Request.Context(), id, *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// GetWaf GET /api/v1/sites/:id/waf
func (a *SiteAPI) GetWaf(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	w, err := a.Sites.GetWaf(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, w)
}

// UpdateWaf PUT /api/v1/sites/:id/waf
func (a *SiteAPI) UpdateWaf(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.SiteWaf](c)
	if !ok {
		return
	}
	if err := a.Sites.UpdateWaf(c.Request.Context(), id, *req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
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
