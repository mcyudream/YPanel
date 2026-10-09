// SnapshotAPI 快照状态机接口（M45）。
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// nodeOf 取节点参数（默认 local）。
func nodeOf(c *gin.Context) string {
	n := c.DefaultQuery("node", "local")
	if n == "" {
		n = "local"
	}
	return n
}

// SnapshotAPI 快照管理。
type SnapshotAPI struct {
	Snap *service.SnapshotService
}

// List GET /api/v1/snapshots?node=
func (a *SnapshotAPI) List(c *gin.Context) {
	out, err := a.Snap.List(c.Request.Context(), nodeOf(c))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Plan GET /api/v1/snapshots/plan?file=
func (a *SnapshotAPI) Plan(c *gin.Context) {
	out, err := a.Snap.PlanRestore(nodeOf(c), c.Query("file"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Execute POST /api/v1/snapshots/execute {file,rollbackBefore}
// 执行期间面板失联属预期，前端轮询 /health。
func (a *SnapshotAPI) Execute(c *gin.Context) {
	req, ok := bind[struct {
		Node           string `json:"node"`
		File           string `json:"file" binding:"required"`
		RollbackBefore bool   `json:"rollbackBefore"`
	}](c)
	if !ok {
		return
	}
	node := req.Node
	if node == "" {
		node = "local"
	}
	if err := a.Snap.ExecuteRestore(c.Request.Context(), node, req.File, req.RollbackBefore); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"started": true, "hint": "面板将短暂失联，请等待 /health 恢复后刷新页面"})
}

// Import POST /api/v1/snapshots/import {node,path}（tmp 内已上传的 .db）
func (a *SnapshotAPI) Import(c *gin.Context) {
	req, ok := bind[struct {
		Node string `json:"node"`
		Path string `json:"path" binding:"required"`
	}](c)
	if !ok {
		return
	}
	node := req.Node
	if node == "" {
		node = "local"
	}
	out, err := a.Snap.Import(c.Request.Context(), node, req.Path)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetKeep GET /api/v1/snapshots/keep
func (a *SnapshotAPI) GetKeep(c *gin.Context) {
	respOK(c, gin.H{"keep": a.Snap.Keep()})
}

// SetKeep PUT /api/v1/snapshots/keep {keep}
func (a *SnapshotAPI) SetKeep(c *gin.Context) {
	req, ok := bind[struct {
		Keep int `json:"keep" binding:"required,min=1,max=100"`
	}](c)
	if !ok {
		return
	}
	if err := a.Snap.SetKeep(req.Keep); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Prune POST /api/v1/snapshots/prune {node?,keep?,all?}
func (a *SnapshotAPI) Prune(c *gin.Context) {
	req, ok := bind[struct {
		Node string `json:"node"`
		Keep int    `json:"keep"`
		All  bool   `json:"all"`
	}](c)
	if !ok {
		return
	}
	if req.All {
		respOK(c, a.Snap.PruneAll(c.Request.Context(), req.Keep))
		return
	}
	node := req.Node
	if node == "" {
		node = "local"
	}
	removed, err := a.Snap.Prune(c.Request.Context(), node, req.Keep)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"removed": removed})
}
