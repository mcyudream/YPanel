package api

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// ContainerFileAPI 容器内文件管理接口（代理 agent，见 agent dockerx/cfiles.go）。
type ContainerFileAPI struct {
	Nodes *service.NodeService
}

func (f *ContainerFileAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := f.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// basePath 容器文件端点前缀。
func (f *ContainerFileAPI) basePath(c *gin.Context) string {
	return "/agent/v1/docker/containers/" + c.Param("id") + "/files"
}

// List GET /api/v1/docker/containers/:id/files/list?path=&node=
func (f *ContainerFileAPI) List(c *gin.Context) {
	out, err := agentclient.GetJSON[dto.FileListResp](f.client(c), c.Request.Context(), f.basePath(c)+"/list?path="+escape(c.Query("path")))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Read GET /api/v1/docker/containers/:id/files/read?path=&raw=&node=
func (f *ContainerFileAPI) Read(c *gin.Context) {
	q := f.basePath(c) + "/read?path=" + escape(c.Query("path"))
	if c.Query("raw") != "" {
		q += "&raw=" + escape(c.Query("raw"))
	}
	out, err := agentclient.GetJSON[dto.FileReadResp](f.client(c), c.Request.Context(), q)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Download GET /api/v1/docker/containers/:id/files/download?path=&node=（流式透传）
func (f *ContainerFileAPI) Download(c *gin.Context) {
	req, err := f.client(c).NewRequest(c.Request.Context(), http.MethodGet, f.basePath(c)+"/download?path="+escape(c.Query("path")), nil)
	if err != nil {
		respErr(c, err)
		return
	}
	resp, err := f.client(c).HTTP.Do(req)
	if err != nil {
		respErr(c, errAgentUnreach(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		respErr(c, errAgentUnreach(nil))
		return
	}
	for _, k := range []string{"Content-Type", "Content-Disposition", "Content-Length"} {
		if v := resp.Header.Get(k); v != "" {
			c.Header(k, v)
		}
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, resp.Body)
}

// Write POST /api/v1/docker/containers/:id/files/write
func (f *ContainerFileAPI) Write(c *gin.Context) {
	req, ok := bind[dto.FileWriteReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/write", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Mkdir POST /api/v1/docker/containers/:id/files/mkdir
func (f *ContainerFileAPI) Mkdir(c *gin.Context) {
	req, ok := bind[dto.FileMkdirReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/mkdir", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Rename POST /api/v1/docker/containers/:id/files/rename
func (f *ContainerFileAPI) Rename(c *gin.Context) {
	req, ok := bind[dto.FileRenameReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileRenameReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/rename", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Delete POST /api/v1/docker/containers/:id/files/delete
func (f *ContainerFileAPI) Delete(c *gin.Context) {
	req, ok := bind[dto.FileDeleteReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileDeleteReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/delete", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Chmod POST /api/v1/docker/containers/:id/files/chmod
func (f *ContainerFileAPI) Chmod(c *gin.Context) {
	req, ok := bind[dto.FileChmodReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileChmodReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/chmod", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Chown POST /api/v1/docker/containers/:id/files/chown
func (f *ContainerFileAPI) Chown(c *gin.Context) {
	req, ok := bind[dto.FileChownReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileChownReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/chown", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// containerUploadReq 容器文件上传（JSON base64）。
type containerUploadReq struct {
	Path       string `json:"path" binding:"required"`
	Name       string `json:"name" binding:"required"`
	ContentB64 string `json:"contentB64" binding:"required"`
}

// Upload POST /api/v1/docker/containers/:id/files/upload（multipart：file 字段 + path query）
func (f *ContainerFileAPI) Upload(c *gin.Context) {
	fileHdr, err := c.FormFile("file")
	if err != nil {
		respErr(c, err)
		return
	}
	fh, err := fileHdr.Open()
	if err != nil {
		respErr(c, err)
		return
	}
	defer func() { _ = fh.Close() }()
	content, err := io.ReadAll(fh)
	if err != nil {
		respErr(c, err)
		return
	}
	req := containerUploadReq{
		Path:       c.Query("path"),
		Name:       fileHdr.Filename,
		ContentB64: base64.StdEncoding.EncodeToString(content),
	}
	if _, err := agentclient.DoJSON[containerUploadReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, f.basePath(c)+"/upload", &req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
