package api

import (
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// NodeExecAPI 多节点批量命令。
type NodeExecAPI struct {
	Nodes *service.NodeService
}

// Exec POST /api/v1/nodes/exec {nodeIds, command, timeoutSecs} → 并发执行汇总
func (a *NodeExecAPI) Exec(c *gin.Context) {
	req, ok := bind[struct {
		NodeIds     []string `json:"nodeIds" binding:"required,min=1"`
		Command     string   `json:"command" binding:"required"`
		TimeoutSecs int      `json:"timeoutSecs"`
	}](c)
	if !ok {
		return
	}
	type nodeResult struct {
		NodeId  string `json:"nodeId"`
		Name    string `json:"name"`
		Ok      bool   `json:"ok"`
		Output  string `json:"output"`
		Error   string `json:"error,omitempty"`
	}
	results := make([]nodeResult, len(req.NodeIds))
	var wg sync.WaitGroup
	for i, nid := range req.NodeIds {
		wg.Add(1)
		go func(idx int, nodeID string) {
			defer wg.Done()
			node, err := a.Nodes.ByID(nodeID)
			if err != nil {
				results[idx] = nodeResult{NodeId: nodeID, Ok: false, Error: err.Error()}
				return
			}
			ac := agentclient.New(node.BaseURL, node.Token)
			out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, c.Request.Context(), "POST", "/agent/v1/exec",
				&dto.ExecReq{Command: req.Command, TimeoutSecs: req.TimeoutSecs})
			if err != nil {
				results[idx] = nodeResult{NodeId: nodeID, Name: node.Name, Ok: false, Error: err.Error()}
				return
			}
			results[idx] = nodeResult{
				NodeId: nodeID, Name: node.Name,
				Ok:     !out.TimedOut && out.ExitCode == 0,
				Output: out.Output,
			}
		}(i, nid)
	}
	wg.Wait()
	respOK(c, results)
}

// ProcProxy 进程/服务代理（本机节点）。
type ProcProxy struct {
	Nodes *service.NodeService
}

func (p *ProcProxy) client(c *gin.Context) *agentclient.Client {
	node, _ := p.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// Processes GET /api/v1/processes
func (p *ProcProxy) Processes(c *gin.Context) {
	out, err := agentclient.GetJSON[[]dto.ProcessItem](p.client(c), c.Request.Context(), "/agent/v1/processes")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// KillProcess POST /api/v1/processes/kill {pid}
func (p *ProcProxy) KillProcess(c *gin.Context) {
	req, ok := bind[struct {
		Pid int32 `json:"pid" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[struct{ Pid int32 }, struct{}](p.client(c), c.Request.Context(), "POST", "/agent/v1/processes/kill", &struct{ Pid int32 }{Pid: req.Pid}); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Services GET /api/v1/services
func (p *ProcProxy) Services(c *gin.Context) {
	out, err := agentclient.GetJSON[[]dto.ServiceItem](p.client(c), c.Request.Context(), "/agent/v1/services")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ServiceAction POST /api/v1/services/:name/:action
func (p *ProcProxy) ServiceAction(c *gin.Context) {
	path := "/agent/v1/services/" + c.Param("name") + "/" + c.Param("action")
	if _, err := agentclient.DoJSON[struct{}, map[string]string](p.client(c), c.Request.Context(), "POST", path, &struct{}{}); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
