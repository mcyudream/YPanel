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
			Desc: "安装商店应用（异步任务：拉镜像/起容器，会先向用户确认参数）。安装前先用 list_database_instances/list_installed_apps 检查已有服务：应用需要的数据库已存在时告知用户选择复用还是新建，存在多个同类实例时列出候选交由用户选择；复用时经 reveal_db_credentials（需用户确认）获取连接信息填入 params（host/port/user/password），并先 create_database/create_db_user 准备库账号。input JSON：{\"sourceId\":1,\"key\":\"wordpress\",\"name\":\"app-blog\",\"params\":{\"PANEL_DB_ROOT_PASSWORD\":\"可选覆盖\"},\"version\":\"可选\",\"domain\":\"可选反代域名\"}。name 规则：小写字母/数字/中划线（如 app-blog）。params 按 store_app_detail 的 formFields envKey 传。注意：密码类参数（type=password，envKey 含 PASSWORD）不要传空串——若包未声明 random，必须自行生成 16 位以上随机强密码填入，并在交付说明中把密码告知用户；声明 random 的可省略由系统生成。用户选择复用已有实例时传 externalDB：{\"instanceId\":选中实例ID,\"database\":\"库名可省略\",\"user\":\"账号可省略\",\"createIfMissing\":true}，系统自动建库建号并注入连接参数（凭据不经过对话）。返回任务 ID，用 get_task 跟踪。",
			Parameters: schObj(map[string]any{
				"sourceId": schInt("应用来源 ID"), "key": schStr("应用 key"),
				"name": schStr("应用实例名（小写字母/数字/中划线，如 app-blog）"),
				"version": schStr("版本，空=最新"), "params": schObj(map[string]any{}, "安装参数（envKey→值）；密码字段必须传自生成的强密码，不要传空串"),
				"domain": schStr("可选：安装后一键反代的域名"),
			}, "sourceId", "key", "name"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[StoreInstallInput](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Name) == "" {
					return "", fmt.Errorf("缺少 name")
				}
				out, err := s.store.Install(ctx, p)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "安装任务已创建", "task": out})
			},
		},
		{
			Name: "list_installed_apps", Module: aiModStore, Risk: aiRiskRead,
			Desc: "列出已安装的应用（状态/端口/版本/安装参数摘要/可升级）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.store.InstalledDetailed(ctx)
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
				"project": schStr("应用项目名（app- 开头，list_installed_apps 可查）"),
				"purgeData": schBool("同时删除应用数据"), "removeImage": schBool("同时删除镜像"), "cascadeDB": schBool("级联移除数据库纳管记录"),
			}, "project"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
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
					PurgeData: p.PurgeData, RemoveImage: p.RemoveImage, CascadeDB: p.CascadeDB,
				})
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "卸载任务已创建", "task": out})
			},
		},
	}
}
