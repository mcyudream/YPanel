package service

import (
	"context"
	"path"
	"regexp"
	"strings"


	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 站点识别（功能清单 §3.6-28）：扫描 conf.d 中未接管的 .conf 与独立 nginx 容器。

var (
	serverNameRe = regexp.MustCompile(`(?m)^\s*server_name\s+([^;]+);`)
	listenRe     = regexp.MustCompile(`(?m)^\s*listen\s+([^;]+);`)
	proxyPassRe  = regexp.MustCompile(`proxy_pass\s+([^;]+);`)
	rootRe       = regexp.MustCompile(`root\s+([^;]+);`)
	ypanelTagRe  = regexp.MustCompile(`(?m)^\s*#\s*ypanel-site:\d+`)
)

// DiscoveredSite 扫描发现的未接管站点。
type DiscoveredSite struct {
	File      string `json:"file"`      // conf 文件名（conf.d 下）
	Domain    string `json:"domain"`    // 首个 server_name
	Port      string `json:"port"`      // listen 值
	Type      string `json:"type"`      // static / proxy（按是否含 proxy_pass 判定）
	ProxyPass string `json:"proxyPass"` // 反代目标（如有）
	Root      string `json:"root"`      // 静态根目录（如有）
}

// ScanSites 扫描未接管的站点配置与独立 nginx 容器。
func (s *SiteService) ScanSites(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	list, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(nginxConfDir))
	if err != nil {
		return nil, err
	}

	// 已接管 conf 文件集合（按当前站点名 + 元数据里登记的原始文件名）
	adopted := map[string]bool{}
	var rows []model.Site
	_ = s.db.Find(&rows).Error
	for _, r := range rows {
		adopted[r.Name+".conf"] = true
		if r.OriginFile != "" {
			adopted[r.OriginFile] = true
		}
	}
	adopted["default.conf"] = true

	sites := []DiscoveredSite{}
	for _, e := range list.Entries {
		if e.IsDir || !strings.HasSuffix(e.Name, ".conf") || adopted[e.Name] {
			continue
		}
		out, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(path.Join(nginxConfDir, e.Name)))
		if err != nil {
			continue
		}
		content := out.Content
		if ypanelTagRe.MatchString(content) {
			continue // 有标记 = 已接管
		}
		ds := DiscoveredSite{File: e.Name}
		if m := serverNameRe.FindStringSubmatch(content); m != nil {
			ds.Domain = strings.Fields(strings.TrimSpace(m[1]))[0]
		}
		if m := listenRe.FindStringSubmatch(content); m != nil {
			ds.Port = strings.TrimSpace(m[1])
		}
		if m := proxyPassRe.FindStringSubmatch(content); m != nil {
			ds.Type = "proxy"
			ds.ProxyPass = strings.TrimSpace(m[1])
		} else {
			ds.Type = "static"
			if m := rootRe.FindStringSubmatch(content); m != nil {
				ds.Root = strings.TrimSpace(m[1])
			}
		}
		sites = append(sites, ds)
	}

	// 独立 nginx 容器（非 ypanel-nginx）
	containers := []map[string]string{}
	if cl, err := agentclient.GetJSON[[]dto.ContainerItem](ac, ctx, "/agent/v1/docker/containers"); err == nil {
		for _, c := range *cl {
			if c.State != "running" || c.Name == nginxContainer {
				continue
			}
			img := strings.ToLower(c.Image)
			if strings.Contains(img, "nginx") || strings.Contains(img, "caddy") || strings.Contains(img, "traefik") {
				containers = append(containers, map[string]string{
					"name": c.Name, "image": c.Image, "ports": portsSummary(c.Ports),
				})
			}
		}
	}

	return map[string]any{"sites": sites, "containers": containers}, nil
}

// Adopt 接管发现的站点：登记元数据（conf 文件沿用原名，不改动内容）。
func (s *SiteService) Adopt(ctx context.Context, file, domain, typ, proxyPass string) (*model.Site, error) {
	if !strings.HasSuffix(file, ".conf") || strings.Contains(file, "/") || strings.Contains(file, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "配置文件名不合法")
	}
	if typ != "static" && typ != "proxy" {
		return nil, errs.Wrap(errs.ErrBadRequest, "类型仅支持 static/proxy")
	}
	if !domainPattern.MatchString(strings.ToLower(domain)) {
		return nil, errs.Wrap(errs.ErrBadRequest, "域名不合法: "+domain)
	}
	domain = strings.ToLower(domain)
	var count int64
	_ = s.db.Model(&model.Site{}).Where("domain = ?", domain).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.siteExists", "该域名已存在接管记录")
	}
	// 从原文件名派生站点名
	base := strings.TrimSuffix(file, ".conf")
	name := base
	if !siteNamePattern.MatchString(name) {
		name = "adopted-" + strings.ToLower(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(base, "-"))
		name = strings.Trim(name, "-")
		if len(name) > 32 {
			name = name[:32]
		}
		if !siteNamePattern.MatchString(name) {
			name = "adopted-site"
		}
	}
	// 域名/名字冲突补丁
	var existing model.Site
	if err := s.db.Where("name = ?", name).First(&existing).Error; err == nil {
		name = name + "-" + randomHex(2)
	}

	site := &model.Site{
		Name: name, Type: typ, Domain: domain, ProxyPass: proxyPass,
		Enabled: true, OriginFile: file,
	}
	if err := s.db.Create(site).Error; err != nil {
		return nil, err
	}
	return site, nil
}

// portsSummary 端口摘要。
func portsSummary(ports []dto.PortBinding) string {
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.HostPort != "" {
			parts = append(parts, p.HostPort+"->"+p.ContainerPort)
		} else {
			parts = append(parts, p.ContainerPort)
		}
	}
	return strings.Join(parts, ",")
}
