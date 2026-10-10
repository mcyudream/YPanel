// FileExtAPI 文件管理扩展接口（M38）：回收站/收藏/分享/远程下载。
package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// FileExtAPI 文件扩展。
type FileExtAPI struct {
	Ext   *service.FileExtService
	Nodes *service.NodeService
}

// Trash POST /api/v1/files/trash {paths}
func (a *FileExtAPI) Trash(c *gin.Context) {
	req, ok := bind[struct {
		Paths []string `json:"paths" binding:"required,min=1"`
	}](c)
	if !ok {
		return
	}
	node, err := a.Nodes.ByID("local")
	if err != nil {
		respErr(c, err)
		return
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	out, err := agentclient.DoJSON[dto.FileTrashReq, dto.FileTrashListResp](ac, c.Request.Context(), "POST", "/agent/v1/files/trash",
		&dto.FileTrashReq{Paths: req.Paths})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// TrashList GET /api/v1/files/trash/list
func (a *FileExtAPI) TrashList(c *gin.Context) {
	node, err := a.Nodes.ByID("local")
	if err != nil {
		respErr(c, err)
		return
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	out, err := agentclient.GetJSON[dto.FileTrashListResp](ac, c.Request.Context(), "/agent/v1/files/trash/list")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out.Items)
}

func (a *FileExtAPI) trashNames(c *gin.Context, action string) {
	req, ok := bind[struct {
		Names []string `json:"names" binding:"required,min=1"`
	}](c)
	if !ok {
		return
	}
	node, err := a.Nodes.ByID("local")
	if err != nil {
		respErr(c, err)
		return
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	if _, err := agentclient.DoJSON[dto.FileTrashNamesReq, struct{}](ac, c.Request.Context(), "POST", "/agent/v1/files/trash/"+action,
		&dto.FileTrashNamesReq{Names: req.Names}); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// TrashRestore POST /api/v1/files/trash/restore
func (a *FileExtAPI) TrashRestore(c *gin.Context) { a.trashNames(c, "restore") }

// TrashPurge POST /api/v1/files/trash/purge
func (a *FileExtAPI) TrashPurge(c *gin.Context) { a.trashNames(c, "purge") }

// TrashClear POST /api/v1/files/trash/clear
func (a *FileExtAPI) TrashClear(c *gin.Context) {
	node, err := a.Nodes.ByID("local")
	if err != nil {
		respErr(c, err)
		return
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	out, err := agentclient.DoJSON[struct{}, map[string]int](ac, c.Request.Context(), "POST", "/agent/v1/files/trash/clear", nil)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Favorites GET /api/v1/files/favorites
func (a *FileExtAPI) Favorites(c *gin.Context) { respOK(c, a.Ext.Favorites()) }

// FavoriteAdd POST /api/v1/files/favorites {path}
func (a *FileExtAPI) FavoriteAdd(c *gin.Context) {
	req, ok := bind[struct {
		Path string `json:"path" binding:"required"`
	}](c)
	if !ok {
		return
	}
	row, err := a.Ext.FavoriteAdd(req.Path)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, row)
}

// FavoriteRemove DELETE /api/v1/files/favorites/:id
func (a *FileExtAPI) FavoriteRemove(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Ext.FavoriteRemove(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Shares GET /api/v1/files/shares
func (a *FileExtAPI) Shares(c *gin.Context) { respOK(c, a.Ext.ShareList()) }

// ShareCreate POST /api/v1/files/shares {path,days}（返回原始 token，仅此一次）
func (a *FileExtAPI) ShareCreate(c *gin.Context) {
	req, ok := bind[struct {
		Path string `json:"path" binding:"required"`
		Days int    `json:"days"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Ext.ShareCreate(c.Request.Context(), req.Path, req.Days)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ShareRevoke DELETE /api/v1/files/shares/:id
func (a *FileExtAPI) ShareRevoke(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Ext.ShareRevoke(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// RemoteDownload POST /api/v1/files/remote-download {url,destDir} → 任务化（立即回 taskId，进度见任务中心）
func (a *FileExtAPI) RemoteDownload(c *gin.Context) {
	req, ok := bind[struct {
		URL     string `json:"url" binding:"required"`
		DestDir string `json:"destDir" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Ext.RemoteDownloadTask(req.URL, req.DestDir)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ShareDownload GET /s/:token（公开无鉴权；agent 流式下载）
func (a *FileExtAPI) ShareDownload(c *gin.Context) {
	name, body, cleanup, err := a.Ext.ShareResolve(c.Param("token"))
	if err != nil {
		cleanup()
		c.String(http.StatusNotFound, "404")
		return
	}
	defer cleanup()
	h := c.Writer.Header()
	h.Set("Content-Type", "application/octet-stream")
	h.Set("Content-Disposition", "attachment; filename*=UTF-8''"+encodeRFC5987(name))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, body)
}

func encodeRFC5987(s string) string {
	out := make([]byte, 0, len(s)*3)
	const hexd = "0123456789ABCDEF"
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' || ch == '_' {
			out = append(out, ch)
		} else {
			out = append(out, '%', hexd[ch>>4], hexd[ch&15])
		}
	}
	return string(out)
}

// CopyAcross POST /api/v1/files/copy-across（M56 跨节点拷贝，任务化）
func (a *FileAPI) CopyAcross(c *gin.Context) {
	req, ok := bind[service.CopyAcrossReq](c)
	if !ok {
		return
	}
	out, err := a.Cross.CopyAcrossTask(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
