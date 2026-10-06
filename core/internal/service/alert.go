package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 去抖窗口。
const alertDebounce = 10 * time.Minute

// AlertService 告警服务：规则 CRUD + 周期评估 + webhook 通知。
type AlertService struct {
	db    *gorm.DB
	nodes *NodeService
	notif *NotificationService

	fired map[string]time.Time // ruleKey:metric → 上次触发时间（去抖）
	breach map[string]int      // ruleKey:metric → 连续超阈采样数
}

// NewAlertService 创建。
func NewAlertService(db *gorm.DB, nodes *NodeService, notif *NotificationService) *AlertService {
	return &AlertService{db: db, nodes: nodes, notif: notif, fired: map[string]time.Time{}, breach: map[string]int{}}
}

// Start 启动评估循环（30s）。
func (s *AlertService) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(30 * time.Second)
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

// metricValue 从概览中取指标百分比。
func metricValue(metric string, ov *dto.SystemOverview) (float64, bool) {
	switch metric {
	case "cpu":
		return ov.CPU.UsagePercent, true
	case "memory":
		return ov.Memory.UsagePercent, true
	case "disk":
		if len(ov.Disks) > 0 {
			return ov.Disks[0].UsagePercent, true
		}
	}
	return 0, false
}

// evaluateOnce 单轮评估。
func (s *AlertService) evaluateOnce(ctx context.Context) {
	var rules []model.AlertRule
	if err := s.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		slog.Warn("alert: 读取规则失败", "err", err)
		return
	}
	if len(rules) == 0 {
		return
	}
	slog.Info("alert: evaluate", "rules", len(rules))
	ac, err := s.client()
	if err != nil {
		return
	}
	ov, err := agentclient.GetJSON[dto.SystemOverview](ac, ctx, "/agent/v1/sysinfo/overview")
	if err != nil {
		slog.Warn("alert: 采集失败", "err", err)
		return
	}
	for _, r := range rules {
		key := fmt.Sprintf("%d:%s", r.ID, r.Metric)
		v, ok := metricValue(r.Metric, ov)
		if !ok {
			continue
		}
		slog.Info("alert: rule check", "rule", r.Name, "metric", r.Metric, "value", v, "threshold", r.Threshold, "breach", s.breach[key])
		if v >= float64(r.Threshold) {
			s.breach[key]++
			if s.breach[key] >= 2 { // 连续 2 个周期确认
				if last, fired := s.fired[key]; !fired || time.Since(last) > alertDebounce {
					s.fired[key] = time.Now()
					msg := fmt.Sprintf("【YPanel 告警】%s %s 使用率 %.1f%%，超过阈值 %d%%", ov.Hostname, strings.ToUpper(r.Metric), v, r.Threshold)
					s.notify(ctx, r.WebhookURL, r.WebhookType, msg)
					if s.notif != nil {
						s.notif.Push("warning", "阈值告警", msg)
					}
					rec := model.Setting{Key: "alert.record:" + key + ":" + fmt.Sprint(time.Now().Unix()), Value: msg}
					if err := s.db.Create(&rec).Error; err != nil {
						slog.Error("alert: 记录落库失败", "err", err)
					} else {
						slog.Info("alert: 已触发并落库", "rule", r.Name)
					}
				}
			}
		} else {
			if s.breach[key] >= 2 {
				if last, fired := s.fired[key]; fired && time.Since(last) <= alertDebounce*2 {
					s.notify(ctx, r.WebhookURL, r.WebhookType,
						fmt.Sprintf("【YPanel 恢复】%s %s 使用率 %.1f%%，已回落至阈值 %d%% 以下", ov.Hostname, strings.ToUpper(r.Metric), v, r.Threshold))
				}
			}
			s.breach[key] = 0
		}
	}
}

// notify 按 webhook 类型发送。
func (s *AlertService) notify(ctx context.Context, url, typ, msg string) {
	var payload any
	switch typ {
	case "feishu":
		payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": msg}}
	case "dingtalk":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": msg}}
	case "wecom":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": msg}}
	default:
		payload = map[string]any{"text": msg, "source": "ypanel", "time": time.Now().Format(time.RFC3339)}
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func (s *AlertService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// ListRules 规则列表。
func (s *AlertService) ListRules() []model.AlertRule {
	rules := []model.AlertRule{}
	_ = s.db.Order("id").Find(&rules).Error
	return rules
}

// CreateRule 新建规则。
func (s *AlertService) CreateRule(name, metric string, threshold int, webhookURL, webhookType string) (*model.AlertRule, error) {
	if !alertMetricOK(metric) {
		return nil, errs.Wrap(errs.ErrBadRequest, "指标仅支持 cpu/memory/disk")
	}
	if threshold < 1 || threshold > 100 {
		return nil, errs.Wrap(errs.ErrBadRequest, "阈值需在 1-100")
	}
	if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
		return nil, errs.Wrap(errs.ErrBadRequest, "webhook 地址不合法")
	}
	row := &model.AlertRule{Name: name, Metric: metric, Threshold: threshold, WebhookURL: webhookURL, WebhookType: webhookType, Enabled: true}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateRule 更新规则（启用/阈值）。
func (s *AlertService) UpdateRule(id uint, updates map[string]any) error {
	return s.db.Model(&model.AlertRule{ID: uint(id)}).Updates(updates).Error
}

// DeleteRule 删除规则。
func (s *AlertService) DeleteRule(id uint) error {
	return s.db.Delete(&model.AlertRule{}, id).Error
}

func alertMetricOK(m string) bool {
	return m == "cpu" || m == "memory" || m == "disk"
}
