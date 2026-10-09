// SystemToolService 系统工具箱（M40）：swap / BBR / 系统清理，全部 agent exec + 回读验证。
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

// sysValuePattern sysctl 值白名单。
var sysValuePattern = regexp.MustCompile(`^[a-zA-Z0-9._/ -]{1,64}$`)

// SystemToolService 系统管理动作。
type SystemToolService struct {
	nodes *NodeService
}

// NewSystemToolService 创建。
func NewSystemToolService(nodes *NodeService) *SystemToolService {
	return &SystemToolService{nodes: nodes}
}

func (s *SystemToolService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// execRun 执行并校验退出码。
func (s *SystemToolService) execRun(ctx context.Context, cmd string, timeout int) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return out.Output, errs.Wrapc(errs.CodeFileOpFailed, "执行失败: "+firstLine(out.Output))
	}
	return out.Output, nil
}

// ---- swap ----

// SwapStatus 查看 swap（大小/文件/开关）。
func (s *SystemToolService) SwapStatus(ctx context.Context) (map[string]any, error) {
	out, err := s.execRun(ctx, `swapon --show=NAME,SIZE --noheadings 2>/dev/null; echo ---; free -b | awk '/Swap/{print $2, $3, $4}'`, 30)
	if err != nil {
		return nil, err
	}
	files := []map[string]any{}
	total, used := int64(0), int64(0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				files = append(files, map[string]any{"file": parts[0], "size": parts[1]})
			}
		} else {
			_, _ = fmt.Sscanf(line, "%d %d", &total, &used)
		}
	}
	return map[string]any{"files": files, "totalBytes": total, "usedBytes": used, "on": total > 0}, nil
}

// SwapApply 调整 swap（sizeGB=0 关闭；档位白名单 1/2/4/8）。/swapfile 方案，不动分区。
func (s *SystemToolService) SwapApply(ctx context.Context, sizeGB int) (map[string]any, error) {
	var cmd string
	if sizeGB == 0 {
		cmd = `swapoff /swapfile 2>/dev/null; sed -i '\|/swapfile|d' /etc/fstab 2>/dev/null; rm -f /swapfile; echo swapped-off`
	} else {
		if sizeGB != 1 && sizeGB != 2 && sizeGB != 4 && sizeGB != 8 {
			return nil, errs.Wrap(errs.ErrBadRequest, "swap 大小仅支持 0(关闭)/1/2/4/8 GB")
		}
		cmd = fmt.Sprintf(`set -e; swapoff /swapfile 2>/dev/null || true; sed -i '\|/swapfile|d' /etc/fstab 2>/dev/null || true; rm -f /swapfile; fallocate -l %dG /swapfile && chmod 600 /swapfile && mkswap /swapfile >/dev/null && swapon /swapfile && echo '/swapfile none swap sw 0 0' >> /etc/fstab && swapon --show=NAME,SIZE --noheadings`, sizeGB)
	}
	out, err := s.execRun(ctx, cmd, 120)
	if err != nil {
		return nil, err
	}
	return map[string]any{"output": tail(out, 500)}, nil
}

// ---- BBR ----

// BBRStatus 拥塞控制算法回读。
func (s *SystemToolService) BBRStatus(ctx context.Context) (map[string]any, error) {
	out, err := s.execRun(ctx, `sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null; sysctl -n net.core.default_qdisc 2>/dev/null`, 30)
	if err != nil {
		return nil, err
	}
	lines := strings.Fields(strings.TrimSpace(out))
	algo := ""
	if len(lines) > 0 {
		algo = lines[0]
	}
	qdisc := ""
	if len(lines) > 1 {
		qdisc = lines[1]
	}
	return map[string]any{"algo": algo, "qdisc": qdisc, "bbr": algo == "bbr"}, nil
}

// BBRApply 开启/关闭 BBR（独立 sysctl 文件 + 即时生效 + 回读）。
func (s *SystemToolService) BBRApply(ctx context.Context, enable bool) (map[string]any, error) {
	var cmd string
	if enable {
		cmd = `set -e; printf 'net.core.default_qdisc=fq\nnet.ipv4.tcp_congestion_control=bbr\n' > /etc/sysctl.d/99-ypanel-bbr.conf; sysctl -p /etc/sysctl.d/99-ypanel-bbr.conf >/dev/null; sysctl -n net.ipv4.tcp_congestion_control`
	} else {
		cmd = `rm -f /etc/sysctl.d/99-ypanel-bbr.conf; sysctl -w net.ipv4.tcp_congestion_control=cubic >/dev/null 2>&1 || true; sysctl -n net.ipv4.tcp_congestion_control`
	}
	out, err := s.execRun(ctx, cmd, 30)
	if err != nil {
		return nil, err
	}
	algo := strings.TrimSpace(out)
	if enable && algo != "bbr" {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "BBR 开启失败（当前算法: "+algo+"，内核可能不支持）")
	}
	return map[string]any{"algo": algo, "bbr": algo == "bbr"}, nil
}

// ---- 系统清理（手动执行，与 cron clean 同语义） ----

// SystemClean 清理悬空镜像/停止容器/构建缓存 + /tmp 旧文件，返回统计输出。
func (s *SystemToolService) SystemClean(ctx context.Context) (string, error) {
	return s.execRun(ctx, `echo "== image prune"; docker image prune -f; echo "== container prune"; docker container prune -f; echo "== builder prune"; docker builder prune -f; echo "== /tmp"; find /tmp -type f -mtime +7 -delete 2>/dev/null; echo done`, 600)
}
