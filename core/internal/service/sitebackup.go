// SiteBackupService 站点备份（B9/M34）：站点目录 tar.gz 打包到备份目录，可选远程上传。
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
	nodes   *NodeService
	storage *StorageService // 可选：备份产物远程上传（M34）
}

// SetStorage 注入远程存储服务。
func (s *SiteBackupService) SetStorage(st *StorageService) { s.storage = st }

// NewSiteBackupService 创建。
func NewSiteBackupService(nodes *NodeService) *SiteBackupService {
	return &SiteBackupService{nodes: nodes}
}

// BackupDir 备份落盘目录（宿主机）。
func (s *SiteBackupService) BackupDir() string {
	return "/opt/ypanel/backups/sites"
}

// Backup 打包站点目录（/var/www/sites/<name>）为 tar.gz，返回文件信息；opts 可选远程上传。
func (s *SiteBackupService) Backup(ctx context.Context, siteName string, opts ...BackupUploadOpts) (map[string]any, error) {
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
	res := map[string]any{"file": target, "site": siteName, "at": time.Now().Format(time.RFC3339)}
	if len(opts) > 0 && opts[0].StorageAccountID > 0 && s.storage != nil {
		key, uerr := s.storage.UploadAgentFile(ctx, opts[0].StorageAccountID, "sites/"+siteName, target, opts[0].Keep)
		if uerr != nil {
			return res, errs.Wrapc(errs.CodeFileOpFailed, "本地备份成功，远端上传失败: "+uerr.Error())
		}
		res["remoteKey"] = key
	}
	return res, nil
}

func (s *SiteBackupService) nodesClient() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}
