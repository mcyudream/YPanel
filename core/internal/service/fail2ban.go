// Fail2banService 入侵防护纳管（fail2ban-client 通道，ufw 模式同构）。
package service

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Fail2banService fail2ban 纳管服务。
type Fail2banService struct {
	nodes *NodeService
}

// NewFail2banService 创建。
func NewFail2banService(nodes *NodeService) *Fail2banService {
	return &Fail2banService{nodes: nodes}
}

func (s *Fail2banService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// exec 执行 fail2ban-client 子命令。
func (s *Fail2banService) exec(ctx context.Context, args string) (string, int, error) {
	ac, err := s.client()
	if err != nil {
		return "", 0, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "fail2ban-client " + args, TimeoutSecs: 30})
	if err != nil {
		return "", 0, err
	}
	return out.Output, out.ExitCode, nil
}

// Status jail 列表与封禁情况。
func (s *Fail2banService) Status(ctx context.Context) (map[string]any, error) {
	out, code, err := s.exec(ctx, "status")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return map[string]any{"available": false, "hint": "目标机未安装/未运行 fail2ban（apt install fail2ban && systemctl enable --now fail2ban）"}, nil
	}
	available := true
	jails := []map[string]any{}
	// 解析 "- Jail list:\tsshd, nginx-badbot"
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "Jail list:") {
			continue
		}
		part := strings.SplitN(line, ":", 2)
		if len(part) != 2 {
			continue
		}
		for _, j := range strings.Split(strings.TrimSpace(part[1]), ",") {
			j = strings.TrimSpace(j)
			if j == "" {
				continue
			}
			banned, total := s.jailBanned(ctx, j)
			jails = append(jails, map[string]any{"name": j, "banned": banned, "total": total})
		}
	}
	return map[string]any{"available": available, "jails": jails}, nil
}

// jailBanned 查询单个 jail 的封禁 IP 列表。
func (s *Fail2banService) jailBanned(ctx context.Context, jail string) ([]string, int) {
	out, code, err := s.exec(ctx, "status "+jail)
	if err != nil || code != 0 {
		return nil, 0
	}
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "Banned IP list:") {
			continue
		}
		part := strings.SplitN(line, ":", 2)
		if len(part) != 2 {
			return []string{}, 0
		}
		ips := strings.Fields(strings.TrimSpace(part[1]))
		return ips, len(ips)
	}
	return []string{}, 0
}

// Unban 解封 IP。
func (s *Fail2banService) Unban(ctx context.Context, jail, ip string) error {
	if err := validateJailIP(ip); err != nil {
		return err
	}
	_, code, err := s.exec(ctx, fmt.Sprintf("set %s unbanip %s", jail, ip))
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "解封失败（IP 可能不在封禁列表）")
	}
	return nil
}

// Ban 手动封禁 IP。
func (s *Fail2banService) Ban(ctx context.Context, jail, ip string) error {
	if err := validateJailIP(ip); err != nil {
		return err
	}
	_, code, err := s.exec(ctx, fmt.Sprintf("set %s banip %s", jail, ip))
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "封禁失败")
	}
	return nil
}

// validateJailIP 校验 IP 与 jail 名。
func validateJailIP(ip string) error {
	if net.ParseIP(strings.TrimSpace(ip)) == nil {
		return errs.Wrap(errs.ErrBadRequest, "IP 不合法: "+ip)
	}
	return nil
}

// ValidateJail 校验 jail 名。
func ValidateJail(jail string) error {
	if jail == "" || len(jail) > 64 || strings.ContainsAny(jail, " ;&|/\\") {
		return errs.Wrap(errs.ErrBadRequest, "jail 名不合法")
	}
	return nil
}
