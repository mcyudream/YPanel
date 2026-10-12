// aitools_middleware.go AI 工具：可复用中间件实例（M68，与面板「中间件」页同域）。
// 节点语义：node 参数经 withAINode 注入 ctx，执行期解析为节点 ID 打到目标节点 agent。
package service

import (
	"context"
	"fmt"
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
			Desc: "列出面板纳管的可复用中间件实例（RabbitMQ/Memcached 等，数据库类实例在数据库域）。input 可选 JSON：{\"search\":\"名称/类型关键词\"}。返回 {total,items}（含版本与可复用判定）。",
			Parameters: schObj(map[string]any{"search": schStr("名称/类型关键词")}),
			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Search string `json:"search"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.mwSvc.List(ctx)
				if err != nil {
					return "", err
				}
				filtered := make([]map[string]any, 0, len(out))
				for _, row := range out {
					if matchAny(p.Search, fmt.Sprint(row["name"]), fmt.Sprint(row["type"]), fmt.Sprint(row["host"])) {
						filtered = append(filtered, row)
					}
				}
				return toolOut(map[string]any{"total": len(filtered), "items": filtered})
			},
		},
		{
			Name: "list_middleware_types", Module: aiModMW, Risk: aiRiskRead,
			Desc: "列出已注册的可复用中间件类型（含默认端口与最低可复用版本基线）。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out := s.mwSvc.Types()
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "test_middleware_instance", Module: aiModMW, Risk: aiRiskRead,
			Desc: "测试中间件实例连通性。input JSON：{\"instanceId\":1}",
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
			Desc: "扫描节点上可发现的中间件容器（按镜像规则匹配，返回版本/网络/compose 项目；不含凭据明文；不落库）。input 可选 JSON：{\"node\":\"local\"}",
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
				for i := range out {
					out[i].Password = ""
				}
				return toolOut(map[string]any{"total": len(out), "items": out})
			},
		},
		{
			Name: "adopt_middleware_container", Module: aiModMW, Risk: aiRiskWrite,
			Desc: "把扫描发现的中间件容器纳管为面板可复用实例。input JSON：{\"node\":\"local\",\"containerName\":\"1Panel-rabbitmq-xxx\",\"name\":\"可选自定义实例名\"}",
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
			Desc: "删除中间件实例的纳管登记（不动实例本体运行；danger：执行前务必向用户确认）。input JSON：{\"instanceId\":1}",
			Parameters: schObj(map[string]any{"instanceId": schInt("实例 ID")}, "instanceId"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					InstanceID uint `json:"instanceId"`
				}](input)
				if err != nil {
					return "", err
				}
				if _, err := s.mwSvc.ByID(p.InstanceID); err != nil {
					return "", err
				}
				if err := s.mwSvc.Delete(p.InstanceID); err != nil {
					return "", err
				}
				return toolOut(map[string]any{"deleted": p.InstanceID})
			},
		},
	}
}
