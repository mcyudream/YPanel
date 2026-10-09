package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// DockerInstallAPI Docker/Compose 一键安装接口（M52）。
type DockerInstallAPI struct {
	Svc *service.DockerInstallService
}

// Precheck POST /api/v1/docker/install/precheck {nodeId?, source?}
// source 非空时附带该源的连通性测试结果。
func (a *DockerInstallAPI) Precheck(c *gin.Context) {
	req, ok := bind[struct {
		NodeID string `json:"nodeId"`
		Source string `json:"source"`
	}](c)
	if !ok {
		return
	}
	pk, err := a.Svc.Precheck(c.Request.Context(), req.NodeID)
	if err != nil {
		respErr(c, err)
		return
	}
	out := map[string]any{"precheck": pk}
	if req.Source != "" {
		t, terr := a.Svc.TestSource(c.Request.Context(), req.NodeID, req.Source)
		if terr != nil {
			respErr(c, terr)
			return
		}
		out["sourceTest"] = t
	}
	respOK(c, out)
}

// Install POST /api/v1/docker/install（admin）：dryRun=true 只生成步骤+语法校验，否则创建安装任务。
func (a *DockerInstallAPI) Install(c *gin.Context) {
	req, ok := bind[service.InstallReq](c)
	if !ok {
		return
	}
	out, err := a.Svc.Install(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetMirrors GET /api/v1/docker/registry-mirrors?node=
func (a *DockerInstallAPI) GetMirrors(c *gin.Context) {
	mirrors, err := a.Svc.GetRegistryMirrors(c.Request.Context(), c.Query("node"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, map[string]any{"mirrors": mirrors})
}

// SetMirrors POST /api/v1/docker/registry-mirrors {nodeId?, mirrors}（合并写 daemon.json + 重启 docker）
func (a *DockerInstallAPI) SetMirrors(c *gin.Context) {
	req, ok := bind[struct {
		NodeID  string   `json:"nodeId"`
		Mirrors []string `json:"mirrors"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Svc.SetRegistryMirrors(c.Request.Context(), req.NodeID, req.Mirrors)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
