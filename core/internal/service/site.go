// Package service 站点管理：容器化 nginx + vhost 配置编排。
package service

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

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
    restart: unless-stopped
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

// confTemplate 生成站点配置。
func confTemplate(site *model.Site, ssl bool) string {
	var b strings.Builder
	if !ssl {
		if site.CertDomain != "" {
			// 有证书：80 跳 443
			fmt.Fprintf(&b, "# ypanel-site:%d\nserver {\n    listen 80;\n    server_name %s;\n    return 301 https://$host$request_uri;\n}\n\n", site.ID, site.Domain)
		}
		b.WriteString(fmt.Sprintf("# ypanel-site:%d\nserver {\n    listen %d", site.ID, map[bool]int{true: 443, false: 80}[ssl]))
		if ssl {
			b.WriteString(" ssl")
		}
		fmt.Fprintf(&b, ";\n    server_name %s;\n", site.Domain)
		if ssl {
			fmt.Fprintf(&b, "    ssl_certificate     /etc/nginx/certs/%s.crt;\n    ssl_certificate_key /etc/nginx/certs/%s.key;\n", site.CertDomain, site.CertDomain)
		}
	} else {
		b.WriteString(fmt.Sprintf("# ypanel-site:%d\nserver {\n    listen 443 ssl;\n    server_name %s;\n", site.ID, site.Domain))
		fmt.Fprintf(&b, "    ssl_certificate     /etc/nginx/certs/%s.crt;\n    ssl_certificate_key /etc/nginx/certs/%s.key;\n", site.CertDomain, site.CertDomain)
	}
	switch site.Type {
	case "static":
		fmt.Fprintf(&b, "    root /var/www/sites/%s;\n    index index.html;\n    location / { try_files $uri $uri/ =404; }\n", site.Name)
	case "proxy":
		fmt.Fprintf(&b, "    location / {\n        proxy_pass %s;\n        proxy_set_header Host $host;\n        proxy_set_header X-Real-IP $remote_addr;\n        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n        proxy_set_header X-Forwarded-Proto $scheme;\n    }\n", site.ProxyPass)
	}
	b.WriteString("}\n")
	return b.String()
}

// List 站点列表（元数据 + conf 对账）。
func (s *SiteService) List(ctx context.Context) ([]map[string]any, error) {
	var rows []model.Site
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
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
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "domain": r.Domain,
			"port": r.Port, "proxyPass": r.ProxyPass, "certDomain": r.CertDomain,
			"enabled": r.Enabled, "onDisk": onDisk, "nginxRunning": nginxRunning,
			"createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// Create 创建站点。
func (s *SiteService) Create(ctx context.Context, name, typ, domain string, port int, proxyPass string) (*model.Site, error) {
	if !siteNamePattern.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "站点名不合法（小写字母/数字/中划线）")
	}
	if typ != "static" && typ != "proxy" {
		return nil, errs.Wrap(errs.ErrBadRequest, "类型仅支持 static/proxy")
	}
	if !domainPattern.MatchString(strings.ToLower(domain)) {
		return nil, errs.Wrap(errs.ErrBadRequest, "域名不合法: "+domain)
	}
	domain = strings.ToLower(domain)
	if typ == "proxy" && !proxyTargetPattern.MatchString(proxyPass) {
		return nil, errs.Wrap(errs.ErrBadRequest, "反代目标不合法（http(s)://host:port）")
	}
	if port <= 0 {
		port = 80
	}
	var count int64
	_ = s.db.Model(&model.Site{}).Where("name = ? OR domain = ?", name, domain).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.siteExists", "站点名或域名已存在")
	}

	if err := s.ensureNginx(ctx); err != nil {
		return nil, err
	}
	site := &model.Site{Name: name, Type: typ, Domain: domain, Port: port, ProxyPass: proxyPass, Enabled: true}
	if err := s.db.Create(site).Error; err != nil {
		return nil, err
	}
	if err := s.writeConf(ctx, site, confTemplate(site, false)); err != nil {
		return nil, err
	}
	if typ == "static" {
		if err := s.writeStaticIndex(ctx, site); err != nil {
			return nil, err
		}
	}
	return site, nil
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
	return s.writeConf(ctx, site, confTemplate(site, true))
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
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "")); err != nil {
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

func (s *SiteService) siteByID(id uint) (*model.Site, error) {
	var site model.Site
	if err := s.db.First(&site, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.siteNotFound", "站点不存在")
	}
	return &site, nil
}

func escapeURL(s string) string { return urlQueryEscape(s) }

func urlQueryEscape(s string) string {
	// 与 api.escape 一致的简化实现（% 与 & 之外的常规字符足够）
	r := strings.NewReplacer(" ", "%20", "?", "%3F", "#", "%23", "&", "%26", "+", "%2B", "%", "%25")
	return r.Replace(s)
}
