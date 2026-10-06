// SiteBackupService 站点备份（B9）：站点目录 tar.gz 打包到备份目录。
package service

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// SiteBackupService 站点备份。
type SiteBackupService struct {
	nodes *NodeService
}

// NewSiteBackupService 创建。
func NewSiteBackupService(nodes *NodeService) *SiteBackupService {
	return &SiteBackupService{nodes: nodes}
}

// BackupDir 备份落盘目录（宿主机）。
func (s *SiteBackupService) BackupDir() string {
	return "/opt/ypanel/backups/sites"
}

// Backup 打包站点目录（/var/www/sites/<name>）为 tar.gz，返回文件信息。
func (s *SiteBackupService) Backup(ctx context.Context, siteName string) (map[string]any, error) {
	if siteName == "" || backupDirEscape(siteName) != siteName {
		return nil, errs.Wrap(errs.ErrBadRequest, "站点名不合法")
	}
	ac, err := s.nodesClient()
	if err != nil {
		return nil, err
	}
	file := fmt.Sprintf("%s_%s.tar.gz", siteName, time.Now().Format("20060102_150405"))
	target := path.Join(s.BackupDir(), file)
	cmd := fmt.Sprintf("mkdir -p %s && docker exec ypanel-nginx tar -czf /tmp/%s -C /var/www/sites %s && docker cp ypanel-nginx:/tmp/%s %s && docker exec ypanel-nginx rm -f /tmp/%s",
		s.BackupDir(), file, siteName, file, target, file)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 600})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "站点备份失败: "+firstLine(out.Output))
	}
	return map[string]any{"file": target, "site": siteName, "at": time.Now().Format(time.RFC3339)}, nil
}

func (s *SiteBackupService) nodesClient() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

