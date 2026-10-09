// aitools_netsec.go AI 工具：防火墙/NAT/Hosts/DNS/计划任务（M31）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ypanel/core/internal/model"
)

// aiToolsNetSec 网络安全工具集。
func (s *AIService) aiToolsNetSec(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "get_network_security", Module: aiModNetSec, Risk: aiRiskRead,
			Desc: "聚合查询网络安全配置：防火墙状态与放行规则、NAT 转发规则、Hosts 解析记录、内网 DNS 记录。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out := map[string]any{}
				if st, err := s.fw.Status(ctx); err == nil {
					out["firewall"] = st
				} else {
					out["firewall"] = "查询失败: " + err.Error()
				}
				if rules, err := s.nat.List("local"); err == nil {
					out["natRules"] = rules
				}
				if hrs, err := s.hosts.ListRecords(); err == nil {
				out["hostsRecords"] = hrs
			}
				if drs, err := s.dns.ListRecords(); err == nil {
				out["dnsRecords"] = drs
			}
				return toolOut(out)
			},
		},
		{
			Name: "firewall_rule_add", Module: aiModNetSec, Risk: aiRiskWrite,
			Desc: "放行防火墙端口。input JSON：{\"port\":\"80 或 3000:3010\",\"proto\":\"tcp|udp\"}",
			Parameters: schObj(map[string]any{
				"port": schStr("端口或端口段"), "proto": schEnum("协议", "tcp", "udp"),
			}, "port", "proto"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Port  string `json:"port"`
					Proto string `json:"proto"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Proto != "tcp" && p.Proto != "udp" {
					p.Proto = "tcp"
				}
				if err := s.fw.Allow(ctx, p.Port, p.Proto); err != nil {
					return "", err
				}
				return fmt.Sprintf("防火墙已放行 %s/%s", p.Port, p.Proto), nil
			},
		},
		{
			Name: "firewall_rule_remove", Module: aiModNetSec, Risk: aiRiskDanger,
			Desc: "删除防火墙放行规则（编号见 get_network_security，误删可能锁死访问，会先向用户确认）。input JSON：{\"number\":规则编号}",
			Parameters: schObj(map[string]any{"number": schInt("规则编号")}, "number"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Number int `json:"number"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.fw.DeleteRule(ctx, p.Number); err != nil {
					return "", err
				}
				return fmt.Sprintf("已删除防火墙规则 #%d", p.Number), nil
			},
		},
		{
			Name: "nat_rule_save", Module: aiModNetSec, Risk: aiRiskWrite,
			Desc: "新建/更新 NAT 端口转发规则（保存后自动下发 iptables）。input JSON：{\"id\":更新时传,\"name\":\"规则名\",\"protocol\":\"tcp|udp\",\"listenPort\":8080,\"targetIp\":\"10.0.0.2\",\"targetPort\":80,\"enabled\":true}",
			Parameters: schObj(map[string]any{
				"id": schInt("更新已有规则时传"), "name": schStr("规则名"), "protocol": schEnum("协议", "tcp", "udp"),
				"listenPort": schInt("监听端口"), "targetIp": schStr("目标 IP"), "targetPort": schInt("目标端口"),
				"enabled": schBool("启用"),
			}, "name", "protocol", "listenPort", "targetIp", "targetPort"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID         uint   `json:"id"`
					Name       string `json:"name"`
					Protocol   string `json:"protocol"`
					ListenPort int    `json:"listenPort"`
					TargetIP   string `json:"targetIp"`
					TargetPort int    `json:"targetPort"`
					Enabled    *bool  `json:"enabled"`
				}](input)
				if err != nil {
					return "", err
				}
				enabled := true
				if p.Enabled != nil {
					enabled = *p.Enabled
				}
				rule := &model.NatForwardRule{
					NodeID: "local", Name: p.Name, Protocol: p.Protocol,
					ListenPort: p.ListenPort, TargetIP: p.TargetIP, TargetPort: p.TargetPort, Enabled: enabled,
				}
				if p.ID > 0 {
					rule.ID = p.ID
				}
				if _, warns, err := s.nat.Save(ctx, rule); err != nil {
					return "", err
				} else if len(warns) > 0 {
					return fmt.Sprintf("规则已保存（警告: %s）", strings.Join(warns, "；")), nil
				}
				return "NAT 规则已保存并下发", nil
			},
		},
		{
			Name: "nat_rule_remove", Module: aiModNetSec, Risk: aiRiskDanger,
			Desc: "删除 NAT 转发规则（对应转发立即失效，会先向用户确认）。input JSON：{\"id\":规则ID}",
			Parameters: schObj(map[string]any{"id": schInt("规则 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if _, err := s.nat.Delete(ctx, p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("NAT 规则 %d 已删除", p.ID), nil
			},
		},
		{
			Name: "hosts_record_save", Module: aiModNetSec, Risk: aiRiskWrite,
			Desc: "新建/更新 Hosts 解析记录（自动下发到托管节点 /etc/hosts）。input JSON：{\"id\":更新时传,\"ip\":\"10.0.0.2\",\"hostnames\":\"a.internal b.internal\",\"comment\":\"备注\",\"enabled\":true}",
			Parameters: schObj(map[string]any{
				"id": schInt("更新已有记录时传"), "ip": schStr("IP 地址"), "hostnames": schStr("主机名，空格分隔多个"),
				"comment": schStr("备注"), "enabled": schBool("启用"),
			}, "ip", "hostnames"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID        uint   `json:"id"`
					IP        string `json:"ip"`
					Hostnames string `json:"hostnames"`
					Comment   string `json:"comment"`
					Enabled   *bool  `json:"enabled"`
				}](input)
				if err != nil {
					return "", err
				}
				enabled := true
				if p.Enabled != nil {
					enabled = *p.Enabled
				}
				rec := &model.HostRecord{IP: p.IP, Hostnames: p.Hostnames, Comment: p.Comment, Enabled: enabled}
				if p.ID > 0 {
					rec.ID = p.ID
				}
				if _, err := s.hosts.SaveRecord(ctx, rec); err != nil {
					return "", err
				}
				return "Hosts 记录已保存并下发", nil
			},
		},
		{
			Name: "hosts_record_remove", Module: aiModNetSec, Risk: aiRiskDanger,
			Desc: "删除 Hosts 解析记录（依赖该记录的服务会解析失败，会先向用户确认）。input JSON：{\"id\":记录ID}",
			Parameters: schObj(map[string]any{"id": schInt("记录 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.hosts.DeleteRecord(ctx, p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("Hosts 记录 %d 已删除", p.ID), nil
			},
		},
		{
			Name: "dns_record_save", Module: aiModNetSec, Risk: aiRiskWrite,
			Desc: "新建/更新内网 DNS 解析记录（dnsmasq，保存后自动部署）。input JSON：{\"id\":更新时传,\"type\":\"address|cname|txt\",\"domain\":\"nas.internal\",\"target\":\"10.0.0.2\",\"enabled\":true}",
			Parameters: schObj(map[string]any{
				"id": schInt("更新已有记录时传"), "type": schEnum("记录类型", "address", "cname", "txt"),
				"domain": schStr("域名"), "target": schStr("目标（IP/主机名/文本）"), "enabled": schBool("启用"),
			}, "type", "domain", "target"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID      uint   `json:"id"`
					Type    string `json:"type"`
					Domain  string `json:"domain"`
					Target  string `json:"target"`
					Enabled *bool  `json:"enabled"`
				}](input)
				if err != nil {
					return "", err
				}
				enabled := true
				if p.Enabled != nil {
					enabled = *p.Enabled
				}
				rec := &model.DnsRecord{Type: p.Type, Domain: p.Domain, Target: p.Target, Enabled: enabled}
				if p.ID > 0 {
					rec.ID = p.ID
				}
				if _, err := s.dns.SaveRecord(ctx, rec); err != nil {
					return "", err
				}
				return "DNS 记录已保存", nil
			},
		},
		{
			Name: "dns_record_remove", Module: aiModNetSec, Risk: aiRiskDanger,
			Desc: "删除内网 DNS 解析记录（依赖该记录的访问会失败，会先向用户确认）。input JSON：{\"id\":记录ID}",
			Parameters: schObj(map[string]any{"id": schInt("记录 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dns.DeleteRecord(ctx, p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("DNS 记录 %d 已删除", p.ID), nil
			},
		},
	}
}

// aiToolsTasks 计划任务工具集。
func (s *AIService) aiToolsTasks(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_tasks", Module: aiModTasks, Risk: aiRiskRead,
			Desc: "列出计划任务（含 cron 表达式/最近执行结果）。input 可选 JSON：{\"search\":\"任务名关键词\"}",
			Parameters: schObj(map[string]any{"search": schStr("任务名关键词")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				rows := []model.CronTask{}
				if err := s.db.Order("id").Find(&rows).Error; err != nil {
					return "", err
				}
				filtered := make([]model.CronTask, 0, len(rows))
				for _, t := range rows {
					if matchAny(p.Search, t.Name, t.Command) {
						filtered = append(filtered, t)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "save_task", Module: aiModTasks, Risk: aiRiskWrite,
			Desc: "新建/更新计划任务（shell 类型）。input JSON：{\"id\":更新时传,\"name\":\"任务名\",\"cron\":\"*/5 * * * *\",\"command\":\"shell 命令\",\"timeoutSecs\":300,\"enabled\":true}",
			Parameters: schObj(map[string]any{
				"id": schInt("更新已有任务时传"), "name": schStr("任务名"), "cron": schStr("cron 表达式（5 段）"),
				"command": schStr("shell 命令"), "timeoutSecs": schInt("超时秒数，默认 300"), "enabled": schBool("启用"),
			}, "name", "cron", "command"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID          uint   `json:"id"`
					Name        string `json:"name"`
					Cron        string `json:"cron"`
					Command     string `json:"command"`
					TimeoutSecs int    `json:"timeoutSecs"`
					Enabled     *bool  `json:"enabled"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Command) == "" {
					return "", fmt.Errorf("name 与 command 必填")
				}
				if p.TimeoutSecs < 1 {
					p.TimeoutSecs = 300
				}
				enabled := true
				if p.Enabled != nil {
					enabled = *p.Enabled
				}
				if p.ID > 0 {
					updates := map[string]any{
						"name": p.Name, "cron": p.Cron, "command": p.Command,
						"timeout_secs": p.TimeoutSecs, "enabled": enabled,
					}
					if err := s.db.Model(&model.CronTask{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
						return "", err
					}
				} else {
					row := model.CronTask{Name: p.Name, Cron: p.Cron, Command: p.Command, Type: "shell", Enabled: enabled, TimeoutSecs: p.TimeoutSecs}
					if err := s.db.Create(&row).Error; err != nil {
						return "", err
					}
				}
				s.cron.Reload()
				return "计划任务已保存并重新加载", nil
			},
		},
		{
			Name: "get_task", Module: aiModTasks, Risk: aiRiskRead,
			Desc: "查询任务中心任务状态与日志（应用安装/源码构建/备份等异步任务的结果跟踪）。input JSON：{\"id\":任务ID}",
			Parameters: schObj(map[string]any{"id": schInt("任务 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				var row model.AppTask
				if err := s.db.First(&row, p.ID).Error; err != nil {
					return "", fmt.Errorf("任务不存在: %d", p.ID)
				}
				log := row.LogText
				if len(log) > 6000 {
					log = "…[截断]\n" + log[len(log)-6000:]
				}
				return toolOut(map[string]any{
					"id": row.ID, "type": row.Type, "title": row.Title, "ref": row.Ref,
					"status": row.Status, "error": row.Error, "createdAt": row.CreatedAt, "log": log,
				})
			},
		},
		{
			Name: "run_task", Module: aiModTasks, Risk: aiRiskWrite,
			Desc: "立即执行一次计划任务（异步执行，结果看任务日志；命令有副作用，会先向用户确认）。input JSON：{\"id\":任务ID}",
			Parameters: schObj(map[string]any{"id": schInt("任务 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.cron.RunNow(p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("任务 %d 已触发执行（异步）", p.ID), nil
			},
		},
		{
			Name: "delete_task", Module: aiModTasks, Risk: aiRiskDanger,
			Desc: "删除计划任务（其备份/清理自动化随之失效，会先向用户确认）。input JSON：{\"id\":任务ID}",
			Parameters: schObj(map[string]any{"id": schInt("任务 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.db.Delete(&model.CronTask{}, p.ID).Error; err != nil {
					return "", err
				}
				s.cron.Reload()
				return fmt.Sprintf("计划任务 %d 已删除", p.ID), nil
			},
		},
	}
}

var (
	_ = json.Marshal
	_ = time.Now
)
