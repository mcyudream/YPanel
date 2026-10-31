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
			Name: "get_compose_auto_update", Module: aiModCompose, Risk: aiRiskRead,
			Desc:       "读取编排镜像自动更新配置及最近检查结果。node 为节点 ID，默认 local；project 为项目名。返回 enabled、intervalMin、前后脚本、hookFailurePolicy(stop/continue)、最近任务 ID 与结果。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID，默认 local"), "project": schStr("编排项目名")}, "project"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node    string `json:"node"`
					Project string `json:"project"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.src2.GetAutoUpdate(ctx, p.Node, p.Project)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "configure_compose_auto_update", Module: aiModCompose, Risk: aiRiskWrite,
			Desc: "保存编排镜像自动更新配置（完整替换配置，先 get_compose_auto_update 读取并保留现有设置）。intervalMin 为 5–1440 分钟整数，常用 60/360/1440；preHook/postHook 为目标节点 shell 脚本，仅发现更新时执行，每个最多 5 分钟。hookFailurePolicy 默认 stop：前脚本失败不重建，后脚本失败报告错误但不回滚已完成更新；continue 记录错误后继续。仅更新当前镜像引用且跳过停止的服务；不下载商店包、不自动跨版本。发现未部署的编排配置变化时中止，先明确部署配置或使用 store_upgrade_app。",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID，默认 local"), "project": schStr("编排项目名"), "dir": schStr("项目工作目录，可省略"),
				"enabled": schBool("是否启用"), "intervalMin": schInt("检查间隔，5–1440 分钟"),
				"preHook": schStr("更新前脚本，可选，最多 8192 字符"), "postHook": schStr("更新后脚本，可选，最多 8192 字符"),
				"hookFailurePolicy": schEnum("脚本失败策略，默认 stop", "stop", "continue"), "remark": schStr("备注，最多 255 字符"),
			}, "project", "enabled", "intervalMin"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					dto.ComposeAutoUpdateSaveReq
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				p.NodeID = p.Node
				if err := s.src2.SaveAutoUpdate(ctx, p.ComposeAutoUpdateSaveReq); err != nil {
					return "", err
				}
				return "自动更新配置已保存", nil
			},
		},
		{
			Name: "pull_compose_update", Module: aiModCompose, Risk: aiRiskWrite,
			Desc:       "为编排项目创建一次镜像更新任务：拉取当前镜像引用，只重建镜像变化的运行服务，拉取失败中止。不会下载商店版本包或切换固定标签；编排配置与运行容器不一致时中止，先部署配置或使用 store_upgrade_app。node 默认 local；project 项目名必填；dir 外部项目工作目录可选。返回 taskId，用 get_task 查看进度和结果。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID，默认 local"), "project": schStr("编排项目名"), "dir": schStr("项目工作目录，可选")}, "project"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node    string `json:"node"`
					Project string `json:"project"`
					Dir     string `json:"dir"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.src2.PullUpdateTask(ctx, dto.ComposePullUpdateReq{NodeID: p.Node, Project: p.Project, Dir: p.Dir})
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "list_compose_projects", Module: aiModCompose, Risk: aiRiskRead,
			Desc:       "列出 compose 编排项目（含运行状态与服务清单）。input 可选 JSON：{\"search\":\"项目名关键词\"}。返回 items 含 name/dir/managed(托管可编辑)/running/total/services。",
			Parameters: schObj(map[string]any{"search": schStr("项目名关键词")}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node   string `json:"node"`
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
				"node": schStr("目标节点 ID（list_nodes 返回数组项的 id；local=本机；不要传节点名）"),
				"name": schStr("项目名"), "service": schStr("服务名，可省略"), "tail": schInt("尾部行数，默认 300"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node    string `json:"node"`
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
				"node": schStr("目标节点 ID（list_nodes 返回数组项的 id；local=本机；不要传节点名）"),
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
				"node": schStr("目标节点 ID（list_nodes 返回数组项的 id；local=本机；不要传节点名）"),
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
				"node": schStr("目标节点 ID（list_nodes 返回数组项的 id；local=本机；不要传节点名）"),
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
			Desc: "对 compose 项目内单个服务执行 start/stop/restart。project 为项目名（兼容旧 name），外部编排可传 dir 工作目录，node 默认 local。input JSON：{\"project\":\"项目名\",\"service\":\"服务名\",\"action\":\"start|stop|restart\",\"dir\":\"外部项目目录可省略\"}",
			Parameters: schObj(map[string]any{
				"node":    schStr("目标节点 ID（list_nodes 返回数组项的 id；local=本机；不要传节点名）"),
				"project": schStr("项目名"), "service": schStr("服务名"), "action": schEnum("操作", "start", "stop", "restart"),
				"dir": schStr("外部编排工作目录，托管项目省略"),
			}, "project", "service", "action"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					dto.ComposeServiceActionReq
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Action != "start" && p.Action != "stop" && p.Action != "restart" {
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				if _, err := RunComposeServiceAction(ctx, s.nodes, p.Node, p.ComposeServiceActionReq); err != nil {
					return "", err
				}
				project := p.Project
				if project == "" {
					project = p.Name
				}
				return fmt.Sprintf("已对 %s/%s 执行 %s", project, p.Service, p.Action), nil
			},
		},
		{
			Name: "compose_down", Module: aiModCompose, Risk: aiRiskDanger,
			Desc: "下线 compose 项目（停止并移除其全部容器，数据卷保留；会先向用户确认）。input JSON：{\"name\":\"项目名\",\"dir\":\"外部项目目录可省略\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID（list_nodes 返回数组项的 id；local=本机；不要传节点名）"),
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
			Desc:       "删除托管 compose 项目（移除容器并删除项目目录内应用包文件；data 数据卷目录保留，会先向用户确认）。input JSON：{\"name\":\"项目名\"}",
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
