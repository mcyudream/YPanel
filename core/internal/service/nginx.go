// NginxService 节点 nginx 管理（M57 站点体系节点化 A 的基座）：可选装/开关/重载，
// 全部按节点路由（WithNode 副本模式，方法签名与 FtpService 同构）。零 agent 改动。
package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// NginxService 节点 nginx 服务。
type NginxService struct {
	nodes      *NodeService
	nodeClient *agentclient.Client
}

// NewNginxService 创建。
func NewNginxService(nodes *NodeService) *NginxService {
	return &NginxService{nodes: nodes}
}

func (s *NginxService) client() (*agentclient.Client, error) {
	if s.nodeClient != nil {
		return s.nodeClient, nil
	}
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// agentclientNew 节点客户端快捷构造。
func agentclientNew(node *Node) *agentclient.Client {
	return agentclient.New(node.BaseURL, node.Token)
}

// agentExec agent exec 快捷通道。
func agentExec(ac *agentclient.Client, ctx context.Context, cmd string, timeout int) (string, error) {
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
	if err != nil {
		return "", err
	}
	return out.Output, nil
}

// WithNode 返回绑定目标节点的副本。
func (s *NginxService) WithNode(nodeId string) (*NginxService, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	cp := *s
	cp.nodeClient = agentclientNew(node)
	return &cp, nil
}

// exec 在绑定节点执行命令。
func (s *NginxService) exec(ctx context.Context, cmd string, timeout int) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	return agentExec(ac, ctx, cmd, timeout)
}

// Status nginx 状态（安装/运行/版本）。
func (s *NginxService) Status(ctx context.Context) (map[string]any, error) {
	out, err := s.exec(ctx, `command -v nginx >/dev/null && echo installed || echo missing; systemctl is-active nginx 2>/dev/null || echo inactive; nginx -v 2>&1 | head -1`, 30)
	if err != nil {
		return nil, err
	}
	st := map[string]any{"installed": false, "running": false, "version": ""}
	lines := strings.Split(out, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "installed" {
		st["installed"] = true
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) == "active" {
		st["running"] = true
	}
	if len(lines) > 2 {
		st["version"] = strings.TrimSpace(lines[2])
	}
	return st, nil
}

// Install 安装 nginx（Debian/Ubuntu apt；RHEL 系 yum/dnf）。
func (s *NginxService) Install(ctx context.Context) (map[string]any, error) {
	_, err := s.exec(ctx, `export DEBIAN_FRONTEND=noninteractive; if command -v apt-get >/dev/null; then apt-get update -qq && apt-get install -y -qq nginx; elif command -v dnf >/dev/null; then dnf install -y nginx; elif command -v yum >/dev/null; then yum install -y nginx; fi; systemctl enable nginx && systemctl start nginx && systemctl enable nginx`, 900)
	if err != nil {
		return nil, err
	}
	// 装后探测判定（apt/dnf 的 SysV 同步消息会污染标记串，标记法误报）
	out, err := s.exec(ctx, `command -v nginx >/dev/null && echo NGINX_INSTALL_OK`, 30)
	if err != nil || !strings.Contains(out, "NGINX_INSTALL_OK") {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "nginx 安装失败")
	}
	return map[string]any{"message": "nginx 已安装并启动"}, nil
}

// Power 开关/重载（start|stop|restart|reload）。
func (s *NginxService) Power(ctx context.Context, action string) error {
	if !regexp.MustCompile(`^(start|stop|restart|reload)$`).MatchString(action) {
		return errs.ErrBadRequest
	}
	_, err := s.exec(ctx, "systemctl "+action+" nginx", 60)
	return err
}

// Reload 平滑重载（配置生效用；失败返回 nginx -t 错误）。
func (s *NginxService) Reload(ctx context.Context) error {
	out, err := s.exec(ctx, `nginx -t 2>&1 && systemctl reload nginx && echo YPOK`, 60)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "YPOK") {
		return errs.Wrapc(errs.CodeFileOpFailed, "nginx -t 校验失败: "+tailStr(out, 300))
	}
	return nil
}

var _ = time.Second
