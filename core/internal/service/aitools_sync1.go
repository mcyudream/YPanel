// aitools_sync1.go AI 工具同步补齐（M33 固化机制第一批）：告警规则管理/日志中心查询/
// 配置快照回滚/系统审计/DNS 账号/Mongo 集合。
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ypanel/core/internal/model"
)

// aiToolsAlertOps 告警规则管理补充。
func (s *AIService) aiToolsAlertOps(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "update_alert_rule", Module: aiModMonitor, Risk: aiRiskWrite,
			Desc: "更新告警规则（阈值/启停/webhook 等，字段可省略=不修改）。input JSON：{\"id\":规则ID,\"threshold\":90,\"enabled\":false}（updates 键值透传）",
			Parameters: schObj(map[string]any{
				"id":       schInt("规则 ID"),
				"updates":  schObj(map[string]any{}, "待更新字段（threshold/enabled/webhookURL 等）"),
			}, "id", "updates"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID      uint           `json:"id"`
					Updates map[string]any `json:"updates"`
				}](input)
				if err != nil {
					return "", err
				}
				if len(p.Updates) == 0 {
					return "", fmt.Errorf("缺少 updates")
				}
				if err := s.alert.UpdateRule(p.ID, p.Updates); err != nil {
					return "", err
				}
				return fmt.Sprintf("告警规则 %d 已更新", p.ID), nil
			},
		},
		{
			Name: "delete_alert_rule", Module: aiModMonitor, Risk: aiRiskDanger,
			Desc: "删除告警规则（该指标将不再告警，会先向用户确认）。input JSON：{\"id\":规则ID}",
			Parameters: schObj(map[string]any{"id": schInt("规则 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.alert.DeleteRule(p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("告警规则 %d 已删除", p.ID), nil
			},
		},
	}
}

// aiToolsLogCentral 日志中心查询（M33 新功能同步）。
func (s *AIService) aiToolsLogCentral(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "query_logs", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "日志中心查询：跨节点聚合检索结构化日志（LogsQL 语法，如 `app=\"nginx\" and level>=error`）。input JSON：{\"query\":\"查询语句\",\"start\":\"可选开始时间\",\"end\":\"可选结束时间\",\"limit\":50}",
			Parameters: schObj(map[string]any{
				"query": schStr("LogsQL 查询语句"), "start": schStr("开始时间，可省略"),
				"end": schStr("结束时间，可省略"), "limit": schInt("条数上限，默认 50"),
			}, "query"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Query string `json:"query"`
					Start string `json:"start"`
					End   string `json:"end"`
					Limit int    `json:"limit"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Limit < 1 || p.Limit > 200 {
					p.Limit = 50
				}
				out, err := s.logc.Query(ctx, CentralQueryArg(p.Query, p.Start, p.End, p.Limit))
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
	}
}

// CentralQueryArg 组装日志中心查询参数。
func CentralQueryArg(query, start, end string, limit int) CentralQuery {
	if limit < 1 || limit > 500 {
		limit = 50
	}
	return CentralQuery{Query: query, Start: start, End: end, Limit: limit}
}

// aiToolsSnapshot 配置快照回滚（受管路径写盘自动快照：编排/daemon.json/站点配置等）。
func (s *AIService) aiToolsSnapshot(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_config_revisions", Module: aiModPanel, Risk: aiRiskRead,
			Desc: "列出受管配置路径的历史版本（写盘前自动快照：编排 yaml/daemon.json/站点 conf 等）。input JSON：{\"node\":\"local 可选\",\"path\":\"/opt/ypanel/compose/app-blog/docker-compose.yml\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("节点，默认 local"), "path": schStr("配置文件绝对路径"),
			}, "path"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Path string `json:"path"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Node == "" {
					p.Node = "local"
				}
				rows, err := s.rev.List(ctx, p.Node, p.Path)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(rows), "items": rows})
			},
		},
		{
			Name: "restore_config_revision", Module: aiModPanel, Risk: aiRiskDanger,
			Desc: "回滚配置到指定历史版本（写盘生效，会先向用户确认）。input JSON：{\"id\":快照ID}",
			Parameters: schObj(map[string]any{"id": schInt("快照 ID（list_config_revisions 可查）")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				rev, err := s.rev.Restore(ctx, p.ID, "ai")
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("已回滚配置：%s（快照 %d）", rev.Scope, p.ID), nil
			},
		},
	}
}

// aiToolsAudit 系统操作审计。
func (s *AIService) aiToolsAudit(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_audit_logs", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "查看系统操作审计日志（谁在什么时间对什么路径做了什么操作）。input 可选 JSON：{\"limit\":30}",
			Parameters: schObj(map[string]any{"limit": schInt("条数，默认 30")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Limit int `json:"limit"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Limit < 1 || p.Limit > 200 {
					p.Limit = 30
				}
				rows := []model.AuditLog{}
				if err := s.db.Order("id desc").Limit(p.Limit).Find(&rows).Error; err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(rows), "items": rows})
			},
		},
	}
}

// aiToolsDnsAccounts 证书 DNS 账号（签发证书的前置配置）。
func (s *AIService) aiToolsDnsAccounts(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_dns_accounts", Module: aiModSites, Risk: aiRiskRead,
			Desc: "列出 DNS 解析账户（ACME DNS 挑战签发证书时使用）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.certs.ListDnsAccounts()
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "create_dns_account", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "创建 DNS 解析账户（ACME DNS 挑战用；provider 支持 aliyun/dnspod/cloudflare，Secret 加密存储）。input JSON：{\"name\":\"账户名\",\"provider\":\"aliyun\",\"accessKey\":\"AK\",\"secret\":\"SK\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("账户名"), "provider": schEnum("服务商", "aliyun", "dnspod", "cloudflare"),
				"accessKey": schStr("AccessKey"), "secret": schStr("Secret"),
			}, "name", "provider", "accessKey", "secret"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[DnsAccountInput](input)
				if err != nil {
					return "", err
				}
				acc, err := s.certs.CreateDnsAccount(p)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("DNS 账户已创建: %s（ID %d）", acc.Name, acc.ID), nil
			},
		},
	}
}

// aiToolsMongoCollections Mongo 集合管理。
func (s *AIService) aiToolsMongoCollections(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "mongo_collection_action", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "Mongo 集合管理动作：create 创建 / drop 删除（数据不可恢复）/ rename 重命名（会先向用户确认）。input JSON：{\"instanceId\":1,\"database\":\"库名\",\"name\":\"集合名\",\"action\":\"create|drop|rename\",\"to\":\"rename 时的新名\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "name": schStr("集合名"),
				"action": schEnum("动作", "create", "drop", "rename"), "to": schStr("rename 的新集合名"),
			}, "instanceId", "database", "name", "action"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Name       string `json:"name"`
					Action     string `json:"action"`
					To         string `json:"to"`
				}](input)
				if err != nil {
					return "", err
				}
				switch p.Action {
				case "create", "drop", "rename":
				default:
					return "", fmt.Errorf("不支持的动作: %s", p.Action)
				}
				if err := s.adminSvc.MongoCollectionAction(ctx, p.InstanceID, "ai", p.Action, p.Database, p.Name, p.To); err != nil {
					return "", err
				}
				return fmt.Sprintf("集合 %s.%s 已执行 %s", p.Database, p.Name, p.Action), nil
			},
		},
	}
}

var _ = json.Marshal
