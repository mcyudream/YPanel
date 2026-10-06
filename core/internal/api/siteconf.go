package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SiteConfAPI 站点配置域接口（对标 1Panel 站点详情子页：每域独立读写）。
type SiteConfAPI struct {
	Sites *service.SiteService
}

func siteIDParam(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errBadRequest("站点 ID 不合法")
	}
	return uint(id), nil
}

// GetDomain GET /api/v1/sites/:id/conf/domain
func (a *SiteConfAPI) GetDomain(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.GetDomainConf(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateDomain PUT /api/v1/sites/:id/conf/domain {domains}
func (a *SiteConfAPI) UpdateDomain(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Domains []string `json:"domains"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Sites.UpdateDomainConf(c.Request.Context(), id, req.Domains)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetDefaults GET /api/v1/sites/:id/conf/defaults
func (a *SiteConfAPI) GetDefaults(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.GetDefaultsConf(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateDefaults PUT /api/v1/sites/:id/conf/defaults
func (a *SiteConfAPI) UpdateDefaults(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.SiteDefaultsConf](c)
	if !ok {
		return
	}
	out, err := a.Sites.UpdateDefaultsConf(c.Request.Context(), id, *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetProxy GET /api/v1/sites/:id/conf/proxy
func (a *SiteConfAPI) GetProxy(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.GetProxyConf(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateProxy PUT /api/v1/sites/:id/conf/proxy
func (a *SiteConfAPI) UpdateProxy(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.SiteProxyConf](c)
	if !ok {
		return
	}
	out, err := a.Sites.UpdateProxyConf(c.Request.Context(), id, *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetRewrite GET /api/v1/sites/:id/conf/rewrite
func (a *SiteConfAPI) GetRewrite(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.GetRewriteConf(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// UpdateRewrite PUT /api/v1/sites/:id/conf/rewrite
func (a *SiteConfAPI) UpdateRewrite(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[service.SiteRewriteConf](c)
	if !ok {
		return
	}
	out, err := a.Sites.UpdateRewriteConf(c.Request.Context(), id, *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetHTTPS GET /api/v1/sites/:id/conf/https
func (a *SiteConfAPI) GetHTTPS(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.GetHTTPSConf(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// EnableHTTPS POST /api/v1/sites/:id/conf/https
func (a *SiteConfAPI) EnableHTTPS(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.EnableHTTPS(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DisableHTTPS DELETE /api/v1/sites/:id/conf/https
func (a *SiteConfAPI) DisableHTTPS(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Sites.DisableHTTPS(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
