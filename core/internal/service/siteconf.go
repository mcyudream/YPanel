// 站点配置域读写（对标 1Panel 站点详情子页模型）：
// 每个配置域独立 GET/PUT，PUT 统一走"校验 → 更新列 → writeConf(nginx -t 失败回滚)"链路。
package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// ---- 域：domain 域名管理 ----

// SiteDomainConf 域名配置（主域名只读，附加域名增删）。
type SiteDomainConf struct {
	Name       string   `json:"name"`
	Domain     string   `json:"domain"`
	Domains    []string `json:"domains"`
	CertDomain string   `json:"certDomain"`
}

// GetDomainConf 读取域名配置。
func (s *SiteService) GetDomainConf(id uint) (SiteDomainConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDomainConf{}, err
	}
	extra, _ := parseExtraDomains(site)
	return SiteDomainConf{Name: site.Name, Domain: site.Domain, Domains: extra, CertDomain: site.CertDomain}, nil
}

// UpdateDomainConf 更新附加域名（主域名不可改；证书域名随主域名时需重签）。
func (s *SiteService) UpdateDomainConf(ctx context.Context, id uint, domains []string) (SiteDomainConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDomainConf{}, err
	}
	cleaned, err := validateDomains(domains)
	if err != nil {
		return SiteDomainConf{}, err
	}
	for _, d := range cleaned {
		if d == site.Domain {
			return SiteDomainConf{}, errs.Wrap(errs.ErrBadRequest, "附加域名不能与主域名重复: "+d)
		}
	}
	raw := ""
	if len(cleaned) > 0 {
		b, _ := json.Marshal(cleaned)
		raw = string(b)
	}
	if err := s.db.Model(site).Update("domains", raw).Error; err != nil {
		return SiteDomainConf{}, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return SiteDomainConf{}, err
	}
	return s.GetDomainConf(id)
}

func parseExtraDomains(site *model.Site) ([]string, error) {
	if site.Domains == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(site.Domains), &out); err != nil {
		return []string{}, nil
	}
	return out, nil
}

// ---- 域：defaults 默认文档 / 404 ----

// SiteDefaultsConf 默认文档配置。
type SiteDefaultsConf struct {
	IndexFiles   string `json:"indexFiles"`
	ErrorPage404 string `json:"errorPage404"`
}

// GetDefaultsConf 读取。
func (s *SiteService) GetDefaultsConf(id uint) (SiteDefaultsConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDefaultsConf{}, err
	}
	return SiteDefaultsConf{IndexFiles: site.IndexFiles, ErrorPage404: site.ErrorPage404}, nil
}

// UpdateDefaultsConf 更新（indexFiles 白名单校验）。
func (s *SiteService) UpdateDefaultsConf(ctx context.Context, id uint, conf SiteDefaultsConf) (SiteDefaultsConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDefaultsConf{}, err
	}
	files := strings.Split(strings.ReplaceAll(conf.IndexFiles, "，", ","), ",")
	cleaned := make([]string, 0, len(files))
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if strings.ContainsAny(f, "/\\ ") || strings.Contains(f, "..") {
			return SiteDefaultsConf{}, errs.Wrap(errs.ErrBadRequest, "默认文档仅允许文件名（不含路径）: "+f)
		}
		cleaned = append(cleaned, f)
	}
	if len(cleaned) == 0 {
		return SiteDefaultsConf{}, errs.Wrap(errs.ErrBadRequest, "至少填写一个默认文档")
	}
	val := strings.Join(cleaned, " ")
	if err := s.db.Model(site).Updates(map[string]any{
		"index_files":   val,
		"error_page404": strings.TrimSpace(conf.ErrorPage404),
	}).Error; err != nil {
		return SiteDefaultsConf{}, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return SiteDefaultsConf{}, err
	}
	return s.GetDefaultsConf(id)
}

// ---- 域：proxy 反代规则（仅 proxy 类型）----

// SiteProxyConf 反代配置。
type SiteProxyConf struct {
	Rules         []ProxyRule `json:"rules"`
	CacheEnable   bool        `json:"cacheEnable"`
	CacheDuration string      `json:"cacheDuration"`
}

// GetProxyConf 读取。
func (s *SiteService) GetProxyConf(id uint) (SiteProxyConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteProxyConf{}, err
	}
	m := parseSiteMeta(site)
	return SiteProxyConf{Rules: m.ProxyRules, CacheEnable: m.CacheEnable, CacheDuration: m.CacheDuration}, nil
}

// UpdateProxyConf 更新反代规则与缓存。
func (s *SiteService) UpdateProxyConf(ctx context.Context, id uint, conf SiteProxyConf) (SiteProxyConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteProxyConf{}, err
	}
	if site.Type != "proxy" {
		return SiteProxyConf{}, errs.Wrap(errs.ErrBadRequest, "仅反向代理站点支持该配置")
	}
	if len(conf.Rules) == 0 {
		return SiteProxyConf{}, errs.Wrap(errs.ErrBadRequest, "至少保留一条反代规则")
	}
	for _, r := range conf.Rules {
		if r.Target == "" || !(strings.HasPrefix(r.Target, "http://") || strings.HasPrefix(r.Target, "https://")) {
			return SiteProxyConf{}, errs.Wrap(errs.ErrBadRequest, "转发目标需为 http(s):// 地址: "+r.Target)
		}
	}
	if conf.CacheDuration == "" {
		conf.CacheDuration = "12h"
	}
	raw, err := json.Marshal(conf.Rules)
	if err != nil {
		return SiteProxyConf{}, err
	}
	if err := s.db.Model(site).Updates(map[string]any{
		"proxy_rules":    string(raw),
		"cache_enable":   conf.CacheEnable,
		"cache_duration": conf.CacheDuration,
	}).Error; err != nil {
		return SiteProxyConf{}, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return SiteProxyConf{}, err
	}
	return s.GetProxyConf(id)
}

// ---- 域：rewrite 伪静态 ----

// SiteRewriteConf 伪静态配置。
type SiteRewriteConf struct {
	RewriteName    string `json:"rewriteName"`
	RewriteContent string `json:"rewriteContent"`
}

// GetRewriteConf 读取。
func (s *SiteService) GetRewriteConf(id uint) (SiteRewriteConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteRewriteConf{}, err
	}
	return SiteRewriteConf{RewriteName: site.RewriteName, RewriteContent: site.RewriteContent}, nil
}

// UpdateRewriteConf 更新伪静态。
func (s *SiteService) UpdateRewriteConf(ctx context.Context, id uint, conf SiteRewriteConf) (SiteRewriteConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteRewriteConf{}, err
	}
	if err := s.db.Model(site).Updates(map[string]any{
		"rewrite_name": conf.RewriteName, "rewrite_content": conf.RewriteContent,
	}).Error; err != nil {
		return SiteRewriteConf{}, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return SiteRewriteConf{}, err
	}
	return s.GetRewriteConf(id)
}

// ---- 域：https HTTPS 管理 ----

// SiteHTTPSConf HTTPS 配置视图（B23 扩展：HTTP 模式/HSTS/TLS 版本/加密算法/HTTP2/证书库绑定）。
type SiteHTTPSConf struct {
	Enable     bool   `json:"enable"`
	CertDomain string `json:"certDomain"`
	CertID     uint   `json:"certId"`

	// 高级设置（存 Site.HTTPSJSON）
	HTTPMode      string   `json:"httpMode"`      // redirect / both / deny
	HSTS          bool     `json:"hsts"`
	HSTSSubdomain bool     `json:"hstsSubdomain"`
	TLSVersions   []string `json:"tlsVersions"`
	Ciphers       string   `json:"ciphers"`
	HTTP2         bool     `json:"http2"`
}

// GetHTTPSConf 读取。
func (s *SiteService) GetHTTPSConf(id uint) (SiteHTTPSConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteHTTPSConf{}, err
	}
	cfg := parseHTTPS(site)
	return SiteHTTPSConf{
		Enable: site.CertDomain != "", CertDomain: site.CertDomain, CertID: site.CertID,
		HTTPMode: cfg.HTTPMode, HSTS: cfg.HSTS, HSTSSubdomain: cfg.HSTSSubdomain,
		TLSVersions: cfg.TLSVersions, Ciphers: cfg.Ciphers, HTTP2: cfg.HTTP2,
	}, nil
}

// SiteHTTPSUpdateInput HTTPS 更新输入：certID 与 selfsigned 二选一（都空=仅更新高级设置）。
type SiteHTTPSUpdateInput struct {
	CertID        uint     `json:"certId"`        // 从证书库选择
	SelfSigned    bool     `json:"selfSigned"`    // 自签
	Disable       bool     `json:"disable"`       // 停用 HTTPS
	HTTPMode      string   `json:"httpMode"`
	HSTS          bool     `json:"hsts"`
	HSTSSubdomain bool     `json:"hstsSubdomain"`
	TLSVersions   []string `json:"tlsVersions"`
	Ciphers       string   `json:"ciphers"`
	HTTP2         bool     `json:"http2"`
}

// UpdateHTTPSConf 更新 HTTPS 配置（证书绑定 / 停用 / 高级设置，统一校验回滚链路）。
func (s *SiteService) UpdateHTTPSConf(ctx context.Context, id uint, in SiteHTTPSUpdateInput) (SiteHTTPSConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteHTTPSConf{}, err
	}

	// 1) 证书来源变更
	if in.Disable {
		if site.CertDomain != "" {
			if err := s.db.Model(site).Updates(map[string]any{"cert_domain": "", "cert_id": 0}).Error; err != nil {
				return SiteHTTPSConf{}, err
			}
			if err := s.writeConf(ctx, site, confTemplate(site, false, parseWaf(site))); err != nil {
				return SiteHTTPSConf{}, err
			}
		}
	} else if in.CertID != 0 {
		var cert model.Certificate
		if err := s.db.First(&cert, in.CertID).Error; err != nil {
			return SiteHTTPSConf{}, errs.Wrap(errs.ErrBadRequest, "所选证书不存在")
		}
		if site.CertID != in.CertID {
			if err := s.ApplyCert(ctx, site, cert.ID, cert.CertName); err != nil {
				return SiteHTTPSConf{}, err
			}
		}
	} else if in.SelfSigned && site.CertDomain == "" {
		if err := s.IssueSelfSigned(ctx, id); err != nil {
			return SiteHTTPSConf{}, err
		}
	}

	// 2) 高级设置持久化
	site, err = s.siteByID(id) // 回读（证书可能已变更）
	if err != nil {
		return SiteHTTPSConf{}, err
	}
	cfg := SiteHTTPSCfg{
		HTTPMode: in.HTTPMode, HSTS: in.HSTS, HSTSSubdomain: in.HSTSSubdomain,
		TLSVersions: in.TLSVersions, Ciphers: strings.TrimSpace(in.Ciphers), HTTP2: in.HTTP2,
	}
	if cfg.HTTPMode == "" {
		cfg.HTTPMode = "redirect"
	}
	if cfg.Ciphers == "" {
		cfg.Ciphers = ""
	}
	b, _ := json.Marshal(cfg)
	if err := s.db.Model(site).Update("https_json", string(b)).Error; err != nil {
		return SiteHTTPSConf{}, err
	}
	// 启用状态下重写 conf 生效
	if site.CertDomain != "" {
		if err := s.writeConf(ctx, site, confTemplate(site, true, parseWaf(site))); err != nil {
			return SiteHTTPSConf{}, err
		}
	}
	return s.GetHTTPSConf(id)
}

// EnableHTTPS 为站点启用 HTTPS（自签证书；兼容旧端点）。
func (s *SiteService) EnableHTTPS(ctx context.Context, id uint) (SiteHTTPSConf, error) {
	if err := s.IssueSelfSigned(ctx, id); err != nil {
		return SiteHTTPSConf{}, err
	}
	return s.GetHTTPSConf(id)
}

// DisableHTTPS 停用 HTTPS（清证书域名并重写 conf；兼容旧端点）。
func (s *SiteService) DisableHTTPS(ctx context.Context, id uint) (SiteHTTPSConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteHTTPSConf{}, err
	}
	if site.CertDomain == "" {
		return s.GetHTTPSConf(id)
	}
	if err := s.db.Model(site).Updates(map[string]any{"cert_domain": "", "cert_id": 0}).Error; err != nil {
		return SiteHTTPSConf{}, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, false, parseWaf(site))); err != nil {
		return SiteHTTPSConf{}, err
	}
	return s.GetHTTPSConf(id)
}
