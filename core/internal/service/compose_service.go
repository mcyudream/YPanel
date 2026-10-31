package service

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// RunComposeServiceAction 供 HTTP 与 AI 共用，按实际目标校验项目、目录和节点范围。
func RunComposeServiceAction(ctx context.Context, nodes *NodeService, nodeID string, req dto.ComposeServiceActionReq) (*dto.ComposeServiceActionResp, error) {
	target := dto.ComposePullUpdateReq{NodeID: nodeID, Project: req.Project, Name: req.Name, Dir: req.Dir}
	if err := NormalizeComposeUpdateTarget(ctx, &target); err != nil {
		return nil, err
	}
	if !src2NameRe.MatchString(req.Service) {
		return nil, svcBadReq("请选择有效的编排服务（名称仅允许字母、数字、中划线、下划线，最多 64 个字符）")
	}
	switch req.Action {
	case "start", "stop", "restart", "pull", "up":
	default:
		return nil, svcBadReq("服务操作只支持 start、stop、restart、pull、up")
	}
	if nodes == nil {
		return nil, errs.Wrapc(errs.CodeInternal, "节点服务不可用")
	}
	node, err := nodes.ByID(target.NodeID)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, errs.ErrNotFound
	}
	query := url.Values{
		"project": {target.Project}, "service": {req.Service}, "action": {req.Action}, "dir": {target.Dir},
	}
	// 保持 agent 查询参数协议，兼容独立升级的旧版节点；此端点返回 output，不是 exec 信封。
	return agentclient.DoJSON[struct{}, dto.ComposeServiceActionResp](
		agentclient.New(node.BaseURL, node.Token), ctx, http.MethodPost,
		"/agent/v1/compose/service-action?"+query.Encode(), nil,
	)
}
