// Package service 站点管理：容器化 nginx + vhost 配置编排。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// nginx 容器与目录约定（agent 主机侧）。
const (
	nginxProject   = "ypanel-nginx"
	nginxContainer = "ypanel-nginx"
	nginxConfDir   = "/opt/ypanel/nginx/conf.d"
	nginxCertDir   = "/opt/ypanel/nginx/certs"
	nginxWwwDir    = "/opt/ypanel/nginx/www"
)

var domainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
var siteNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)
var proxyTargetPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9._-]+(:[0-9]{1,5})?$`)

// SiteService 站点服务。
type SiteService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewSiteService 创建站点服务。
func NewSiteService(db *gorm.DB, nodes *NodeService) *SiteService {
	return &SiteService{db: db, nodes: nodes}
}

func (s *SiteService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// nginxComposeTemplate nginx 容器 compose 配置。
func nginxComposeTemplate() string {
	return `services:
  nginx:
    image: nginx:stable-alpine
    container_name: ` + nginxContainer + `
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /opt/ypanel/nginx/conf.d:/etc/nginx/conf.d
      - /opt/ypanel/nginx/certs:/etc/nginx/certs
      - /opt/ypanel/nginx/www:/var/www
      - /opt/ypanel/nginx/logs:/var/log/nginx
      - /opt/ypanel/nginx/cache:/var/cache/nginx
    networks:
      - 1panel-network
    restart: unless-stopped

networks:
  1panel-network:
    external: true
`
}

// ensureNginx 安装并启动 nginx 容器（幂等）。
func (s *SiteService) ensureNginx(ctx context.Context) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	// 目录
	for _, d := range []string{nginxConfDir, nginxCertDir, nginxWwwDir} {
		if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir",
			&dto.FileMkdirReq{Path: d}); err != nil {
			return err
		}
	}
	// compose 配置 + up（已存在则覆盖无害）
	if _, err := agentclient.DoJSON[dto.ComposeWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/compose/config",
		&dto.ComposeWriteReq{Name: nginxProject, Content: nginxComposeTemplate()}); err != nil {
		return err
	}
	if _, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
		&dto.ComposeActionReq{Name: nginxProject}); err != nil {
		return err
	}
	// 默认配置（仅首次）
	if _, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL(path.Join(nginxConfDir, "default.conf"))); err == nil {
		return nil // 已有默认配置
	}
	defaultConf := `# YPanel nginx 默认配置：未匹配站点一律 404
server {
    listen 80 default_server;
    listen 443 ssl default_server;
    ssl_certificate     /etc/nginx/certs/default.crt;
    ssl_certificate_key /etc/nginx/certs/default.key;
    return 404;
}
`
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: path.Join(nginxConfDir, "default.conf"), Content: defaultConf}); err != nil {
		return err
	}
	// 默认自签证书（保证 443 default_server 可启动）
	return s.selfSign(ctx, "default", nil)
}

// Status nginx 运行状态。
func (s *SiteService) Status(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	projects, err := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects")
	if err != nil {
		return nil, err
	}
	installed, running := false, false
	for _, p := range projectsSafe(projects) {
		if p.Name == nginxProject {
			installed = true
			running = p.Running > 0
		}
	}
	var count int64
	_ = s.db.Model(&model.Site{}).Count(&count).Error
	return map[string]any{
		"installed": installed, "running": running,
		"sites": count,
		"hostIP": "",
	}, nil
}

// Install 安装 nginx。
func (s *SiteService) Install(ctx context.Context) error {
	return s.ensureNginx(ctx)
}

// ProxyRule 反代规则（location 前缀 → 后端）。
type ProxyRule struct {
	Prefix   string `json:"prefix"` // 如 /api（空 = "/"）
	Target   string `json:"target"` // http(s)://host:port
	WebSocket bool  `json:"ws"`
}

// siteMeta 站点生成期元数据（多域名/规则/日志/伪静态/缓存/自定义 location）。
type siteMeta struct {
	ExtraDomains    []string        `json:"extraDomains,omitempty"`
	ProxyRules      []ProxyRule     `json:"proxyRules,omitempty"`
	IndexFiles      string          `json:"indexFiles,omitempty"`
	LogsEnabled     bool            `json:"logsEnabled"`
	RewriteName     string          `json:"rewriteName,omitempty"`
	RewriteContent  string          `json:"rewriteContent,omitempty"`
	CustomLocations []CustomLocation `json:"customLocations,omitempty"`
	ErrorPage404    string          `json:"errorPage404,omitempty"`
	CacheEnable     bool            `json:"cacheEnable"`
	CacheDuration   string          `json:"cacheDuration,omitempty"`
	RuntimeID       uint            `json:"runtimeId,omitempty"`
	RuntimeContainer string         `json:"runtimeContainer,omitempty"`
}

// parseSiteMeta 解析站点附加元数据（兼容旧数据：仅主域名 + 默认规则）。
func parseSiteMeta(site *model.Site) siteMeta {
	m := siteMeta{IndexFiles: "index.html", LogsEnabled: site.LogsEnabled}
	if site.IndexFiles != "" {
		m.IndexFiles = site.IndexFiles
	}
	if site.Domains != "" {
		_ = json.Unmarshal([]byte(site.Domains), &m.ExtraDomains)
	}
	if site.ProxyRules != "" {
		_ = json.Unmarshal([]byte(site.ProxyRules), &m.ProxyRules)
	}
	if site.RewriteName != "" {
		m.RewriteName = site.RewriteName
	}
	if site.RewriteContent != "" {
		m.RewriteContent = site.RewriteContent
	}
	if site.CustomLocations != "" {
		_ = json.Unmarshal([]byte(site.CustomLocations), &m.CustomLocations)
	}
	m.ErrorPage404 = site.ErrorPage404
	m.CacheEnable = site.CacheEnable
	m.CacheDuration = site.CacheDuration
	m.RuntimeID = site.RuntimeID
	m.RuntimeContainer = site.RuntimeContainer
	// 兼容：旧数据默认规则 = 主域名 "/" -> ProxyPass
	if len(m.ProxyRules) == 0 && site.ProxyPass != "" && site.Type == "proxy" {
		m.ProxyRules = []ProxyRule{{Prefix: "/", Target: site.ProxyPass}}
	}
	return m
}

// validateDomains 校验域名列表（去重、合法）。
func validateDomains(domains []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if !domainPattern.MatchString(d) {
			return nil, errs.Wrap(errs.ErrBadRequest, "域名不合法: "+d)
		}
		if seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	if len(out) == 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "至少需要一个域名")
	}
	return out, nil
}

// SiteWaf 站点 WAF 配置。
type SiteWaf struct {
	DenyIPs    []string `json:"denyIps"`
	AllowIPs   []string `json:"allowIps"`
	DenyUAs    []string `json:"denyUAs"`
	RateEnable bool     `json:"rateEnable"`
	Rate       int      `json:"rate"`
	Burst      int      `json:"burst"`
}

// parseWaf 解析站点 WAF 配置（空 = 默认关闭）。
func parseWaf(site *model.Site) SiteWaf {
	w := SiteWaf{Rate: 10, Burst: 20}
	if site.WafJSON == "" {
		return w
	}
	_ = json.Unmarshal([]byte(site.WafJSON), &w)
	return w
}

var ipPattern = regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)

// sanitizeUA 片段去除 nginx 配置危险字符。
func sanitizeUA(s string) string {
	bad := func(r rune) bool {
		switch r {
		case '\\', '"', '\'', '{', '}', ';', '(', ')':
			return true
		}
		return false
	}
	return strings.Map(func(r rune) rune {
		if bad(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
}

// validateWaf 校验 WAF 配置。
func validateWaf(w SiteWaf) error {
	for _, ip := range append(append([]string{}, w.DenyIPs...), w.AllowIPs...) {
		if !ipPattern.MatchString(ip) {
			return errs.Wrap(errs.ErrBadRequest, "IP 不合法: "+ip)
		}
	}
	for _, ua := range w.DenyUAs {
		if sanitizeUA(ua) == "" || len(ua) > 64 {
			return errs.Wrap(errs.ErrBadRequest, "UA 关键字不合法（禁含引号/花括号等）")
		}
	}
	if w.RateEnable && (w.Rate < 1 || w.Rate > 10000 || w.Burst < 0 || w.Burst > 10000) {
		return errs.Wrap(errs.ErrBadRequest, "限流参数不合法")
	}
	return nil
}

// wafSection 生成 WAF 配置段（zone 在 conf.d 顶层=http 上下文；server 段由调用方缩进包裹使用）。
func wafSection(site *model.Site, w SiteWaf) (zoneTop, serverPart string) {
	zoneName := "rl_" + site.Name
	if w.RateEnable {
		zoneTop = fmt.Sprintf("limit_req_zone $binary_remote_addr zone=%s:1m rate=%dr/s;\n", zoneName, w.Rate)
	}
	var b strings.Builder
	for _, ip := range w.DenyIPs {
		fmt.Fprintf(&b, "    deny %s;\n", ip)
	}
	if len(w.AllowIPs) > 0 {
		for _, ip := range w.AllowIPs {
			fmt.Fprintf(&b, "    allow %s;\n", ip)
		}
		b.WriteString("    deny all;\n")
	}
	var uas []string
	for _, ua := range w.DenyUAs {
		if u := sanitizeUA(ua); u != "" {
			uas = append(uas, regexp.QuoteMeta(u))
		}
	}
	if len(uas) > 0 {
		fmt.Fprintf(&b, "    if ($http_user_agent ~* \"(%s)\") { return 403; }\n", strings.Join(uas, "|"))
	}
	if w.RateEnable {
		burst := w.Burst
		if burst == 0 {
			burst = w.Rate * 2
		}
		fmt.Fprintf(&b, "    limit_req zone=%s burst=%d nodelay;\n", zoneName, burst)
		b.WriteString("    limit_req_status 503;\n")
	}
	return zoneTop, b.String()
}

// SiteHTTPSCfg HTTPS 高级设置（B23，持久化于 Site.HTTPSJSON）。
type SiteHTTPSCfg struct {
	HTTPMode      string   `json:"httpMode"`      // redirect（默认）/ both / deny
	HSTS          bool     `json:"hsts"`          // Strict-Transport-Security
	HSTSSubdomain bool     `json:"hstsSubdomain"` // includeSubDomains
	TLSVersions   []string `json:"tlsVersions"`   // ["1.3","1.2"]（空=默认 1.3+1.2）
	Ciphers       string   `json:"ciphers"`       // 空=默认现代套件
	HTTP2         bool     `json:"http2"`
}

// parseHTTPS 解析站点 HTTPS 高级设置（含默认值）。
func parseHTTPS(site *model.Site) SiteHTTPSCfg {
	cfg := SiteHTTPSCfg{HTTPMode: "redirect", TLSVersions: []string{"1.3", "1.2"}, HTTP2: true}
	if site.HTTPSJSON == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(site.HTTPSJSON), &cfg)
	if cfg.HTTPMode != "redirect" && cfg.HTTPMode != "both" && cfg.HTTPMode != "deny" {
		cfg.HTTPMode = "redirect"
	}
	if len(cfg.TLSVersions) == 0 {
		cfg.TLSVersions = []string{"1.3", "1.2"}
	}
	return cfg
}

// defaultCiphers 现代 TLS 默认加密套件（对齐 1Panel 默认值）。
const defaultCiphers = "ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256"

// runDirSafe 校验运行目录（/ 开头相对站点 root 的路径，禁止穿越）。
func runDirSafe(d string) bool {
	if d == "" || d == "/" {
		return true
	}
	if !strings.HasPrefix(d, "/") || strings.Contains(d, "..") || strings.Contains(d, "\\") {
		return false
	}
	for _, seg := range strings.Split(strings.Trim(d, "/"), "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

// siteRoot 站点 root 指令（含运行目录）。
func siteRoot(site *model.Site) string {
	p := "/var/www/sites/" + site.Name
	if d := strings.TrimSuffix(site.RunDir, "/"); d != "" && d != "/" {
		p += d
	}
	return p
}

// confTemplate 生成站点配置。
func confTemplate(site *model.Site, ssl bool, waf SiteWaf) string {
	meta := parseSiteMeta(site)
	wafCfg := parseWaf(site)
	httpsCfg := parseHTTPS(site)
	zoneTop, wafServer := wafSection(site, wafCfg)
	var b strings.Builder
	b.WriteString(zoneTop)

	// S20 第二批配置域：http 上下文指令（limit_conn_zone / upstream）
	extra := parseSiteExtra(site)
	if lc := extra.LimitConn; lc != nil && lc.Enable {
		fmt.Fprintf(&b, "limit_conn_zone $binary_remote_addr zone=conn_%s:1m;\n", site.Name)
	}
	if lb := extra.LoadBalance; lb != nil && lb.Enable && site.Type == "proxy" {
		b.WriteString("upstream lb_" + site.Name + " {\n")
		if lb.Strategy != "" {
			b.WriteString("    " + lb.Strategy + ";\n")
		}
		for _, u := range lb.Upstreams {
			// upstream server 指令不允许 http:// 前缀
			addr := strings.TrimPrefix(strings.TrimPrefix(u.Address, "http://"), "https://")
			if u.Weight > 0 {
				fmt.Fprintf(&b, "    server %s weight=%d;\n", addr, u.Weight)
			} else {
				fmt.Fprintf(&b, "    server %s;\n", addr)
			}
		}
		b.WriteString("}\n")
	}

	// 日志（站点级落盘，logs 卷）
	if meta.LogsEnabled {
		fmt.Fprintf(&b, "access_log /var/log/nginx/%s.access.log;\n", site.Name)
		fmt.Fprintf(&b, "error_log  /var/log/nginx/%s.error.log;\n", site.Name)
	}

	names := append([]string{site.Domain}, meta.ExtraDomains...)
	serverNames := strings.Join(names, " ")

	// 反代缓存 zone（conf.d 顶层 = http 上下文，目录挂载于 cache 卷）
	if meta.CacheEnable && site.Type == "proxy" {
		dur := meta.CacheDuration
		if dur == "" {
			dur = "12h"
		}
		fmt.Fprintf(&b, "proxy_cache_path /var/cache/nginx/%s levels=1:2 keys_zone=cache_%s:10m inactive=%s max_size=1g;\n", site.Name, site.Name, dur)
	}

	// 伪静态（模板或自定义）
	rewriteContent := meta.RewriteContent
	if meta.RewriteName != "" && meta.RewriteName != "custom" && meta.RewriteName != "none" {
		if t, err := ResolveRewrite(meta.RewriteName); err == nil {
			rewriteContent = t
		}
	}

	// 有证书：80 段按 HTTP 模式（redirect 跳转 / both 共存 / deny 拒绝）+ 443 主段
	if site.CertDomain != "" {
		switch httpsCfg.HTTPMode {
		case "both":
			listenSSL := ""
			if ssl {
				listenSSL = " ssl"
			}
			b.WriteString(fmt.Sprintf("# ypanel-site:%d\nserver {\n    listen %d%s;\n    server_name %s;\n", site.ID, site.Port, listenSSL, serverNames))
			b.WriteString(wafServer)
			b.WriteString("}\n\n")
		case "deny":
			fmt.Fprintf(&b, "# ypanel-site:%d\nserver {\n    listen 80;\n    server_name %s;\n    return 444;\n}\n\n", site.ID, serverNames)
		default: // redirect
			fmt.Fprintf(&b, "# ypanel-site:%d\nserver {\n    listen 80;\n    server_name %s;\n    return 301 https://$host$request_uri;\n}\n\n", site.ID, serverNames)
		}
		listenSSLFlag := " ssl"
		if httpsCfg.HTTP2 {
			listenSSLFlag += " http2"
		}
		b.WriteString(fmt.Sprintf("# ypanel-site:%d\nserver {\n    listen 443%s;\n    server_name %s;\n", site.ID, listenSSLFlag, serverNames))
		fmt.Fprintf(&b, "    ssl_certificate     /etc/nginx/certs/%s.crt;\n    ssl_certificate_key /etc/nginx/certs/%s.key;\n", site.CertDomain, site.CertDomain)
		// TLS 协议版本与加密套件
		protos := make([]string, 0, len(httpsCfg.TLSVersions))
		for _, v := range httpsCfg.TLSVersions {
			if v == "1.3" || v == "1.2" || v == "1.1" || v == "1.0" {
				protos = append(protos, "TLSv"+v)
			}
		}
		if len(protos) == 0 {
			protos = []string{"TLSv1.3", "TLSv1.2"}
		}
		fmt.Fprintf(&b, "    ssl_protocols %s;\n", strings.Join(protos, " "))
		ciphers := httpsCfg.Ciphers
		if ciphers == "" {
			ciphers = defaultCiphers
		}
		fmt.Fprintf(&b, "    ssl_ciphers %s;\n", ciphers)
		b.WriteString("    ssl_prefer_server_ciphers off;\n")
		b.WriteString("    ssl_session_cache shared:SSL:10m;\n")
		b.WriteString("    ssl_session_timeout 10m;\n")
		if httpsCfg.HSTS {
			v := `add_header Strict-Transport-Security "max-age=31536000`
			if httpsCfg.HSTSSubdomain {
				v += "; includeSubDomains"
			}
			v += `" always;` + "\n"
			b.WriteString("    " + v)
		}
		// HTTP 共存模式下 443 段同样生成站点内容，主流程继续（下方 switch）
	} else {
		// 整站重定向：80 段直接 return（有证书时 80 已被 301 到 https，重定向域不生效）
		if rd := extra.Redirect; rd != nil && rd.Enable && rd.Target != "" {
			code := rd.Code
			if code == 0 {
				code = 301
			}
			fmt.Fprintf(&b, "# ypanel-site:%d\nserver {\n    listen %d;\n    server_name %s;\n    return %d %s;\n}\n", site.ID, site.Port, serverNames, code, rd.Target)
			return b.String()
		}
		listenSSL := ""
		if ssl {
			listenSSL = " ssl"
		}
		b.WriteString(fmt.Sprintf("# ypanel-site:%d\nserver {\n    listen %d%s;\n    server_name %s;\n", site.ID, site.Port, listenSSL, serverNames))
	}

	b.WriteString(wafServer)

	// S20 第二批：server 上下文指令
	if al := extra.AntiLeech; al != nil && al.Enable {
		parts := []string{}
		if al.AllowNone {
			parts = append(parts, "none")
		}
		if al.AllowBlocked {
			parts = append(parts, "blocked")
		}
		parts = append(parts, "server_names")
		parts = append(parts, al.ValidReferers...)
		fmt.Fprintf(&b, "    valid_referers %s;\n", strings.Join(parts, " "))
		code := al.ReturnCode
		if code != 403 && code != 404 {
			code = 403
		}
		fmt.Fprintf(&b, "    if ($invalid_referer) { return %d; }\n", code)
	}
	if ab := extra.AuthBasic; ab != nil && ab.Enable && len(ab.Users) > 0 {
		realm := ab.Realm
		if realm == "" {
			realm = "Restricted"
		}
		fmt.Fprintf(&b, "    auth_basic %s;\n", quoteGo(realm))
		fmt.Fprintf(&b, "    auth_basic_user_file /etc/nginx/conf.d/%s.htpasswd;\n", site.Name)
	}
	if co := extra.CORS; co != nil && co.Enable && len(co.AllowOrigins) > 0 {
		b.WriteString("    # CORS\n")
		if len(co.AllowOrigins) == 1 && co.AllowOrigins[0] == "*" {
			b.WriteString("    add_header Access-Control-Allow-Origin \"*\" always;\n")
		} else {
			b.WriteString("    add_header Access-Control-Allow-Origin $http_origin always;\n")
		}
		if co.AllowCredentials {
			b.WriteString("    add_header Access-Control-Allow-Credentials \"true\" always;\n")
		}
		fmt.Fprintf(&b, "    add_header Access-Control-Allow-Methods \"%s\" always;\n", strings.Join(co.AllowMethods, ", "))
		if len(co.AllowHeaders) > 0 {
			fmt.Fprintf(&b, "    add_header Access-Control-Allow-Headers \"%s\" always;\n", strings.Join(co.AllowHeaders, ", "))
		}
		if co.MaxAge > 0 {
			fmt.Fprintf(&b, "    add_header Access-Control-Max-Age \"%d\" always;\n", co.MaxAge)
		}
		b.WriteString("    if ($request_method = 'OPTIONS') { return 204; }\n")
	}
	if rp := extra.RealIP; rp != nil && rp.Enable && len(rp.TrustedProxies) > 0 {
		b.WriteString("    # real IP\n")
		for _, p := range rp.TrustedProxies {
			fmt.Fprintf(&b, "    set_real_ip_from %s;\n", p)
		}
		fmt.Fprintf(&b, "    real_ip_header %s;\n", rp.Header)
	}
	if lc := extra.LimitConn; lc != nil && lc.Enable {
		fmt.Fprintf(&b, "    limit_conn conn_%s %d;\n", site.Name, lc.ConnPerIP)
	}

	// 自定义 404 页（static 类型）
	if site.Type == "static" && meta.ErrorPage404 != "" {
		fmt.Fprintf(&b, "    error_page 404 %s;\n", meta.ErrorPage404)
	}

	// 反代缓存指令开关（proxy location 生成时注入）
	cacheOn := meta.CacheEnable && site.Type == "proxy"

	switch site.Type {
	case "static":
		index := meta.IndexFiles
		if index == "" {
			index = "index.html"
		}
		fmt.Fprintf(&b, "    root %s;\n", siteRoot(site))
		if index != "" {
			fmt.Fprintf(&b, "    index %s;\n", index)
		}
		// 伪静态含 location / 时替代默认块（SPA 前端路由场景）
		if strings.Contains(rewriteContent, "location /") {
			b.WriteString(rewriteContent)
			b.WriteString("\n")
			rewriteContent = ""
		} else {
			b.WriteString("    location / { try_files $uri $uri/ =404; }\n")
		}
	case "php":
		fmt.Fprintf(&b, "    root %s;\n", siteRoot(site))
		if meta.IndexFiles != "" {
			fmt.Fprintf(&b, "    index %s;\n", meta.IndexFiles)
		}
		if meta.RuntimeContainer != "" {
			fmt.Fprintf(&b, "    location ~ \\.php$ {\n")
			fmt.Fprintf(&b, "        fastcgi_pass %s:9000;\n", meta.RuntimeContainer)
			b.WriteString("        fastcgi_index index.php;\n")
			b.WriteString("        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
			b.WriteString("        include fastcgi_params;\n")
			b.WriteString("    }\n")
		}
		if strings.Contains(rewriteContent, "location /") {
			b.WriteString(rewriteContent)
			b.WriteString("\n")
			rewriteContent = ""
		} else {
			b.WriteString("    location / { try_files $uri $uri/ /index.php?$query_string; }\n")
		}
	case "proxy":
		for _, r := range meta.ProxyRules {
			prefix := r.Prefix
			if prefix == "" {
				prefix = "/"
			}
			fmt.Fprintf(&b, "    location %s {\n", prefix)
			// LB: 
			if lb2 := extra.LoadBalance; lb2 != nil && lb2.Enable && len(lb2.Upstreams) > 1 {
				fmt.Fprintf(&b, "        proxy_pass http://lb_%s;\n", site.Name)
			} else {
				fmt.Fprintf(&b, "        proxy_pass %s;\n", r.Target)
			}
			b.WriteString("        proxy_set_header Host $host;\n")
			b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
			if r.WebSocket {
				b.WriteString("        proxy_http_version 1.1;\n")
				b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
				b.WriteString("        proxy_set_header Connection \"upgrade\";\n")
				b.WriteString("        proxy_read_timeout 300s;\n")
			}
			if cacheOn {
				dur := meta.CacheDuration
				if dur == "" {
					dur = "12h"
				}
				b.WriteString("        proxy_cache cache_" + site.Name + ";\n")
				b.WriteString("        proxy_cache_valid 200 " + dur + ";\n")
				b.WriteString("        add_header X-Cache-Status $upstream_cache_status;\n")
			}
			b.WriteString("    }\n")
		}
	}

	// 伪静态 / 自定义 location（server 段尾部）
	if rewriteContent != "" {
		b.WriteString(rewriteContent)
		b.WriteString("\n")
	}
	for _, cl := range meta.CustomLocations {
		if strings.TrimSpace(cl.Content) != "" {
			if cl.Comment != "" {
				b.WriteString("    # " + cl.Comment + "\n")
			}
			b.WriteString(cl.Content)
			b.WriteString("\n")
		}
	}

	b.WriteString("}\n")
	return b.String()
}

// writeConf 写配置 + 校验 + 重载（失败回滚）。
func (s *SiteService) writeConf(ctx context.Context, site *model.Site, content string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	confPath := path.Join(nginxConfDir, site.Name+".conf")
	orig := ""
	if out, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL(confPath)); err == nil {
		orig = out.Content
	}
	if err := s.writeViaFiles(ctx, confPath, content); err != nil {
		return err
	}
	if err := s.reloadNginx(ctx); err != nil {
		// 回滚
		if orig != "" {
			_ = s.writeViaFiles(ctx, confPath, orig)
		} else {
			_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
				&dto.FileDeleteReq{Paths: []string{confPath}})
		}
		_ = s.reloadNginx(ctx)
		return err
	}
	return nil
}

// writeViaFiles 经 agent 文件通道写配置（core 进程内无法直写远端，统一走 HTTP）。
func (s *SiteService) writeViaFiles(ctx context.Context, p, content string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: p, Content: content})
	return err
}

// reloadNginx 校验并重载 nginx。
func (s *SiteService) reloadNginx(ctx context.Context) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "docker exec " + nginxContainer + " nginx -t", TimeoutSecs: 30})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "nginx 配置校验失败: "+firstLine(out.Output))
	}
	out, err = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "docker exec " + nginxContainer + " nginx -s reload", TimeoutSecs: 30})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "nginx 重载失败: "+firstLine(out.Output))
	}
	return nil
}

// selfSign 自签证书（openssl）。
func (s *SiteService) selfSign(ctx context.Context, domain string, _ any) error {
	if !domainPattern.MatchString(domain) && domain != "default" {
		return errs.Wrap(errs.ErrBadRequest, "证书域名不合法")
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	key := path.Join(nginxCertDir, domain+".key")
	crt := path.Join(nginxCertDir, domain+".crt")
	cmd := fmt.Sprintf("openssl req -x509 -nodes -newkey rsa:2048 -days 365 -keyout %s -out %s -subj '/CN=%s'", key, crt, domain)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 60})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "自签证书生成失败: "+firstLine(out.Output))
	}
	return nil
}

// SiteExtConfig 站点增强配置（M15 聚合读写）。
type SiteExtConfig struct {
	RewriteName     string           `json:"rewriteName"`
	RewriteContent  string           `json:"rewriteContent"`
	CustomLocations []CustomLocation `json:"customLocations"`
	ErrorPage404    string           `json:"errorPage404"`
	CacheEnable     bool             `json:"cacheEnable"`
	CacheDuration   string           `json:"cacheDuration"`
}

// GetExt 读取增强配置。
func (s *SiteService) GetExt(id uint) (SiteExtConfig, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteExtConfig{}, err
	}
	m := parseSiteMeta(site)
	return SiteExtConfig{
		RewriteName: m.RewriteName, RewriteContent: m.RewriteContent,
		CustomLocations: m.CustomLocations, ErrorPage404: m.ErrorPage404,
		CacheEnable: m.CacheEnable, CacheDuration: m.CacheDuration,
	}, nil
}

// UpdateExt 更新增强配置并重写 conf（校验回滚链路）。
func (s *SiteService) UpdateExt(ctx context.Context, id uint, ext SiteExtConfig) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	if err := validateCustomLocations(ext.CustomLocations); err != nil {
		return err
	}
	if ext.CacheDuration == "" {
		ext.CacheDuration = "12h"
	}
	meta := parseSiteMeta(site)
	meta.RewriteName = ext.RewriteName
	meta.RewriteContent = ext.RewriteContent
	meta.CustomLocations = ext.CustomLocations
	meta.ErrorPage404 = ext.ErrorPage404
	meta.CacheEnable = ext.CacheEnable
	meta.CacheDuration = ext.CacheDuration

	// 持久化
	clJSON := ""
	if len(ext.CustomLocations) > 0 {
		b, _ := json.Marshal(ext.CustomLocations)
		clJSON = string(b)
	}
	updates := map[string]any{
		"rewrite_name": ext.RewriteName, "rewrite_content": ext.RewriteContent,
		"custom_locations": clJSON, "error_page404": ext.ErrorPage404,
		"cache_enable": ext.CacheEnable, "cache_duration": ext.CacheDuration,
	}
	if err := s.db.Model(site).Updates(updates).Error; err != nil {
		return err
	}
	return s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site)))
}

// GetWaf 读取站点 WAF 配置。
func (s *SiteService) GetWaf(id uint) (SiteWaf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SiteWaf{}, err
	}
	return parseWaf(site), nil
}

// UpdateWaf 更新 WAF 配置并重写 conf（校验回滚链路）。
func (s *SiteService) UpdateWaf(ctx context.Context, id uint, w SiteWaf) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	if err := validateWaf(w); err != nil {
		return err
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return err
	}
	if err := s.db.Model(site).Update("waf_json", string(raw)).Error; err != nil {
		return err
	}
	return s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", w))
}

// IssueSelfSigned 为站点签发自签证书并启用 443。
func (s *SiteService) IssueSelfSigned(ctx context.Context, id uint) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	if err := s.selfSign(ctx, site.Domain, nil); err != nil {
		return err
	}
	site.CertDomain = site.Domain
	if err := s.db.Model(site).Update("cert_domain", site.CertDomain).Error; err != nil {
		return err
	}
	return s.writeConf(ctx, site, confTemplate(site, true, parseWaf(site)))
}

// UpdateConfig 手动编辑配置（保存即校验+重载，失败回滚）。
func (s *SiteService) UpdateConfig(ctx context.Context, id uint, content string) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	return s.writeConf(ctx, site, content)
}

// Config 读取站点配置。
func (s *SiteService) Config(ctx context.Context, id uint) (string, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return "", err
	}
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL(path.Join(nginxConfDir, site.Name+".conf")))
	if err != nil {
		return "", err
	}
	return out.Content, nil
}

// SetEnabled 启用/禁用（conf 删除/重建）。
func (s *SiteService) SetEnabled(ctx context.Context, id uint, enabled bool) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	confPath := path.Join(nginxConfDir, site.Name+".conf")
	if !enabled {
		if _, err := agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{confPath}}); err != nil {
			return err
		}
		if err := s.reloadNginx(ctx); err != nil {
			return err
		}
		return s.db.Model(site).Update("enabled", false).Error
	}
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return err
	}
	return s.db.Model(site).Update("enabled", true).Error
}

// Delete 删除站点。
func (s *SiteService) Delete(ctx context.Context, id uint, purgeFiles bool) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
		&dto.FileDeleteReq{Paths: []string{path.Join(nginxConfDir, site.Name + ".conf")}})
	if purgeFiles {
		_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{path.Join(nginxWwwDir, "sites", site.Name)}})
	}
	_ = s.reloadNginx(ctx)
	return s.db.Delete(&model.Site{}, id).Error
}

// GetByIDF 单条站点查询（F8：详情页免拉全量列表）。
func (s *SiteService) GetByIDF(id uint) (*model.Site, error) {
	return s.siteByID(id)
}

func (s *SiteService) siteByID(id uint) (*model.Site, error) {
	var site model.Site
	if err := s.db.First(&site, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.siteNotFound", "站点不存在")
	}
	return &site, nil
}

// escapeURL query 转义（% 与 & 之外的常规字符足够）。
func escapeURL(s string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "?", "%3F", "#", "%23", "&", "%26", "+", "%2B")
	return r.Replace(s)
}

// List 站点列表（元数据 + conf 对账 + 分组/备注/证书过期时间）。
func (s *SiteService) List(ctx context.Context) ([]map[string]any, error) {
	var rows []model.Site
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	// 分组名与证书过期时间映射（B23）
	groupNames := map[uint]string{}
	var groups []model.SiteGroup
	_ = s.db.Find(&groups).Error
	for _, g := range groups {
		groupNames[g.ID] = g.Name
	}
	certExpiry := map[uint]*time.Time{}
	var certs []model.Certificate
	_ = s.db.Select("id, not_after").Find(&certs).Error
	for i := range certs {
		certExpiry[certs[i].ID] = certs[i].NotAfter
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	projects, _ := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects")
	nginxRunning := false
	for _, p := range projectsSafe(projects) {
		if p.Name == nginxProject {
			nginxRunning = p.Running > 0
		}
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		confPath := path.Join(nginxConfDir, r.Name+".conf")
		onDisk := false
		if _, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL(confPath)); err == nil {
			onDisk = true
		}
		extra := []string{}
		if r.Domains != "" {
			_ = json.Unmarshal([]byte(r.Domains), &extra)
		}
		groupName := groupNames[r.GroupID]
		if groupName == "" {
			groupName = "默认"
		}
		var notAfter *time.Time
		if r.CertID != 0 {
			notAfter = certExpiry[r.CertID]
		}
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type,
			"domain": r.Domain, "domains": extra,
			"port": r.Port, "proxyPass": r.ProxyPass, "certDomain": r.CertDomain,
			"indexFiles": r.IndexFiles, "logsEnabled": r.LogsEnabled,
			"groupId": r.GroupID, "groupName": groupName, "remark": r.Remark,
			"runDir": r.RunDir, "certId": r.CertID, "certNotAfter": notAfter,
			"enabled": r.Enabled, "onDisk": onDisk, "nginxRunning": nginxRunning,
			"createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// Create 创建站点（M13：多域名/反代规则/默认文档/日志）。
func (s *SiteService) Create(ctx context.Context, req SiteCreateInput) (*model.Site, error) {
	if !siteNamePattern.MatchString(req.Name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "站点名不合法（小写字母/数字/中划线）")
	}
	if req.Type != "static" && req.Type != "proxy" && req.Type != "php" {
		return nil, errs.Wrap(errs.ErrBadRequest, "类型仅支持 static/proxy/php")
	}
	domains, err := validateDomains(append([]string{req.Domain}, req.ExtraDomains...))
	if err != nil {
		return nil, err
	}
	domain := domains[0]
	extraJSON := ""
	if len(domains) > 1 {
		b, _ := json.Marshal(domains[1:])
		extraJSON = string(b)
	}
	runtimeContainer := ""
	if req.Type == "php" {
		if req.RuntimeID == 0 {
			return nil, errs.Wrap(errs.ErrBadRequest, "PHP 站点需绑定运行环境")
		}
		var rt model.Runtime
		if err := s.db.First(&rt, req.RuntimeID).Error; err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "运行环境不存在")
		}
		runtimeContainer = "php-" + rt.Name
	}
	if req.Type == "proxy" {
		if len(req.ProxyRules) == 0 && req.ProxyPass == "" {
			return nil, errs.Wrap(errs.ErrBadRequest, "反代站点需要至少一条规则")
		}
		for _, r := range req.ProxyRules {
			if !proxyTargetPattern.MatchString(r.Target) {
				return nil, errs.Wrap(errs.ErrBadRequest, "反代目标不合法: "+r.Target)
			}
		}
	}
	var count int64
	_ = s.db.Model(&model.Site{}).Where("name = ? OR domain = ?", req.Name, domain).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.siteExists", "站点名或域名已存在")
	}
	if err := s.ensureNginx(ctx); err != nil {
		return nil, err
	}
	rulesJSON := ""
	if len(req.ProxyRules) > 0 {
		b, _ := json.Marshal(req.ProxyRules)
		rulesJSON = string(b)
	}
	indexFiles := req.IndexFiles
	if indexFiles == "" {
		indexFiles = "index.html"
	}
	if req.RunDir != "" && !runDirSafe(req.RunDir) {
		return nil, errs.Wrap(errs.ErrBadRequest, "运行目录不合法（需 / 开头且不含 ..）")
	}
	site := &model.Site{
		Name: req.Name, Type: req.Type, Domain: domain, Domains: extraJSON,
		Port: req.Port, ProxyPass: req.ProxyPass, ProxyRules: rulesJSON,
		IndexFiles: indexFiles, LogsEnabled: true, Enabled: true,
		RuntimeID: req.RuntimeID, RuntimeContainer: runtimeContainer,
		GroupID: req.GroupID, Remark: strings.TrimSpace(req.Remark), RunDir: req.RunDir,
	}
	if err := s.db.Create(site).Error; err != nil {
		return nil, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, false, parseWaf(site))); err != nil {
		return nil, err
	}
	if req.Type == "static" {
		if err := s.writeStaticIndex(ctx, site); err != nil {
			return nil, err
		}
	}
	if req.Type == "php" {
		if err := s.writePhpProbe(ctx, site); err != nil {
			return nil, err
		}
	}
	return site, nil
}

// writePhpProbe 写 PHP 探针（验收 php-fpm 链路）。
func (s *SiteService) writePhpProbe(ctx context.Context, site *model.Site) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	dir := path.Join(nginxWwwDir, "sites", site.Name)
	if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir", &dto.FileMkdirReq{Path: dir}); err != nil {
		return err
	}
	probe := "<?php\necho 'PHP ' . PHP_VERSION . ' | YPanel Runtime OK';\n"
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: path.Join(dir, "index.php"), Content: probe})
	return err
}

// writeStaticIndex 生成静态站欢迎页。
func (s *SiteService) writeStaticIndex(ctx context.Context, site *model.Site) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	dir := path.Join(nginxWwwDir, "sites", site.Name)
	if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir", &dto.FileMkdirReq{Path: dir}); err != nil {
		return err
	}
	html := fmt.Sprintf(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>%s</title></head>
<body style="font-family:system-ui;display:flex;align-items:center;justify-content:center;height:100vh;margin:0">
<div style="text-align:center"><h1>%s</h1><p>由 YPanel 创建的静态站点</p></div></body></html>`, site.Domain, site.Domain)
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: path.Join(dir, "index.html"), Content: html})
	return err
}

// SiteCreateInput 创建站点输入。
type SiteCreateInput struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Domain       string      `json:"domain"`
	ExtraDomains []string    `json:"extraDomains"`
	Port         int         `json:"port"`
	ProxyPass    string      `json:"proxyPass"`
	ProxyRules   []ProxyRule `json:"proxyRules"`
	IndexFiles   string      `json:"indexFiles"`
	RuntimeID    uint        `json:"runtimeId"`
	GroupID      uint        `json:"groupId"`
	Remark       string      `json:"remark"`
	RunDir       string      `json:"runDir"`
}

// SiteMetaInput 站点元信息编辑（分组/备注）。
type SiteMetaInput struct {
	GroupID *uint   `json:"groupId"`
	Remark  *string `json:"remark"`
}

// UpdateMeta 更新分组/备注（不触发 nginx 变更）。
func (s *SiteService) UpdateMeta(id uint, in SiteMetaInput) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	updates := map[string]any{}
	if in.GroupID != nil {
		if *in.GroupID != 0 {
			var g model.SiteGroup
			if err := s.db.First(&g, *in.GroupID).Error; err != nil {
				return errs.Wrap(errs.ErrBadRequest, "分组不存在")
			}
		}
		updates["group_id"] = *in.GroupID
	}
	if in.Remark != nil {
		updates["remark"] = strings.TrimSpace(*in.Remark)
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(site).Updates(updates).Error
}

// GetRunDir 运行目录读取（含 root 与子目录列表）。
func (s *SiteService) GetRunDir(ctx context.Context, id uint) (map[string]any, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	hostDir := path.Join(nginxWwwDir, "sites", site.Name)
	// 列出站点根目录下的子目录（经 agent files/list）
	resp, err := agentclient.GetJSON[struct {
		Entries []struct {
			Name  string `json:"name"`
			IsDir bool   `json:"isDir"`
		} `json:"entries"`
	}](ac, ctx, "/agent/v1/files/list?path="+escapeURL(hostDir))
	subdirs := []string{}
	if err == nil {
		for _, e := range resp.Entries {
			if e.IsDir && !strings.HasPrefix(e.Name, ".") {
				subdirs = append(subdirs, "/"+e.Name)
			}
		}
	}
	return map[string]any{
		"root":     siteRoot(site),
		"hostRoot": hostDir,
		"runDir":   site.RunDir,
		"subdirs":  subdirs,
	}, nil
}

// UpdateRunDir 更新运行目录并重载（PHP 框架二级目录场景，如 /public）。
func (s *SiteService) UpdateRunDir(ctx context.Context, id uint, runDir string) error {
	site, err := s.siteByID(id)
	if err != nil {
		return err
	}
	runDir = strings.TrimSpace(runDir)
	if runDir == "/" {
		runDir = ""
	}
	if !runDirSafe(runDir) {
		return errs.Wrap(errs.ErrBadRequest, "运行目录不合法（需 / 开头、不含 ..，如 /public）")
	}
	if err := s.db.Model(site).Update("run_dir", runDir).Error; err != nil {
		return err
	}
	return s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site)))
}

// ApplyCert 将证书库证书绑定到站点并启用 HTTPS（B23）。
func (s *SiteService) ApplyCert(ctx context.Context, site *model.Site, certID uint, certName string) error {
	site.CertDomain = certName
	site.CertID = certID
	if err := s.db.Model(site).Updates(map[string]any{"cert_domain": certName, "cert_id": certID}).Error; err != nil {
		return err
	}
	return s.writeConf(ctx, site, confTemplate(site, true, parseWaf(site)))
}

// SiteLogs 读取站点日志（logs 卷内站点级文件，tail 通道）。
func (s *SiteService) SiteLogs(ctx context.Context, id uint, logType, tail string) (string, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return "", err
	}
	if logType != "access" && logType != "error" {
		logType = "access"
	}
	if tail == "" {
		tail = "200"
	}
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("tail -n %s /opt/ypanel/nginx/logs/%s.%s.log 2>/dev/null", tail, site.Name, logType)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 30})
	if err != nil {
		return "", err
	}
	return out.Output, nil
}
