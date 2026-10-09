// aitools_dbmore.go AI 工具：数据库补全——列结构/Redis/Mongo/备份恢复（M31 二批）。
package service

import (
	"context"

	"github.com/ypanel/core/internal/dbdriver"
)

// aiToolsDBMore 数据库第二批工具（模块归属 databases，与 aiToolsDatabases 同模块聚合）。
func (s *AIService) aiToolsDBMore(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_db_columns", Module: aiModDB, Risk: aiRiskRead,
			Desc: "列出表的列结构（列名/类型/可空/键，写 SQL 前必看）。input JSON：{\"instanceId\":1,\"database\":\"库名\",\"table\":\"表名\",\"schema\":\"PG 可选\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "table": schStr("表名"), "schema": schStr("PG schema"),
			}, "instanceId", "database", "table"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Table      string `json:"table"`
					Schema     string `json:"schema"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.adminSvc.Columns(ctx, p.InstanceID, p.Database, p.Schema, p.Table)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "redis_scan_keys", Module: aiModDB, Risk: aiRiskRead,
			Desc: "浏览 Redis 键（pattern 匹配，如 app:*，返回键名与类型）。input JSON：{\"instanceId\":1,\"database\":\"0\",\"pattern\":\"*\",\"cursor\":0,\"count\":50}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("逻辑库编号，默认 0"),
				"pattern": schStr("键模式，默认 *"), "cursor": schInt("游标（上页返回，0=开头）"), "count": schInt("单页数量，默认 50"),
			}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Pattern    string `json:"pattern"`
					Cursor     uint64 `json:"cursor"`
					Count      int64  `json:"count"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Pattern == "" {
					p.Pattern = "*"
				}
				if p.Database == "" {
					p.Database = "0"
				}
				if p.Count < 1 || p.Count > 200 {
					p.Count = 50
				}
				keys, next, err := s.adminSvc.ScanKeys(ctx, p.InstanceID, p.Database, p.Pattern, p.Cursor, p.Count)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(keys), "nextCursor": next, "items": keys})
			},
		},
		{
			Name: "redis_get_key", Module: aiModDB, Risk: aiRiskRead,
			Desc: "读取 Redis 键的值与元信息（类型/TTL/内容，hash/list/set/zset/stream 全支持）。input JSON：{\"instanceId\":1,\"database\":\"0\",\"name\":\"key\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("逻辑库编号"), "name": schStr("键名"),
			}, "instanceId", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Name       string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.adminSvc.KeyDetail(ctx, p.InstanceID, p.Database, p.Name)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "redis_write_key", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "写入 Redis 键（新建/覆盖）。input JSON：{\"instanceId\":1,\"database\":\"0\",\"name\":\"key\",\"type\":\"string|hash|list|set|zset\",\"value\":值(hash 传对象/list·set 传数组/zset 传 [{member,score}])}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("逻辑库编号"),
				"name": schStr("键名"), "type": schEnum("键类型", "string", "hash", "list", "set", "zset"),
				"value": map[string]any{"description": "键值（类型对应结构）"},
			}, "instanceId", "name", "type", "value"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Name       string `json:"name"`
					Type       string `json:"type"`
					Value      any    `json:"value"`
				}](input)
				if err != nil {
					return "", err
				}
				req := dbdriver.RedisWrite{DB: p.Database, Name: p.Name, Type: p.Type, Value: p.Value}
				if err := s.adminSvc.WriteRedisKey(ctx, p.InstanceID, "ai", req); err != nil {
					return "", err
				}
				return "已写入键: " + p.Name, nil
			},
		},
		{
			Name: "redis_delete_key", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "删除 Redis 键（不可恢复，会先向用户确认）。input JSON：{\"instanceId\":1,\"database\":\"0\",\"name\":\"key\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("逻辑库编号"), "name": schStr("键名"),
			}, "instanceId", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Name       string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.adminSvc.DeleteRedisKey(ctx, p.InstanceID, "ai", p.Database, p.Name); err != nil {
					return "", err
				}
				return "已删除键: " + p.Name, nil
			},
		},
		{
			Name: "redis_exec", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "对 Redis 实例执行任意命令（TTL/EXPIRE/DEL/CONFIG 等面板工具未覆盖的操作；写命令会先向用户确认）。input JSON：{\"instanceId\":1,\"database\":\"0\",\"command\":\"TTL app:token\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("逻辑库编号"), "command": schStr("redis 命令"),
			}, "instanceId", "command"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Command    string `json:"command"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.adminSvc.RedisExec(ctx, p.InstanceID, "ai", p.Database, p.Command)
				if err != nil {
					return "", err
				}
				return out, nil
			},
		},
		{
			Name: "mongo_find_docs", Module: aiModDB, Risk: aiRiskRead,
			Desc: "查询 MongoDB 集合文档（filter 为 ExtJSON 如 {\"status\":\"ok\"}）。input JSON：{\"instanceId\":1,\"database\":\"库名\",\"collection\":\"集合\",\"filter\":{},\"sort\":\"可选字段\",\"sortDir\":\"asc|desc\",\"skip\":0,\"limit\":20}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "collection": schStr("集合名"),
				"filter": schObj(map[string]any{}, "查询条件 ExtJSON"), "sort": schStr("排序字段"),
				"sortDir": schEnum("排序方向", "asc", "desc"), "skip": schInt("跳过数"), "limit": schInt("条数上限，默认 20"),
			}, "instanceId", "database", "collection"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Collection string `json:"collection"`
					Filter     string `json:"filter"`
					Sort       string `json:"sort"`
					SortDir    string `json:"sortDir"`
					Skip       int    `json:"skip"`
					Limit      int    `json:"limit"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Limit < 1 || p.Limit > 50 {
					p.Limit = 20
				}
				out, err := s.adminSvc.MongoFindDocs(ctx, p.InstanceID, p.Database, p.Collection, p.Filter, "", p.Sort, p.SortDir, p.Skip, p.Limit)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "mongo_insert_doc", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "插入 MongoDB 文档（doc 为 JSON 对象文本）。input JSON：{\"instanceId\":1,\"database\":\"库\",\"collection\":\"集合\",\"doc\":\"{\\\"k\\\":\\\"v\\\"}\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "collection": schStr("集合名"),
				"doc": schStr("文档 JSON 字符串"),
			}, "instanceId", "database", "collection", "doc"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Collection string `json:"collection"`
					Doc        string `json:"doc"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.adminSvc.MongoInsertDoc(ctx, p.InstanceID, "ai", p.Database, p.Collection, p.Doc); err != nil {
					return "", err
				}
				return "文档已插入 " + p.Database + "/" + p.Collection, nil
			},
		},
		{
			Name: "mongo_update_doc", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "替换 MongoDB 文档（按 id 定位整文档替换，会先向用户确认）。input JSON：{\"instanceId\":1,\"database\":\"库\",\"collection\":\"集合\",\"id\":\"{\\\"$oid\\\":\\\"...\\\"}\",\"doc\":\"{...完整新文档...}\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "collection": schStr("集合名"),
				"id": schStr("文档 _id ExtJSON"), "doc": schStr("完整新文档 JSON"),
			}, "instanceId", "database", "collection", "id", "doc"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Collection string `json:"collection"`
					ID         string `json:"id"`
					Doc        string `json:"doc"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.adminSvc.MongoUpdateDoc(ctx, p.InstanceID, "ai", p.Database, p.Collection, p.ID, p.Doc); err != nil {
					return "", err
				}
				return "文档已更新", nil
			},
		},
		{
			Name: "mongo_delete_doc", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "删除 MongoDB 文档（不可恢复，会先向用户确认）。input JSON：{\"instanceId\":1,\"database\":\"库\",\"collection\":\"集合\",\"id\":\"{\\\"$oid\\\":\\\"...\\\"}\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "collection": schStr("集合名"),
				"id": schStr("文档 _id ExtJSON"),
			}, "instanceId", "database", "collection", "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Collection string `json:"collection"`
					ID         string `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.adminSvc.MongoDeleteDoc(ctx, p.InstanceID, "ai", p.Database, p.Collection, p.ID); err != nil {
					return "", err
				}
				return "文档已删除", nil
			},
		},
		{
			Name: "create_db_backup", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "创建数据库实例备份（mysqldump/pg_dump/redis·mongo 导出，任务化）。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.CreateBackup(ctx, p.InstanceID)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "备份完成", "detail": out})
			},
		},
		{
			Name: "list_db_backups", Module: aiModDB, Risk: aiRiskRead,
			Desc: "列出数据库实例的备份文件。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.Backups(ctx, p.InstanceID)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "reveal_db_credentials", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "查看数据库实例的连接凭据（含 root 密码；敏感操作会弹窗向用户确认）。用途：复用已有实例部署应用时，获取连接信息填入应用参数。结果中的密码在界面与审计中自动打码；严禁把密码写入记忆或复述给无关上下文。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.ConnectInfo(ctx, p.InstanceID)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{
					"name": out.RootUser, "rootUser": out.RootUser, "rootPass": out.RootPass,
					"container": out.Container, "innerPort": out.InnerPort, "networks": out.Networks,
					"lanIp": out.LanIP, "mapPort": out.MapPort,
					"hint": "同网络容器用 container:innerPort；外部/跨网络用 lanIp:mapPort（root 账号见 rootUser/rootPass）",
				})
			},
		},
		{
			Name: "restore_db_backup", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "从备份文件恢复数据库（覆盖当前数据，不可逆，会先向用户确认）。input JSON：{\"instanceId\":1,\"file\":\"备份文件名\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "file": schStr("备份文件名（list_db_backups 可查）"),
			}, "instanceId", "file"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					File       string `json:"file"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.Restore(ctx, p.InstanceID, p.File); err != nil {
					return "", err
				}
				return "备份恢复已触发: " + p.File, nil
			},
		},
	}
}
