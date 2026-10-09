// BackupService 通用目录备份（M34）：任意目录 tar.gz 打包 + 可选远程上传。
// compose 项目备份 = 项目目录打包（YPanel compose 约定数据卷在项目目录内，./data 随包）。
package service

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// dirBackupBase 目录备份落盘根。
const dirBackupBase = "/opt/ypanel/backups/dirs"

// composeProjectPattern compose 项目名白名单。
var composeProjectPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

// composeDir compose 项目根。
const composeDir = "/opt/ypanel/compose"

// BackupService 目录备份。
type BackupService struct {
	nodes   *NodeService
	storage *StorageService
}

// NewBackupService 创建。
func NewBackupService(nodes *NodeService, storage *StorageService) *BackupService {
	return &BackupService{nodes: nodes, storage: storage}
}

// DirBackup 打包目录为 tar.gz（name 为归档名基准），可选远程上传；返回产物信息。
func (s *BackupService) DirBackup(ctx context.Context, srcDir, name string, opts ...BackupUploadOpts) (map[string]any, error) {
	srcDir = path.Clean(srcDir)
	if !path.IsAbs(srcDir) || srcDir == "/" || strings.Contains(srcDir, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "源目录不合法（需绝对路径且非根目录）")
	}
	if name == "" {
		name = path.Base(srcDir)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return nil, errs.Wrap(errs.ErrBadRequest, "归档名不合法")
	}
	ac, err := s.nodesClient()
	if err != nil {
		return nil, err
	}
	file := fmt.Sprintf("%s_%s.tar.gz", name, time.Now().Format("20060102_150405"))
	target := path.Join(dirBackupBase, file)
	// 多源打包以源 basename 为顶层条目（exp：tar 语义），解包还原目录名
	if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir",
		&dto.FileMkdirReq{Path: dirBackupBase}); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "mkdir: "+err.Error())
	}
	if _, err := agentclient.DoJSON[dto.FileCompressReq, struct{}](ac, ctx, "POST", "/agent/v1/files/compress",
		&dto.FileCompressReq{Srcs: []string{srcDir}, Dest: target}); err != nil {
		return nil, err
	}
	res := map[string]any{"file": file, "path": target, "src": srcDir, "at": time.Now().Format(time.RFC3339)}
	if len(opts) > 0 && opts[0].StorageAccountID > 0 && s.storage != nil {
		key, uerr := s.storage.UploadAgentFile(ctx, opts[0].StorageAccountID, "dirs/"+name, target, opts[0].Keep)
		if uerr != nil {
			return res, errs.Wrapc(errs.CodeFileOpFailed, "本地备份成功，远端上传失败: "+uerr.Error())
		}
		res["remoteKey"] = key
	}
	return res, nil
}

// ComposeBackup 备份 compose 项目目录。
func (s *BackupService) ComposeBackup(ctx context.Context, project string, opts ...BackupUploadOpts) (map[string]any, error) {
	if !composeProjectPattern.MatchString(project) {
		return nil, errs.Wrap(errs.ErrBadRequest, "项目名不合法")
	}
	return s.DirBackup(ctx, path.Join(composeDir, project), project, opts...)
}

func (s *BackupService) nodesClient() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}
