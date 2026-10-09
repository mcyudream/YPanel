// SystemSnapshotAPI 系统级快照接口（M47，对齐 1Panel 快照）。
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// SystemSnapshotAPI 系统快照。
type SystemSnapshotAPI struct {
	Snap *service.SystemSnapshotService
}

func snapNodeOf(c *gin.Context) string {
	n := c.DefaultQuery("node", "local")
	if n == "" {
		n = "local"
	}
	return n
}

// List GET /api/v1/system/snapshots?node=
func (a *SystemSnapshotAPI) List(c *gin.Context) {
	out, err := a.Snap.List(c.Request.Context(), snapNodeOf(c))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/system/snapshots {node,websites,databases}（任务化）
func (a *SystemSnapshotAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Node      string `json:"node"`
		Websites  bool   `json:"websites"`
		Databases bool   `json:"databases"`
	}](c)
	if !ok {
		return
	}
	node := req.Node
	if node == "" {
		node = "local"
	}
	out, err := a.Snap.Create(c.Request.Context(), node, service.SystemSnapOptions{
		Websites: req.Websites, Databases: req.Databases,
	})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Delete DELETE /api/v1/system/snapshots?node=&file=
func (a *SystemSnapshotAPI) Delete(c *gin.Context) {
	if err := a.Snap.Delete(c.Request.Context(), snapNodeOf(c), c.Query("file")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Restore POST /api/v1/system/snapshots/restore {node,file}（异步任务；面板会重启）
func (a *SystemSnapshotAPI) Restore(c *gin.Context) {
	req, ok := bind[struct {
		Node string `json:"node"`
		File string `json:"file" binding:"required"`
	}](c)
	if !ok {
		return
	}
	node := req.Node
	if node == "" {
		node = "local"
	}
	out, err := a.Snap.Restore(c.Request.Context(), node, req.File)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// GetKeep GET /api/v1/system/snapshots/keep
func (a *SystemSnapshotAPI) GetKeep(c *gin.Context) {
	respOK(c, gin.H{"keep": a.Snap.GetKeep()})
}

// SetKeep PUT /api/v1/system/snapshots/keep {keep}
func (a *SystemSnapshotAPI) SetKeep(c *gin.Context) {
	req, ok := bind[struct {
		Keep int `json:"keep" binding:"required,min=1,max=50"`
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
