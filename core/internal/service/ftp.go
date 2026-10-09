// FtpService FTP 服务管理（M49）：vsftpd 安装/启停/端口/被动范围，agent exec + 回读验证。
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// FtpService FTP（vsftpd）管理。
type FtpService struct {
	nodes *NodeService
}

// NewFtpService 创建。
func NewFtpService(nodes *NodeService) *FtpService {
	return &FtpService{nodes: nodes}
}

func (s *FtpService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

func (s *FtpService) exec(ctx context.Context, cmd string, timeout int) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
	if err != nil {
		return "", err
	}
	return out.Output, nil
}

// portPattern FTP 端口白名单。
var ftpPortPattern = regexp.MustCompile(`^\d{1,5}$`)

// Status FTP 状态（安装/运行/端口/被动范围）。
func (s *FtpService) Status(ctx context.Context) (map[string]any, error) {
	out, err := s.exec(ctx, `command -v vsftpd >/dev/null && echo installed || echo missing; systemctl is-active vsftpd 2>/dev/null || echo inactive; grep -E '^(listen_port|pasv_min_port|pasv_max_port|anonymous_enable)' /etc/vsftpd.conf 2>/dev/null`, 30)
	if err != nil {
		return nil, err
	}
	st := map[string]any{"installed": false, "running": false, "port": 21, "pasvMin": 40000, "pasvMax": 40100, "anonymous": false}
	lines := strings.Split(out, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "installed" {
		st["installed"] = true
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) == "active" {
		st["running"] = true
	}
	for _, line := range lines[2:] {
		kv := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "listen_port":
			var n int
			fmt.Sscanf(kv[1], "%d", &n)
			st["port"] = n
		case "pasv_min_port":
			var n int
			fmt.Sscanf(kv[1], "%d", &n)
			st["pasvMin"] = n
		case "pasv_max_port":
			var n int
			fmt.Sscanf(kv[1], "%d", &n)
			st["pasvMax"] = n
		case "anonymous_enable":
			st["anonymous"] = kv[1] == "YES"
		}
	}
	return st, nil
}

// Install 安装 vsftpd（Ubuntu/Debian apt；失败给出发行版提示）。
func (s *FtpService) Install(ctx context.Context) (map[string]any, error) {
	out, err := s.exec(ctx, `export DEBIAN_FRONTEND=noninteractive; apt-get install -y vsftpd 2>&1 | tail -3 && systemctl enable --now vsftpd && echo FTP_INSTALL_OK`, 600)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(out, "FTP_INSTALL_OK") {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "安装失败（仅支持 apt 系发行版）: "+firstLine(tail(out, 300)))
	}
	return s.Status(ctx)
}

// Power 启动/停止/重启。
func (s *FtpService) Power(ctx context.Context, action string) error {
	if action != "start" && action != "stop" && action != "restart" {
		return errs.Wrap(errs.ErrBadRequest, "不支持的动作: "+action)
	}
	_, err := s.exec(ctx, fmt.Sprintf("systemctl %s vsftpd && systemctl is-active vsftpd", action), 30)
	return err
}

// SetPort 设置监听端口 + 被动范围（写 /etc/vsftpd.conf + 重启 + 回读）。
func (s *FtpService) SetPort(ctx context.Context, port, pasvMin, pasvMax int) error {
	if port < 1024 || port > 65535 {
		return errs.Wrap(errs.ErrBadRequest, "监听端口需在 1024-65535")
	}
	if pasvMin < 1024 || pasvMax > 65535 || pasvMin >= pasvMax {
		return errs.Wrap(errs.ErrBadRequest, "被动端口范围不合法")
	}
	script := fmt.Sprintf(`set -e
cp /etc/vsftpd.conf /etc/vsftpd.conf.ypbak 2>/dev/null || true
sed -i '/^listen_port/d;/^pasv_min_port/d;/^pasv_max_port/d;/^listen=/d;/^listen_ipv6/d' /etc/vsftpd.conf
printf '\nlisten=YES\nlisten_port=%d\npasv_enable=YES\npasv_min_port=%d\npasv_max_port=%d\n' %d %d %d >> /etc/vsftpd.conf
systemctl restart vsftpd
grep -E '^(listen_port|pasv_min_port|pasv_max_port)' /etc/vsftpd.conf`, port, pasvMin, pasvMax, port, pasvMin, pasvMax)
	out, err := s.exec(ctx, script, 60)
	if err != nil {
		return err
	}
	if !strings.Contains(out, fmt.Sprintf("listen_port=%d", port)) {
		return errs.Wrapc(errs.CodeFileOpFailed, "端口设置回读验证失败")
	}
	return nil
}
