// aitools_middleware.go AI 工具：可复用中间件实例（M68，与面板「中间件」页同域）。
// 节点语义：node 参数经 withAINode 注入 ctx，执行期解析为节点 ID 打到目标节点 agent。
package service

import (
	"context"
)

// mwNodeIDFromCtx 工具执行期的目标节点 ID（缺省 local）。
func mwNodeIDFromCtx(ctx context.Context) string {
	if node, ok := ctx.Value(aiNodeCtxKey{}).(string); ok && node != "" {
		return node
	}
	return "local"
}

// aiToolsMiddleware 中间件工具集。
func (s *AIService) aiToolsMiddleware(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "list_middleware_instances", Module: aiModMW, Risk: aiRiskRead,
			Desc:       "列出登记中间件、当前节点自动发现容器，以及关联的数据库纳管记录（含 S3/Neo4j/Redis 等）。返回 {total,items,discoveryError?}，含 source/managedBy/nodeId/state；source=database 的 ID 仅用于数据库域，discovered 条目尚未登记，发现失败保留已有记录。不返回密码、不自动纳管。",
			Parameters: schObj(map[string]any{"search": schStr("名称/类型关键词"), "node": schStr("目标节点，缺省 local")}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
					Node   string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				out, err := s.mwSvc.List(ctx, mwNodeIDFromCtx(ctx))
				if err != nil {
					return "", err
				}
				filtered := make([]MiddlewareListItem, 0, len(out.Items))
				for _, row := range out.Items {
					if matchAny(p.Search, row.Name, row.Type, row.Host) {
						filtered = append(filtered, row)
					}
				}
				out.Items, out.Total = filtered, len(filtered)
				return toolOut(out)
			},
		},
		{
			Name: "list_middleware_types", Module: aiModMW, Risk: aiRiskRead,
			Desc:       "列出已注册的可复用中间件类型（含默认端口与最低可复用版本基线）。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out := s.mwSvc.Types()
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "test_middleware_instance", Module: aiModMW, Risk: aiRiskRead,
			Desc:       "测试中间件实例连通性。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.mwSvc.Test(ctx, p.InstanceID); err != nil {
					return toolOut(map[string]any{"ok": false, "error": err.Error()})
				}
				return toolOut(map[string]any{"ok": true})
			},
		},
		{
			Name: "scan_middleware_containers", Module: aiModMW, Risk: aiRiskRead,
			Desc:       "扫描节点运行中与停止的中间件/数据库容器，返回版本、网络、项目、状态和 managedBy/adoptable。数据库域类型请在数据库管理接入；不读取凭据、不落库，扫描失败返回错误。input 可选 JSON：{\"node\":\"local\"}",
			Parameters: schObj(map[string]any{"node": schStr("目标节点，缺省 local")}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				out, err := s.mwSvc.Scan(ctx, mwNodeIDFromCtx(ctx))
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "adopt_middleware_container", Module: aiModMW, Risk: aiRiskWrite,
			Desc: "把扫描结果中 adoptable=true 的 RabbitMQ/Memcached 容器纳管为可复用实例，记录节点并在此时读取、加密凭据。数据库域类型不通过此工具接入。input JSON：{\"node\":\"local\",\"containerName\":\"rabbitmq-main\",\"name\":\"可选自定义实例名\"}",
			Parameters: schObj(map[string]any{
				"node":          schStr("目标节点，缺省 local"),
				"containerName": schStr("容器名（scan 结果中的 containerName）"),
				"name":          schStr("自定义实例名，缺省自动生成"),
			}, "containerName"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node          string `json:"node"`
					ContainerName string `json:"containerName"`
					Name          string `json:"name"`
				}](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
				inst, err := s.mwSvc.Adopt(ctx, mwNodeIDFromCtx(ctx), p.ContainerName, p.Name)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"id": inst.ID, "name": inst.Name, "type": inst.Type})
			},
		},
		{
			Name: "create_middleware_instance", Module: aiModMW, Risk: aiRiskWrite,
			Desc: "手动登记外部中间件实例（带连通性预检，失败即拒）。input JSON：{\"name\":\"实例名\",\"type\":\"rabbitmq|memcached\",\"host\":\"主机\",\"port\":5672,\"user\":\"可选\",\"password\":\"可选\",\"remark\":\"可选\"}",
			Parameters: schObj(map[string]any{
				"name": schStr("实例名（小写字母/数字/中划线）"), "type": schStr("类型（list_middleware_types 查询）"),
				"host": schStr("主机"), "port": schInt("端口，缺省用类型默认"), "user": schStr("用户名可选"),
				"password": schStr("密码可选"), "remark": schStr("备注可选"),
			}, "name", "type", "host"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Name     string `json:"name"`
					Type     string `json:"type"`
					Host     string `json:"host"`
					Port     int    `json:"port"`
					User     string `json:"user"`
					Password string `json:"password"`
					Remark   string `json:"remark"`
				}](input)
				if err != nil {
					return "", err
				}
				inst, err := s.mwSvc.Create(ctx, p.Name, p.Type, p.Host, p.Port, p.User, p.Password, p.Remark)
				if err != nil {
					return "", err
				}
				return toolOut(map[string]any{"id": inst.ID, "name": inst.Name})
			},
		},
		{
			Name: "delete_middleware_instance", Module: aiModMW, Risk: aiRiskDanger,
			Desc:       "删除中间件实例的纳管登记（不动实例本体运行；danger：执行前务必向用户确认）。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.mwSvc.Delete(ctx, p.InstanceID); err != nil {
					return "", err
				}
				return toolOut(map[string]any{"deleted": p.InstanceID})
			},
		},
	}
}
