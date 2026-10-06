package service

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// MarketApp 内置应用定义。
type MarketApp struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Description string            `json:"description"`
	Params      []MarketAppParam  `json:"params"`
	render      func(app string, params map[string]string) string
}

// MarketAppParam 安装参数（端口/密码等）。
type MarketAppParam struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Default     string `json:"default"`
	IsPassword  bool   `json:"isPassword"`
}

// 内置精选应用（compose 模板复用 M3 引擎；镜像走 registry-mirrors）。
var builtinApps = []MarketApp{
	{
		ID: "nginx", Name: "Nginx", Category: "Web 服务",
		Description: "高性能 Web 服务器与反向代理",
		Params:      []MarketAppParam{{Key: "port", Label: "HTTP 端口", Default: "8081"}},
		render: func(app string, p map[string]string) string {
			return fmt.Sprintf(`services:
  nginx:
    image: nginx:stable-alpine
    container_name: app-%s
    ports:
      - "%s:80"
    restart: unless-stopped
`, app, p["port"])
		},
	},
	{
		ID: "uptime-kuma", Name: "Uptime Kuma", Category: "监控",
		Description: "开箱即用的服务可用性监控（自带拨测与状态页）",
		Params:      []MarketAppParam{{Key: "port", Label: "访问端口", Default: "3001"}},
		render: func(app string, p map[string]string) string {
			return fmt.Sprintf(`services:
  kuma:
    image: louislam/uptime-kuma:1
    container_name: app-%s
    ports:
      - "%s:3001"
    volumes:
      - ./data:/app/data
    restart: unless-stopped
`, app, p["port"])
		},
	},
	{
		ID: "wordpress", Name: "WordPress", Category: "建站",
		Description: "博客/CMS（含 MySQL）",
		Params: []MarketAppParam{
			{Key: "port", Label: "访问端口", Default: "8082"},
			{Key: "dbpassword", Label: "数据库密码", Default: "wp_db_2026", IsPassword: true},
		},
		render: func(app string, p map[string]string) string {
			return fmt.Sprintf(`services:
  wp:
    image: wordpress:latest
    container_name: app-%s
    ports:
      - "%s:80"
    environment:
      WORDPRESS_DB_HOST: %s-db
      WORDPRESS_DB_NAME: wordpress
      WORDPRESS_DB_USER: wp
      WORDPRESS_DB_PASSWORD: "%s"
    volumes:
      - ./wp:/var/www/html
    restart: unless-stopped
  db:
    image: mysql:8
    container_name: app-%s-db
    environment:
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wp
      MYSQL_PASSWORD: "%s"
      MYSQL_ROOT_PASSWORD: "%s"
    volumes:
      - ./db:/var/lib/mysql
    restart: unless-stopped
`, app, p["port"], app, p["dbpassword"], app, p["dbpassword"], p["dbpassword"])
		},
	},
	{
		ID: "memcached", Name: "Memcached", Category: "缓存",
		Description: "分布式内存缓存",
		Params:      []MarketAppParam{{Key: "port", Label: "端口", Default: "11211"}},
		render: func(app string, p map[string]string) string {
			return fmt.Sprintf(`services:
  memcached:
    image: memcached:alpine
    container_name: app-%s
    ports:
      - "%s:11211"
    restart: unless-stopped
`, app, p["port"])
		},
	},
}

// appIDPattern 应用安装 ID 白名单。
var appIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)

// MarketService 应用市场。
type MarketService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewMarketService 创建。
func NewMarketService(db *gorm.DB, nodes *NodeService) *MarketService {
	return &MarketService{db: db, nodes: nodes}
}

// List 应用列表（内置 + 1Panel 格式目录扫描）。
func (s *MarketService) List(ctx context.Context) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(builtinApps)+4)
	for _, a := range builtinApps {
		out = append(out, map[string]any{
			"id": a.ID, "name": a.Name, "category": a.Category,
			"description": a.Description, "params": a.Params, "source": "builtin",
		})
	}
	// 1Panel 格式：/opt/ypanel/apps/<app>/<version>/{docker-compose.yml,data.yml}
	ac, err := s.client()
	if err != nil {
		return out, nil
	}
	appsRoot, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path=%2Fopt%2Fypanel%2Fapps")
	if err == nil {
		for _, dir := range appsRoot.Entries {
			if !dir.IsDir {
				continue
			}
			versions, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(path.Join("/opt/ypanel/apps", dir.Name)))
			if err != nil {
				continue
			}
			for _, v := range versions.Entries {
				if !v.IsDir {
					continue
				}
				out = append(out, map[string]any{
					"id": dir.Name + "/" + v.Name, "name": dir.Name, "category": "1Panel 目录",
					"description": "版本 " + v.Name, "params": []MarketAppParam{}, "source": "1panel",
				})
				break // 每应用只列最新（目录序）
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	return out, nil
}

// Install 安装应用：写 compose 项目并 up（复用 M3 引擎，同名视为重建）。
func (s *MarketService) Install(ctx context.Context, appID string, params map[string]string) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	var content string
	project := "app-" + appID
	switch {
	case strings.Contains(appID, "/"):
		// 1Panel 格式：直接指向其 compose 文件目录安装
		parts := strings.SplitN(appID, "/", 2)
		if len(parts) != 2 || !appIDPattern.MatchString(parts[0]) {
			return nil, errs.Wrap(errs.ErrBadRequest, "应用 ID 不合法")
		}
		dir := filepath.ToSlash(filepath.Join("/opt/ypanel/apps", appID))
		// 拷贝 compose 文件内容到托管目录
		cfg, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(filepath.ToSlash(filepath.Join(dir, "docker-compose.yml"))))
		if err != nil {
			cfg, err = agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(filepath.ToSlash(filepath.Join(dir, "docker-compose.yaml"))))
			if err != nil {
				return nil, errs.Wrap(errs.ErrNotFound, "未找到 docker-compose.yml")
			}
		}
		content = cfg.Content
		project = "app-" + parts[0]
	default:
		var app *MarketApp
		for i := range builtinApps {
			if builtinApps[i].ID == appID {
				app = &builtinApps[i]
				break
			}
		}
		if app == nil {
			return nil, errs.Wrap(errs.ErrNotFound, "应用不存在")
		}
		if !appIDPattern.MatchString(appID) {
			return nil, errs.Wrap(errs.ErrBadRequest, "应用 ID 不合法")
		}
		merged := map[string]string{}
		for _, pm := range app.Params {
			merged[pm.Key] = pm.Default
		}
		for k, v := range params {
			merged[k] = v
		}
		content = app.render(appID, merged)
	}
	if _, err := agentclient.DoJSON[dto.ComposeWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/compose/config",
		&dto.ComposeWriteReq{Name: project, Content: content}); err != nil {
		return nil, err
	}
	out, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
		&dto.ComposeActionReq{Name: project})
	if err != nil {
		return nil, err
	}
	return map[string]any{"project": project, "output": (*out)["output"]}, nil
}

// Installed 已安装应用（compose 项目中 app- 前缀）。
func (s *MarketService) Installed(ctx context.Context) ([]dto.ComposeProject, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	projects, err := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects")
	if err != nil {
		return nil, err
	}
	out := []dto.ComposeProject{}
	for _, p := range projectsSafe(projects) {
		if strings.HasPrefix(p.Name, "app-") {
			out = append(out, p)
		}
	}
	return out, nil
}

// escapeURL2 query 转义（复用 site.go 的实现语义）。
func escapeURL2(s string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "?", "%3F", "#", "%23", "&", "%26", "+", "%2B", "/", "%2F")
	return r.Replace(s)
}

func (s *MarketService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}
