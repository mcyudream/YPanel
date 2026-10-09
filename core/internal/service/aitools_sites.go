// aitools_sites.go AI 工具：网站/证书/运行环境（M31）。
package service

import (
	"context"
	"fmt"
)

// aiToolsSites 网站与证书工具集。
func (s *AIService) aiToolsSites(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_sites", Module: aiModSites, Risk: aiRiskRead,
			Desc: "列出网站站点（分页/搜索）。input 可选 JSON：{\"search\":\"站点名/域名关键词\"}。返回 {total,items}，含 id/name/type/domain/enabled。",
			Parameters: schObj(map[string]any{"search": schStr("站点名/域名关键词")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.sites.List(ctx)
				if err != nil {
					return "", err
				}
				filtered := make([]map[string]any, 0, len(out))
				for _, row := range out {
					if matchAny(p.Search, fmt.Sprint(row["name"]), fmt.Sprint(row["domain"]), fmt.Sprint(row["domains"]), fmt.Sprint(row["type"])) {
						filtered = append(filtered, row)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "site_logs", Module: aiModSites, Risk: aiRiskRead,
			Desc: "查看站点访问/错误日志（尾部）。input JSON：{\"id\":站点ID,\"logType\":\"access|error\",\"tail\":200}",
			Parameters: schObj(map[string]any{
				"id": schInt("站点 ID"), "logType": schEnum("日志类型", "access", "error"), "tail": schInt("尾部行数"),
			}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID      uint   `json:"id"`
					LogType string `json:"logType"`
					Tail    string `json:"tail"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.sites.SiteLogs(ctx, p.ID, p.LogType, p.Tail)
				if err != nil {
					return "", err
				}
				return truncText(out, 6000), nil
			},
		},
		{
			Name: "create_site", Module: aiModSites, Risk: aiRiskWrite,
				Desc: "创建站点（静态/反向代理/PHP 等，按面板类型体系）。非 80/443 端口自动追加容器映射并放行防火墙（被占用时报错）。input JSON：{\"name\":\"站点名\",\"type\":\"static|proxy|php\",\"domain\":\"主域名\",\"port\":80,\"proxyPass\":\"反代目标 http://127.0.0.1:8080（proxy 类型）\",\"remark\":\"备注\"}",
				Parameters: schObj(map[string]any{
					"name": schStr("站点名"), "type": schStr("类型 static/proxy/php"), "domain": schStr("主域名"),
					"port": schInt("监听端口，默认 80；非 80/443 会自动映射容器端口并放行防火墙"), "proxyPass": schStr("proxy 类型的反代目标地址"), "remark": schStr("备注"),
				}, "name", "type", "domain"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[SiteCreateInput](input)
				if err != nil {
					return "", err
				}
				if p.Name == "" || p.Domain == "" {
					return "", fmt.Errorf("name 与 domain 必填")
				}
				site, err := s.sites.Create(ctx, p)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("站点已创建: %s（ID %d）", site.Name, site.ID), nil
			},
		},
		{
			Name: "update_site_config", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "读取并整段替换站点 nginx 配置（替换后自动 reload；改错可能导致站点不可用）。input JSON：{\"id\":站点ID,\"content\":\"完整 server 配置文本\"}。content 为空则仅返回当前配置。",
			Parameters: schObj(map[string]any{
				"id": schInt("站点 ID"), "content": schStr("完整 nginx server 配置文本；空 = 只读"),
			}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID      uint   `json:"id"`
					Content string `json:"content"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Content == "" {
					return s.sites.Config(ctx, p.ID)
				}
				if err := s.sites.UpdateConfig(ctx, p.ID, p.Content); err != nil {
					return "", err
				}
				return fmt.Sprintf("站点 %d 配置已保存并 reload", p.ID), nil
			},
		},
		{
			Name: "site_toggle", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "启用/停用站点（停用后该站点返回默认页）。input JSON：{\"id\":站点ID,\"enabled\":true|false}",
			Parameters: schObj(map[string]any{
				"id": schInt("站点 ID"), "enabled": schBool("true 启用 / false 停用"),
			}, "id", "enabled"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID      uint `json:"id"`
					Enabled bool `json:"enabled"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.sites.SetEnabled(ctx, p.ID, p.Enabled); err != nil {
					return "", err
				}
				state := "停用"
				if p.Enabled {
					state = "启用"
				}
				return "站点已" + state, nil
			},
		},
		{
			Name: "delete_site", Module: aiModSites, Risk: aiRiskDanger,
			Desc: "删除站点（可选同时删除站点目录与备份，不可恢复，会先向用户确认）。input JSON：{\"id\":站点ID,\"purgeFiles\":false,\"purgeBackups\":false}",
			Parameters: schObj(map[string]any{
				"id": schInt("站点 ID"), "purgeFiles": schBool("同时删除站点目录"), "purgeBackups": schBool("同时删除站点备份"),
			}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID           uint `json:"id"`
					PurgeFiles   bool `json:"purgeFiles"`
					PurgeBackups bool `json:"purgeBackups"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.sites.Delete(ctx, p.ID, SiteDeleteOptions{PurgeFiles: p.PurgeFiles, PurgeBackups: p.PurgeBackups}); err != nil {
					return "", err
				}
				return fmt.Sprintf("站点 %d 已删除", p.ID), nil
			},
		},
		{
			Name: "list_certs", Module: aiModSites, Risk: aiRiskRead,
			Desc: "列出证书库证书（域名/签发方/有效期/自动续签状态）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.certs.List(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "renew_cert", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "手动续签指定证书（走 acme.sh --renew --force）。input JSON：{\"id\":证书ID}",
			Parameters: schObj(map[string]any{"id": schInt("证书 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.certs.Renew(ctx, p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("证书 %d 续签完成", p.ID), nil
			},
		},
		{
			Name: "issue_cert", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "签发 HTTPS 证书（ACME DNS 挑战，自动附带泛域名；DNS 账号缺省用面板环境变量）。签发耗时 1-3 分钟。input JSON：{\"domain\":\"yudream.cn\",\"altDomains\":\"可选，逗号分隔\",\"autoRenew\":true}",
			Parameters: schObj(map[string]any{
				"domain": schStr("主域名（自动含 *.domain 泛域名）"), "altDomains": schStr("附加域名，逗号分隔"),
				"autoRenew": schBool("自动续签，默认 true"), "remark": schStr("备注"),
			}, "domain"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[CertIssueInput](input)
				if err != nil {
					return "", err
				}
				if p.Domain == "" {
					return "", fmt.Errorf("缺少 domain")
				}
				cert, err := s.certs.Issue(ctx, p)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("证书已签发入库: %s（ID %d，到期 %s）", cert.Domain, cert.ID, cert.NotAfter.Format("2006-01-02")), nil
			},
		},
		{
			Name: "create_site_backup", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "创建站点备份（站点目录 tar.gz）。input JSON：{\"siteName\":\"站点名\"}",
			Parameters: schObj(map[string]any{"siteName": schStr("站点名")}, "siteName"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					SiteName string `json:"siteName"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := NewSiteBackupService(s.nodes).Backup(ctx, p.SiteName)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "站点备份完成", "detail": out})
			},
		},
		{
			Name: "delete_cert", Module: aiModSites, Risk: aiRiskDanger,
			Desc: "从证书库删除证书文件与记录（不影响已部署站点，但无法再自动续签；会先向用户确认）。input JSON：{\"id\":证书ID}",
			Parameters: schObj(map[string]any{"id": schInt("证书 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.certs.Delete(ctx, p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("证书 %d 已删除", p.ID), nil
			},
		},
	}
}

// aiToolsRuntimes 运行环境工具集。
func (s *AIService) aiToolsRuntimes(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_runtimes", Module: aiModRuntime, Risk: aiRiskRead,
			Desc: "列出运行环境（PHP/Node/Java 等，含状态）。input 可选 JSON：{\"search\":\"名称关键词\"}",
			Parameters: schObj(map[string]any{"search": schStr("名称关键词")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.runtimes.List(ctx)
				if err != nil {
					return "", err
				}
				filtered := make([]map[string]any, 0, len(out))
				for _, row := range out {
					if matchAny(p.Search, fmt.Sprint(row["name"]), fmt.Sprint(row["type"]), fmt.Sprint(row["version"])) {
						filtered = append(filtered, row)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "runtime_operate", Module: aiModRuntime, Risk: aiRiskWrite,
			Desc: "对运行环境执行 start/stop/restart（restart 会短暂中断其上站点）。input JSON：{\"id\":运行环境ID,\"action\":\"start|stop|restart\"}",
			Parameters: schObj(map[string]any{
				"id": schInt("运行环境 ID"), "action": schEnum("操作", "start", "stop", "restart"),
			}, "id", "action"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID     uint   `json:"id"`
					Action string `json:"action"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Action != "start" && p.Action != "stop" && p.Action != "restart" {
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				if err := s.runtimes.Operate(ctx, p.ID, p.Action); err != nil {
					return "", err
				}
				return fmt.Sprintf("运行环境 %d 已执行 %s", p.ID, p.Action), nil
			},
		},
		{
			Name: "delete_runtime", Module: aiModRuntime, Risk: aiRiskDanger,
			Desc: "删除运行环境（其上站点将无法解析 PHP 等运行时，会先向用户确认）。input JSON：{\"id\":运行环境ID}",
			Parameters: schObj(map[string]any{"id": schInt("运行环境 ID")}, "id"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ID uint `json:"id"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.runtimes.Delete(ctx, p.ID); err != nil {
					return "", err
				}
				return fmt.Sprintf("运行环境 %d 已删除", p.ID), nil
			},
		},
	}
}
