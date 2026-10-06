// FirewallService 防火墙管理（ufw 形态，agent exec 通道）。
// 安全红线：启用 ufw 前强制放行 SSH(22) 与面板端口，防止自锁。
package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

var portPattern = regexp.MustCompile(`^[0-9]{1,5}$`)

// FirewallService 防火墙服务。
type FirewallService struct {
	nodes     *NodeService
	panelPort int
}

// NewFirewallService 创建（panelPort 用于自锁保护）。
func NewFirewallService(nodes *NodeService, panelPort int) *FirewallService {
	return &FirewallService{nodes: nodes, panelPort: panelPort}
}

func (s *FirewallService) client() (*agentclient.Client, error) {
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

// Status 防火墙状态与规则列表。
func (s *FirewallService) Status(ctx context.Context) (map[string]any, error) {
	out, code, err := s.exec(ctx, "which ufw && ufw status numbered")
	if err != nil {
		return nil, err
	}
	if code != 0 || !strings.Contains(out, "Status:") {
		return map[string]any{"available": false, "hint": "目标机未安装 ufw（apt install ufw）"}, nil
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
	return map[string]any{"available": true, "enabled": enabled, "rules": rules}, nil
}

// validatePort 端口白名单。
func validatePort(p string) error {
	v, err := strconv.Atoi(p)
	if err != nil || v < 1 || v > 65535 {
		return errs.Wrap(errs.ErrBadRequest, "端口不合法: "+p)
	}
	return nil
}

// Allow 放行端口。
func (s *FirewallService) Allow(ctx context.Context, port, proto string) error {
	if !portPattern.MatchString(port) {
		return errs.Wrap(errs.ErrBadRequest, "端口不合法")
	}
	if err := validatePort(port); err != nil {
		return err
	}
	if proto != "tcp" && proto != "udp" {
		proto = "tcp"
	}
	_, code, err := s.exec(ctx, "ufw allow "+port+"/"+proto)
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "ufw allow 失败")
	}
	return nil
}

// DeleteRule 删除规则（按 Status 列表编号，u fw --force 免交互）。
func (s *FirewallService) DeleteRule(ctx context.Context, number int) error {
	if number < 1 {
		return errs.ErrBadRequest
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
func (s *FirewallService) SetEnabled(ctx context.Context, enabled bool) error {
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
