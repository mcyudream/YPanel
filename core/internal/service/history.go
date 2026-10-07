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

// retention 保留时长。
const retention = 30 * 24 * time.Hour

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
	// 过期清理（概率性执行减少写放大）
	if time.Now().Second()%10 == 0 {
		if err := h.db.Where("at < ?", time.Now().Add(-retention)).Delete(&model.MetricRecord{}).Error; err != nil {
			slog.Warn("history: 清理失败", "err", err)
		}
	}
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
