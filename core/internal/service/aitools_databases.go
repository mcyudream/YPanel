// aitools_databases.go AI 工具：数据库管理（M31，行级编辑复用 DB Admin 通道自带审计）。
package service

import (
	"context"
	"fmt"

	"github.com/ypanel/core/internal/dbdriver"
)

// aiToolsDatabases 数据库工具集。
func (s *AIService) aiToolsDatabases(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_database_instances", Module: aiModDB, Risk: aiRiskRead,
			Desc: "列出面板管理的数据库实例（MySQL/PG/Redis/Mongo，分页/搜索）。input 可选 JSON：{\"search\":\"名称/类型/备注关键词\"}。返回 {total,items}；实例 id 用于本模块其他工具。",
			Parameters: schObj(map[string]any{"search": schStr("名称/类型/备注关键词")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.List(ctx)
				if err != nil {
					return "", err
				}
				filtered := make([]map[string]any, 0, len(out))
				for _, row := range out {
					if matchAny(p.Search, fmt.Sprint(row["name"]), fmt.Sprint(row["type"]), fmt.Sprint(row["remark"]), fmt.Sprint(row["host"])) {
						filtered = append(filtered, row)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "list_databases", Module: aiModDB, Risk: aiRiskRead,
			Desc: "列出实例下的数据库/键空间。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.Databases(ctx, p.InstanceID)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "list_tables", Module: aiModDB, Risk: aiRiskRead,
			Desc: "列出数据库中的表（含行数；PG 传 schema）。input JSON：{\"instanceId\":1,\"database\":\"库名\",\"schema\":\"PG 可选\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "schema": schStr("PG schema，默认 public"),
			}, "instanceId", "database"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					Schema     string `json:"schema"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.adminSvc.Tables(ctx, p.InstanceID, p.Database, p.Schema)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "query_database", Module: aiModDB, Risk: aiRiskRead,
			Desc: "对实例执行只读 SQL（仅 SELECT/SHOW/DESC/EXPLAIN 白名单，最多返回 40 行）。Redis 用 key 模式浏览不适用本工具。input JSON：{\"instanceId\":1,\"database\":\"库名\",\"sql\":\"SELECT ...\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "database": schStr("库名"), "sql": schStr("只读 SQL"),
			}, "instanceId", "database", "sql"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Database   string `json:"database"`
					SQL        string `json:"sql"`
				}](input)
				if err != nil {
					return "", err
				}
				return s.aiQueryDatabase(ctx, p.InstanceID, p.Database, p.SQL)
			},
		},
		{
			Name: "list_db_users", Module: aiModDB, Risk: aiRiskRead,
			Desc: "列出实例的数据库账号。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.dbSvc.Users(ctx, p.InstanceID)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "create_database", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "在实例中创建数据库。input JSON：{\"instanceId\":1,\"name\":\"新库名\",\"charset\":\"utf8mb4 可选\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "name": schStr("新库名"), "charset": schStr("字符集，MySQL 建议 utf8mb4"),
			}, "instanceId", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Name       string `json:"name"`
					Charset    string `json:"charset"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.CreateDatabase(ctx, p.InstanceID, p.Name, p.Charset); err != nil {
					return "", err
				}
				return "已创建数据库: " + p.Name, nil
			},
		},
		{
			Name: "create_db_user", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "创建数据库账号（密码走 AES 加密存储不会回显；host 为 MySQL 允许来源，PG 留空）。input JSON：{\"instanceId\":1,\"name\":\"用户名\",\"host\":\"%\",\"password\":\"强密码\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "name": schStr("用户名"),
				"host": schStr("允许来源主机（MySQL，如 % 或 10.0.0.%）"), "password": schStr("强密码"),
			}, "instanceId", "name", "password"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Name       string `json:"name"`
					Host       string `json:"host"`
					Password   string `json:"password"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.CreateUser(ctx, p.InstanceID, p.Name, p.Host, p.Password); err != nil {
					return "", err
				}
				return "已创建用户: " + p.Name, nil
			},
		},
		{
			Name: "row_insert", Module: aiModDB, Risk: aiRiskWrite,
			Desc: "向表插入一行（值全部参数化，走 DB Admin 通道有审计）。input JSON：{\"instanceId\":1,\"db\":\"库名\",\"table\":\"表名\",\"values\":{\"列\":\"值\",...}}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "db": schStr("库名"), "table": schStr("表名"),
				"values": schObj(map[string]any{}, "列名与值键值对"),
			}, "instanceId", "db", "table", "values"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint          `json:"instanceId"`
					DB         string        `json:"db"`
					Table      string        `json:"table"`
					Values     map[string]any `json:"values"`
				}](input)
				if err != nil {
					return "", err
				}
				req := dbdriver.RowEdit{DB: p.DB, Table: p.Table, Values: p.Values}
				if err := s.adminSvc.InsertRow(ctx, p.InstanceID, "ai", req); err != nil {
					return "", err
				}
				return "已插入一行到 " + p.DB + "." + p.Table, nil
			},
		},
		{
			Name: "row_update", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "按主键更新表行（影响运行数据，会先向用户确认）。input JSON：{\"instanceId\":1,\"db\":\"库名\",\"table\":\"表名\",\"pk\":{\"id\":3},\"values\":{\"列\":\"新值\"}}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "db": schStr("库名"), "table": schStr("表名"),
				"pk": schObj(map[string]any{}, "主键列与值"), "values": schObj(map[string]any{}, "待更新列与值"),
			}, "instanceId", "db", "table", "pk", "values"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint           `json:"instanceId"`
					DB         string         `json:"db"`
					Table      string         `json:"table"`
					PK         map[string]any `json:"pk"`
					Values     map[string]any `json:"values"`
				}](input)
				if err != nil {
					return "", err
				}
				if len(p.PK) == 0 {
					return "", fmt.Errorf("缺少主键定位 pk")
				}
				req := dbdriver.RowEdit{DB: p.DB, Table: p.Table, PK: p.PK, Values: p.Values}
				if err := s.adminSvc.UpdateRow(ctx, p.InstanceID, "ai", req); err != nil {
					return "", err
				}
				return fmt.Sprintf("已更新 %s.%s 主键 %v", p.DB, p.Table, p.PK), nil
			},
		},
		{
			Name: "row_delete", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "按主键删除表行（不可恢复，会先向用户确认）。input JSON：{\"instanceId\":1,\"db\":\"库名\",\"table\":\"表名\",\"pk\":{\"id\":3}}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "db": schStr("库名"), "table": schStr("表名"),
				"pk": schObj(map[string]any{}, "主键列与值"),
			}, "instanceId", "db", "table", "pk"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint           `json:"instanceId"`
					DB         string         `json:"db"`
					Table      string         `json:"table"`
					PK         map[string]any `json:"pk"`
				}](input)
				if err != nil {
					return "", err
				}
				if len(p.PK) == 0 {
					return "", fmt.Errorf("缺少主键定位 pk")
				}
				req := dbdriver.RowEdit{DB: p.DB, Table: p.Table, PK: p.PK}
				if err := s.adminSvc.DeleteRow(ctx, p.InstanceID, "ai", req); err != nil {
					return "", err
				}
				return fmt.Sprintf("已删除 %s.%s 主键 %v", p.DB, p.Table, p.PK), nil
			},
		},
		{
			Name: "drop_database", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "删除实例中的数据库（全部数据不可恢复，会先向用户确认）。input JSON：{\"instanceId\":1,\"name\":\"库名\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "name": schStr("要删除的库名"),
			}, "instanceId", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Name       string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.DropDatabase(ctx, p.InstanceID, p.Name); err != nil {
					return "", err
				}
				return "已删除数据库: " + p.Name, nil
			},
		},
		{
			Name: "drop_db_user", Module: aiModDB, Risk: aiRiskDanger,
			Desc: "删除数据库账号（依赖该账号的服务会断连，会先向用户确认）。input JSON：{\"instanceId\":1,\"name\":\"用户名\",\"host\":\"%\"}",
			Parameters: schObj(map[string]any{
				"instanceId": schInt("实例 ID"), "name": schStr("用户名"), "host": schStr("MySQL 来源主机"),
			}, "instanceId", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint   `json:"instanceId"`
					Name       string `json:"name"`
					Host       string `json:"host"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.dbSvc.DropUser(ctx, p.InstanceID, p.Name, p.Host); err != nil {
					return "", err
				}
				return "已删除用户: " + p.Name, nil
			},
		},
	}
}
