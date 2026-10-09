// 站点配置域读写（对标 1Panel 站点详情子页模型）：
// 每个配置域独立 GET/PUT，PUT 统一走"校验 → 更新列 → writeConf(nginx -t 失败回滚)"链路。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// ---- 域：domain 域名管理 ----

// SiteDomainConf 域名配置（主域名可通过 primary 切换，附加域名增删）。
type SiteDomainConf struct {
	Name       string   `json:"name"`
	Domain     string   `json:"domain"`
	Domains    []string `json:"domains"`
	CertDomain string   `json:"certDomain"`
	CertNotice string   `json:"certNotice"` // 主域名切换后的证书适配提示（空 = 无需处理）
}

// GetDomainConf 读取域名配置。
func (s *SiteService) GetDomainConf(id uint) (SiteDomainConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDomainConf{}, err
	}
	s = s.forSite(site)
	extra, _ := parseExtraDomains(site)
	return SiteDomainConf{Name: site.Name, Domain: site.Domain, Domains: extra, CertDomain: site.CertDomain}, nil
}

// UpdateDomainConf 更新域名。primary 为空时仅更新附加域名；primary 为站点已有域名且异于
// 当前主域名时切换主域名（位置交换，nginx server_name 集合不变）：自签证书自动按新主域名
// 重签，证书库条目不覆盖新主域名时在 CertNotice 返回提示。
func (s *SiteService) UpdateDomainConf(ctx context.Context, id uint, domains []string, primary string) (SiteDomainConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDomainConf{}, err
	}
	s = s.forSite(site)
	cleaned, err := validateDomains(domains)
	if err != nil {
		return SiteDomainConf{}, err
	}
	newPrimary := strings.ToLower(strings.TrimSpace(primary))
	certNotice := ""
	switching := newPrimary != "" && newPrimary != site.Domain
	if switching {
		// 新主域名必须是本站已有域名：库中附加域名之一，或仍在提交列表中（兼容两种提交形态——
		// 前端交换预览会把新主域名从列表移除、旧主域名放回列表；直调 API 则可能仍留在列表里）
		owned := false
		extra, _ := parseExtraDomains(site)
		for _, d := range extra {
			if d == newPrimary {
				owned = true
				break
			}
		}
		inSubmitted := false
		for _, d := range cleaned {
			if d == newPrimary {
				inSubmitted = true
				break
			}
		}
		if !owned && !inSubmitted {
			return SiteDomainConf{}, errs.New(errs.CodeBadRequest, "error.badRequest", "新主域名必须是站点已有域名（主域名或附加域名）: "+newPrimary)
		}
		taken, err := s.domainTakenByOther(site.ID, newPrimary)
		if err != nil {
			return SiteDomainConf{}, err
		}
		if taken {
			return SiteDomainConf{}, errs.New(errs.CodeBadRequest, "error.badRequest", "域名已被其他站点占用: "+newPrimary)
		}
		cleaned = switchedExtras(cleaned, site.Domain, newPrimary)
	}
	for _, d := range cleaned {
		if d == newPrimary {
			return SiteDomainConf{}, errs.Wrap(errs.ErrBadRequest, "附加域名不能与主域名重复: "+d)
		}
	}
	raw := ""
	if len(cleaned) > 0 {
		b, _ := json.Marshal(cleaned)
		raw = string(b)
	}
	updates := map[string]any{"domains": raw}
	if switching {
		// 证书联动：自签先按新主域名重签（失败整体不落库）；证书库条目仅检查并提示
		if site.CertDomain != "" {
			if site.CertID == 0 {
				if err := s.selfSign(ctx, newPrimary, nil); err != nil {
					return SiteDomainConf{}, err
				}
				updates["cert_domain"] = newPrimary
			} else {
				var cert model.Certificate
				if err := s.db.First(&cert, site.CertID).Error; err == nil && !certCovers(&cert, newPrimary) {
					certNotice = fmt.Sprintf("主域名已切换为 %s，但证书「%s」未覆盖该域名，HTTPS 访问新域名将告警，请重新签发或更换证书", newPrimary, cert.CertName)
				}
			}
		}
		updates["domain"] = newPrimary
	}
	if err := s.db.Model(site).Updates(updates).Error; err != nil {
		return SiteDomainConf{}, err
	}
	site, err = s.siteByID(id) // 回读：confTemplate 需要切换后的 Domain/CertDomain/Domains
	if err != nil {
		return SiteDomainConf{}, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return SiteDomainConf{}, err
	}
	out, err := s.GetDomainConf(id)
	if err != nil {
		return SiteDomainConf{}, err
	}
	out.CertNotice = certNotice
	return out, nil
}

// switchedExtras 计算切换主域名后的附加列表（纯函数）：移除新主域名（调用方仍留在列表的形态），
// 旧主域名缺失时补入尾部，保证站点域名集合守恒。
func switchedExtras(submitted []string, oldPrimary, newPrimary string) []string {
	kept := make([]string, 0, len(submitted)+1)
	hasOld := false
	for _, d := range submitted {
		if d == newPrimary {
			continue
		}
		if d == oldPrimary {
			hasOld = true
		}
		kept = append(kept, d)
	}
	if !hasOld {
		kept = append(kept, oldPrimary)
	}
	return kept
}

// domainTakenByOther 检查域名是否被其他站点占用（主域名或附加域名；站点量级小，内存比对）。
func (s *SiteService) domainTakenByOther(siteID uint, domain string) (bool, error) {
	var sites []model.Site
	if err := s.db.Where("id <> ?", siteID).Find(&sites).Error; err != nil {
		return false, err
	}
	for i := range sites {
		st := &sites[i]
		if st.Domain == domain {
			return true, nil
		}
		extra, _ := parseExtraDomains(st)
		for _, d := range extra {
			if strings.ToLower(strings.TrimSpace(d)) == domain {
				return true, nil
			}
		}
	}
	return false, nil
}

// certCovers 判断证书是否覆盖域名（主域名/其他域名/一级泛域名，泛域名不覆盖裸域与多级子域）。
func certCovers(cert *model.Certificate, domain string) bool {
	if cert.Domain == domain {
		return true
	}
	var alts []string
	if cert.AltDomains != "" {
		_ = json.Unmarshal([]byte(cert.AltDomains), &alts)
	}
	for _, a := range alts {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == domain {
			return true
		}
		if strings.HasPrefix(a, "*.") {
			if i := strings.IndexByte(domain, '.'); i > 0 && domain[i+1:] == a[2:] {
				return true
			}
		}
	}
	return false
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
	s = s.forSite(site)
	return SiteDefaultsConf{IndexFiles: site.IndexFiles, ErrorPage404: site.ErrorPage404}, nil
}

// UpdateDefaultsConf 更新（indexFiles 白名单校验）。
func (s *SiteService) UpdateDefaultsConf(ctx context.Context, id uint, conf SiteDefaultsConf) (SiteDefaultsConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteDefaultsConf{}, err
	}
	s = s.forSite(site)
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
	s = s.forSite(site)
	m := parseSiteMeta(site)
	return SiteProxyConf{Rules: m.ProxyRules, CacheEnable: m.CacheEnable, CacheDuration: m.CacheDuration}, nil
}

// UpdateProxyConf 更新反代规则与缓存。
func (s *SiteService) UpdateProxyConf(ctx context.Context, id uint, conf SiteProxyConf) (SiteProxyConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteProxyConf{}, err
	}
	s = s.forSite(site)
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
	s = s.forSite(site)
	return SiteRewriteConf{RewriteName: site.RewriteName, RewriteContent: site.RewriteContent}, nil
}

// UpdateRewriteConf 更新伪静态。
func (s *SiteService) UpdateRewriteConf(ctx context.Context, id uint, conf SiteRewriteConf) (SiteRewriteConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteRewriteConf{}, err
	}
	s = s.forSite(site)
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
	s = s.forSite(site)
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
	s = s.forSite(site)

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
	s = s.forSite(site)
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
