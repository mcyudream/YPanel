// DiskGuardService 磁盘空间保护（M55）。
// 周期检测各节点根分区与 /var/lib/docker 所在分区的剩余空间；连续 2 个周期低于阈值即触发——
// 停掉该节点全部容器（豁免名单除外）并改写重启策略为 no，强制管理员清理空间后一键恢复。
// 触发态不自动解除（空间回升也保持压制，防止刚恢复又被写满）；触发期间发现非豁免容器运行 → 追加停机。
// 触发判断与事件状态都在 core；agent 只执行 stopall/restore 两个动作（旧版 agent 无此端点时状态标记需升级）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 设置键与默认值。
const (
	diskGuardEnabledKey     = "diskguard.enabled"      // "1"（默认开启）/ "0"
	diskGuardThresholdKey   = "diskguard.threshold_gb" // 默认 5，可配 1-500
	diskGuardExcludeKey     = "diskguard.exclude"      // 逗号分隔容器名
	diskGuardDefaultExclude = "ypanel-nginx,ypanel-dnsmasq"
)

const (
	diskGuardInterval     = 60 * time.Second
	diskGuardConfirmTicks = 2 // 连续低水位周期数（防瞬时误判）
	diskGuardDefaultGB    = 5
	diskGuardMinGB        = 1
	diskGuardMaxGB        = 500
	gbBytes               = uint64(1) << 30
)

// DiskGuardConfig 保护配置。
type DiskGuardConfig struct {
	Enabled     bool     `json:"enabled"`
	ThresholdGB int      `json:"thresholdGB"`
	Exclude     []string `json:"exclude"`
}

// DiskGuardEventView 事件视图（快照已解析）。
type DiskGuardEventView struct {
	ID             uint                   `json:"id"`
	NodeID         string                 `json:"nodeId"`
	NodeName       string                 `json:"nodeName"`
	TriggeredAt    time.Time              `json:"triggeredAt"`
	RestoredAt     *time.Time             `json:"restoredAt"`
	FreeBytes      uint64                 `json:"freeBytes"`
	ThresholdBytes uint64                 `json:"thresholdBytes"`
	Containers     []dto.GuardStoppedItem `json:"containers"`
	Remark         string                 `json:"remark"`
}

// DiskGuardNodeStatus 节点实时状态（guard 相关挂载点用量）。
type DiskGuardNodeStatus struct {
	NodeID      string          `json:"nodeId"`
	NodeName    string          `json:"nodeName"`
	Online      bool            `json:"online"`
	Triggered   bool            `json:"triggered"`
	AgentTooOld bool            `json:"agentTooOld"` // agent 缺 guard 端点（需升级）
	Disks       []dto.DiskUsage `json:"disks"`
}

// DiskGuardStatus status 接口聚合。
type DiskGuardStatus struct {
	Config DiskGuardConfig       `json:"config"`
	Events []DiskGuardEventView  `json:"events"`
	Nodes  []DiskGuardNodeStatus `json:"nodes"`
}

// guardNode 内部节点描述。
type guardNode struct{ id, name, baseURL, token string }

// DiskGuardService 磁盘空间保护服务。
type DiskGuardService struct {
	db       *gorm.DB
	nodes    *NodeService
	settings *SettingService
	notif    *NotificationService

	lowStreak map[string]int  // nodeID → 连续低水位周期数
	tooOld    map[string]bool // nodeID → agent 缺 guard 端点（触发/压制时探得）
}

// NewDiskGuardService 创建。
func NewDiskGuardService(db *gorm.DB, nodes *NodeService, settings *SettingService, notif *NotificationService) *DiskGuardService {
	return &DiskGuardService{
		db: db, nodes: nodes, settings: settings, notif: notif,
		lowStreak: map[string]int{}, tooOld: map[string]bool{},
	}
}

// Start 启动检测循环（60s，启动即先跑一轮）。
func (s *DiskGuardService) Start(ctx context.Context) {
	go func() {
		s.evaluateOnce(ctx)
		t := time.NewTicker(diskGuardInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.evaluateOnce(ctx)
			}
		}
	}()
}

// ---- 配置 ----

// GetConfig 读配置（SettingService 惰性默认值；exclude 用哨兵区分"从未设置/已清空"）。
func (s *DiskGuardService) GetConfig() DiskGuardConfig {
	cfg := DiskGuardConfig{
		Enabled:     s.settings.Get(diskGuardEnabledKey, "1") != "0",
		ThresholdGB: diskGuardDefaultGB,
		Exclude:     parseExcludeList(diskGuardDefaultExclude),
	}
	if v := parseIntSafe(s.settings.Get(diskGuardThresholdKey, "")); v >= diskGuardMinGB && v <= diskGuardMaxGB {
		cfg.ThresholdGB = v
	}
	if raw := s.settings.Get(diskGuardExcludeKey, "\x00unset"); raw != "\x00unset" {
		cfg.Exclude = parseExcludeList(raw)
	}
	return cfg
}

// UpdateConfig 校验并写配置。
func (s *DiskGuardService) UpdateConfig(enabled bool, thresholdGB int, exclude []string) (DiskGuardConfig, error) {
	if thresholdGB < diskGuardMinGB || thresholdGB > diskGuardMaxGB {
		return DiskGuardConfig{}, errs.New(errs.CodeBadRequest, "error.badRequest",
			fmt.Sprintf("阈值需在 %d-%d GB", diskGuardMinGB, diskGuardMaxGB))
	}
	exc := sanitizeExclude(exclude)
	sets := [][2]string{
		{diskGuardEnabledKey, boolToStr(enabled)},
		{diskGuardThresholdKey, strconv.Itoa(thresholdGB)},
		{diskGuardExcludeKey, strings.Join(exc, ",")},
	}
	for _, kv := range sets {
		if err := s.settings.Set(kv[0], kv[1]); err != nil {
			return DiskGuardConfig{}, err
		}
	}
	return DiskGuardConfig{Enabled: enabled, ThresholdGB: thresholdGB, Exclude: exc}, nil
}

// ---- 检测循环 ----

func (s *DiskGuardService) evaluateOnce(ctx context.Context) {
	cfg := s.GetConfig()
	if !cfg.Enabled {
		return // 关闭只停检测；已有触发态仍需手动恢复
	}
	threshold := uint64(cfg.ThresholdGB) * gbBytes
	for _, node := range s.allNodes() {
		ac := agentclient.New(node.baseURL, node.token)
		ovCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ovCtx, "/agent/v1/sysinfo/overview")
		cancel()
		if err != nil {
			s.lowStreak[node.id] = 0
			continue
		}
		lowFree, low := guardMountsLow(ov.Disks, threshold)
		if !low {
			s.lowStreak[node.id] = 0
		} else {
			s.lowStreak[node.id]++
		}
		if ev := s.openEvent(node.id); ev != nil {
			s.suppress(ctx, ac, node, ev, cfg, lowFree)
			continue
		}
		if low && s.lowStreak[node.id] >= diskGuardConfirmTicks {
			s.trigger(ctx, ac, node, cfg, lowFree, threshold)
		}
	}
}

// trigger 触发保护：调 agent stopall → 事件落库 → 通知。
func (s *DiskGuardService) trigger(ctx context.Context, ac *agentclient.Client, node guardNode, cfg DiskGuardConfig, freeBytes, threshold uint64) {
	callCtx, cancel := context.WithTimeout(ctx, 120*time.Second) // 容器 stop 各自最多 30s
	defer cancel()
	resp, err := agentclient.DoJSON[dto.GuardStopAllReq, dto.GuardStopAllResp](ac, callCtx, http.MethodPost,
		"/agent/v1/docker/guard/stopall", &dto.GuardStopAllReq{Exclude: cfg.Exclude})
	if err != nil {
		s.noteGuardCallError(node.id, err)
		slog.Warn("diskguard: 触发停止容器失败", "node", node.name, "err", err)
		return // 下轮重试；lowStreak 保持
	}
	delete(s.tooOld, node.id)
	remark := ""
	if len(resp.Failed) > 0 {
		remark = "部分容器停止失败: " + strings.Join(resp.Failed, "；")
	}
	severity := "error"
	if len(resp.Stopped) == 0 {
		severity = "warning" // 无容器可停（纯宿主写入占满），仍需管理员清理
	}
	ev := &model.DiskGuardEvent{
		NodeID: node.id, NodeName: node.name, TriggeredAt: time.Now(),
		FreeBytes: freeBytes, ThresholdBytes: threshold,
		ContainersJSON: snapshotEncode(resp.Stopped), Remark: remark,
	}
	if err := s.db.Create(ev).Error; err != nil {
		slog.Error("diskguard: 事件落库失败", "node", node.name, "err", err)
		return
	}
	slog.Warn("diskguard: 已触发保护", "node", node.name, "stopped", len(resp.Stopped), "skipped", len(resp.Skipped), "freeBytes", freeBytes)
	if s.notif != nil {
		msg := fmt.Sprintf("节点 %s 磁盘剩余 %s，低于阈值 %s。已停止 %d 个容器（豁免 %d 个），面板不受影响；请清理空间后到「系统 → 磁盘保护」一键恢复。",
			node.name, humanBytes(freeBytes), humanBytes(threshold), len(resp.Stopped), len(resp.Skipped))
		if remark != "" {
			msg += " " + remark
		}
		s.notif.Push(severity, "磁盘空间保护已触发", msg)
	}
}

// suppress 触发态压制：发现非豁免容器又在运行 → 追加停机并入快照（防手动拉起/compose up 绕过）。
func (s *DiskGuardService) suppress(ctx context.Context, ac *agentclient.Client, node guardNode, ev *model.DiskGuardEvent, cfg DiskGuardConfig, freeBytes uint64) {
	callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	resp, err := agentclient.DoJSON[dto.GuardStopAllReq, dto.GuardStopAllResp](ac, callCtx, http.MethodPost,
		"/agent/v1/docker/guard/stopall", &dto.GuardStopAllReq{Exclude: cfg.Exclude})
	if err != nil {
		s.noteGuardCallError(node.id, err)
		return
	}
	delete(s.tooOld, node.id)
	changed := false
	remark := ev.Remark
	if freeBytes > 0 && freeBytes < ev.FreeBytes {
		ev.FreeBytes = freeBytes
		changed = true
	}
	if len(resp.Stopped) > 0 || len(resp.Failed) > 0 {
		snap := snapshotDecode(ev.ContainersJSON)
		seen := map[string]bool{}
		for _, it := range snap {
			seen[it.Name] = true
		}
		added := []dto.GuardStoppedItem{}
		for _, it := range resp.Stopped {
			if !seen[it.Name] {
				snap = append(snap, it)
				added = append(added, it)
			}
		}
		if len(added) > 0 {
			remark = joinRemark(remark, fmt.Sprintf("压制期间追停 %d 个容器（%s）", len(added), strings.Join(namesOf(added), "、")))
		}
		if len(resp.Failed) > 0 {
			remark = joinRemark(remark, "停止失败: "+strings.Join(resp.Failed, "；"))
		}
		ev.ContainersJSON = snapshotEncode(snap)
		ev.Remark = remark
		changed = true
	}
	if !changed {
		return
	}
	if err := s.db.Model(ev).Updates(map[string]any{
		"free_bytes": ev.FreeBytes, "containers_json": ev.ContainersJSON, "remark": ev.Remark,
	}).Error; err != nil {
		slog.Warn("diskguard: 压制状态落库失败", "node", node.name, "err", err)
		return
	}
	if len(resp.Stopped) > 0 && s.notif != nil {
		s.notif.Push("warning", "磁盘保护压制中", fmt.Sprintf("节点 %s 触发期间检测到容器被拉起，已再次停止：%s", node.name, strings.Join(namesOf(resp.Stopped), "、")))
	}
}

// Restore 一键恢复：eventID=0 恢复全部未恢复事件；按快照还原重启策略并拉起容器。
func (s *DiskGuardService) Restore(ctx context.Context, eventID uint) (started, failed []string, err error) {
	events := []model.DiskGuardEvent{}
	q := s.db.Where("restored_at IS NULL").Order("id").Find(&events)
	if q.Error != nil {
		return nil, nil, q.Error
	}
	for _, ev := range events {
		if eventID != 0 && ev.ID != eventID {
			continue
		}
		node, nerr := s.nodeByID(ev.NodeID, ev.NodeName)
		if nerr != nil {
			failed = append(failed, fmt.Sprintf("[%s] %v", ev.NodeName, nerr))
			continue
		}
		ac := agentclient.New(node.baseURL, node.token)
		callCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
		resp, rerr := agentclient.DoJSON[dto.GuardRestoreReq, dto.GuardRestoreResp](ac, callCtx, http.MethodPost,
			"/agent/v1/docker/guard/restore", &dto.GuardRestoreReq{Containers: snapshotDecode(ev.ContainersJSON)})
		cancel()
		if rerr != nil {
			failed = append(failed, fmt.Sprintf("[%s] agent 不可达: %v", ev.NodeName, rerr))
			continue
		}
		now := time.Now()
		remark := joinRemark(ev.Remark, fmt.Sprintf("已恢复：启动 %d 个，失败 %d 个", len(resp.Started), len(resp.Failed)))
		if len(resp.Failed) > 0 {
			remark = joinRemark(remark, strings.Join(resp.Failed, "；"))
		}
		if uerr := s.db.Model(&model.DiskGuardEvent{}).Where("id = ?", ev.ID).
			Updates(map[string]any{"restored_at": &now, "remark": remark}).Error; uerr != nil {
			failed = append(failed, fmt.Sprintf("[%s] 事件状态更新失败: %v", ev.NodeName, uerr))
			continue
		}
		started = append(started, resp.Started...)
		failed = append(failed, resp.Failed...)
		slog.Info("diskguard: 事件已恢复", "node", ev.NodeName, "event", ev.ID, "started", len(resp.Started), "failed", resp.Failed)
	}
	if s.notif != nil && len(events) > 0 {
		s.notif.Push("info", "磁盘保护已恢复", fmt.Sprintf("已按快照恢复容器：启动 %d 个，失败 %d 个。", len(started), len(failed)))
	}
	return started, failed, nil
}

// ---- 状态查询 ----

// Status 配置 + 事件历史 + 各节点实时磁盘（前端拿 freeBytes 与配置阈值自行比较）。
func (s *DiskGuardService) Status(ctx context.Context) DiskGuardStatus {
	out := DiskGuardStatus{Config: s.GetConfig(), Events: s.ListEvents(), Nodes: []DiskGuardNodeStatus{}}
	for _, node := range s.allNodes() {
		entry := DiskGuardNodeStatus{NodeID: node.id, NodeName: node.name, Disks: []dto.DiskUsage{}}
		entry.Triggered = s.openEvent(node.id) != nil
		entry.AgentTooOld = s.tooOld[node.id]
		ac := agentclient.New(node.baseURL, node.token)
		ovCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ovCtx, "/agent/v1/sysinfo/overview")
		cancel()
		if err == nil {
			entry.Online = true
			entry.Disks = guardMounts(ov.Disks)
		}
		out.Nodes = append(out.Nodes, entry)
	}
	return out
}

// ListEvents 事件视图（未恢复在前，其余按触发时间倒序）。
func (s *DiskGuardService) ListEvents() []DiskGuardEventView {
	rows := []model.DiskGuardEvent{}
	if err := s.db.Order("triggered_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return []DiskGuardEventView{}
	}
	out := make([]DiskGuardEventView, 0, len(rows))
	for _, r := range rows {
		out = append(out, DiskGuardEventView{
			ID: r.ID, NodeID: r.NodeID, NodeName: r.NodeName,
			TriggeredAt: r.TriggeredAt, RestoredAt: r.RestoredAt,
			FreeBytes: r.FreeBytes, ThresholdBytes: r.ThresholdBytes,
			Containers: snapshotDecode(r.ContainersJSON), Remark: r.Remark,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := out[i].RestoredAt == nil, out[j].RestoredAt == nil
		if ai != aj {
			return ai
		}
		return out[i].TriggeredAt.After(out[j].TriggeredAt)
	})
	return out
}

// ---- 内部 ----

// openEvent 节点当前未恢复事件（无则 nil）。
func (s *DiskGuardService) openEvent(nodeID string) *model.DiskGuardEvent {
	var ev model.DiskGuardEvent
	if err := s.db.Where("node_id = ? AND restored_at IS NULL", nodeID).Order("id DESC").First(&ev).Error; err != nil {
		return nil
	}
	return &ev
}

// noteGuardCallError 识别旧版 agent（404 = 无 guard 端点），状态页提示升级。
func (s *DiskGuardService) noteGuardCallError(nodeID string, err error) {
	if err != nil && strings.Contains(err.Error(), "agent HTTP 404") {
		s.tooOld[nodeID] = true
	}
}

// allNodes local + 全部远程节点。
func (s *DiskGuardService) allNodes() []guardNode {
	out := []guardNode{}
	if l := s.nodes.Local(); l != nil {
		out = append(out, guardNode{id: l.ID, name: l.Name, baseURL: l.BaseURL, token: l.Token})
	}
	rows := []model.Node{}
	if err := s.db.Order("id").Find(&rows).Error; err == nil {
		for _, r := range rows {
			out = append(out, guardNode{id: fmt.Sprintf("%d", r.ID), name: r.Name, baseURL: r.Addr, token: r.Token})
		}
	}
	return out
}

// nodeByID 按 ID 找节点；远程节点已删除则拒绝（快照容器无从恢复，明确报错优于静默）。
func (s *DiskGuardService) nodeByID(id, fallbackName string) (guardNode, error) {
	for _, n := range s.allNodes() {
		if n.id == id {
			return n, nil
		}
	}
	return guardNode{}, fmt.Errorf("节点 %s 已不存在，无法恢复其容器", fallbackName)
}

// guardMounts 取 guard 相关挂载点：根分区 / 与 /var/lib/docker 所在分区（按最长前缀匹配，去重）。
func guardMounts(disks []dto.DiskUsage) []dto.DiskUsage {
	const dockerRoot = "/var/lib/docker"
	out := []dto.DiskUsage{}
	best := ""
	for _, d := range disks {
		if strings.HasPrefix(dockerRoot, d.Mountpoint) && len(d.Mountpoint) > len(best) {
			best = d.Mountpoint
		}
	}
	seen := map[string]bool{}
	for _, want := range []string{"/", best} {
		if want == "" || seen[want] {
			continue
		}
		seen[want] = true
		for _, d := range disks {
			if d.Mountpoint == want {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

// guardMountsLow guard 挂载点任一剩余低于阈值；返回观测到的最低剩余。
func guardMountsLow(disks []dto.DiskUsage, threshold uint64) (uint64, bool) {
	mounts := guardMounts(disks)
	if len(mounts) == 0 {
		return 0, false
	}
	minFree := mounts[0].Free
	for _, d := range mounts {
		if d.Free < minFree {
			minFree = d.Free
		}
	}
	return minFree, minFree < threshold
}

// ---- 小工具 ----

func parseIntSafe(v string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// parseExcludeList 解析逗号分隔豁免清单并净化。
func parseExcludeList(raw string) []string { return sanitizeExclude(strings.Split(raw, ",")) }

// sanitizeExclude 去空白/去重/截断（单名 ≤128 字符，最多 50 项）。
func sanitizeExclude(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, n := range in {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] || len(n) > 128 {
			continue
		}
		seen[n] = true
		out = append(out, n)
		if len(out) >= 50 {
			break
		}
	}
	return out
}

func snapshotEncode(items []dto.GuardStoppedItem) string {
	b, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func snapshotDecode(raw string) []dto.GuardStoppedItem {
	out := []dto.GuardStoppedItem{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func namesOf(items []dto.GuardStoppedItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func joinRemark(prev, note string) string {
	if prev == "" {
		return note
	}
	return prev + "；" + note
}

// humanBytes 字节数人性化（通知/日志用）。
func humanBytes(n uint64) string {
	switch {
	case n >= gbBytes:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gbBytes))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
