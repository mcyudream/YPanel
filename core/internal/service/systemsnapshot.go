// SystemSnapshotService 系统级快照（M47）：整机关键数据打包。
// 内容清单（选项化）：系统元数据 + Docker 元数据（必含）；面板 db + 编排目录 + nginx 配置（默认含）；
// 网站目录 / 数据库 dump（可选，重量级）。产出 /opt/ypanel/backups/system/snapshot-<node>-<ts>.tar.gz。
// 保留份数 system_snapshot.keep（默认 5），超出自动删除最旧。
// 恢复：面板 db/compose/nginx conf 自动；网站目录自动回灌容器；数据库 dump 落位到标准备份目录（用既有恢复入口导入）。
package service

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// systemSnapDir 系统快照落盘目录（节点侧）。
const systemSnapDir = "/opt/ypanel/backups/system"

// SystemSnapshotKeepKey 系统快照保留份数设置键。
const SystemSnapshotKeepKey = "system_snapshot.keep"

// SystemSnapshotService 系统快照。
type SystemSnapshotService struct {
	nodes *NodeService
	tasks *TaskService
	dbsvc *DatabaseService
	db    *gorm.DB
	bp    *PanelBackupService
}

// NewSystemSnapshotService 创建。
func NewSystemSnapshotService(nodes *NodeService, tasks *TaskService, dbsvc *DatabaseService, db *gorm.DB, bp *PanelBackupService) *SystemSnapshotService {
	return &SystemSnapshotService{nodes: nodes, tasks: tasks, dbsvc: dbsvc, db: db, bp: bp}
}

func (s *SystemSnapshotService) clientFor(nodeID string) (*agentclient.Client, error) {
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// SystemSnapOptions 快照内容选项。
type SystemSnapOptions struct {
	Websites  bool `json:"websites"`  // 含网站目录（/var/www/sites）
	Databases bool `json:"databases"` // 含全部数据库实例 dump
}

// List 系统快照列表（新→旧）。
func (s *SystemSnapshotService) List(ctx context.Context, nodeID string) ([]map[string]any, error) {
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(systemSnapDir))
	if err != nil {
		return []map[string]any{}, nil
	}
	entries := []map[string]any{}
	for _, e := range out.Entries {
		if e.IsDir || !strings.HasPrefix(e.Name, "snapshot-") || !strings.HasSuffix(e.Name, ".tar.gz") {
			continue
		}
		entries = append(entries, map[string]any{
			"file": e.Name, "sizeMb": float64(e.Size) / 1024 / 1024, "modTime": e.ModTime,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i]["file"].(string) > entries[j]["file"].(string)
	})
	return entries, nil
}

// Delete 删除系统快照。
func (s *SystemSnapshotService) Delete(ctx context.Context, nodeID, file string) error {
	if !strings.HasPrefix(file, "snapshot-") || strings.Contains(file, "..") {
		return errs.ErrPathInvalid
	}
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
		&dto.FileDeleteReq{Paths: []string{path.Join(systemSnapDir, file)}})
	return err
}

// buildCreateScript 生成节点侧打包脚本（全部动作在节点 shell 内完成，core 只编排）。
func buildCreateScript(nodeID string, opts SystemSnapOptions) string {
	ts := time.Now().Format("20060102-150405")
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString(fmt.Sprintf("TS=%s\nW=/opt/ypanel/tmp/systnap-$TS\n", ts))
	b.WriteString("mkdir -p $W/snapshot/panel $W/snapshot/meta /opt/ypanel/backups/system\n")
	// 系统元数据（变量赋值 + 单 printf，避免多行花括号组）
	b.WriteString("H=$(hostname); K=$(uname -r); A=$(uname -m); IP=$(hostname -I 2>/dev/null | awk '{print $1}'); N=$(date -Iseconds)\n")
	b.WriteString("printf '{\"hostname\":\"%s\",\"kernel\":\"%s\",\"arch\":\"%s\",\"ip\":\"%s\",\"collectedAt\":\"%s\"}\\n' \"$H\" \"$K\" \"$A\" \"$IP\" \"$N\" > $W/snapshot/meta/system.json\n")
	// Docker 元数据（子 shell 圆括号组重定向）
	b.WriteString("( echo 'containers:'; docker ps -a --format '{{.Names}}|{{.Image}}|{{.Status}}'; echo 'images:'; docker images --format '{{.Repository}}:{{.Tag}}|{{.Size}}'; echo 'volumes:'; docker volume ls --format '{{.Name}}'; echo 'networks:'; docker network ls --format '{{.Name}}|{{.Driver}}' ) > $W/snapshot/meta/docker.txt 2>/dev/null || true\n")
	// 面板数据（db 仅在运行 core 的节点存在；纯 agent 节点自动跳过）
	b.WriteString("cp /opt/ypanel/data/ypanel.db $W/snapshot/panel/ 2>/dev/null || true\n")
	b.WriteString("tar -czf $W/snapshot/panel/compose.tar.gz -C /opt/ypanel compose 2>/dev/null || true\n")
	b.WriteString("tar -czf $W/snapshot/panel/nginx.tar.gz -C /opt/ypanel nginx 2>/dev/null || true\n")
	b.WriteString("cp /etc/docker/daemon.json $W/snapshot/panel/ 2>/dev/null || true\n")
	// 网站目录（经 nginx 容器）
	if opts.Websites {
		b.WriteString("docker exec ypanel-nginx tar -czf /tmp/yp-sites.tgz -C /var/www/sites . 2>/dev/null && docker cp ypanel-nginx:/tmp/yp-sites.tgz $W/snapshot/websites.tar.gz && docker exec ypanel-nginx rm -f /tmp/yp-sites.tgz || true\n")
	}
	// 数据库 dump（core 已先行逐实例 CreateBackup 到 /opt/ypanel/backups/<type>/<name>/）
	if opts.Databases {
		b.WriteString("mkdir -p $W/snapshot/databases\n")
		b.WriteString("cp -r /opt/ypanel/backups/mysql /opt/ypanel/backups/postgres /opt/ypanel/backups/mongo /opt/ypanel/backups/redis $W/snapshot/databases/ 2>/dev/null || true\n")
	}
	// 打包
	b.WriteString(fmt.Sprintf("tar -czf /opt/ypanel/backups/system/snapshot-%s-$TS.tar.gz -C $W snapshot\n", nodeID))
	b.WriteString("rm -rf $W\necho SNAPSHOT_OK\n")
	return b.String()
}

// Create 创建系统快照（任务化，长耗时）。
func (s *SystemSnapshotService) Create(ctx context.Context, nodeID string, opts SystemSnapOptions) (map[string]any, error) {
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return nil, err
	}
	script := buildCreateScript(nodeID, opts)
	keep := 5
	if s.db != nil {
		var st model.Setting
		if err := s.db.Where("`key` = ?", "system_snapshot.keep").First(&st).Error; err == nil {
			_, _ = fmt.Sscanf(st.Value, "%d", &keep)
		}
		if keep <= 0 {
			keep = 5
		}
	}
	// 1P 对齐：快照包含数据库 dump——先逐实例生成备份（跳过外部实例客户端缺失场景）
	if opts.Databases && s.dbsvc != nil {
		var insts []model.DatabaseInstance
		if err := s.db.Find(&insts).Error; err == nil {
			for _, inst := range insts {
				if inst.Type != "mysql" && inst.Type != "postgres" && inst.Type != "mongo" && inst.Type != "redis" {
					continue
				}
				if _, err := s.dbsvc.CreateBackup(ctx, inst.ID); err != nil {
					// 单实例失败不阻塞快照
					continue
				}
			}
		}
	}
	task, err := s.tasks.StartTask("system-snapshot", "系统快照："+node.Name, node.Name, 60*time.Minute, func(ctx context.Context, logf TaskLogf) error {
		ac := agentclient.New(node.BaseURL, node.Token)
		if _, werr := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
			&dto.FileWriteReq{Path: "/opt/ypanel/tmp/systnap.sh", Content: script}); werr != nil {
			return werr
		}
		logf("info", "开始打包系统快照（websites=%v databases=%v）", opts.Websites, opts.Databases)
		out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: "bash /opt/ypanel/tmp/systnap.sh", TimeoutSecs: 3600})
		if err != nil {
			return err
		}
		if out.ExitCode != 0 || !strings.Contains(out.Output, "SNAPSHOT_OK") {
			return fmt.Errorf("打包失败: %s", firstLine(out.Output))
		}
		// 保留份数：超出 keep 删除最旧（core 侧列出删除）
		entries, lerr := s.List(ctx, nodeID)
		if lerr == nil {
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e["file"].(string))
			}
			sort.Strings(names)
			ac2 := agentclient.New(node.BaseURL, node.Token)
			for i := 0; i < len(names)-keep; i++ {
				_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac2, ctx, "POST", "/agent/v1/files/delete",
					&dto.FileDeleteReq{Paths: []string{path.Join(systemSnapDir, names[i])}})
				logf("info", "已删除超出保留份数的旧快照: %s", names[i])
			}
		}
		logf("info", "系统快照打包完成")
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID, "dir": systemSnapDir}, nil
}

// buildRestoreScript 生成恢复脚本（面板数据自动恢复；网站回灌容器；数据库 dump 落位标准备份目录）。
func buildRestoreScript(nodeID, file string) string {
	return fmt.Sprintf(`#!/bin/bash
set -x
SNAP=/opt/ypanel/backups/system/%s
W=/opt/ypanel/tmp/sysrestore
rm -rf $W && mkdir -p $W
tar -xzf $SNAP -C $W
S=$W/snapshot
systemctl stop ypanel
# 回滚留底
cp /opt/ypanel/data/ypanel.db /opt/ypanel/backups/panel/rollback-$(date +%%Y%%m%%d-%%H%%M%%S).db 2>/dev/null || true
# 面板 db
cp $S/panel/ypanel.db /opt/ypanel/data/ypanel.db 2>/dev/null || true
rm -f /opt/ypanel/data/ypanel.db-wal /opt/ypanel/data/ypanel.db-shm
# 编排目录 / nginx 配置 / daemon.json
if [ -f $S/panel/compose.tar.gz ]; then rm -rf /opt/ypanel/compose && tar -xzf $S/panel/compose.tar.gz -C /opt/ypanel; fi
if [ -f $S/panel/nginx.tar.gz ]; then mkdir -p /opt/ypanel/nginx && tar -xzf $S/panel/nginx.tar.gz -C /opt/ypanel; fi
if [ -f $S/panel/daemon.json ]; then cp $S/panel/daemon.json /etc/docker/daemon.json; fi
# 网站目录回灌 nginx 容器
if [ -f $S/websites.tar.gz ]; then docker cp $S/websites.tar.gz ypanel-nginx:/tmp/yp-sites.tgz && docker exec ypanel-nginx sh -c 'rm -rf /var/www/sites/* && tar -xzf /tmp/yp-sites.tgz -C /var/www/sites && rm -f /tmp/yp-sites.tgz'; fi
# 数据库 dump 落位标准备份目录（实例由恢复后的面板 db 记录，用既有恢复入口导入）
if [ -d $S/databases ]; then cp -r $S/databases/. /opt/ypanel/backups/ 2>/dev/null || true; fi
rm -rf $W
systemctl start ypanel
echo RESTORE_OK`, file)
}

// Restore 恢复系统快照（任务化；面板失联属预期）。
func (s *SystemSnapshotService) Restore(ctx context.Context, nodeID, file string) (map[string]any, error) {
	if !strings.HasPrefix(file, "snapshot-") || strings.Contains(file, "..") {
		return nil, errs.ErrPathInvalid
	}
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return nil, err
	}
	script := buildRestoreScript(nodeID, file)
	task, err := s.tasks.StartTask("system-restore", "系统快照恢复："+node.Name, file, 30*time.Minute, func(ctx context.Context, logf TaskLogf) error {
		ac := agentclient.New(node.BaseURL, node.Token)
		if _, werr := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
			&dto.FileWriteReq{Path: "/opt/ypanel/tmp/sysrestore.sh", Content: script}); werr != nil {
			return werr
		}
		logf("info", "开始恢复（面板将停止并重启；数据库 dump 已落位标准备份目录，请用各实例的恢复入口导入）")
		// M48 修复：systemd-run 瞬态单元执行——脚本内 stop ypanel 会杀死 agent（含 nohup 子进程），
		// 瞬态单元独立于 ypanel cgroup，恢复全程不受影响。
		unit := fmt.Sprintf("yp-sysrestore-%d", time.Now().UnixNano()%1e6)
		_, _ = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: fmt.Sprintf("systemd-run --unit=%s --collect bash /opt/ypanel/tmp/sysrestore.sh >> /opt/ypanel/tmp/sysrestore.log 2>&1", unit), TimeoutSecs: 15})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID, "hint": "恢复异步执行：面板会停止/重启，数据库 dump 已落位标准备份目录，请用各实例恢复入口导入；compose 项目需手动 up 重建"}, nil
}

// SetKeep 设置系统快照保留份数（settings 全局）。
func (s *SystemSnapshotService) SetKeep(keep int) error {
	if keep < 1 || keep > 50 {
		return errs.Wrap(errs.ErrBadRequest, "保留份数需在 1-50")
	}
	if s.db == nil {
		return errs.New(errs.CodeInternal, "error.internal", "设置存储未就绪")
	}
	return s.db.Where("`key` = ?", "system_snapshot.keep").Assign(&model.Setting{Key: "system_snapshot.keep", Value: fmt.Sprintf("%d", keep)}).FirstOrCreate(&model.Setting{}).Error
}

// GetKeep 读取保留份数（缺省 5）。
func (s *SystemSnapshotService) GetKeep() int {
	keep := 5
	if s.db != nil {
		var st model.Setting
		if err := s.db.Where("`key` = ?", "system_snapshot.keep").First(&st).Error; err == nil {
			_, _ = fmt.Sscanf(st.Value, "%d", &keep)
		}
	}
	if keep <= 0 {
		keep = 5
	}
	return keep
}
