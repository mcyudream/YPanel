package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// SystemAPI 主机监控接口（代理 agent）。
type SystemAPI struct {
	Nodes *service.NodeService
	Notif *service.NotificationService // 概览附带最近通知（可空）
}

func (s *SystemAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := s.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// Overview GET /api/v1/system/overview（含 recentNotifications 最近 5 条）
func (s *SystemAPI) Overview(c *gin.Context) {
	out, err := agentclient.GetJSON[dto.SystemOverview](s.client(c), c.Request.Context(), "/agent/v1/sysinfo/overview")
	if err != nil {
		respErr(c, err)
		return
	}
	// 展开为 map 以便附加 recentNotifications 且不破坏原字段
	var resp map[string]any
	if b, merr := json.Marshal(out); merr == nil {
		_ = json.Unmarshal(b, &resp)
	} else {
		resp = map[string]any{}
	}
	if s.Notif != nil {
		resp["recentNotifications"] = s.Notif.List(5)
	}
	respOK(c, resp)
}

// HostEntries GET /api/v1/system/hosts（读宿主机 /etc/hosts，过滤 localhost/回环，供容器 hosts 映射导入）。
func (s *SystemAPI) HostEntries(c *gin.Context) {
	resp, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](s.client(c), c.Request.Context(), "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "cat /etc/hosts 2>/dev/null || true", TimeoutSecs: 30})
	if err != nil {
		respErr(c, err)
		return
	}
	type hostEntry struct {
		IP    string   `json:"ip"`
		Hosts []string `json:"hosts"`
	}
	entries := []hostEntry{}
	for _, line := range strings.Split(resp.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ip := fields[0]
		lip := strings.ToLower(ip)
		if lip == "127.0.0.1" || strings.HasPrefix(lip, "127.") ||
			// IPv6 仅保留全局单播（2xxx/3xxx），滤掉 ::1/链路本地/多播等无意义条目
			(strings.Contains(lip, ":") && !strings.HasPrefix(lip, "2") && !strings.HasPrefix(lip, "3")) {
			continue
		}
		hosts := []string{}
		for _, h := range fields[1:] {
			h = strings.ToLower(h)
			if h == "localhost" || strings.HasSuffix(h, ".localdomain") || strings.HasSuffix(h, ".local") {
				continue
			}
			hosts = append(hosts, h)
		}
		if len(hosts) > 0 {
			entries = append(entries, hostEntry{IP: ip, Hosts: hosts})
		}
	}
	respOK(c, entries)
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

// DiskUsageTree GET /api/v1/disk/usage?path=/var[&refresh=1]（du 语义目录占用；agent 24h 缓存，refresh=1 绕过重算）
func (s *SystemAPI) DiskUsageTree(c *gin.Context) {
	p := filepath.Clean(c.Query("path"))
	if !filepath.IsAbs(p) || c.Query("path") == "" {
		respErr(c, errs.ErrBadRequest)
		return
	}
	q := "/agent/v1/disk/usage?path=" + url.QueryEscape(p)
	if c.Query("refresh") == "1" {
		q += "&refresh=1"
	}
	out, err := agentclient.GetJSON[dto.DiskUsageTree](s.client(c), c.Request.Context(), q)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// FileAPI 文件管理接口（代理 agent）。
type FileAPI struct {
	Nodes *service.NodeService
	Rev   *service.RevisionService // 受管路径写盘前自动快照（M23），可空
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
	node := c.DefaultQuery("node", "local")
	if f.Rev != nil && service.ScopeFor(node, req.Path) != "" {
		f.Rev.SnapshotBefore(c.Request.Context(), node, req.Path, "save", c.GetString(middleware.CtxUsername))
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

// Copy POST /api/v1/files/copy
func (f *FileAPI) Copy(c *gin.Context) {
	req, ok := bind[dto.FileCopyReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileCopyReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/copy", req); err != nil {
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

// Chmod POST /api/v1/files/chmod {path, mode, recursive}
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

// Chown POST /api/v1/files/chown {path, owner, group, recursive}
func (f *FileAPI) Chown(c *gin.Context) {
	req, ok := bind[dto.FileChownReq](c)
	if !ok {
		return
	}
	if _, err := agentclient.DoJSON[dto.FileChownReq, struct{}](f.client(c), c.Request.Context(), http.MethodPost, "/agent/v1/files/chown", req); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Owners GET /api/v1/files/owners（系统用户/组枚举）
func (f *FileAPI) Owners(c *gin.Context) {
	out, err := agentclient.GetJSON[dto.FileOwnersResp](f.client(c), c.Request.Context(), "/agent/v1/files/owners")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Compress POST /api/v1/files/compress {src?, srcs?, dest}
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

// 容器最近一次操作失败记录（内存，重启即清）：id → {action, message, at}。
// Docker/agent 不保存历史错误，面板留存供详情页排障展示。
var (
	containerOpErrMu sync.Mutex
	containerOpErr   = map[string]containerOpErrEntry{}
)

type containerOpErrEntry struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	At      time.Time `json:"at"`
}

func recordContainerOpErr(id, action string, err error) {
	containerOpErrMu.Lock()
	defer containerOpErrMu.Unlock()
	containerOpErr[id] = containerOpErrEntry{Action: action, Message: errs.From(err).Message, At: time.Now()}
}

// ContainerLastOpErr GET /api/v1/docker/containers/:id/last-op-err
func (d *DockerAPI) ContainerLastOpErr(c *gin.Context) {
	containerOpErrMu.Lock()
	defer containerOpErrMu.Unlock()
	e, ok := containerOpErr[c.Param("id")]
	if !ok {
		respOK(c, (*containerOpErrEntry)(nil))
		return
	}
	respOK(c, &e)
}

// Action POST /api/v1/docker/containers/:id/:action
func (d *DockerAPI) Action(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")
	path := "/agent/v1/docker/containers/" + id + "/" + action
	if _, err := agentclient.DoJSON[struct{}, struct{}](d.client(c), c.Request.Context(), http.MethodPost, path, &struct{}{}); err != nil {
		if action == "start" || action == "restart" {
			recordContainerOpErr(id, action, err)
		}
		respErr(c, err)
		return
	}
	containerOpErrMu.Lock()
	delete(containerOpErr, id)
	containerOpErrMu.Unlock()
	respOK(c, struct{}{})
}

// Logs GET /api/v1/docker/containers/:id/logs?tail=&follow=&timestamps=
func (d *DockerAPI) Logs(c *gin.Context) {
	path := "/agent/v1/docker/containers/" + c.Param("id") + "/logs?tail=" + c.DefaultQuery("tail", "500") + "&follow=" + c.DefaultQuery("follow", "0") + "&timestamps=" + c.DefaultQuery("timestamps", "1")
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
