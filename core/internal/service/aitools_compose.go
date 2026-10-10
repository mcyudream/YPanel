// aitools_compose.go AI 工具：编排（compose）项目（M31）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ypanel/shared/dto"
)

// aiToolsCompose 编排应用工具集。
func (s *AIService) aiToolsCompose(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_compose_projects", Module: aiModCompose, Risk: aiRiskRead,
			Desc: "列出 compose 编排项目（含运行状态与服务清单）。input 可选 JSON：{\"search\":\"项目名关键词\"}。返回 items 含 name/dir/managed(托管可编辑)/running/total/services。",
			Parameters: schObj(map[string]any{"search": schStr("项目名关键词")}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				out, err := agentGetJSON[[]dto.ComposeProject](s, ctx, "/agent/v1/compose/projects")
				if err != nil {
					return "", err
				}
				items := []dto.ComposeProject{}
				if out != nil {
					items = *out
				}
				filtered := make([]dto.ComposeProject, 0, len(items))
				for _, pr := range items {
					if matchAny(p.Search, pr.Name) {
						filtered = append(filtered, pr)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "compose_logs", Module: aiModCompose, Risk: aiRiskRead,
			Desc: "查看 compose 项目日志（尾部）。input JSON：{\"name\":\"项目名\",\"service\":\"可选服务名\",\"tail\":300}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("项目名"), "service": schStr("服务名，可省略"), "tail": schInt("尾部行数，默认 300"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name    string `json:"name"`
					Service string `json:"service"`
					Tail    int    `json:"tail"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				if p.Tail < 1 {
					p.Tail = 300
				}
				if p.Tail > 2000 {
					p.Tail = 2000
				}
				return s.agentText(ctx, fmt.Sprintf("/agent/v1/compose/logs?name=%s&service=%s&tail=%d",
					url.QueryEscape(p.Name), url.QueryEscape(p.Service), p.Tail))
			},
		},
		{
			Name: "get_compose_config", Module: aiModCompose, Risk: aiRiskRead,
			Desc: "读取 compose 项目 docker-compose.yml 内容（托管项目可编辑，返回内容可直接作为 save_compose_config 的入参）。input JSON：{\"name\":\"项目名\",\"dir\":\"外部项目目录，可省略\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("项目名"), "dir": schStr("外部项目工作目录，托管项目省略"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
					Dir  string `json:"dir"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				out, err := agentGetJSON[dto.ComposeConfigResp](s, ctx, "/agent/v1/compose/config?name="+url.QueryEscape(p.Name)+"&dir="+url.QueryEscape(p.Dir))
				if err != nil {
					return "", err
				}
				// content 直接可作 save_compose_config 的入参
				return toolOut(map[string]any{"name": out.Name, "dir": out.Dir, "file": out.File, "content": out.Content})
			},
		},
		{
			Name: "save_compose_config", Module: aiModCompose, Risk: aiRiskWrite,
			Desc: "保存托管 compose 项目的 docker-compose.yml（仅托管项目可写；保存后建议 compose_up 生效）。input JSON：{\"name\":\"项目名\",\"content\":\"完整 yaml 文本\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("托管项目名"), "content": schStr("完整 docker-compose.yml 文本"),
			}, "name", "content"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					dto.ComposeWriteReq
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if _, err := agentPostJSON[dto.ComposeWriteReq, any](s, ctx, "/agent/v1/compose/config", &p.ComposeWriteReq); err != nil {
					return "", err
				}
				return "配置已保存: " + p.Name, nil
			},
		},
		{
			Name: "compose_up", Module: aiModCompose, Risk: aiRiskWrite,
			Desc: "上线/应用 compose 项目（up -d，创建缺失容器并应用变更）。input JSON：{\"name\":\"项目名\",\"dir\":\"外部项目目录可省略\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("项目名"), "dir": schStr("外部项目工作目录，托管项目省略"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					dto.ComposeActionReq
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				out, err := agentPostJSON[dto.ComposeActionReq, map[string]string](s, ctx, "/agent/v1/compose/up", &p.ComposeActionReq)
				if err != nil {
					return "", err
				}
				return "compose up 完成\n" + truncText((*out)["output"], 1500), nil
			},
		},
		{
			Name: "compose_service_action", Module: aiModCompose, Risk: aiRiskWrite,
			Desc: "对 compose 项目内单个服务执行 start/stop/restart。input JSON：{\"name\":\"项目名\",\"service\":\"服务名\",\"action\":\"start|stop|restart\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("项目名"), "service": schStr("服务名"), "action": schEnum("操作", "start", "stop", "restart"),
			}, "name", "service", "action"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name    string `json:"name"`
					Service string `json:"service"`
					Action  string `json:"action"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Action != "start" && p.Action != "stop" && p.Action != "restart" {
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				path := fmt.Sprintf("/agent/v1/compose/service-action?project=%s&service=%s&action=%s",
					url.QueryEscape(p.Name), url.QueryEscape(p.Service), p.Action)
				if _, err := agentPostJSON[struct{}, any](s, ctx, path, nil); err != nil {
					return "", err
				}
				return fmt.Sprintf("已对 %s/%s 执行 %s", p.Name, p.Service, p.Action), nil
			},
		},
		{
			Name: "compose_down", Module: aiModCompose, Risk: aiRiskDanger,
			Desc: "下线 compose 项目（停止并移除其全部容器，数据卷保留；会先向用户确认）。input JSON：{\"name\":\"项目名\",\"dir\":\"外部项目目录可省略\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("项目名"), "dir": schStr("外部项目工作目录，托管项目省略"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					dto.ComposeActionReq
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				out, err := agentPostJSON[dto.ComposeActionReq, map[string]string](s, ctx, "/agent/v1/compose/down", &p.ComposeActionReq)
				if err != nil {
					return "", err
				}
				return "compose down 完成\n" + truncText((*out)["output"], 1000), nil
			},
		},
		{
			Name: "delete_compose_project", Module: aiModCompose, Risk: aiRiskDanger,
			Desc: "删除托管 compose 项目（移除容器并删除项目目录内应用包文件；data 数据卷目录保留，会先向用户确认）。input JSON：{\"name\":\"项目名\"}",
			Parameters: schObj(map[string]any{"name": schStr("托管项目名")}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if _, err := s.agentDeleteJSON(ctx, "/agent/v1/compose/projects/"+url.PathEscape(p.Name)); err != nil {
					return "", err
				}
				return "已删除编排项目: " + p.Name, nil
			},
		},
	}
}

var _ = json.Marshal
