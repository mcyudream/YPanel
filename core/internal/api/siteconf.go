package api

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SiteConfAPI 站点配置域接口（对标 1Panel 站点详情子页：每域独立读写）。
type SiteConfAPI struct {
	Sites *service.SiteService
	DNS   *service.DnsService // 域名变更后 best-effort 重载内网 DNS 对齐
}

// dnsAlignRefresh 域名变更后重载内网 DNS 对齐（best-effort，失败仅记日志）。
func (a *SiteConfAPI) dnsAlignRefresh(c *gin.Context) {
	if a.DNS == nil {
		return
	}
	if err := a.DNS.OnSitesChanged(c.Request.Context()); err != nil {
		slog.Warn("站点对齐 DNS 重载失败", "err", err.Error())
	}
}

// confGet GET 泛型代理。
func confGet[T any](c *gin.Context, get func(uint) (T, error)) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := get(id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// confPut PUT 泛型代理（apply 首参为 request context）。
func confPut[Req any, Out any](c *gin.Context, apply func(context.Context, uint, Req) (Out, error)) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[Req](c)
	if !ok {
		return
	}
	out, err := apply(c.Request.Context(), id, *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
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

// UpdateDomain PUT /api/v1/sites/:id/conf/domain {domains, primary}
// primary 为空 = 仅更新附加域名；为站点已有域名且异于当前主域名 = 切换主域名。
func (a *SiteConfAPI) UpdateDomain(c *gin.Context) {
	id, err := siteIDParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Domains []string `json:"domains"`
		Primary string   `json:"primary"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Sites.UpdateDomainConf(c.Request.Context(), id, req.Domains, req.Primary)
	if err != nil {
		respErr(c, err)
		return
	}
	a.dnsAlignRefresh(c)
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

// UpdateHTTPS PUT /api/v1/sites/:id/conf/https（B23：证书绑定/停用/HTTP 模式/HSTS/TLS 版本/加密算法）
func (a *SiteConfAPI) UpdateHTTPS(c *gin.Context) {
	confPut(c, a.Sites.UpdateHTTPSConf)
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

// ---- S20 第二批配置域 ----

// GetAntiLeech GET /api/v1/sites/:id/conf/antileech
func (a *SiteConfAPI) GetAntiLeech(c *gin.Context) { confGet(c, a.Sites.GetAntiLeech) }

// UpdateAntiLeech PUT /api/v1/sites/:id/conf/antileech
func (a *SiteConfAPI) UpdateAntiLeech(c *gin.Context) {
	confPut(c, a.Sites.UpdateAntiLeech)
}

// GetAuthBasic GET /api/v1/sites/:id/conf/authbasic
func (a *SiteConfAPI) GetAuthBasic(c *gin.Context) { confGet(c, a.Sites.GetAuthBasic) }

// UpdateAuthBasic PUT /api/v1/sites/:id/conf/authbasic
func (a *SiteConfAPI) UpdateAuthBasic(c *gin.Context) {
	confPut(c, a.Sites.UpdateAuthBasic)
}

// GetCORS GET /api/v1/sites/:id/conf/cors
func (a *SiteConfAPI) GetCORS(c *gin.Context) { confGet(c, a.Sites.GetCORS) }

// UpdateCORS PUT /api/v1/sites/:id/conf/cors
func (a *SiteConfAPI) UpdateCORS(c *gin.Context) { confPut(c, a.Sites.UpdateCORS) }

// GetRedirect GET /api/v1/sites/:id/conf/redirect
func (a *SiteConfAPI) GetRedirect(c *gin.Context) { confGet(c, a.Sites.GetRedirect) }

// UpdateRedirect PUT /api/v1/sites/:id/conf/redirect
func (a *SiteConfAPI) UpdateRedirect(c *gin.Context) {
	confPut(c, a.Sites.UpdateRedirect)
}

// GetRealIP GET /api/v1/sites/:id/conf/realip
func (a *SiteConfAPI) GetRealIP(c *gin.Context) { confGet(c, a.Sites.GetRealIP) }

// UpdateRealIP PUT /api/v1/sites/:id/conf/realip
func (a *SiteConfAPI) UpdateRealIP(c *gin.Context) { confPut(c, a.Sites.UpdateRealIP) }

// GetLimitConn GET /api/v1/sites/:id/conf/limitconn
func (a *SiteConfAPI) GetLimitConn(c *gin.Context) { confGet(c, a.Sites.GetLimitConn) }

// UpdateLimitConn PUT /api/v1/sites/:id/conf/limitconn
func (a *SiteConfAPI) UpdateLimitConn(c *gin.Context) {
	confPut(c, a.Sites.UpdateLimitConn)
}

// GetLoadBalance GET /api/v1/sites/:id/conf/loadbalance
func (a *SiteConfAPI) GetLoadBalance(c *gin.Context) { confGet(c, a.Sites.GetLoadBalance) }

// UpdateLoadBalance PUT /api/v1/sites/:id/conf/loadbalance
func (a *SiteConfAPI) UpdateLoadBalance(c *gin.Context) {
	confPut(c, a.Sites.UpdateLoadBalance)
}

// GetPort GET /api/v1/sites/:id/conf/port（M50 监听端口）
func (a *SiteConfAPI) GetPort(c *gin.Context) { confGet(c, a.Sites.GetPortConf) }

// UpdatePort PUT /api/v1/sites/:id/conf/port {port}
// 非 80/443 端口保存后自动落点：container 模式追加容器映射并重建，host 模式联动防火墙放行。
func (a *SiteConfAPI) UpdatePort(c *gin.Context) {
	confPut(c, func(ctx context.Context, id uint, req struct {
		Port int `json:"port"`
	}) (service.SitePortConf, error) {
		return a.Sites.UpdatePortConf(ctx, id, req.Port)
	})
}
