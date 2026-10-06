// RuntimeService PHP 运行环境（php-fpm 容器化，compose 项目 rt-php-<name>）。
package service

import (
	"context"
	"fmt"
	"regexp"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// phpVersions 支持的 PHP 版本。
var phpVersions = map[string]string{
	"8.2": "php:8.2-fpm-alpine",
	"8.3": "php:8.3-fpm-alpine",
}

// RuntimeService 运行环境服务。
type RuntimeService struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewRuntimeService 创建。
func NewRuntimeService(db *gorm.DB, nodes *NodeService) *RuntimeService {
	return &RuntimeService{db: db, nodes: nodes}
}

func (s *RuntimeService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// phpComposeTemplate 生成 php-fpm compose（www 卷与 nginx 一致，路径对齐 fastcgi）。
func phpComposeTemplate(name, image string) string {
	return fmt.Sprintf(`services:
  php:
    image: %s
    container_name: php-%s
    volumes:
      - /opt/ypanel/nginx/www:/var/www
    networks:
      - 1panel-network
    restart: unless-stopped

networks:
  1panel-network:
    external: true
`, image, name)
}

// Create 创建运行环境（容器化）。
func (s *RuntimeService) Create(ctx context.Context, name, version string) (*model.Runtime, error) {
	if !siteNamePattern.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "名称不合法（小写字母/数字/中划线）")
	}
	image, ok := phpVersions[version]
	if !ok {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的 PHP 版本（8.2/8.3）")
	}
	var count int64
	_ = s.db.Model(&model.Runtime{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.runtimeExists", "运行环境名已存在")
	}
	row := &model.Runtime{Name: name, Version: version, Origin: "container", ComposeProject: "rt-php-" + name}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	if _, err := agentclient.DoJSON[dto.ComposeWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/compose/config",
		&dto.ComposeWriteReq{Name: row.ComposeProject, Content: phpComposeTemplate(name, image)}); err != nil {
		return nil, err
	}
	if _, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
		&dto.ComposeActionReq{Name: row.ComposeProject}); err != nil {
		return nil, err
	}
	return row, nil
}

// fcgiAddrPattern 外部 fastcgi 地址（host:port 或 unix:/path/socket.sock）。
var fcgiAddrPattern = regexp.MustCompile(`^([a-zA-Z0-9._-]+:[0-9]{1,5}|unix:/[^\s]{1,200})$`)

// AttachExternal 接管本机已有 php-fpm（fastcgi 直连，不创建容器）。
func (s *RuntimeService) AttachExternal(name, version, fcgiAddr, remark string) (*model.Runtime, error) {
	if !siteNamePattern.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "名称不合法（小写字母/数字/中划线）")
	}
	if version == "" {
		version = "unknown"
	}
	if !fcgiAddrPattern.MatchString(fcgiAddr) {
		return nil, errs.Wrap(errs.ErrBadRequest, "FastCGI 地址需为 host:port 或 unix:/path 形式")
	}
	var count int64
	_ = s.db.Model(&model.Runtime{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.runtimeExists", "运行环境名已存在")
	}
	row := &model.Runtime{
		Name: name, Version: version, Origin: "external", FCGIAddr: fcgiAddr,
		Remark: truncStr(remark, 255), ComposeProject: "-",
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// List 运行环境列表（含容器状态）。
func (s *RuntimeService) List(ctx context.Context) ([]map[string]any, error) {
	var rows []model.Runtime
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	projects, _ := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects")
	runningMap := map[string]bool{}
	for _, p := range projectsSafe(projects) {
		runningMap[p.Name] = p.Running > 0
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		running := false
		switch r.Origin {
		case "external":
			// 外部 fastcgi 以连通性为准（列表不做阻塞探测，站点绑定时校验）
			running = r.FCGIAddr != ""
		default:
			running = runningMap[r.ComposeProject]
		}
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "version": r.Version,
			"origin": r.Origin, "fcgiAddr": r.FCGIAddr, "remark": r.Remark,
			"composeProject": r.ComposeProject,
			"running":        running, "createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// Delete 删除运行环境（down + 元数据；www 数据保留）。
func (s *RuntimeService) Delete(ctx context.Context, id uint) error {
	var row model.Runtime
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.runtimeNotFound", "运行环境不存在")
	}
	if row.Origin == "external" {
		// 外部接管仅解除纳管
		return s.db.Delete(&model.Runtime{}, id).Error
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, _ = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/down",
		&dto.ComposeActionReq{Name: row.ComposeProject})
	return s.db.Delete(&model.Runtime{}, id).Error
}

// SetEnabled 启停。
func (s *RuntimeService) SetEnabled(ctx context.Context, id uint, up bool) error {
	var row model.Runtime
	if err := s.db.First(&row, id).Error; err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.Wrap(errs.ErrBadRequest, "外部运行环境由其所在主机管理，面板不支持启停")
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	action := "down"
	if up {
		action = "up"
	}
	_, err = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/"+action,
		&dto.ComposeActionReq{Name: row.ComposeProject})
	return err
}
