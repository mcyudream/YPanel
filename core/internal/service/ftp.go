// FtpService FTP 服务管理（M49）：vsftpd 安装/启停/端口/被动范围，agent exec + 回读验证。
package service

import (
	"context"
	"encoding/base64"
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
	nodeClient *agentclient.Client // WithNode 绑定的节点客户端（空=本机）
}

// NewFtpService 创建。
func NewFtpService(nodes *NodeService) *FtpService {
	return &FtpService{nodes: nodes}
}

// WithNode 返回绑定目标节点的副本（M55 主机域节点化：client 按节点路由，方法签名不变）。
func (s *FtpService) WithNode(nodeId string) (*FtpService, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	cp := *s
	cp.nodeClient = agentclient.New(node.BaseURL, node.Token)
	return &cp, nil
}

func (s *FtpService) client() (*agentclient.Client, error) {
	if s.nodeClient != nil {
		return s.nodeClient, nil
	}
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

// Install 安装 vsftpd（apt/dnf/yum 多通道；apt 先刷新索引——新机 lists 为空时直接
// install 会报 Unable to locate package；装后以 command -v 回读验证，不依赖管道
// 退出码（`| tail` 会吞掉 apt 的失败退出码）。
func (s *FtpService) Install(ctx context.Context) (map[string]any, error) {
	out, err := s.exec(ctx, `export DEBIAN_FRONTEND=noninteractive; `+
		`if command -v apt-get >/dev/null 2>&1; then apt-get update -qq && apt-get install -y -qq vsftpd; `+
		`elif command -v dnf >/dev/null 2>&1; then dnf install -y vsftpd; `+
		`elif command -v yum >/dev/null 2>&1; then yum install -y vsftpd; `+
		`else echo FTP_NO_PKG_MGR; fi; `+
		`if command -v vsftpd >/dev/null 2>&1; then systemctl enable --now vsftpd && echo FTP_INSTALL_OK; `+
		`else echo FTP_INSTALL_MISSING; fi`, 600)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(out, "FTP_INSTALL_OK") {
		msg := "安装失败: " + lastLine(tail(out, 300))
		switch {
		case strings.Contains(out, "FTP_NO_PKG_MGR"):
			msg = "安装失败：未识别到 apt/dnf/yum 包管理器，暂不支持该发行版"
		case strings.Contains(out, "FTP_INSTALL_MISSING"):
			msg = "安装失败（软件源不可用或安装未成功）: " + lastLine(tail(out, 300))
		}
		return nil, errs.Wrapc(errs.CodeFileOpFailed, msg)
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

// ftpConfPathProbe 探测配置文件路径（Debian /etc/vsftpd.conf 优先，RHEL /etc/vsftpd/vsftpd.conf 兜底）。
const ftpConfPathProbe = `f=/etc/vsftpd.conf; [ -f "$f" ] || f=/etc/vsftpd/vsftpd.conf; if [ -f "$f" ]; then cat "$f"; else echo __NO_CONF__; fi`

// GetConfig 读 vsftpd.conf 原文（M53：配置编辑器）。
func (s *FtpService) GetConfig(ctx context.Context) (string, error) {
	out, err := s.exec(ctx, ftpConfPathProbe, 15)
	if err != nil {
		return "", err
	}
	if strings.Contains(out, "__NO_CONF__") {
		return "", errs.Wrap(errs.ErrBadRequest, "未找到 vsftpd 配置文件（服务未安装？）")
	}
	return out, nil
}

// PutConfig 写回 vsftpd.conf（base64 传递防注入）+ 重启校验，失败自动回滚备份。
func (s *FtpService) PutConfig(ctx context.Context, content string) error {
	if strings.TrimSpace(content) == "" {
		return errs.Wrap(errs.ErrBadRequest, "配置内容不能为空")
	}
	if len(content) > 512*1024 {
		return errs.Wrap(errs.ErrBadRequest, "配置内容过大（>512KB）")
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	script := `set -e
f=/etc/vsftpd.conf; [ -f "$f" ] || f=/etc/vsftpd/vsftpd.conf
[ -f "$f" ] || { echo __NO_CONF__; exit 1; }
cp "$f" /tmp/yp-vsftpd.bak
printf '%s' '` + b64 + `' | base64 -d > "$f"
if systemctl restart vsftpd 2>/tmp/yp-ftp.err; then
  echo FTP_CONF_OK
else
  cp /tmp/yp-vsftpd.bak "$f"
  systemctl restart vsftpd 2>/dev/null || true
  echo FTP_CONF_ROLLBACK
  cat /tmp/yp-ftp.err
  exit 1
fi`
	out, err := s.exec(ctx, script, 60)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "FTP_CONF_OK") {
		return errs.Wrapc(errs.CodeFileOpFailed, "重启 vsftpd 失败，已回滚: "+lastLine(tail(out, 300)))
	}
	return nil
}
