// aitools_more.go AI 工具第二批补充：监控告警/面板运维/数据库实例管理/容器文件与配置/
// 站点域名/PHP 扩展/商店操作/组网（M32 按系统功能全貌补齐）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
)

// aiToolsMonitor 监控告警模块。
func (s *AIService) aiToolsMonitor(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_alert_rules", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "列出告警规则（指标/阈值/通知渠道/静默时段）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				rules := s.alert.ListRules()
				return toolOut(map[string]any{"total": len(rules), "items": rules})
			},
		},
		{
			Name: "create_alert_rule", Module: aiModMonitor, Risk: aiRiskWrite,
			Desc: "创建告警规则（指标超阈值推送通知）。input JSON：{\"name\":\"规则名\",\"metric\":\"cpu|mem|disk\",\"threshold\":85,\"webhookURL\":\"可选\",\"webhookType\":\"可选\",\"windowSec\":300}",
			Parameters: schObj(map[string]any{
				"name": schStr("规则名"), "metric": schEnum("监控指标", "cpu", "mem", "disk"), "threshold": schInt("阈值（百分比）"),
				"webhookURL": schStr("webhook 地址可省略"), "webhookType": schStr("webhook 类型可省略"),
				"windowSec": schInt("采样窗口秒，默认 300"),
			}, "name", "metric", "threshold"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name        string `json:"name"`
					Metric      string `json:"metric"`
					Threshold   int    `json:"threshold"`
					WebhookURL  string `json:"webhookURL"`
					WebhookType string `json:"webhookType"`
					SilentStart string `json:"silentStart"`
					SilentEnd   string `json:"silentEnd"`
					WindowSec   int    `json:"windowSec"`
				}](input)
				if err != nil {
					return "", err
				}
				rule, err := s.alert.CreateRule(p.Name, p.Metric, p.Threshold, p.WebhookURL, p.WebhookType, p.SilentStart, p.SilentEnd, "", p.WindowSec)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("告警规则已创建: %s（%s > %d%%）", rule.Name, rule.Metric, rule.Threshold), nil
			},
		},
		{
			Name: "query_metrics_history", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "查询节点历史监控指标（CPU/内存/磁盘/网络，按采样点返回）。input JSON：{\"seconds\":3600,\"node\":\"local 可选\"}（seconds 回看秒数，默认 3600）",
			Parameters: schObj(map[string]any{
				"seconds": schInt("回看秒数，默认 3600"), "node": schStr("节点 ID，默认 local"),
			}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Seconds int    `json:"seconds"`
					Node    string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Seconds < 60 {
					p.Seconds = 3600
				}
				if p.Node == "" {
					p.Node = "local"
				}
				rows := s.hist.Query(ctx, p.Seconds, p.Node)
				type pt struct {
					Time string  `json:"time"`
					CPU  float64 `json:"cpu"`
					Mem  float64 `json:"mem"`
				}
				out := make([]pt, 0, len(rows))
				for _, r := range rows {
					out = append(out, pt{Time: r.CreatedAt.Format("15:04"), CPU: r.CPU, Mem: r.Mem})
				}
				if len(out) > 60 {
					out = out[len(out)-60:]
				}
				return toolOut(map[string]any{"points": len(out), "items": out})
			},
		},
		{
			Name: "list_notifications", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "查看面板通知（告警/任务/系统事件）与未读数。input 可选 JSON：{\"limit\":20}",
			Parameters: schObj(map[string]any{"limit": schInt("条数，默认 20")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Limit int `json:"limit"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Limit < 1 || p.Limit > 100 {
					p.Limit = 20
				}
				rows := s.notif.List(p.Limit)
				return toolOut(map[string]any{"unread": s.notif.UnreadCount(), "total": len(rows), "items": rows})
			},
		},
		{
			Name: "list_login_logs", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "查看面板登录日志（时间/IP/账号/成败），安全审计用。input 可选 JSON：{\"limit\":20}",
			Parameters: schObj(map[string]any{"limit": schInt("条数，默认 20")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Limit int `json:"limit"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Limit < 1 || p.Limit > 100 {
					p.Limit = 20
				}
				rows := []model.LoginLog{}
				if err := s.db.Order("id desc").Limit(p.Limit).Find(&rows).Error; err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(rows), "items": rows})
			},
		},
	}
}

// aiToolsPanelOps 面板运维模块（备份/自更新）。
func (s *AIService) aiToolsPanelOps(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "create_panel_backup", Module: aiModPanel, Risk: aiRiskWrite,
			Desc: "创建面板备份（配置/数据库打包）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.panelBk.Create(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "面板备份完成", "detail": out})
			},
		},
		{
			Name: "list_panel_backups", Module: aiModPanel, Risk: aiRiskRead,
			Desc: "列出面板备份文件。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.panelBk.List(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "delete_panel_backup", Module: aiModPanel, Risk: aiRiskDanger,
			Desc: "删除面板备份文件（不可恢复，会先向用户确认）。input JSON：{\"file\":\"备份文件名\"}",
			Parameters: schObj(map[string]any{"file": schStr("备份文件名")}, "file"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					File string `json:"file"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.panelBk.Delete(ctx, p.File); err != nil {
					return "", err
				}
				return "已删除备份: " + p.File, nil
			},
		},
		{
			Name: "check_panel_update", Module: aiModPanel, Risk: aiRiskRead,
			Desc: "检查面板更新（当前版本/最新版本/是否可升级/可用源）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				online, err := s.su.CheckOnline(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(online)
			},
		},
		{
			Name: "upgrade_panel", Module: aiModPanel, Risk: aiRiskDanger,
			Desc: "升级面板到最新版本（重启面板服务，会断开所有会话，会先向用户确认）。input JSON：{\"source\":\"源名可省略（默认可用源）\"}",
			Parameters: schObj(map[string]any{"source": schStr("更新源名，省略则自动")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Source string `json:"source"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.su.Upgrade(ctx, p.Source)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "升级任务已触发，面板将自动重启", "detail": out})
			},
		},
	}
}
