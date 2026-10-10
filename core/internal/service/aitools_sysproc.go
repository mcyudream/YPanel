// aitools_sysproc.go AI 工具：系统概览/进程服务/命令执行（M31）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ypanel/shared/dto"
)

// aiToolsSystem 系统概览工具集（常驻）。
func (s *AIService) aiToolsSystem(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "get_overview", Module: aiModSystem, Risk: aiRiskRead,
			Desc: "获取服务器实时概览：CPU/内存/磁盘/负载/网络速率。input JSON：{\"node\":\"节点 ID 可选，默认 local\"}。",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID，默认 local"),
			}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				ov, err := agentGetJSON[dto.SystemOverview](s, ctx, "/agent/v1/sysinfo/overview")
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{
					"cpuPercent": ov.CPU.UsagePercent, "cores": ov.CPU.LogicalCount,
					"memPercent": ov.Memory.UsagePercent, "memUsedGB": float64(ov.Memory.Used) / 1e9,
					"load1": ov.Load.Load1, "disks": ov.Disks,
					"rxSpeedBps": ov.Network.RxSpeedBps, "txSpeedBps": ov.Network.TxSpeedBps,
				})
			},
		},
		{
			Name: "get_disk_usage", Module: aiModSystem, Risk: aiRiskRead,
			Desc: "分析磁盘目录占用（usage tree，用于定位大文件/目录）。input JSON：{\"path\":\"/\",\"depth\":2}（path 默认 /，depth 默认 2）",
			Parameters: schObj(map[string]any{
				"path": schStr("起始目录绝对路径"), "depth": schInt("展开层级，默认 2"), "node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
			}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Path  string `json:"path"`
					Depth int    `json:"depth"`
					Node  string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Path == "" {
					p.Path = "/"
				}
				if p.Depth < 1 {
					p.Depth = 2
				}
				if p.Depth > 4 {
					p.Depth = 4
				}
				ctx = withAINode(ctx, p.Node)
				raw, err := agentGetJSON[json.RawMessage](s, ctx, fmt.Sprintf("/agent/v1/disk/usage?path=%s&depth=%d", p.Path, p.Depth))
				if err != nil {
					return "", err
				}
				return truncText(string(*raw), 8000), nil
			},
		},
		{
			Name: "list_nodes", Module: aiModSystem, Risk: aiRiskRead,
			Desc: "列出面板纳管的服务器节点（名称/地址/在线状态）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				rows := s.nodes.ListNodes()
				out := make([]map[string]any, 0, len(rows))
				for _, r := range rows {
					out = append(out, map[string]any{
						"id": r["id"], "name": r["name"], "baseURL": r["baseURL"],
						"online": r["online"], "version": r["version"],
					})
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
	}
}

// aiToolsProc 进程与服务工具集。
func (s *AIService) aiToolsProc(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_processes", Module: aiModProc, Risk: aiRiskRead,
			Desc: "列出主机进程（按 CPU/内存排序取头部）。input 可选 JSON：{\"sort\":\"cpu|mem\",\"order\":\"desc\",\"limit\":30,\"search\":\"进程名关键词\"}",
			Parameters: schObj(map[string]any{
				"node": schStr("目标节点 ID，默认 local"),
				"sort": schEnum("排序字段", "cpu", "mem"), "order": schEnum("方向", "asc", "desc"),
				"limit": schInt("条数上限，默认 30"), "search": schStr("进程名/命令行关键词"),
			}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node   string `json:"node"`
					Sort   string `json:"sort"`
					Order  string `json:"order"`
					Limit  int    `json:"limit"`
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Sort == "" {
					p.Sort = "cpu"
				}
				if p.Order == "" {
					p.Order = "desc"
				}
				if p.Limit < 1 {
					p.Limit = 30
				}
				if p.Limit > 200 {
					p.Limit = 200
				}
				ctx = withAINode(ctx, p.Node)
				out, err := agentGetJSON[[]dto.ProcessItem](s, ctx, fmt.Sprintf("/agent/v1/processes?sort=%s&order=%s&limit=%d", p.Sort, p.Order, p.Limit))
				if err != nil {
					return "", err
				}
				items := []dto.ProcessItem{}
				if out != nil {
					items = *out
				}
				filtered := make([]dto.ProcessItem, 0, len(items))
				for _, pr := range items {
					if matchAny(p.Search, pr.Name, pr.Cmdline) {
						filtered = append(filtered, pr)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "list_services", Module: aiModProc, Risk: aiRiskRead,
			Desc: "列出 systemd 服务（加载/活动状态）。input 可选 JSON：{\"search\":\"服务名关键词\"}",
			Parameters: schObj(map[string]any{"search": schStr("服务名关键词"), "node": schStr("目标节点，默认 local")}),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
					Node   string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				out, err := agentGetJSON[[]dto.ServiceItem](s, ctx, "/agent/v1/services")
				if err != nil {
					return "", err
				}
				items := []dto.ServiceItem{}
				if out != nil {
					items = *out
				}
				filtered := make([]dto.ServiceItem, 0, len(items))
				for _, sv := range items {
					if matchAny(p.Search, sv.Name, sv.Desc) {
						filtered = append(filtered, sv)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "kill_process", Module: aiModProc, Risk: aiRiskDanger,
			Desc: "终止进程（SIGKILL，可能造成服务中断/数据丢失，会先向用户确认）。input JSON：{\"pid\":1234}",
			Parameters: schObj(map[string]any{"pid": schInt("进程 PID"), "node": schStr("目标节点，默认 local")}, "pid", "node"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Pid  int32  `json:"pid"`
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Pid <= 1 {
					return "", fmt.Errorf("拒绝终止 PID %d（系统关键进程）", p.Pid)
				}
				body := struct {
					Pid int32 `json:"pid"`
				}{Pid: p.Pid}
				if _, err := agentPostJSON[struct {
					Pid int32 `json:"pid"`
				}, any](s, ctx, "/agent/v1/processes/kill", &body); err != nil {
					return "", err
				}
				return fmt.Sprintf("已终止进程 %d", p.Pid), nil
			},
		},
		{
			Name: "service_action", Module: aiModProc, Risk: aiRiskDanger,
			Desc: "对 systemd 服务执行 start/stop/restart（stop/restart 会中断该服务，会先向用户确认）。input JSON：{\"name\":\"服务名\",\"action\":\"start|stop|restart\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("服务名（如 nginx）"), "action": schEnum("操作", "start", "stop", "restart"),
				"node": schStr("目标节点，默认 local"),
			}, "name", "action", "node"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name   string `json:"name"`
					Action string `json:"action"`
					Node   string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Action != "start" && p.Action != "stop" && p.Action != "restart" {
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				if _, err := agentPostJSON[struct{}, map[string]string](s, ctx, "/agent/v1/services/"+p.Name+"/"+p.Action, nil); err != nil {
					return "", err
				}
				return fmt.Sprintf("已对服务 %s 执行 %s", p.Name, p.Action), nil
			},
		},
	}
}

// aiToolsExec 命令执行工具集（最高风险，全部走 ask）。
func (s *AIService) aiToolsExec(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "run_command", Module: aiModExec, Risk: aiRiskDanger,
			Desc: "在服务器上执行任意 shell 命令（systemd 服务环境，无终端交互；每次执行都会请求用户确认）。仅在没有更合适的面板工具时使用。input JSON：{\"command\":\"shell 命令\",\"timeoutSecs\":120}",
			Parameters: schObj(map[string]any{
				"command": schStr("要执行的 shell 命令"), "timeoutSecs": schInt("超时秒数，默认 120 上限 300"),
				"node": schStr("目标节点，默认 local"),
			}, "command"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Command     string `json:"command"`
					TimeoutSecs int    `json:"timeoutSecs"`
					Node        string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Command) == "" {
					return "", fmt.Errorf("命令为空")
				}
				if p.TimeoutSecs < 1 || p.TimeoutSecs > 300 {
					p.TimeoutSecs = 120
				}
				ctx = withAINode(ctx, p.Node)
				return s.hostExecTimeout(ctx, p.Command, p.TimeoutSecs)
			},
		},
		{
			Name: "run_in_workspace", Module: aiModExec, Risk: aiRiskDanger,
			Desc: "在 AI 工作空间目录（" + workspaceDir + "）中执行 shell 命令（写临时脚本/代码实验；每次执行都会请求用户确认）。input JSON：{\"command\":\"shell 命令\"}",
			Parameters: schObj(map[string]any{"command": schStr("shell 命令"), "node": schStr("目标节点，默认 local")}, "command"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Command string `json:"command"`
					Node    string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Command) == "" {
					return "", fmt.Errorf("命令为空")
				}
				ctx = withAINode(ctx, p.Node)
				return s.hostExec(ctx, fmt.Sprintf("mkdir -p '%s' && cd '%s' && %s", workspaceDir, workspaceDir, p.Command))
			},
		},
	}
}
