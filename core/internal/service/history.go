package service

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
)

// HistoryRecorder 历史监控记录器。
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
	ac, err := h.client()
	if err != nil {
		return
	}
	ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ctx, "/agent/v1/sysinfo/overview")
	if err != nil {
		return
	}
	row := model.MetricRecord{
		At: time.Now(), Cpu: ov.CPU.UsagePercent, Mem: ov.Memory.UsagePercent,
		RxSpeed: ov.Network.RxSpeedBps, TxSpeed: ov.Network.TxSpeedBps, Load1: ov.Load.Load1,
	}
	if err := h.db.Create(&row).Error; err != nil {
		slog.Warn("history: 落库失败", "err", err)
		return
	}
	// 过期清理（概率性执行减少写放大）
	if time.Now().Second()%10 == 0 {
		if err := h.db.Where("at < ?", time.Now().Add(-retention)).Delete(&model.MetricRecord{}).Error; err != nil {
			slog.Warn("history: 清理失败", "err", err)
		}
	}
}

func (h *HistoryRecorder) client() (*agentclient.Client, error) {
	node, err := h.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// Query 查询区间历史（秒）。
func (h *HistoryRecorder) Query(ctx context.Context, seconds int) []model.MetricRecord {
	if seconds <= 0 || seconds > 30*24*3600 {
		seconds = 3600
	}
	out := []model.MetricRecord{}
	_ = h.db.Where("at > ?", time.Now().Add(-time.Duration(seconds)*time.Second)).
		Order("at").Limit(5000).Find(&out).Error
	return out
}


