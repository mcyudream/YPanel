// runtime_backup.go 运行环境备份与恢复：编排目录整体 tar.gz（含 conf/supervisor 配置与日志清单），
// 恢复前自动回滚备份；www 站点数据不在备份范围（与 1P 运行环境备份同语义）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

const runtimeBackupDir = "/opt/ypanel/runtime-backup"

var backupFileRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.tar\.gz$`)

// runtimeBackupMeta 备份元数据（随包内 .ypanel-runtime.json 存档）。
type runtimeBackupMeta struct {
	Name    string     `json:"name"`
	Type    string     `json:"type"`
	Version string     `json:"version"`
	Image   string     `json:"image"`
	Env     runtimeEnv `json:"env"`
	At      time.Time  `json:"at"`
}

func (s *RuntimeService) backupPaths(row *model.Runtime) (dir string) {
	return runtimeBackupDir + "/" + row.Type + "/" + row.Name
}

// RuntimeBackupInfo 备份条目。
type RuntimeBackupInfo struct {
	File string `json:"file"`
	Size string `json:"size"`
	At   string `json:"at"`
}

// BackupList 备份列表。
func (s *RuntimeService) BackupList(ctx context.Context, id uint) ([]RuntimeBackupInfo, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	out, err := s.exec(ctx, 20, "ls -lh %s 2>/dev/null | grep '\\.tar\\.gz' || true", s.backupPaths(row))
	if err != nil {
		return nil, err
	}
	list := make([]RuntimeBackupInfo, 0, 4)
	for _, line := range strings.Split(out.Output, "\n") {
		f := strings.Fields(line)
		if len(f) < 8 {
			continue
		}
		list = append(list, RuntimeBackupInfo{File: f[len(f)-1], Size: f[4], At: strings.Join(f[5:8], " ")})
	}
	return list, nil
}

// writeBackupMeta 把运行时元数据写进编排目录（随后打进 tar 包）。
func (s *RuntimeService) writeBackupMeta(ctx context.Context, row *model.Runtime) error {
	meta, _ := json.Marshal(runtimeBackupMeta{
		Name: row.Name, Type: row.Type, Version: row.Version, Image: row.Image,
		Env: envJSON(row), At: time.Now(),
	})
	return s.writeRemote(ctx, s.dir(row)+"/.ypanel-runtime.json", string(meta))
}

// BackupCreate 创建备份（任务化）。
func (s *RuntimeService) BackupCreate(ctx context.Context, id uint) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部运行环境无编排目录可备份")
	}
	file := fmt.Sprintf("%s.%s.tar.gz", row.Name, time.Now().Format("20060102150405"))
	task, err := s.tasks.StartTask("runtime-backup", fmt.Sprintf("备份运行环境 %s", row.Name), file, 10*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			if err := s.writeBackupMeta(tctx, row); err != nil {
				return err
			}
			logf("info", "打包 %s → %s", s.dir(row), file)
			out, err := s.exec(tctx, 300, "mkdir -p %s && tar -czf %s/%s -C %s . && ls -lh %s/%s | awk '{print $5}'", s.backupPaths(row), s.backupPaths(row), file, s.dir(row), s.backupPaths(row), file)
			if err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "打包失败: "+tailOutput(out.Output, 400))
			}
			logf("info", "备份完成，大小 %s", strings.TrimSpace(out.Output))
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID, "file": file}, nil
}

// BackupRestore 恢复备份（任务化；恢复前自动做一次安全备份，失败自动回退）。
func (s *RuntimeService) BackupRestore(ctx context.Context, id uint, file string) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部运行环境无备份可恢复")
	}
	if !backupFileRe.MatchString(file) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "备份文件名不合法")
	}
	if s.extTaskRunning(id) {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "该运行环境已有任务进行中，请稍后再试")
	}
	dir := s.dir(row)
	bdir := s.backupPaths(row)
	task, err := s.tasks.StartTask("runtime-restore", fmt.Sprintf("恢复运行环境 %s ← %s", row.Name, file), file, 20*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			// 恢复前安全备份（当前目录状态）
			safety := fmt.Sprintf("auto-before-restore.%s.tar.gz", time.Now().Format("20060102150405"))
			logf("info", "恢复前安全备份 → %s", safety)
			if err := s.writeBackupMeta(tctx, row); err != nil {
				return err
			}
			if out, err := s.exec(tctx, 300, "mkdir -p %s && tar -czf %s/%s -C %s .", bdir, bdir, safety, dir); err != nil || out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "安全备份失败: "+tailOutput(out.Output, 400))
			}
			// 停止 → 换目录 → 解包 → 启动；失败回退安全备份
			logf("info", "停止容器")
			if cf, err := s.locateCompose(tctx, row); err == nil {
				_, _ = s.exec(tctx, 300, "%s 2>/dev/null; true", composeCmd(cf, row.ComposeProject, "down --remove-orphans"))
			}
			logf("info", "替换编排目录")
			backupScript := fmt.Sprintf(
				"set -e; mv %s %s.bak; (mkdir -p %s && tar -xzf %s/%s -C %s) || { rm -rf %s; mv %s.bak %s; docker compose -f %s/docker-compose.yml -p %s up -d; exit 1; }; rm -rf %s.bak",
				dir, dir, dir, bdir, file, dir, dir, dir, dir, dir, row.ComposeProject, dir)
			out, err := s.exec(tctx, 120, "%s 2>&1", backupScript)
			if err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "解包失败（已回退）: "+tailOutput(out.Output, 400))
			}
			logf("info", "启动容器")
			upOut, err := s.exec(tctx, 600, "%s", composeCmd(dir+"/docker-compose.yml", row.ComposeProject, "up -d"))
			if err != nil {
				return err
			}
			if upOut.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "容器启动失败: "+tailOutput(upOut.Output, 600))
			}
			state, err := s.containerState(tctx, containerNameOr(row))
			if err != nil || state != "running" {
				return errs.Wrapc(errs.CodeFileOpFailed, "容器未进入运行状态: "+state)
			}
			row.Status = RuntimeStatusRunning
			row.Message = ""
			_ = s.db.Save(row).Error
			logf("info", "恢复完成")
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// BackupDelete 删除备份文件。
func (s *RuntimeService) BackupDelete(ctx context.Context, id uint, file string) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if !backupFileRe.MatchString(file) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "备份文件名不合法")
	}
	_, err = s.exec(ctx, 30, "rm -f %s/%s", s.backupPaths(row), file)
	return err
}
