package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// panelBackupDir 面板备份目录（agent 主机侧）。
const panelBackupDir = "/opt/ypanel/backups/panel"

// PanelBackupService 面板备份：SQLite 快照（VACUUM INTO）+ 配置说明。
type PanelBackupService struct {
	nodes *NodeService
}

// NewPanelBackupService 创建。
func NewPanelBackupService(nodes *NodeService) *PanelBackupService {
	return &PanelBackupService{nodes: nodes}
}

func (s *PanelBackupService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// Create 创建面板备份（SQLite VACUUM INTO 一致性快照）。
func (s *PanelBackupService) Create(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	_, _ = agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir",
		&dto.FileMkdirReq{Path: panelBackupDir})
	file := fmt.Sprintf("ypanel-backup-%s.db", time.Now().Format("20060102-150405"))
	target := path.Join(panelBackupDir, file)
	cmd := fmt.Sprintf(`sqlite3 /opt/ypanel/data/ypanel.db "VACUUM INTO '%s'"`, target)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 600})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return map[string]any{"output": out.Output}, errs.Wrapc(errs.CodeFileOpFailed, "面板备份失败: "+firstLine(out.Output))
	}
	return map[string]any{"file": file, "path": target}, nil
}

// List 备份列表。
func (s *PanelBackupService) List(ctx context.Context) ([]map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(panelBackupDir))
	if err != nil {
		return []map[string]any{}, nil
	}
	files := make([]map[string]any, 0, len(out.Entries))
	for _, e := range out.Entries {
		if e.IsDir || !strings.HasSuffix(e.Name, ".db") {
			continue
		}
		files = append(files, map[string]any{
			"name": e.Name, "sizeMb": float64(e.Size) / 1024 / 1024, "modTime": e.ModTime, "path": e.Path,
		})
	}
	return files, nil
}

// Delete 删除备份。
func (s *PanelBackupService) Delete(ctx context.Context, file string) error {
	if strings.Contains(file, "/") || strings.Contains(file, "..") {
		return errs.ErrPathInvalid
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
		&dto.FileDeleteReq{Paths: []string{path.Join(panelBackupDir, file)}})
	return err
}

// RestoreHint 恢复说明（自动恢复有停服风险，M9 提供指引而非一键）。
const RestoreHint = "恢复步骤：1) systemctl stop ypanel；2) 用备份文件替换 /opt/ypanel/data/ypanel.db；3) systemctl start ypanel。可通过下载备份后在服务器执行。"
