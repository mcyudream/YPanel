// 站点非标端口对账（M50）：站点表为单一事实源，创建/停用/删除/改端口/模式切换/启动统一走对账。
// container 模式：重写 nginx compose ports 段（80/443 + 附加端口）并 up -d（compose 仅在配置
// 变化时重建容器）；host 模式：逐端口联动防火墙放行/收回。SitePortLease 登记系统侧落点，
// 供防火墙页展示"站点托管"放行项与跨模式切换回收。
package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 站点基础端口：compose 固定映射，不进登记表。
var siteBasePorts = map[int]bool{80: true, 443: true}

// siteDesiredExtraPorts 站点行 → 附加端口集（纯函数）：enabled 且 port 为非标端口，升序去重。
func siteDesiredExtraPorts(rows []model.Site) []int {
	seen := map[int]bool{}
	ports := []int{}
	for _, r := range rows {
		if !r.Enabled || siteBasePorts[r.Port] || r.Port < 1 || r.Port > 65535 || seen[r.Port] {
			continue
		}
		seen[r.Port] = true
		ports = append(ports, r.Port)
	}
	sort.Ints(ports)
	return ports
}

// siteLeaseDiff 对账纯函数：登记表与 desired 的差异（toAdd 待分配 / toRemove 待回收）。
func siteLeaseDiff(desired []int, leases []model.SitePortLease) (toAdd, toRemove []int) {
	desiredSet := map[int]bool{}
	for _, p := range desired {
		desiredSet[p] = true
	}
	leaseSet := map[int]bool{}
	for _, l := range leases {
		leaseSet[l.Port] = true
		if !desiredSet[l.Port] {
			toRemove = append(toRemove, l.Port)
		}
	}
	for _, p := range desired {
		if !leaseSet[p] {
			toAdd = append(toAdd, p)
		}
	}
	return toAdd, toRemove
}

// SetPortDeps 注入端口对账依赖（main 装配期调用；未注入时对账降级为仅登记表同步）。
func (s *SiteService) SetPortDeps(fw *FirewallService, nat *NatForwardService) {
	s.fw, s.nat = fw, nat
}

// validateSitePort 站点 listen 端口校验（0 = 默认 80）。
func validateSitePort(port int) (int, error) {
	if port == 0 {
		return 80, nil
	}
	if port < 1 || port > 65535 {
		return 0, errs.New(errs.CodeBadRequest, "error.badRequest", "端口不合法（1-65535）")
	}
	return port, nil
}

// nginxComposeTemplate nginx 容器 compose 配置（extraPorts = 附加映射端口，宿主与容器同端口）。
func nginxComposeTemplate(extraPorts []int) string {
	portLines := `- "80:80"
      - "443:443"`
	for _, p := range extraPorts {
		portLines += fmt.Sprintf("\n      - \"%d:%d\"", p, p)
	}
	return `services:
  nginx:
    image: nginx:stable-alpine
    container_name: ` + nginxContainer + `
    ports:
      ` + portLines + `
    volumes:
      - /opt/ypanel/nginx/conf.d:/etc/nginx/conf.d
      - /opt/ypanel/nginx/certs:/etc/nginx/certs
      - /opt/ypanel/nginx/www:/var/www
      - /opt/ypanel/nginx/logs:/var/log/nginx
      - /opt/ypanel/nginx/cache:/var/cache/nginx
    networks:
      - ypanel_default
    restart: unless-stopped

networks:
  ypanel_default:
    external: true
`
}

// nginxHostPorts ypanel-nginx 当前映射的宿主端口集合（tcp）；容器未安装/未列出时返回空集。
func (s *SiteService) nginxHostPorts(ctx context.Context) (map[int]bool, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	cl, err := agentclient.GetJSON[[]dto.ContainerItem](ac, ctx, "/agent/v1/docker/containers")
	if err != nil {
		return nil, err
	}
	out := map[int]bool{}
	for _, c := range *cl {
		if c.Name != nginxContainer {
			continue
		}
		for _, p := range c.Ports {
			if p.Proto != "" && p.Proto != "tcp" {
				continue
			}
			if n, err := strconv.Atoi(p.HostPort); err == nil && n > 0 {
				out[n] = true
			}
		}
	}
	return out, nil
}

// checkPortFree 附加端口分配预检（创建/改端口前调用）：已被 ypanel-nginx 映射的端口放行
// （nginx 多站点可共用同一 listen 端口），其余宿主占用（面板/SSH/其他容器映射/裸进程）一律拒绝。
func (s *SiteService) checkPortFree(ctx context.Context, port int) error {
	if siteBasePorts[port] {
		return nil
	}
	if s.nat == nil {
		return nil
	}
	if mine, err := s.nginxHostPorts(ctx); err == nil && mine[port] {
		return nil
	}
	occupied, err := s.nat.CheckPort(ctx, "local", "tcp", port, 0)
	if err != nil {
		return err
	}
	for _, o := range occupied {
		if o.Port != port {
			continue
		}
		proc := o.Process
		if proc == "" {
			proc = "未知进程"
		}
		return errs.New(errs.CodeConflict, "error.badRequest", fmt.Sprintf("端口 %d 已被占用（%s），无法分配给站点", port, proc))
	}
	return nil
}

// ReconcilePorts 站点端口对账（幂等，可重放）：
// 站点表推导 desired → container 模式对账 compose ports 段 / host 模式对账防火墙放行 → 同步登记表。
func (s *SiteService) ReconcilePorts(ctx context.Context) error {
	s.reconcileMu.Lock()
	defer s.reconcileMu.Unlock()

	var rows []model.Site
	if err := s.db.Find(&rows).Error; err != nil {
		return err
	}
	desired := siteDesiredExtraPorts(rows)
	var leases []model.SitePortLease
	if err := s.db.Find(&leases).Error; err != nil {
		return err
	}
	if s.nginxMode() == "container" {
		if err := s.applyComposePorts(ctx, desired); err != nil {
			return err
		}
	} else if err := s.applyFirewallPorts(ctx, desired, leases); err != nil {
		return err
	}
	return s.syncLeaseTable(desired, rows)
}

// applyComposePorts container 模式：附加映射与 desired 一致则跳过（零重建）；否则重写 compose
// ports 段并 up -d（端口集变化触发容器重建，全部站点瞬断约 1s）。up 失败回滚旧内容再拉起。
func (s *SiteService) applyComposePorts(ctx context.Context, desired []int) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	mine, err := s.nginxHostPorts(ctx)
	if err != nil {
		return err
	}
	if len(mine) == 0 {
		return nil // 容器未安装：ensureNginx 安装时写基础模板，无附加端口可对账
	}
	curExtra := make([]int, 0, len(mine))
	for p := range mine {
		if !siteBasePorts[p] {
			curExtra = append(curExtra, p)
		}
	}
	sort.Ints(curExtra)
	if intsEqual(curExtra, desired) {
		return nil
	}
	// 回滚锚点：优先读回磁盘上的 compose 内容，读不到按当前映射重建等价模板
	prev := nginxComposeTemplate(curExtra)
	if cfg, err := agentclient.GetJSON[dto.ComposeConfigResp](ac, ctx, "/agent/v1/compose/config?name="+nginxProject); err == nil && cfg.Content != "" {
		prev = cfg.Content
	}
	if _, err := agentclient.DoJSON[dto.ComposeWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/compose/config",
		&dto.ComposeWriteReq{Name: nginxProject, Content: nginxComposeTemplate(desired)}); err != nil {
		return err
	}
	if _, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
		&dto.ComposeActionReq{Name: nginxProject}); err != nil {
		_, _ = agentclient.DoJSON[dto.ComposeWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/compose/config",
			&dto.ComposeWriteReq{Name: nginxProject, Content: prev})
		_, _ = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
			&dto.ComposeActionReq{Name: nginxProject})
		return err
	}
	return nil
}

// applyFirewallPorts host 模式：以登记表为"已放行"锚点，增量放行/收回防火墙端口。
func (s *SiteService) applyFirewallPorts(ctx context.Context, desired []int, leases []model.SitePortLease) error {
	if s.fw == nil {
		return nil
	}
	toAdd, toRemove := siteLeaseDiff(desired, leases)
	for _, p := range toAdd {
		if _, err := s.fw.EnsurePortAllowed(ctx, p, "tcp"); err != nil {
			return err
		}
	}
	for _, p := range toRemove {
		if err := s.fw.RevokePortAllowed(ctx, p, "tcp"); err != nil {
			return err
		}
	}
	return nil
}

// syncLeaseTable 登记表对齐 desired（两种模式统一维护）：来源站点名每次刷新，消失端口删行。
func (s *SiteService) syncLeaseTable(desired []int, rows []model.Site) error {
	names := map[int][]string{}
	for _, r := range rows {
		if r.Enabled && !siteBasePorts[r.Port] && r.Port >= 1 && r.Port <= 65535 {
			names[r.Port] = append(names[r.Port], r.Name)
		}
	}
	desiredSet := map[int]bool{}
	for _, p := range desired {
		desiredSet[p] = true
	}
	var leases []model.SitePortLease
	if err := s.db.Find(&leases).Error; err != nil {
		return err
	}
	leaseSet := map[int]bool{}
	for _, l := range leases {
		leaseSet[l.Port] = true
	}
	for _, p := range desired {
		sites := strings.Join(names[p], ",")
		if leaseSet[p] {
			// map 形式 Updates：避免 struct 零值更新被 GORM 吞掉
			if err := s.db.Model(&model.SitePortLease{}).Where("port = ?", p).
				Updates(map[string]any{"sites": sites, "proto": "tcp"}).Error; err != nil {
				return err
			}
			continue
		}
		if err := s.db.Create(&model.SitePortLease{Port: p, Proto: "tcp", Sites: sites}).Error; err != nil {
			return err
		}
	}
	for _, l := range leases {
		if desiredSet[l.Port] {
			continue
		}
		if err := s.db.Delete(&model.SitePortLease{}, l.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

// SitePortConf 监听端口配置域。
type SitePortConf struct {
	Port    int    `json:"port"`
	Mode    string `json:"mode"`    // container / host（提示映射形态）
	Applied bool   `json:"applied"` // 端口已在系统侧落点（映射/放行）；基础端口恒为 true
}

// GetPortConf 读取监听端口配置。
func (s *SiteService) GetPortConf(id uint) (SitePortConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SitePortConf{}, err
	}
	applied := siteBasePorts[site.Port]
	var count int64
	_ = s.db.Model(&model.SitePortLease{}).Where("port = ?", site.Port).Count(&count).Error
	return SitePortConf{Port: site.Port, Mode: s.nginxMode(), Applied: applied || count > 0}, nil
}

// UpdatePortConf 修改站点监听端口：校验 → 冲突预检 → 落库 → writeConf → 端口对账。
func (s *SiteService) UpdatePortConf(ctx context.Context, id uint, port int) (SitePortConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return SitePortConf{}, err
	}
	newPort, err := validateSitePort(port)
	if err != nil {
		return SitePortConf{}, err
	}
	if newPort != site.Port {
		if err := s.checkPortFree(ctx, newPort); err != nil {
			return SitePortConf{}, err
		}
	}
	if err := s.db.Model(site).Update("port", newPort).Error; err != nil {
		return SitePortConf{}, err
	}
	site.Port = newPort
	if err := s.writeConf(ctx, site, confTemplate(site, site.CertDomain != "", parseWaf(site))); err != nil {
		return SitePortConf{}, err
	}
	if err := s.ReconcilePorts(ctx); err != nil {
		return SitePortConf{}, err
	}
	return s.GetPortConf(id)
}

// intsEqual 升序整型切片相等判断（纯函数）。
func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
