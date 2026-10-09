// SnapshotService 快照状态机（M45/M46）：多节点 + 保留份数限定。
// 快照 = 各节点 /opt/ypanel/backups/panel/*.db；恢复脚本经目标节点 agent 执行。
// 保留策略：全局 settings snapshot.keep（默认 10）；导入后与手动 prune 均按名字倒序保留 keep 份。
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
	"gorm.io/gorm"
)

const snapshotStateReady = "ready"

// SnapshotKeepKey 全局保留份数设置键。
const SnapshotKeepKey = "snapshot.keep"

// SnapshotService 面板快照状态机（多节点）。
type SnapshotService struct {
	db       *gorm.DB
	nodes    *NodeService
	bp       *PanelBackupService
	settings *SettingService
}

// NewSnapshotService 创建。
func NewSnapshotService(db *gorm.DB, nodes *NodeService, bp *PanelBackupService, settings *SettingService) *SnapshotService {
	return &SnapshotService{db: db, nodes: nodes, bp: bp, settings: settings}
}

// clientFor 按节点取 agent 客户端。
func (s *SnapshotService) clientFor(nodeID string) (*agentclient.Client, error) {
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// Keep 全局保留份数。
func (s *SnapshotService) Keep() int {
	n := 0
	if s.settings != nil {
		_, _ = fmt.Sscanf(s.settings.Get(SnapshotKeepKey, "10"), "%d", &n)
	}
	if n <= 0 {
		n = 10
	}
	return n
}

// SetKeep 设置全局保留份数（1-100）。
func (s *SnapshotService) SetKeep(n int) error {
	if n < 1 || n > 100 {
		return errs.Wrap(errs.ErrBadRequest, "保留份数需在 1-100")
	}
	return s.settings.Set(SnapshotKeepKey, fmt.Sprintf("%d", n))
}

// snapEntry 快照条目。
type snapEntry struct {
	Name    string    `json:"file"`
	SizeMb  float64   `json:"sizeMb"`
	ModTime time.Time `json:"modTime"`
	State   string    `json:"state"`
}

// client 取 local agent 客户端（兼容旧引用）。
func (s *SnapshotService) client() (*agentclient.Client, error) {
	return s.clientFor("local")
}

// List 指定节点的快照列表（名字倒序 = 新→旧）。
func (s *SnapshotService) List(ctx context.Context, nodeID string) ([]snapEntry, error) {
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(panelBackupDir))
	if err != nil {
		return []snapEntry{}, nil // 目录不存在 = 空
	}
	entries := []snapEntry{}
	for _, e := range out.Entries {
		if e.IsDir || !strings.HasSuffix(e.Name, ".db") {
			continue
		}
		entries = append(entries, snapEntry{
			Name: e.Name, SizeMb: float64(e.Size) / 1024 / 1024, ModTime: e.ModTime, State: snapshotStateReady,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name > entries[j].Name })
	return entries, nil
}

// Prune 按保留份数清理指定节点多余快照（名字倒序保留 keep 份；rollback- 前缀不自动清理）。
func (s *SnapshotService) Prune(ctx context.Context, nodeID string, keep int) (int, error) {
	if keep <= 0 {
		keep = s.Keep()
	}
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return 0, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(panelBackupDir))
	if err != nil {
		return 0, nil
	}
	names := []string{}
	for _, e := range out.Entries {
		if !e.IsDir && strings.HasSuffix(e.Name, ".db") {
			names = append(names, e.Name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	removed := 0
	for i := keep; i < len(names); i++ {
		if strings.HasPrefix(names[i], "rollback-") {
			continue
		}
		if _, err := agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/delete",
			&dto.FileDeleteReq{Paths: []string{path.Join(panelBackupDir, names[i])}}); err == nil {
			removed++
		}
	}
	return removed, nil
}

// PruneAll 全部节点 prune（节点离线跳过）。
func (s *SnapshotService) PruneAll(ctx context.Context, keep int) map[string]int {
	res := map[string]int{}
	for _, n := range s.nodes.RemoteIDs() {
		if removed, err := s.Prune(ctx, n, keep); err == nil {
			res[n] = removed
		}
	}
	if removed, err := s.Prune(ctx, "local", keep); err == nil {
		res["local"] = removed
	}
	return res
}

// Import 导入外部 .db 文件为快照（校验 SQLite 头 + 落备份目录 + 触发保留清理）。
func (s *SnapshotService) Import(ctx context.Context, nodeID, agentPath string) (map[string]any, error) {
	if !strings.HasPrefix(agentPath, "/opt/ypanel/tmp/") || strings.Contains(agentPath, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "导入文件须先上传到 /opt/ypanel/tmp/")
	}
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return nil, err
	}
	head, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("head -c 15 '%s'", agentPath), TimeoutSecs: 30})
	if err != nil || head.ExitCode != 0 || !strings.Contains(head.Output, "SQLite format 3") {
		return nil, errs.Wrap(errs.ErrBadRequest, "不是有效的 SQLite 数据库文件")
	}
	name := "imported-" + path.Base(agentPath)
	target := path.Join(panelBackupDir, name)
	_, _ = agentclient.DoJSON[dto.FileMkdirReq, struct{}](ac, ctx, "POST", "/agent/v1/files/mkdir",
		&dto.FileMkdirReq{Path: panelBackupDir})
	mv, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("mv '%s' '%s'", agentPath, target), TimeoutSecs: 60})
	if err != nil || mv.ExitCode != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "导入失败")
	}
	removed, _ := s.Prune(ctx, nodeID, 0)
	return map[string]any{"file": name, "path": target, "state": snapshotStateReady, "pruned": removed}, nil
}

// RestorePlan 恢复计划。
type RestorePlan struct {
	Node     string `json:"node"`
	File     string `json:"file"`
	Script   string `json:"script"`
	Rollback string `json:"rollback"`
}

// PlanRestore 生成恢复计划（stop → 留回滚快照 → 替换 → start）。
func (s *SnapshotService) PlanRestore(nodeID, file string) (*RestorePlan, error) {
	if strings.Contains(file, "/") || strings.Contains(file, "..") {
		return nil, errs.ErrPathInvalid
	}
	src := path.Join(panelBackupDir, file)
	rb := "rollback-" + time.Now().Format("20060102-150405") + ".db"
	script := fmt.Sprintf(`set -x
systemctl stop ypanel
cp /opt/ypanel/data/ypanel.db /opt/ypanel/backups/panel/%s
cp '%s' /opt/ypanel/data/ypanel.db
rm -f /opt/ypanel/data/ypanel.db-wal /opt/ypanel/data/ypanel.db-shm
systemctl start ypanel`, rb, src)
	return &RestorePlan{Node: nodeID, File: file, Script: script, Rollback: rb}, nil
}

// ExecuteRestore 执行恢复/回滚脚本（经目标节点 agent；面板失联属预期）。
func (s *SnapshotService) ExecuteRestore(ctx context.Context, nodeID, file string, rollbackBefore bool) error {
	target := file
	if rollbackBefore {
		entries, err := s.List(ctx, nodeID)
		if err != nil {
			return err
		}
		latest := ""
		for _, e := range entries {
			if strings.HasPrefix(e.Name, "rollback-") && (latest == "" || e.Name > latest) {
				latest = e.Name
			}
		}
		if latest == "" {
			return errs.Wrap(errs.ErrNotFound, "未找到恢复前留存的回滚快照")
		}
		target = latest
	}
	if strings.Contains(target, "/") || strings.Contains(target, "..") {
		return errs.ErrPathInvalid
	}
	src := path.Join(panelBackupDir, target)
	script := fmt.Sprintf(`set -x
systemctl stop ypanel
cp /opt/ypanel/data/ypanel.db /opt/ypanel/backups/panel/rollback-auto.db
cp '%s' /opt/ypanel/data/ypanel.db
rm -f /opt/ypanel/data/ypanel.db-wal /opt/ypanel/data/ypanel.db-shm
systemctl start ypanel`, src)
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return err
	}
	if _, werr := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: "/opt/ypanel/tmp/restore.sh", Content: script}); werr != nil {
		return werr
	}
	// M48 修复：恢复脚本必须脱离 ypanel cgroup 执行——脚本内 systemctl stop ypanel 会杀死
	// agent 自身（exec 子进程连带被杀，真机实测 nohup 也逃不掉）。用 systemd-run 瞬态单元
	// 独立运行，恢复全程不受影响；日志落 /opt/ypanel/tmp/restore.log，靠 /health 轮询确认完成。
	unit := fmt.Sprintf("yp-restore-%d", time.Now().UnixNano()%1e6)
	_, _ = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("systemd-run --unit=%s --collect bash /opt/ypanel/tmp/restore.sh >> /opt/ypanel/tmp/restore.log 2>&1", unit), TimeoutSecs: 15})
	return nil
}
