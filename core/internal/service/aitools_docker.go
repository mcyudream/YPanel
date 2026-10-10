// aitools_docker.go AI 工具：容器/镜像/网络/卷（M31 模块化注册表）。
package service

import (
	"context"

	"github.com/ypanel/shared/dto"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// aiToolsContainers 容器生命周期工具集。
func (s *AIService) aiToolsContainers(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_containers", Module: aiModContainers, Risk: aiRiskRead,
			Desc: "列出 Docker 容器（分页，勿假设一次返回全部）。input 可选 JSON：{\"search\":\"名称/镜像/ID 关键词\",\"state\":\"running 或 stopped\",\"sort\":\"name|state|image|created\",\"order\":\"asc|desc\",\"page\":1,\"pageSize\":20}，全部可省略（默认第 1 页 20 条，按名称排序）。返回 {total,page,pageSize,items}；total 超过当前页时按需翻页或加 search 收窄。",
			Parameters: schObj(map[string]any{
				"search": schStr("名称/镜像/ID 关键词"), "state": schStr("running / stopped / paused / all，默认 all"),
				"sort": schEnum("排序字段", "name", "state", "image", "created"),
				"order": schEnum("排序方向", "asc", "desc"), "page": schInt("页码，从 1 起"), "pageSize": schInt("每页条数，默认 20 上限 100"),
			}),
			Fn: func(ctx context.Context, input string) (string, error) {
				out, err := agentGetJSON[[]dto.ContainerItem](s, ctx, "/agent/v1/docker/containers")
				if err != nil {
					return "", err
				}
				items := []dto.ContainerItem{}
				if out != nil {
					items = *out
				}
				q := parseListQuery(input)
				filtered := make([]dto.ContainerItem, 0, len(items))
				for _, c := range items {
					if !matchAny(q.Search, c.Name, c.Image, c.ID) {
						continue
					}
					if q.State != "" && q.State != "all" && c.State != q.State {
						continue
					}
					filtered = append(filtered, c)
				}
				var less func(a, b dto.ContainerItem) bool
				switch q.Sort {
				case "state":
					less = func(a, b dto.ContainerItem) bool { return a.State < b.State }
				case "image":
					less = func(a, b dto.ContainerItem) bool { return a.Image < b.Image }
				case "created":
					less = func(a, b dto.ContainerItem) bool { return a.Created.Before(b.Created) }
				default:
					less = func(a, b dto.ContainerItem) bool { return a.Name < b.Name }
				}
				pageItems, total := paginateList(filtered, q, less)
				return toolOut(map[string]any{"total": total, "page": q.Page, "pageSize": q.PageSize, "items": pageItems})
			},
		},
		{
			Name: "inspect_container", Module: aiModContainers, Risk: aiRiskRead,
			Desc: "查看容器详情（inspect 全量 JSON 截断）。input JSON：{\"name\":\"容器名或ID\"}",
			Parameters: schObj(map[string]any{"name": schStr("容器名或 ID")}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				out, err := agentGetJSON[json.RawMessage](s, ctx, "/agent/v1/docker/containers/"+url.PathEscape(p.Name)+"/inspect")
				if err != nil {
					return "", err
				}
				return truncText(string(*out), 8000), nil
			},
		},
		{
			Name: "container_logs", Module: aiModContainers, Risk: aiRiskRead,
			Desc: "查看容器日志（尾部）。input JSON：{\"name\":\"容器名或ID\",\"tail\":200}",
			Parameters: schObj(map[string]any{
				"name": schStr("容器名或 ID"), "tail": schInt("尾部行数，默认 200，上限 2000"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name string `json:"name"`
					Tail int    `json:"tail"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				if p.Tail < 1 {
					p.Tail = 200
				}
				if p.Tail > 2000 {
					p.Tail = 2000
				}
				return s.agentText(ctx, fmt.Sprintf("/agent/v1/docker/containers/%s/logs?tail=%d&timestamps=0", url.PathEscape(p.Name), p.Tail))
			},
		},
		{
			Name: "container_stats", Module: aiModContainers, Risk: aiRiskRead,
			Desc: "查看容器实时资源占用（CPU/内存/网络 IO）。input JSON：{\"name\":\"容器名或ID\"}",
			Parameters: schObj(map[string]any{"name": schStr("容器名或 ID")}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				out, err := agentGetJSON[json.RawMessage](s, ctx, "/agent/v1/docker/containers/"+url.PathEscape(p.Name)+"/stats")
				if err != nil {
					return "", err
				}
				return truncText(string(*out), 3000), nil
			},
		},
		{
			Name: "create_container", Module: aiModContainers, Risk: aiRiskWrite,
			Desc: "创建并启动容器。input JSON：{\"name\":\"容器名\",\"image\":\"镜像:tag\",\"ports\":[{\"hostPort\":8080,\"containerPort\":80,\"proto\":\"tcp\"}],\"mounts\":[\"/host/path:/container/path\"],\"env\":[\"KEY=value\"],\"restart\":\"unless-stopped\",\"network\":\"可选网络名\"}。至少 name+image；建议先 list_images 确认镜像存在（不存在先 pull_image）。",
			Parameters: schObj(map[string]any{
					"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("容器名"), "image": schStr("镜像名:标签"),
				"ports": schArr("端口映射", map[string]any{"type": "object", "properties": map[string]any{
					"hostPort": schInt("宿主端口"), "containerPort": schInt("容器端口"), "proto": schEnum("协议", "tcp", "udp"),
				}}),
				"mounts": schArr("卷挂载 host:container[:ro]", schStr("挂载串")),
				"env":    schArr("环境变量 KEY=value", schStr("环境变量")),
				"restart": schEnum("重启策略", "no", "always", "unless-stopped", "on-failure"),
				"network": schStr("加入的网络名"),
			}, "name", "image"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					ExtContainerCreateReq
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if p.Name == "" || p.Image == "" {
					return "", fmt.Errorf("name 与 image 必填")
				}
				id, err := dx.ContainerCreate(ctx, p.ExtContainerCreateReq)
				if err != nil {
					return "", err
				}
				return "容器已创建并启动，ID: " + id, nil
			},
		},
		{
			Name: "container_action", Module: aiModContainers, Risk: aiRiskDanger,
			Desc: "对容器执行 start/stop/restart（停启影响运行中的服务，会先向用户确认）。input JSON：{\"name\":\"容器名\",\"action\":\"start|stop|restart\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("容器名或 ID"), "action": schEnum("操作", "start", "stop", "restart"),
			}, "name", "action"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name   string `json:"name"`
					Action string `json:"action"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Action != "start" && p.Action != "stop" && p.Action != "restart" {
					return "", fmt.Errorf("不支持的操作: %s", p.Action)
				}
				if _, err := agentPostJSON[struct{}, json.RawMessage](s, ctx,
					"/agent/v1/docker/containers/"+url.PathEscape(p.Name)+"/"+p.Action, nil); err != nil {
					return "", err
				}
				return "已执行 " + p.Action + ": " + p.Name, nil
			},
		},
		{
			Name: "container_remove", Module: aiModContainers, Risk: aiRiskDanger,
			Desc: "删除容器（不可恢复，会先向用户确认）。input JSON：{\"name\":\"容器名\",\"force\":false}",
			Parameters: schObj(map[string]any{
				"name": schStr("容器名或 ID"), "force": schBool("运行中也强制删除"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name  string `json:"name"`
					Force bool   `json:"force"`
				}](input)
				if err != nil {
					return "", err
				}
				if _, err := s.dockerX.Delete(ctx, "/agent/v1/docker/containers/"+url.PathEscape(p.Name)+"?force="+boolInt(p.Force)); err != nil {
					return "", err
				}
				return "已删除容器: " + p.Name, nil
			},
		},
		{
			Name: "containers_prune", Module: aiModContainers, Risk: aiRiskDanger,
			Desc: "清理全部已停止的容器（不可恢复，会先向用户确认）。input 传 {}。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}

				out, err := dx.ContainersPrune(ctx)
				if err != nil {
					return "", err
				}
				return out, nil
			},
		},
	}
}

// aiToolsImages 镜像工具集。
func (s *AIService) aiToolsImages(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_images", Module: aiModImages, Risk: aiRiskRead,
			Desc: "列出本机 Docker 镜像（分页/搜索）。input 可选 JSON：{\"search\":\"名称关键词\",\"page\":1,\"pageSize\":20}。返回 {total,page,pageSize,items}。",
			Parameters: schObj(map[string]any{
					"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"search": schStr("镜像名关键词"), "page": schInt("页码"), "pageSize": schInt("每页条数"),
			}),
			Fn: func(ctx context.Context, input string) (string, error) {
				raw, err := s.dockerX.Passthrough(ctx, "/agent/v1/docker/images")
				if err != nil {
					return "", err
				}
				items := []map[string]any{}
				if err := json.Unmarshal(raw, &items); err != nil {
					return truncText(string(raw), 12000), nil
				}
				q := parseListQuery(input)
				filtered := make([]map[string]any, 0, len(items))
				for _, im := range items {
					if matchAny(q.Search, fmt.Sprint(im["name"]), fmt.Sprint(im["id"]), fmt.Sprint(im["repository"])) {
						filtered = append(filtered, im)
					}
				}
				pageItems, total := paginateList(filtered, q, nil)
				return toolOut(map[string]any{"total": total, "page": q.Page, "pageSize": q.PageSize, "items": pageItems})
			},
		},
		{
			Name: "pull_image", Module: aiModImages, Risk: aiRiskWrite,
			Desc: "拉取镜像（下载需要时间与磁盘空间，会先向用户确认）。input JSON：{\"ref\":\"nginx:latest\"}",
			Parameters: schObj(map[string]any{"ref": schStr("镜像名:标签")}, "ref"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Ref string `json:"ref"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if p.Ref == "" {
					return "", fmt.Errorf("缺少 ref")
				}
				out, err := dx.ImagePull(ctx, p.Ref)
				if err != nil {
					return "", err
				}
				return "镜像拉取完成: " + p.Ref + "\n" + truncText(out, 500), nil
			},
		},
		{
			Name: "remove_image", Module: aiModImages, Risk: aiRiskDanger,
			Desc: "删除镜像（会先向用户确认）。input JSON：{\"id\":\"镜像ID或完整引用名\",\"force\":false}。id 建议传 list_images 返回的 ID（sha256/短 ID），带斜杠的引用名（如 louislam/uptime-kuma:1）亦可",
			Parameters: schObj(map[string]any{
					"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"id": schStr("镜像 ID 或名称"), "force": schBool("被容器引用时强制删除"),
			}, "id"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					ID    string `json:"id"`
					Force bool   `json:"force"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if err := dx.ImageRemove(ctx, p.ID, p.Force); err != nil {
					return "", err
				}
				return "已删除镜像: " + p.ID, nil
			},
		},
		{
			Name: "images_prune", Module: aiModImages, Risk: aiRiskDanger,
			Desc: "清理全部悬空（未被标签引用的）镜像（会先向用户确认）。input 传 {}。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}

				out, err := dx.PassthroughPost(ctx, "/agent/v1/docker/images/prune")
				if err != nil {
					return "", err
				}
				return truncText(string(out), 1000), nil
			},
		},
	}
}

// aiToolsNetworks 网络工具集。
func (s *AIService) aiToolsNetworks(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_networks", Module: aiModNetworks, Risk: aiRiskRead,
			Desc: "列出 Docker 网络。input 传 {}。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				raw, err := s.dockerX.Passthrough(ctx, "/agent/v1/docker/networks")
				if err != nil {
					return "", err
				}
				return truncText(string(raw), 8000), nil
			},
		},
		{
			Name: "create_network", Module: aiModNetworks, Risk: aiRiskWrite,
			Desc: "创建 Docker 网络。input JSON：{\"name\":\"网络名\",\"driver\":\"bridge\"}",
			Parameters: schObj(map[string]any{
					"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),
				"name": schStr("网络名"), "driver": schEnum("驱动", "bridge", "overlay", "macvlan", "host"),
			}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name   string `json:"name"`
					Driver string `json:"driver"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if p.Name == "" {
					return "", fmt.Errorf("缺少 name")
				}
				if err := dx.NetworkCreate(ctx, p.Name, p.Driver); err != nil {
					return "", err
				}
				return "已创建网络: " + p.Name, nil
			},
		},
		{
			Name: "remove_network", Module: aiModNetworks, Risk: aiRiskDanger,
			Desc: "删除 Docker 网络（内置网络不可删，会先向用户确认）。input JSON：{\"name\":\"网络名\"}",
			Parameters: schObj(map[string]any{"name": schStr("网络名")}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if err := dx.NetworkRemove(ctx, p.Name); err != nil {
					return "", err
				}
				return "已删除网络: " + p.Name, nil
			},
		},
	}
}

// aiToolsVolumes 卷工具集。
func (s *AIService) aiToolsVolumes(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_volumes", Module: aiModVolumes, Risk: aiRiskRead,
			Desc: "列出 Docker 存储卷。input 传 {}。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				raw, err := s.dockerX.Passthrough(ctx, "/agent/v1/docker/volumes")
				if err != nil {
					return "", err
				}
				return truncText(string(raw), 8000), nil
			},
		},
		{
			Name: "create_volume", Module: aiModVolumes, Risk: aiRiskWrite,
			Desc: "创建 Docker 存储卷。input JSON：{\"name\":\"卷名\"}",
			Parameters: schObj(map[string]any{"name": schStr("卷名")}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if err := dx.VolumeCreate(ctx, p.Name); err != nil {
					return "", err
				}
				return "已创建卷: " + p.Name, nil
			},
		},
		{
			Name: "remove_volume", Module: aiModVolumes, Risk: aiRiskDanger,
			Desc: "删除存储卷（卷内数据不可恢复，会先向用户确认）。input JSON：{\"name\":\"卷名\"}",
			Parameters: schObj(map[string]any{"name": schStr("卷名")}, "name"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
					Name string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
				if err := dx.VolumeRemove(ctx, p.Name); err != nil {
					return "", err
				}
				return "已删除卷: " + p.Name, nil
			},
		},
		{
			Name: "volumes_prune", Module: aiModVolumes, Risk: aiRiskDanger,
			Desc: "清理全部未被容器引用的存储卷（数据不可恢复，会先向用户确认）。input 传 {}。",
			Parameters: schObj(map[string]any{"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}

				out, err := dx.PassthroughPost(ctx, "/agent/v1/docker/volumes/prune")
				if err != nil {
					return "", err
				}
				return truncText(string(out), 1000), nil
			},
		},
	}
}

// boolInt 布尔转 "1"/"0"。
func boolInt(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

var _ = strings.TrimSpace // 占位：包级引用保持 import 稳定（实际使用见其他工具文件）
