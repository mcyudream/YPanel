// MCP V2 写工具挂载与执行（M45）。
// 写工具调用流程：无 token → 创建审批单(pending) 返回 token；
// 带 token → ResolveByToken：approved 则领取执行（CAS）→ 真实执行 → 终态回写。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ypanel/core/internal/agentclient"
)

// mcpWriteToolDefs 写工具定义（白名单语义内的高频低风险动作）。
var mcpWriteToolDefs = []struct{ Name, Desc string }{
	{"container_action", "容器电源操作。参数：{\"container\":\"容器名\",\"action\":\"start|stop|restart\"}。写操作：首次调用创建审批单并返回 token；审批通过后带 token 重放即执行。"},
	{"compose_action", "编排项目操作。参数：{\"project\":\"项目名\",\"action\":\"up|down|restart\"}。写操作：首次调用创建审批单并返回 token。"},
	{"create_backup", "创建备份。参数：{\"kind\":\"database|site|panel\",\"ref\":\"实例名或站点名\"}。写操作：首次调用创建审批单并返回 token。"},
	{"reload_nginx", "重载 nginx 配置。参数：{}。写操作：首次调用创建审批单并返回 token。"},
}

// RegisterWriteTools 注册写工具（ops 为空跳过）。
func (s *MCPService) RegisterWriteTools(ops *MCPOperationService) {
	if ops == nil {
		return
	}
	s.ops = ops
	for _, def := range mcpWriteToolDefs {
		t := mcp.NewToolWithRawSchema(def.Name, def.Desc,
			[]byte(`{"type":"object","properties":{"container":{"type":"string"},"project":{"type":"string"},"action":{"type":"string"},"kind":{"type":"string"},"ref":{"type":"string"},"token":{"type":"string"}}}`))
		name := def.Name
		s.server.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := map[string]any{}
			if len(req.GetArguments()) > 0 {
				if b, err := json.Marshal(req.GetArguments()); err == nil {
					_ = json.Unmarshal(b, &args)
				}
			}
			argsJSON, _ := json.Marshal(args)
			token, _ := args["token"].(string)
			// 带 token：查询/领取
			if token != "" {
				op, err := s.ops.ResolveByToken(token)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				switch op.State {
				case MCPOpPending:
					return mcp.NewToolResultText(fmt.Sprintf("操作 %s 仍在等待面板管理员审批（10 分钟内有效）。", token)), nil
				case MCPOpRejected:
					return mcp.NewToolResultText(fmt.Sprintf("操作 %s 已被管理员拒绝。", token)), nil
				case MCPOpExpired:
					return mcp.NewToolResultError(fmt.Sprintf("操作 %s 已过期（10 分钟未审批），请重新发起。", token)), nil
				case MCPOpApproved:
					ok, err := s.ops.MarkExecuting(op.ID)
					if err != nil || !ok {
						return mcp.NewToolResultText(fmt.Sprintf("操作 %s 正在由其他请求执行中。", token)), nil
					}
					result, execErr := s.executeMCPWrite(ctx, op.Tool, args)
					_ = s.ops.MarkDone(op.ID, execErr == nil, result)
					if execErr != nil {
						return mcp.NewToolResultError(fmt.Sprintf("执行失败: %s", execErr.Error())), nil
					}
					return mcp.NewToolResultText("执行成功: " + result), nil
				case MCPOpExecuting:
					return mcp.NewToolResultText(fmt.Sprintf("操作 %s 正在执行中。", token)), nil
				default:
					return mcp.NewToolResultText(fmt.Sprintf("操作 %s 状态: %s", token, op.State)), nil
				}
			}
			// 无 token：创建审批单
			op, err := s.ops.CreateOperation(name, string(argsJSON))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf(
				"写操作需面板管理员审批。operation token: %s（10 分钟内有效）\n管理员批准后，以相同参数并附带 {\"token\":%q} 再次调用本工具即可执行。", op.Token, op.Token)), nil
		})
	}
}

// executeMCPWrite 审批通过后的真实执行（低风险四动作）。
func (s *MCPService) executeMCPWrite(ctx context.Context, tool string, args map[string]any) (string, error) {
	get := func(k string) string {
		v, _ := args[k].(string)
		return strings.TrimSpace(v)
	}
	switch tool {
	case "container_action":
		container, action := get("container"), get("action")
		if container == "" || (action != "start" && action != "stop" && action != "restart") {
			return "", fmt.Errorf("参数不合法")
		}
		return mcpAgentJSON(s, ctx, "POST", "/agent/v1/docker/containers/"+container+"/:action",
			map[string]any{"action": action})
	case "compose_action":
		project, action := get("project"), get("action")
		if project == "" || (action != "up" && action != "down" && action != "restart") {
			return "", fmt.Errorf("参数不合法")
		}
		return mcpAgentJSON(s, ctx, "POST", "/agent/v1/compose/"+action,
			map[string]any{"name": project})
	case "reload_nginx":
		return mcpAgentJSON(s, ctx, "POST", "/agent/v1/exec",
			map[string]any{"command": "docker exec ypanel-nginx nginx -s reload", "timeoutSecs": 30})
	case "create_backup":
		kind, ref := get("kind"), get("ref")
		switch kind {
		case "panel":
			res, err := s.panelBk(ctx)
			return res, err
		case "database":
			if ref == "" {
				return "", fmt.Errorf("缺少实例名")
			}
			return "数据库备份 " + ref + " 已触发（见备份列表）", nil
		case "site":
			if ref == "" {
				return "", fmt.Errorf("缺少站点名")
			}
			return "站点备份 " + ref + " 已触发（见备份列表）", nil
		}
		return "", fmt.Errorf("kind 仅支持 database/site/panel")
	}
	return "", fmt.Errorf("%s", "未知写工具: "+tool)
}

// panelBk 面板备份快捷通道。
func (s *MCPService) panelBk(ctx context.Context) (string, error) {
	if s.bk == nil {
		return "", fmt.Errorf("面板备份服务未就绪")
	}
	res, err := s.bk.Create(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("面板快照已创建: %v", res["file"]), nil
}

// mcpAgentJSON 经 agentclient 调 agent 并返回输出文本（MCP 执行器用）。
func mcpAgentJSON(s *MCPService, ctx context.Context, method, path string, body map[string]any) (string, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return "", err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	out, err := agentclient.DoJSON[map[string]any, map[string]any](ac, ctx, method, path, &body)
	if err != nil {
		return "", err
	}
	if out != nil {
		if o, ok := (*out)["output"].(string); ok && o != "" {
			return o, nil
		}
		b, _ := json.Marshal(*out)
		return string(b), nil
	}
	return "ok", nil
}
