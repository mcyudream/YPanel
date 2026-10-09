// FirewallService 防火墙管理（ufw / firewalld 自适应，agent exec 通道）。
// 安全红线：启用防火墙前强制放行 SSH(22) 与面板端口，防止自锁。
package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

var portPattern = regexp.MustCompile(`^[0-9]{1,5}$`)

// FirewallService 防火墙服务。
type FirewallService struct {
	db        *gorm.DB
	nodes     *NodeService
	panelPort int
	nodeClient *agentclient.Client // WithNode 绑定的节点客户端（空=本机）
}

// NewFirewallService 创建（panelPort 用于自锁保护）。
func NewFirewallService(db *gorm.DB, nodes *NodeService, panelPort int) *FirewallService {
	return &FirewallService{db: db, nodes: nodes, panelPort: panelPort}
}

// WithNode 返回绑定目标节点的副本（M55 主机域节点化：client 按节点路由，方法签名不变）。
func (s *FirewallService) WithNode(nodeId string) (*FirewallService, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	cp := *s
	cp.nodeClient = agentclient.New(node.BaseURL, node.Token)
	return &cp, nil
}

func (s *FirewallService) client() (*agentclient.Client, error) {
	if s.nodeClient != nil {
		return s.nodeClient, nil
	}
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// exec 执行命令并返回 (output, exitCode)。
func (s *FirewallService) exec(ctx context.Context, cmd string) (string, int, error) {
	ac, err := s.client()
	if err != nil {
		return "", 0, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 60})
	if err != nil {
		return "", 0, err
	}
	return out.Output, out.ExitCode, nil
}

// detectBackend 防火墙后端探测：ufw 优先，其次 firewalld；空 = 均不可用（探测的是 active 态）。
func (s *FirewallService) detectBackend(ctx context.Context) string {
	if out, code, _ := s.exec(ctx, "command -v ufw >/dev/null && systemctl is-active ufw 2>/dev/null || echo inactive"); code == 0 && strings.Contains(out, "active") {
		return "ufw"
	}
	if out, code, _ := s.exec(ctx, "command -v firewall-cmd >/dev/null && systemctl is-active firewalld 2>/dev/null || echo inactive"); code == 0 && strings.Contains(out, "active") {
		return "firewalld"
	}
	return ""
}

// siteManaged 站点托管放行清单（M50 SitePortLease）。
func (s *FirewallService) siteManaged() []map[string]any {
	managed := []map[string]any{}
	if s.db == nil {
		return managed
	}
	var leases []model.SitePortLease
	_ = s.db.Find(&leases).Error
	for _, l := range leases {
		managed = append(managed, map[string]any{"port": l.Port, "proto": l.Proto, "sites": l.Sites})
	}
	return managed
}

// Status 防火墙状态与规则列表。
func (s *FirewallService) Status(ctx context.Context) (map[string]any, error) {
	backend := s.detectBackend(ctx)
	if backend == "firewalld" {
		out, err := s.statusFirewalld(ctx)
		if out != nil {
			out["siteManaged"] = s.siteManaged()
		}
		return out, err
	}
	out, code, err := s.exec(ctx, "which ufw && ufw status numbered")
	if err != nil {
		return nil, err
	}
	if code != 0 || !strings.Contains(out, "Status:") {
		return map[string]any{"available": false, "backend": "none", "hint": "目标机未安装 ufw / firewalld", "siteManaged": s.siteManaged()}, nil
	}
	enabled := strings.Contains(out, "Status: active")
	rules := []map[string]any{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		rules = append(rules, map[string]any{"raw": line})
	}
	return map[string]any{"available": true, "backend": "ufw", "enabled": enabled, "rules": rules, "siteManaged": s.siteManaged()}, nil
}

// validatePort 端口白名单。
func validatePort(p string) error {
	v, err := strconv.Atoi(p)
	if err != nil || v < 1 || v > 65535 {
		return errs.Wrap(errs.ErrBadRequest, "端口不合法: "+p)
	}
	return nil
}

// Allow 放行端口（按后端分发；ufw / firewalld 均幂等）。
func (s *FirewallService) Allow(ctx context.Context, port, proto string) error {
	if !portPattern.MatchString(port) {
		return errs.Wrap(errs.ErrBadRequest, "端口不合法")
	}
	if err := validatePort(port); err != nil {
		return err
	}
	p, _ := strconv.Atoi(port)
	if _, err := s.EnsurePortAllowed(ctx, p, normalizeProto(proto)); err != nil {
		return err
	}
	return nil
}

func normalizeProto(proto string) string {
	if proto != "tcp" && proto != "udp" {
		return "tcp"
	}
	return proto
}

// EnsurePortAllowed 确保端口放行（M50 站点托管端口，幂等）。
// 返回是否实际变更；后端缺失（未装防火墙）= 无需放行，返回 (false, nil) 不阻断站点流程。
func (s *FirewallService) EnsurePortAllowed(ctx context.Context, port int, proto string) (bool, error) {
	proto = normalizeProto(proto)
	backend := s.detectBackend(ctx)
	switch backend {
	case "ufw":
		_, code, err := s.exec(ctx, "ufw allow "+strconv.Itoa(port)+"/"+proto)
		if err != nil {
			return false, err
		}
		if code != 0 {
			return false, errs.Wrapc(errs.CodeFileOpFailed, "ufw allow 失败")
		}
		return true, nil
	case "firewalld":
		return s.firewalldEnsurePort(ctx, port, proto)
	default:
		return false, nil
	}
}

// RevokePortAllowed 收回端口放行（M50 站点托管回收）：按端口形式删除，不依赖规则编号，幂等。
func (s *FirewallService) RevokePortAllowed(ctx context.Context, port int, proto string) error {
	proto = normalizeProto(proto)
	backend := s.detectBackend(ctx)
	switch backend {
	case "ufw":
		// 规则不存在时 ufw 也可能非零退出（Could not delete non-existent rule），目标态幂等，忽略退出码
		_, _, _ = s.exec(ctx, "ufw --force delete allow "+strconv.Itoa(port)+"/"+proto)
		return nil
	case "firewalld":
		return s.firewalldRevokePort(ctx, port, proto)
	default:
		return nil
	}
}

// firewalldEnsurePort firewalld 永久放行端口（已放行则跳过），变更后 reload；返回是否实际变更。
func (s *FirewallService) firewalldEnsurePort(ctx context.Context, port int, proto string) (bool, error) {
	spec := strconv.Itoa(port) + "/" + proto
	out, code, err := s.exec(ctx, "firewall-cmd --permanent --query-port="+spec)
	if err == nil && code == 0 && strings.TrimSpace(out) == "yes" {
		return false, nil
	}
	if _, code, err := s.exec(ctx, "firewall-cmd --permanent --add-port="+spec); err != nil || code != 0 {
		if err == nil {
			err = errs.Wrapc(errs.CodeFileOpFailed, "firewall-cmd add-port 失败: "+spec)
		}
		return false, err
	}
	return true, s.firewalldReload(ctx)
}

// firewalldRevokePort firewalld 永久收回端口（未放行视为已收回），变更后 reload。
func (s *FirewallService) firewalldRevokePort(ctx context.Context, port int, proto string) error {
	spec := strconv.Itoa(port) + "/" + proto
	out, code, err := s.exec(ctx, "firewall-cmd --permanent --query-port="+spec)
	if err != nil {
		return err
	}
	if code != 0 || strings.TrimSpace(out) != "yes" {
		return nil
	}
	if _, code, err := s.exec(ctx, "firewall-cmd --permanent --remove-port="+spec); err != nil || code != 0 {
		if err == nil {
			err = errs.Wrapc(errs.CodeFileOpFailed, "firewall-cmd remove-port 失败: "+spec)
		}
		return err
	}
	return s.firewalldReload(ctx)
}

func (s *FirewallService) firewalldReload(ctx context.Context) error {
	_, code, err := s.exec(ctx, "firewall-cmd --reload")
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "firewall-cmd reload 失败")
	}
	return nil
}

// DeleteRule 删除规则：ufw 按 Status 列表编号（--force 免交互）；
// firewalld 前端行 number 即端口号（见 statusFirewalld），按端口收回（tcp/udp 各自幂等尝试）。
func (s *FirewallService) DeleteRule(ctx context.Context, number int) error {
	if number < 1 {
		return errs.ErrBadRequest
	}
	if s.detectBackend(ctx) == "firewalld" {
		if err := s.firewalldRevokePort(ctx, number, "tcp"); err != nil {
			return err
		}
		return s.firewalldRevokePort(ctx, number, "udp")
	}
	_, code, err := s.exec(ctx, "ufw --force delete "+strconv.Itoa(number))
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "删除规则失败")
	}
	return nil
}

// SetEnabled 启停防火墙（启用前强制放行 SSH 与面板端口，防自锁）。
// 后端按"已安装"判定而非 active 态（启用动作本身是从 inactive 到 active）。
func (s *FirewallService) SetEnabled(ctx context.Context, enabled bool) error {
	out, code, _ := s.exec(ctx, "command -v firewall-cmd")
	firewalldInstalled := code == 0 && strings.TrimSpace(out) != ""
	if firewalldInstalled {
		return s.setEnabledFirewalld(ctx, enabled)
	}
	if enabled {
		for _, p := range []string{"22", strconv.Itoa(s.panelPort)} {
			_, _, _ = s.exec(ctx, "ufw allow "+p+"/tcp")
		}
		_, code, err := s.exec(ctx, "ufw --force enable")
		if err != nil {
			return err
		}
		if code != 0 {
			return errs.Wrapc(errs.CodeFileOpFailed, "ufw enable 失败")
		}
		return nil
	}
	_, code, err := s.exec(ctx, "ufw disable")
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "ufw disable 失败")
	}
	return nil
}

// setEnabledFirewalld firewalld 启停（服务级），启用前放行 SSH 与面板端口防自锁。
func (s *FirewallService) setEnabledFirewalld(ctx context.Context, enabled bool) error {
	if !enabled {
		_, code, err := s.exec(ctx, "systemctl disable --now firewalld")
		if err != nil {
			return err
		}
		if code != 0 {
			return errs.Wrapc(errs.CodeFileOpFailed, "firewalld 停用失败")
		}
		return nil
	}
	for _, p := range []string{"22/tcp", strconv.Itoa(s.panelPort) + "/tcp"} {
		_, _, _ = s.exec(ctx, "firewall-cmd --permanent --add-port="+p)
	}
	_ = s.firewalldReload(ctx)
	_, code, err := s.exec(ctx, "systemctl enable --now firewalld")
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "firewalld 启用失败")
	}
	return nil
}

// statusFirewalld firewalld 后端状态与端口规则（M40 自适应；number=端口号供前端删除回传）。
func (s *FirewallService) statusFirewalld(ctx context.Context) (map[string]any, error) {
	_, _, _ = s.exec(ctx, "firewall-cmd --state")
	out, code, err := s.exec(ctx, "firewall-cmd --permanent --list-ports")
	if err != nil || code != 0 {
		return map[string]any{"available": false, "backend": "firewalld", "hint": "firewall-cmd 不可用"}, nil
	}
	ports := []map[string]any{}
	for _, p := range strings.Split(strings.TrimSpace(out), " ") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		parts := strings.Split(p, "/")
		proto := "tcp"
		if len(parts) > 1 {
			proto = parts[1]
		}
		num, _ := strconv.Atoi(parts[0])
		ports = append(ports, map[string]any{"port": parts[0], "proto": proto, "number": num})
	}
	running := false
	if o, c, _ := s.exec(ctx, "systemctl is-active firewalld"); c == 0 && strings.Contains(o, "active") {
		running = true
	}
	return map[string]any{"available": true, "backend": "firewalld", "running": running, "ports": ports}, nil
}
