package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
)

// HistoryRecorder 历史监控记录器（本机 + 全部远程节点，60s 采样）。
type HistoryRecorder struct {
	db    *gorm.DB
	nodes *NodeService
}

// NewHistoryRecorder 创建。
func NewHistoryRecorder(db *gorm.DB, nodes *NodeService) *HistoryRecorder {
	return &HistoryRecorder{db: db, nodes: nodes}
}

// retention 保留时长（原始明细）。
const retention = 30 * 24 * time.Hour

// hourlyRetention 小时聚合保留时长（M37）。
const hourlyRetention = 365 * 24 * time.Hour

// Start 启动 60s 采集循环。
func (h *HistoryRecorder) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.recordOnce(ctx)
			}
		}
	}()
}

func (h *HistoryRecorder) recordOnce(ctx context.Context) {
	ids := []string{"local"}
	rows := []model.Node{}
	if err := h.db.Find(&rows).Error; err == nil {
		for _, r := range rows {
			ids = append(ids, fmt.Sprintf("%d", r.ID))
		}
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()
			h.sampleNode(ctx, nodeID)
		}(id)
	}
	wg.Wait()
	// M37：整点回填上一小时聚合（幂等 upsert）
	h.rollupPreviousHour()
	// 过期清理（概率性执行减少写放大）
	if time.Now().Second()%10 == 0 {
		if err := h.db.Where("at < ?", time.Now().Add(-retention)).Delete(&model.MetricRecord{}).Error; err != nil {
			slog.Warn("history: 清理失败", "err", err)
		}
		if err := h.db.Where("hour < ?", time.Now().Add(-hourlyRetention)).Delete(&model.MetricHourly{}).Error; err != nil {
			slog.Warn("history: 小时聚合清理失败", "err", err)
		}
	}
}

// rollupPreviousHour 上一小时原始采样聚合入 MetricHourly（同 (node,hour) 唯一，重复执行覆盖更新）。
func (h *HistoryRecorder) rollupPreviousHour() {
	hourEnd := time.Now().Truncate(time.Hour)
	hourStart := hourEnd.Add(-time.Hour)
	var rows []model.MetricRecord
	if err := h.db.Where("at >= ? AND at < ?", hourStart, hourEnd).Find(&rows).Error; err != nil || len(rows) == 0 {
		return
	}
	agg := map[string]*model.MetricHourly{}
	for _, r := range rows {
		nid := r.NodeId
		a, ok := agg[nid]
		if !ok {
			a = &model.MetricHourly{NodeId: nid, Hour: hourStart}
			agg[nid] = a
		}
		a.CpuAvg += r.Cpu
		if r.Cpu > a.CpuMax {
			a.CpuMax = r.Cpu
		}
		a.MemAvg += r.Mem
		if r.Mem > a.MemMax {
			a.MemMax = r.Mem
		}
		// 60s 采样速率 B/s → 该采样窗口累计 MB（速率×60/1e6）
		a.RxMB += r.RxSpeed * 60 / 1e6
		a.TxMB += r.TxSpeed * 60 / 1e6
		if r.Load1 > a.LoadMax {
			a.LoadMax = r.Load1
		}
	}
	for _, a := range agg {
		var n int64
		_ = h.db.Model(&model.MetricRecord{}).Where("at >= ? AND at < ? AND node_id = ?", hourStart, hourEnd, a.NodeId).Count(&n).Error
		if n > 0 {
			a.CpuAvg /= float64(n)
			a.MemAvg /= float64(n)
		}
		// upsert：存在即覆盖（UniqueIndex node_id+hour）
		if err := h.db.Where("node_id = ? AND hour = ?", a.NodeId, a.Hour).FirstOrCreate(a).Error; err == nil {
			_ = h.db.Model(a).Updates(map[string]any{
				"cpu_avg": a.CpuAvg, "cpu_max": a.CpuMax, "mem_avg": a.MemAvg, "mem_max": a.MemMax,
				"rx_mb": a.RxMB, "tx_mb": a.TxMB, "load_max": a.LoadMax,
			}).Error
		} else {
			slog.Warn("history: 小时聚合落库失败", "err", err)
		}
	}
}

// QueryHourly 查询小时聚合（秒→小时数），转换为 MetricRecord 形状（avg 值，At=整点）供前端复用。
func (h *HistoryRecorder) QueryHourly(ctx context.Context, seconds int, nodeID string) []model.MetricRecord {
	nodeID = normalizeNodeID(nodeID)
	q := h.db.Where("hour > ?", time.Now().Add(-time.Duration(seconds)*time.Second))
	if nodeID == "local" {
		q = q.Where("node_id IN ?", []string{"", "local"})
	} else {
		q = q.Where("node_id = ?", nodeID)
	}
	var rows []model.MetricHourly
	_ = q.Order("hour").Find(&rows).Error
	out := make([]model.MetricRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.MetricRecord{
			NodeId: r.NodeId, At: r.Hour,
			Cpu: r.CpuAvg, Mem: r.MemAvg, RxSpeed: r.RxMB * 1e6 / 3600, TxSpeed: r.TxMB * 1e6 / 3600, Load1: r.LoadMax,
		})
	}
	return out
}

func (h *HistoryRecorder) sampleNode(ctx context.Context, nodeID string) {
	node, err := h.nodes.ByID(nodeID)
	if err != nil {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	ov, err := agentclient.GetJSON[dto.SystemOverview](agentclient.New(node.BaseURL, node.Token), cctx, "/agent/v1/sysinfo/overview")
	if err != nil {
		return // 离线节点本轮跳过
	}
	row := model.MetricRecord{
		NodeId: nodeID, At: time.Now(),
		Cpu: ov.CPU.UsagePercent, Mem: ov.Memory.UsagePercent, Swap: ov.Swap.UsagePercent,
		RxSpeed: ov.Network.RxSpeedBps, TxSpeed: ov.Network.TxSpeedBps, Load1: ov.Load.Load1,
	}
	if err := h.db.Create(&row).Error; err != nil {
		slog.Warn("history: 落库失败", "node", nodeID, "err", err)
	}
}

// normalizeNodeID 空串归一为 local。
func normalizeNodeID(nodeID string) string {
	if nodeID == "" {
		return "local"
	}
	return nodeID
}

// Query 查询区间历史（秒），按节点过滤，均匀抽样至 ≤1440 点。
func (h *HistoryRecorder) Query(ctx context.Context, seconds int, nodeID string) []model.MetricRecord {
	if seconds <= 0 || seconds > 30*24*3600 {
		seconds = 3600
	}
	nodeID = normalizeNodeID(nodeID)
	q := h.db.Where("at > ?", time.Now().Add(-time.Duration(seconds)*time.Second))
	if nodeID == "local" {
		// 多节点化前的旧数据 node_id 为空串
		q = q.Where("node_id IN ?", []string{"", "local"})
	} else {
		q = q.Where("node_id = ?", nodeID)
	}
	out := []model.MetricRecord{}
	_ = q.Order("at").Limit(30*24*3600 / 60).Find(&out).Error
	return decimateRecords(out, 1440)
}

// decimateRecords 等步长抽样，保留首尾点。
func decimateRecords(list []model.MetricRecord, max int) []model.MetricRecord {
	if len(list) <= max {
		return list
	}
	step := len(list) / max
	out := make([]model.MetricRecord, 0, max+1)
	for i := 0; i < len(list); i += step {
		out = append(out, list[i])
	}
	if out[len(out)-1].At != list[len(list)-1].At {
		out = append(out, list[len(list)-1])
	}
	return out
}
