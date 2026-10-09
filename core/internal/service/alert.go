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
	db       *gorm.DB
	nodes    *NodeService
	notif    *NotificationService
	settings *SettingService    // SMTP 全局配置（M37 email 渠道）
	lc       *LogCentralService // metric=log：VL 聚合计数（P3，装配兜底注入）

	fired  map[string]time.Time // ruleKey:metric → 上次触发时间（去抖）
	breach map[string]int       // ruleKey:metric → 连续超阈采样数
}

// SetLogCentral 注入日志聚合服务（metric=log 规则用；main 装配兜底）。
func (s *AlertService) SetLogCentral(lc *LogCentralService) { s.lc = lc }

// NewAlertService 创建。
func NewAlertService(db *gorm.DB, nodes *NodeService, notif *NotificationService, settings *SettingService) *AlertService {
	return &AlertService{db: db, nodes: nodes, notif: notif, settings: settings, fired: map[string]time.Time{}, breach: map[string]int{}}
}

// SetSettings 注入设置服务（main 装配兜底）。
func (s *AlertService) SetSettings(st *SettingService) { s.settings = st }

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
	case "load":
		return ov.Load.Load1, true
	case "network":
		// 出+入取大者，单位 MB/s（阈值口径一致）
		rx := ov.Network.RxSpeedBps / 1e6
		tx := ov.Network.TxSpeedBps / 1e6
		if tx > rx {
			return tx, true
		}
		return rx, true
	}
	return 0, false
}

// certExpiryDaysMin 证书库最早到期天数（无证书返回 ok=false）。
func (s *AlertService) certExpiryDaysMin() (float64, bool) {
	var rows []model.Certificate
	if err := s.db.Where("not_after IS NOT NULL").Find(&rows).Error; err != nil || len(rows) == 0 {
		return 0, false
	}
	minD := time.Duration(1 << 62)
	found := false
	for _, r := range rows {
		if r.NotAfter == nil {
			continue
		}
		d := time.Until(*r.NotAfter)
		if d < minD {
			minD = d
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return minD.Hours() / 24, true
}

// inSilentWindow 当前时间是否在规则静默时段（HH:MM-HH:MM，支持跨零点）。
func inSilentWindow(r model.AlertRule, now time.Time) bool {
	if r.SilentStart == "" || r.SilentEnd == "" {
		return false
	}
	lay := "15:04"
	start, err1 := time.Parse(lay, r.SilentStart)
	end, err2 := time.Parse(lay, r.SilentEnd)
	if err1 != nil || err2 != nil {
		return false
	}
	cur := now.Hour()*60 + now.Minute()
	sm := start.Hour()*60 + start.Minute()
	em := end.Hour()*60 + end.Minute()
	if sm <= em {
		return cur >= sm && cur < em
	}
	return cur >= sm || cur < em // 跨零点
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
	var ov *dto.SystemOverview
	for _, r := range rules {
		key := fmt.Sprintf("%d:%s", r.ID, r.Metric)
		var v float64
		var ok bool
		switch {
		case r.Metric == "log":
			v, ok = s.logCount(ctx, r)
		case r.Metric == "cert_expiry":
			v, ok = s.certExpiryDaysMin()
		case r.Metric == "site_expiry":
			v, ok = s.siteExpiryDaysMin()
		default:
			// 系统指标按需懒采集：纯日志规则集不依赖 agent 采集
			if ov == nil {
				ac, err := s.client()
				if err != nil {
					return
				}
				ov, err = agentclient.GetJSON[dto.SystemOverview](ac, ctx, "/agent/v1/sysinfo/overview")
				if err != nil {
					slog.Warn("alert: 采集失败", "err", err)
					return
				}
			}
			v, ok = metricValue(r.Metric, ov)
		}
		if !ok {
			continue
		}
		slog.Info("alert: rule check", "rule", r.Name, "metric", r.Metric, "value", v, "threshold", r.Threshold, "breach", s.breach[key])
		if v >= float64(r.Threshold) {
			s.breach[key]++
			if s.breach[key] >= 2 { // 连续 2 个周期确认
				if last, fired := s.fired[key]; !fired || time.Since(last) > alertDebounce {
					if inSilentWindow(r, time.Now()) {
						s.fired[key] = time.Now() // 静默期记为已触发，出窗口后重新计去抖
						continue
					}
					s.fired[key] = time.Now()
					msg := s.alertMessage(r, ov, v, "超过阈值 %d", false)
					s.notify(ctx, r.WebhookURL, r.WebhookType, msg)
					if s.notif != nil {
						s.notif.Push("warning", alertTitle(r), msg)
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
					s.notify(ctx, r.WebhookURL, r.WebhookType, s.alertMessage(r, ov, v, "已回落至阈值 %d 以下", true))
				}
			}
			s.breach[key] = 0
		}
	}
}

// alertTitle 站内通知标题（日志规则与阈值规则区分）。
func alertTitle(r model.AlertRule) string {
	if r.Metric == "log" {
		return "日志量告警"
	}
	return "阈值告警"
}

// alertMessage 触发/恢复消息文案（日志规则带查询与窗口，系统指标带主机名）。
func (s *AlertService) alertMessage(r model.AlertRule, ov *dto.SystemOverview, v float64, tail string, recovering bool) string {
	prefix := "【YPanel 告警】"
	if recovering {
		prefix = "【YPanel 恢复】"
	}
	if r.Metric == "log" {
		return fmt.Sprintf("%s日志量告警[%s] 最近 %d 分钟命中 %d 条，%s", prefix, r.Name, r.WindowSec/60, int(v), fmt.Sprintf(tail, r.Threshold))
	}
	host := "面板"
	if ov != nil {
		host = ov.Hostname
	}
	return fmt.Sprintf("%s%s %s 当前值 %.1f，%s", prefix, host, strings.ToUpper(r.Metric), v, fmt.Sprintf(tail, r.Threshold))
}

// logCount 统计日志规则窗口内命中总量（跨节点 VL 聚合；VL 不可用跳过本轮）。
func (s *AlertService) logCount(ctx context.Context, r model.AlertRule) (float64, bool) {
	if s.lc == nil {
		return 0, false
	}
	window := time.Duration(r.WindowSec) * time.Second
	if window < time.Minute {
		window = time.Minute
	}
	start := time.Now().Add(-window).Format(time.RFC3339)
	hits, err := s.lc.Hits(ctx, r.Query, start, "", "")
	if err != nil {
		slog.Warn("alert: 日志计数失败", "rule", r.Name, "err", err)
		return 0, false
	}
	return float64(hits.Total), true
}

// notify 按渠道类型发送（M37：bark/email + 既有 webhook 族）。
func (s *AlertService) notify(ctx context.Context, url, typ, msg string) {
	if typ == "bark" {
		_ = SendBark(ctx, url, "YPanel 告警", msg)
		return
	}
	if typ == "email" {
		cfg := SMTPConfig{}
		if s.settings != nil {
			cfg = LoadSMTPConfig(s.settings)
		}
		_ = SendMail(ctx, cfg, "【YPanel 告警】"+firstLine(msg), msg)
		return
	}
	var payload any
	switch typ {
	case "feishu":
		payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": msg}}
	case "dingtalk":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": msg}}
	case "wecom":
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": msg}}
	case "telegram":
		// B15：TG bot API——url 即 https://api.telegram.org/bot<token>/sendMessage，chat_id 经 query
		if i := strings.Index(url, "?chat_id="); i > 0 {
			payload = map[string]any{"chat_id": url[i+len("?chat_id="):], "text": msg}
			url = url[:i]
		} else {
			return
		}
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
func (s *AlertService) CreateRule(name, metric string, threshold int, webhookURL, webhookType, silentStart, silentEnd, query string, windowSec int) (*model.AlertRule, error) {
	if !alertMetricOK(metric) {
		return nil, errs.Wrap(errs.ErrBadRequest, "指标仅支持 cpu/memory/disk/load/network/cert_expiry/site_expiry/log")
	}
	if err := validateThreshold(metric, threshold); err != nil {
		return nil, err
	}
	if metric == "log" {
		if strings.TrimSpace(query) == "" {
			return nil, errs.Wrap(errs.ErrBadRequest, "日志规则需要 LogsQL 查询")
		}
		if len(query) > 512 {
			return nil, errs.Wrap(errs.ErrBadRequest, "LogsQL 查询过长（上限 512 字符）")
		}
		if windowSec <= 0 {
			windowSec = 300
		}
		if windowSec < 60 || windowSec > 86400 {
			return nil, errs.Wrap(errs.ErrBadRequest, "统计窗口需在 60-86400 秒")
		}
	}
	if webhookType != "email" {
		if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
			return nil, errs.Wrap(errs.ErrBadRequest, "webhook 地址不合法")
		}
	}
	row := &model.AlertRule{Name: name, Metric: metric, Threshold: threshold, WebhookURL: webhookURL, WebhookType: webhookType, SilentStart: silentStart, SilentEnd: silentEnd, Query: query, WindowSec: windowSec, Enabled: true}
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
	switch m {
	case "cpu", "memory", "disk", "load", "network", "cert_expiry", "site_expiry", "node_expiry", "log":
		return true
	}
	return false
}

// siteExpiryDaysMin 网站最早到期天数（M39）。
func (s *AlertService) siteExpiryDaysMin() (float64, bool) {
	var rows []model.Site
	if err := s.db.Where("expire_at IS NOT NULL").Find(&rows).Error; err != nil || len(rows) == 0 {
		return 0, false
	}
	minD := time.Duration(1 << 62)
	found := false
	for _, r := range rows {
		if r.ExpireAt == nil {
			continue
		}
		d := time.Until(*r.ExpireAt)
		if d < minD {
			minD = d
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return minD.Hours() / 24, true
}

// validateThreshold 按指标校验阈值量纲。
func validateThreshold(metric string, threshold int) error {
	switch metric {
	case "cpu", "memory", "disk":
		if threshold < 1 || threshold > 100 {
			return errs.Wrap(errs.ErrBadRequest, "百分比阈值需在 1-100")
		}
	case "load":
		if threshold < 1 || threshold > 1000 {
			return errs.Wrap(errs.ErrBadRequest, "load 阈值需在 1-1000")
		}
	case "network":
		if threshold < 1 || threshold > 10000 {
			return errs.Wrap(errs.ErrBadRequest, "网速阈值（MB/s）需在 1-10000")
		}
	case "cert_expiry", "site_expiry":
		if threshold < 1 || threshold > 365 {
			return errs.Wrap(errs.ErrBadRequest, "证书到期阈值（天）需在 1-365")
		}
	case "log":
		if threshold < 1 || threshold > 10000000 {
			return errs.Wrap(errs.ErrBadRequest, "日志量阈值（条）需在 1-10000000")
		}
	}
	return nil
}


// nodeExpiryDaysMin 节点最早到期天数（M43；无到期节点返回 ok=false）。
func (s *AlertService) nodeExpiryDaysMin() (float64, bool) {
	var rows []model.Node
	if err := s.db.Where("expire_date IS NOT NULL").Find(&rows).Error; err != nil || len(rows) == 0 {
		return 0, false
	}
	minD := time.Duration(1 << 62)
	found := false
	for _, r := range rows {
		if r.ExpireDate == nil {
			continue
		}
		d := time.Until(*r.ExpireDate)
		if d < minD {
			minD = d
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return minD.Hours() / 24, true
}
