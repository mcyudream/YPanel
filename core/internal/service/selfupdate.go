// SelfUpdateService 面板自更新（本地文件通道）。
// 通道：/opt/ypanel/updates/ 目录（运维将新版本二进制放入）；apply = detached 脚本
// （备份当前二进制 → 替换 → systemctl restart），脚本脱离 core 进程生命周期。
package service

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// updateChannelDir 更新通道目录（agent 主机侧）。
const updateChannelDir = "/opt/ypanel/updates"

// SelfUpdateService 自更新服务。
type SelfUpdateService struct {
	nodes     *NodeService
	currentVer string
}

// NewSelfUpdateService 创建。
func NewSelfUpdateService(nodes *NodeService, currentVersion string) *SelfUpdateService {
	return &SelfUpdateService{nodes: nodes, currentVer: currentVersion}
}

func (s *SelfUpdateService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// Status 更新通道状态：当前版本 + 可用更新文件列表。
func (s *SelfUpdateService) Status(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(updateChannelDir))
	if err != nil {
		return map[string]any{
			"currentVersion": s.currentVer,
			"channelDir":     updateChannelDir,
			"available":      []map[string]any{},
		}, nil
	}
	files := []map[string]any{}
	for _, e := range out.Entries {
		if e.IsDir || e.Size < 1<<20 { // 二进制至少 1MB，过滤杂项
			continue
		}
		files = append(files, map[string]any{
			"name": e.Name, "sizeMb": float64(e.Size) / 1024 / 1024,
			"modTime": e.ModTime, "path": e.Path,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		a, _ := files[i]["modTime"].(time.Time)
		b, _ := files[j]["modTime"].(time.Time)
		return a.After(b)
	})
	return map[string]any{
		"currentVersion": s.currentVer,
		"channelDir":     updateChannelDir,
		"available":      files,
	}, nil
}

// validateUpdateFile 更新文件名白名单。
func validateUpdateFile(file string) error {
	if file == "" || strings.Contains(file, "/") || strings.Contains(file, "..") ||
		!strings.HasPrefix(file, "ypanel") {
		return errs.Wrap(errs.ErrBadRequest, "更新文件名不合法（需 ypanel 前缀且无路径分隔）")
	}
	return nil
}

// Apply 应用更新：写 detached 脚本（备份→替换→重启），立即返回。
func (s *SelfUpdateService) Apply(ctx context.Context, file string) (map[string]any, error) {
	if err := validateUpdateFile(file); err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	src := path.Join(updateChannelDir, file)
	applyScript := path.Join("/opt/ypanel/updates", "apply-update.sh")
	script := fmt.Sprintf(`#!/bin/sh
sleep 2
cp /opt/ypanel/ypanel /opt/ypanel/ypanel.bak || exit 1
mv %s /opt/ypanel/ypanel || exit 1
chmod +x /opt/ypanel/ypanel
systemctl restart ypanel
`, src)
	// 写脚本
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: applyScript, Content: script}); err != nil {
		return nil, err
	}
	// detached 执行（core 将在 2s 后被重启，脚本独立存活）
	cmd := fmt.Sprintf("setsid nohup sh %s >/tmp/ypanel-apply.log 2>&1 </dev/null &", applyScript)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 15})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return map[string]any{"output": out.Output}, errs.Wrapc(errs.CodeFileOpFailed, "apply 脚本启动失败")
	}
	return map[string]any{
		"message": "更新已启动：约 2 秒后替换二进制并重启服务，请稍后通过 /health 确认新版本",
		"file":    file,
	}, nil
}
