package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
)

// SystemAPI 主机监控接口（代理 agent）。
type SystemAPI struct {
	Nodes *service.NodeService
}

func (s *SystemAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := s.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// Overview GET /api/v1/system/overview
func (s *SystemAPI) Overview(c *gin.Context) {
	out, err := agentclient.GetJSON[dto.SystemOverview](s.client(c), c.Request.Context(), "/agent/v1/sysinfo/overview")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// History GET /api/v1/system/history?seconds=
func (s *SystemAPI) History(c *gin.Context) {
	seconds := c.DefaultQuery("seconds", "600")
	out, err := agentclient.GetJSON[[]dto.MetricSample](s.client(c), c.Request.Context(), "/agent/v1/sysinfo/history?seconds="+seconds)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// FileAPI 文件管理接口（代理 agent）。
type FileAPI struct {
	Nodes *service.NodeService
}

func (f *FileAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := f.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// List GET /api/v1/files/list?path=
func (f *FileAPI) List(c *gin.Context) {
	out, err := agentclient.GetJSON[dto.FileListResp](f.client(c), c.Request.Context(), "/agent/v1/files/list?path="+escape(c.Query("path")))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Read GET /api/v1/files/read?path=&raw=&encoding=
func (f *FileAPI) Read(c *gin.Context) {
	q := "/agent/v1/files/read?path=" + escape(c.Query("path"))
	if c.Query("raw") != "" {
		q += "&raw=" + escape(c.Query("raw"))
	}
	if c.Query("encoding") != "" {
		q += "&encoding=" + escape(c.Query("encoding"))
	}
	out, err := agentclient.GetJSON[dto.FileReadResp](f.client(c), c.Request.Context(), q)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Write POST /api/v1/files/write
func (f *FileAPI) Write(c *gin.Context) {
	req, ok := bind[dto.FileWriteReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/write", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Mkdir POST /api/v1/files/mkdir
func (f *FileAPI) Mkdir(c *gin.Context) {
	req, ok := bind[dto.FileMkdirReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileMkdirReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/mkdir", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Rename POST /api/v1/files/rename
func (f *FileAPI) Rename(c *gin.Context) {
	req, ok := bind[dto.FileRenameReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileRenameReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/rename", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Delete POST /api/v1/files/delete
func (f *FileAPI) Delete(c *gin.Context) {
	req, ok := bind[dto.FileDeleteReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileDeleteReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/delete", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Chmod POST /api/v1/files/chmod {path, mode}
func (f *FileAPI) Chmod(c *gin.Context) {
	req, ok := bind[dto.FileChmodReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileChmodReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/chmod", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Compress POST /api/v1/files/compress {src, dest}
func (f *FileAPI) Compress(c *gin.Context) {
	req, ok := bind[dto.FileCompressReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileCompressReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/compress", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Decompress POST /api/v1/files/decompress {archive, destDir}
func (f *FileAPI) Decompress(c *gin.Context) {
	req, ok := bind[dto.FileDecompressReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileDecompressReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/decompress", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Search GET /api/v1/files/search?dir=&keyword=
func (f *FileAPI) Search(c *gin.Context) {
	out, err := agentclient.GetJSON[[]dto.FileEntry](f.client(c), c.Request.Context(),
		"/agent/v1/files/search?dir="+escape(c.Query("dir"))+"&keyword="+escape(c.Query("keyword")))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Upload POST /api/v1/files/upload?path=（multipart 直传转发）
func (f *FileAPI) Upload(c *gin.Context) {
	fileHdr, err := c.FormFile("file")
	if err != nil {
		respErr(c, errBadRequest("缺少 file 字段"))
		return
	}
	fh, err := fileHdr.Open()
	if err != nil {
		respErr(c, errBadRequest(err.Error()))
		return
	}
	defer func() { _ = fh.Close() }()

	// 重建 multipart 转发到 agent
	out, err := f.uploadMultipart(c, fh, fileHdr.Filename, c.Query("path"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Download GET /api/v1/files/download?path=（文件原样流式；目录 tar.gz）
func (f *FileAPI) Download(c *gin.Context) {
	req, err := f.client(c).NewRequest(c.Request.Context(), http.MethodGet, "/agent/v1/files/download?path="+escape(c.Query("path")), nil)
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

// DockerAPI 容器接口（代理 agent）。
type DockerAPI struct {
	Nodes *service.NodeService
}

func (d *DockerAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := d.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// List GET /api/v1/docker/containers
func (d *DockerAPI) List(c *gin.Context) {
	out, err := agentclient.GetJSON[[]dto.ContainerItem](d.client(c), c.Request.Context(), "/agent/v1/docker/containers")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Action POST /api/v1/docker/containers/:id/:action
func (d *DockerAPI) Action(c *gin.Context) {
	path := "/agent/v1/docker/containers/" + c.Param("id") + "/" + c.Param("action")
	if _, err := agentclient.DoJSON[struct{}, struct{}](d.client(c), c.Request.Context(), http.MethodPost, path, &struct{}{}); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Logs GET /api/v1/docker/containers/:id/logs?tail=&follow=
func (d *DockerAPI) Logs(c *gin.Context) {
	path := "/agent/v1/docker/containers/" + c.Param("id") + "/logs?tail=" + c.DefaultQuery("tail", "500") + "&follow=" + c.DefaultQuery("follow", "0")
	req, err := d.client(c).NewRequest(c.Request.Context(), http.MethodGet, path, nil)
	if err != nil {
		respErr(c, err)
		return
	}
	resp, err := d.client(c).HTTP.Do(req)
	if err != nil {
		respErr(c, errAgentUnreach(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		respErr(c, errAgentUnreach(nil))
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 8192)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			return
		}
	}
}

// SysManageAPI 主机管理（B21）：代理 agent 受控系统设置。
type SysManageAPI struct {
	Nodes *service.NodeService
}

func (m *SysManageAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := m.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// Manage POST /api/v1/system/manage {action: hostname|timezone|dns, value}
func (m *SysManageAPI) Manage(c *gin.Context) {
	req, ok := bind[map[string]string](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[map[string]string, map[string]string](m.client(c), c.Request.Context(),
		http.MethodPost, "/agent/v1/sysmanage", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}
