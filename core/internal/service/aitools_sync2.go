// aitools_sync2.go AI 工具同步补齐（第二批）：nginx 管理/探针/Git 凭据/MCP 状态。
package service

import (
	"context"
	"fmt"
	"strings"
)

// aiToolsNginx nginx 管理面板机唯一 web 入口（安装/启停/重载）。
func (s *AIService) aiToolsNginx(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "nginx_status", Module: aiModSites, Risk: aiRiskRead,
			Desc: "查看 nginx（OpenResty）运行状态与安装模式。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				ngx, err := s.nginx.WithNode("")
				if err != nil {
					return "", err
				}
				out, err := ngx.Status(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "nginx_install", Module: aiModSites, Risk: aiRiskWrite,
			Desc: "安装 nginx（OpenResty 容器，任务化，会先向用户确认）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				ngx, err := s.nginx.WithNode("")
				if err != nil {
					return "", err
				}
				out, err := ngx.Install(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"message": "nginx 安装任务已创建", "detail": out})
			},
		},
		{
			Name: "nginx_power", Module: aiModSites, Risk: aiRiskDanger,
			Desc: "nginx 电源操作：start/stop/reload（stop 会下线全部站点，会先向用户确认）。input JSON：{\"action\":\"start|stop|reload\"}",
			Parameters: schObj(map[string]any{"action": schEnum("操作", "start", "stop", "reload")}, "action"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Action string `json:"action"`
				}](input)
				if err != nil {
					return "", err
				}
				ngx, nerr := s.nginx.WithNode("")
				if nerr != nil {
					return "", nerr
				}
				if err := ngx.Power(ctx, p.Action); err != nil {
					return "", err
				}
				return "nginx 已执行 " + p.Action, nil
			},
		},
	}
}

// aiToolsProbeGitCred 探针与 Git 凭据查询。
func (s *AIService) aiToolsProbeGitCred(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_probes", Module: aiModMonitor, Risk: aiRiskRead,
			Desc: "列出拨测探针（目标/频率/最近状态）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.probe.List()
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "list_git_credentials", Module: aiModSrcBuild, Risk: aiRiskRead,
			Desc: "列出 Git 凭据库条目（源码构建/商店 yp-git 源的私有仓库凭据；Secret 不回显）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.creds.List(ctx)
				if err != nil {
					return "", err
				}
				items := make([]map[string]any, 0, len(out))
				for _, c := range out {
					items = append(items, map[string]any{
						"id": c.ID, "name": c.Name, "type": c.Type, "host": c.Host, "username": c.Username,
					})
				}
				return toolOut(map[string]any{"total": len(items), "items": items})
			},
		},
		{
			Name: "create_git_credential", Module: aiModSrcBuild, Risk: aiRiskWrite,
			Desc: "创建 Git 凭据（源码构建/私有仓库拉取用；Secret 加密存储不回显）。input JSON：{\"name\":\"凭据名\",\"type\":\"token|ssh\",\"host\":\"github.com\",\"username\":\"可选\",\"secret\":\"token 或私钥\",\"remark\":\"备注\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("凭据名"), "type": schEnum("类型", "token", "ssh"), "host": schStr("匹配主机（如 github.com，* 兜底）"),
				"username": schStr("用户名（https 用）"), "secret": schStr("token/私钥"), "remark": schStr("备注"),
			}, "name", "type", "host", "secret"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name    string `json:"name"`
					Type    string `json:"type"`
					Host    string `json:"host"`
					Username string `json:"username"`
					Secret  string `json:"secret"`
					Remark  string `json:"remark"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Secret) == "" {
					return "", fmt.Errorf("secret 为空")
				}
				cred, err := s.creds.Create(ctx, p.Name, p.Type, p.Host, p.Username, p.Secret, p.Remark)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Git 凭据已创建: %s（ID %d）", cred.Name, cred.ID), nil
			},
		},
	}
}

// aiToolsMCP MCP 开放状态查询。
func (s *AIService) aiToolsMCP(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "mcp_status", Module: aiModPanel, Risk: aiRiskRead,
			Desc: "查看 MCP 对外开放状态（AI 注册表中开放给外部 MCP 客户端的只读工具集）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(ctx context.Context, _ string) (string, error) {
				tools := s.aiToolsAll(ctx)
				readonly := 0
				for _, d := range tools {
					if d.Risk == aiRiskRead {
						readonly++
					}
				}
				return toolOut(map[string]any{
					"note":        "MCP 对外开放复用本注册表的 read 级工具；配置入口在 智能→MCP 页",
					"totalTools":  len(tools),
					"mcpReadable": readonly,
				})
			},
		},
	}
}
