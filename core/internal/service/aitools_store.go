// aitools_store.go AI 工具：应用商店场景（检索/详情/安装/卸载/已装清单）（M31 二批）。
package service

import (
	"context"
	"fmt"
	"strings"
)

// aiToolsStore 应用商店工具集（场景：用户说「我想要个博客」，AI 检索→看参数→安装→验证）。
func (s *AIService) aiToolsStore(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "store_search_apps", Module: aiModStore, Risk: aiRiskRead,
			Desc: "按需求关键词检索应用商店（如「博客」「图床」「网盘」「论坛」「监控」）。input JSON：{\"search\":\"关键词\",\"tag\":\"可选分类\",\"status\":\"all|installed|notInstalled|upgradable\",\"page\":1,\"pageSize\":20}。返回 {total,items}（含 name/key/sourceId/description/installed/upgradable）。",
			Parameters: schObj(map[string]any{
				"search": schStr("需求关键词"), "tag": schStr("分类标签"),
				"status": schEnum("安装状态过滤", "all", "installed", "notInstalled", "upgradable"),
				"page":   schInt("页码"), "pageSize": schInt("每页条数"),
			}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search   string `json:"search"`
					Tag      string `json:"tag"`
					Status   string `json:"status"`
					Page     int    `json:"page"`
					PageSize int    `json:"pageSize"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.PageSize < 1 || p.PageSize > 50 {
					p.PageSize = 20
				}
				if p.Page < 1 {
					p.Page = 1
				}
				out, err := s.store.Apps(StoreListQuery{
					Search: p.Search, Tag: p.Tag, Status: p.Status,
					Page: p.Page, PageSize: p.PageSize, OrderBy: "name", Order: "asc",
				})
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "store_app_detail", Module: aiModStore, Risk: aiRiskRead,
			Desc: "查看应用详情与安装参数定义（formFields：哪些参数必填/默认值/随机密码项），安装前必看。input JSON：{\"sourceId\":1,\"key\":\"wordpress\"}",
			Parameters: schObj(map[string]any{
				"sourceId": schInt("应用来源 ID（store_search_apps 返回）"), "key": schStr("应用 key"),
			}, "sourceId", "key"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					SourceID uint   `json:"sourceId"`
					Key      string `json:"key"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.store.App(p.SourceID, p.Key)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "store_install_app", Module: aiModStore, Risk: aiRiskWrite,
			Desc: "安装或重新部署商店应用，返回异步任务 ID。先查看 store_app_detail 的 formFields、middleware、optionalServices，再确认实例名与目标节点。name 为实例名（如 yudream-admin-prod），项目名前缀 app- 由系统添加。params 按 envKey 填写，random 字段可省略由系统生成，base64-32 会生成标准 Base64 的 32 字节密钥。同名部署沿用旧凭据和数据；升级既有应用优先使用 store_upgrade_app。externalDB 选择纳管实例后自动注入凭据，无需在对话中获取密码。固定镜像参数由所选商店版本管理。",
			Parameters: schObj(map[string]any{
				"node":     schStr("目标节点 ID，默认 local；不要传节点名称"),
				"sourceId": schInt("应用来源 ID"), "key": schStr("应用 key"),
				"name":           schStr("实例名，小写字母/数字/中划线，例如 yudream-admin-prod"),
				"version":        schStr("商店版本，空=最新"),
				"params":         map[string]any{"type": "object", "description": "envKey 到安装值，随机凭据可省略", "additionalProperties": map[string]any{"type": "string"}},
				"domain":         schStr("安装后一键反代的域名"),
				"network":        schStr("容器网络，默认 ypanel_default；host 为宿主网络"),
				"createNetwork":  schBool("不存在时创建所选网络"),
				"timezone":       schStr("时区，例如 Asia/Shanghai"),
				"extraHosts":     map[string]any{"type": "array", "items": schStr("域名:IP 或 host.docker.internal:host-gateway")},
				"mountHostsFile": schBool("只读挂载宿主 /etc/hosts"),
				"externalDB": schObj(map[string]any{
					"instanceId": schInt("SQL/Mongo 纳管实例 ID"), "redisInstanceId": schInt("Redis 实例 ID"),
					"neo4jInstanceId": schInt("Neo4j 实例 ID"), "s3InstanceId": schInt("S3 实例 ID"),
					"database": schStr("数据库/桶名称"), "user": schStr("应用数据库账号"), "createIfMissing": schBool("自动准备库与账号"),
					"middlewareInstanceIds": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "integer"}},
					"optionalServices":      map[string]any{"type": "array", "items": schStr("可选伴生服务 ID")},
				}),
			}, "sourceId", "key", "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					StoreInstallInput
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Name) == "" {
					return "", fmt.Errorf("缺少 name")
				}
				if p.Node != "" {
					if p.NodeID != "" && p.NodeID != p.Node {
						return "", fmt.Errorf("node 与 nodeId 不一致")
					}
					p.NodeID = p.Node
				}
				out, err := s.store.Install(ctx, p.StoreInstallInput)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "安装任务已创建", "task": out})
			},
		},
		{
			Name: "store_upgrade_app", Module: aiModStore, Risk: aiRiskWrite,
			Desc:       "把已安装应用升级到商店最新版本包。会更新模板和版本包指定的镜像引用，保留真实参数、安装选项与持久化数据；配置失败会尝试恢复原部署，应用自身的数据库迁移无法自动撤销。先向用户确认目标实例和备份；先查看 list_installed_apps。与镜像自动更新互斥。返回 task，用 get_task 跟踪；仅下载镜像用镜像工具，沿用当前标签更新运行服务用 pull_compose_update。",
			Parameters: schObj(map[string]any{"project": schStr("Compose 项目名"), "node": schStr("目标节点 ID，默认 local")}, "project"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Node    string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.store.UpgradeApp(ctx, p.Project, p.Node)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "list_installed_apps", Module: aiModStore, Risk: aiRiskRead,
			Desc:       "列出已安装的应用（状态/端口/版本/安装参数摘要/可升级）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.store.InstalledDetailed(ctx, "")
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "uninstall_app", Module: aiModStore, Risk: aiRiskDanger,
			Desc: "卸载商店应用（compose down 停容器；数据/镜像/数据库纳管按选项删除，默认保留数据，会先向用户确认）。input JSON：{\"project\":\"app-blog\",\"purgeData\":false,\"removeImage\":false,\"cascadeDB\":false}",
			Parameters: schObj(map[string]any{
				"node":      schStr("目标节点 ID，默认 local"),
				"project":   schStr("应用项目名（app- 开头，list_installed_apps 可查）"),
				"purgeData": schBool("同时删除应用数据"), "removeImage": schBool("同时删除镜像"), "cascadeDB": schBool("级联移除数据库纳管记录"),
			}, "project"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node        string `json:"node"`
					Project     string `json:"project"`
					PurgeData   bool   `json:"purgeData"`
					RemoveImage bool   `json:"removeImage"`
					CascadeDB   bool   `json:"cascadeDB"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Project) == "" {
					return "", fmt.Errorf("缺少 project")
				}
				out, err := s.store.Uninstall(ctx, p.Project, StoreUninstallOptions{
					NodeID: p.Node, PurgeData: p.PurgeData, RemoveImage: p.RemoveImage, CascadeDB: p.CascadeDB,
				})
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "卸载任务已创建", "task": out})
			},
		},
	}
}
