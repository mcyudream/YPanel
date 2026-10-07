package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/ypanel/core/internal/service"
)

func json2any(raw json.RawMessage) any {
	var v any
	_ = json.Unmarshal(raw, &v)
	return v
}

// DockerExtAPI Docker 管理扩展接口（镜像/网络/卷/容器详情/清理/exec/创建）。
type DockerExtAPI struct {
	Ext *service.DockerExtService
}

// Images GET /api/v1/docker/images
func (a *DockerExtAPI) Images(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/images")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// ImagePull POST /api/v1/docker/images/pull {ref}（任务化：返回 taskId，日志在任务中心轮询）
func (a *DockerExtAPI) ImagePull(c *gin.Context) {
	req, ok := bind[struct {
		Ref string `json:"ref" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Ext.ImagePullTask(req.Ref)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ImageRemove DELETE /api/v1/docker/images/:id
func (a *DockerExtAPI) ImageRemove(c *gin.Context) {
	if err := a.Ext.ImageRemove(c.Request.Context(), c.Param("id"), c.Query("force") == "1"); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ImagesPrune POST /api/v1/docker/images/prune
func (a *DockerExtAPI) ImagesPrune(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/images/prune")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// Networks GET /api/v1/docker/networks
func (a *DockerExtAPI) Networks(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/networks")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// NetworkCreate POST /api/v1/docker/networks {name, driver}
func (a *DockerExtAPI) NetworkCreate(c *gin.Context) {
	req, ok := bind[struct {
		Name   string `json:"name" binding:"required"`
		Driver string `json:"driver"`
	}](c)
	if !ok {
		return
	}
	if err := a.Ext.NetworkCreate(c.Request.Context(), req.Name, req.Driver); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// NetworkRemove DELETE /api/v1/docker/networks/:name
func (a *DockerExtAPI) NetworkRemove(c *gin.Context) {
	if err := a.Ext.NetworkRemove(c.Request.Context(), c.Param("name")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Volumes GET /api/v1/docker/volumes
func (a *DockerExtAPI) Volumes(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/volumes")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// VolumeCreate POST /api/v1/docker/volumes {name}
func (a *DockerExtAPI) VolumeCreate(c *gin.Context) {
	req, ok := bind[struct {
		Name string `json:"name" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Ext.VolumeCreate(c.Request.Context(), req.Name); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// VolumeRemove DELETE /api/v1/docker/volumes/:name
func (a *DockerExtAPI) VolumeRemove(c *gin.Context) {
	if err := a.Ext.VolumeRemove(c.Request.Context(), c.Param("name")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// VolumesPrune POST /api/v1/docker/volumes/prune
func (a *DockerExtAPI) VolumesPrune(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/volumes/prune")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// ContainersPrune POST /api/v1/docker/containers/prune
func (a *DockerExtAPI) ContainersPrune(c *gin.Context) {
	out, err := a.Ext.ContainersPrune(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"output": out})
}

// ContainerInspect GET /api/v1/docker/containers/:id/inspect
func (a *DockerExtAPI) ContainerInspect(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/containers/"+c.Param("id")+"/inspect")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// ContainerStats GET /api/v1/docker/containers/:id/stats
func (a *DockerExtAPI) ContainerStats(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/containers/"+c.Param("id")+"/stats")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// ContainerExecWS GET /api/v1/docker/containers/:id/exec?cmd=/bin/sh（WS 双向代理，浏览器 ↔ agent）
func (a *DockerExtAPI) ContainerExecWS(c *gin.Context) {
	id := c.Param("id")
	cmd := c.DefaultQuery("cmd", "/bin/sh")
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     func(*http.Request) bool { return true }, // 鉴权已由中间件完成
	}
	browserWS, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = browserWS.Close() }()

	ac, err := a.Ext.Client()
	if err != nil {
		_ = browserWS.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "节点不可用: "+err.Error()),
			time.Now().Add(2*time.Second))
		return
	}
	agentURL := ac.WSURL("/agent/v1/docker/containers/" + id + "/exec?cmd=" + url.QueryEscape(cmd))
	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL, ac.Header())
	if err != nil {
		_ = browserWS.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "agent 连接失败: "+err.Error()),
			time.Now().Add(2*time.Second))
		return
	}
	defer func() { _ = agentConn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, data, err := agentConn.ReadMessage()
			if err != nil {
				return
			}
			if err := browserWS.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()
	for {
		mt, payload, err := browserWS.ReadMessage()
		if err != nil {
			return
		}
		if err := agentConn.WriteMessage(mt, payload); err != nil {
			return
		}
	}
}

// ContainerCreate POST /api/v1/docker/containers
func (a *DockerExtAPI) ContainerCreate(c *gin.Context) {
	req, ok := bind[service.ExtContainerCreateReq](c)
	if !ok {
		return
	}
	id, err := a.Ext.ContainerCreate(c.Request.Context(), *req)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": id})
}

// ContainerRemove DELETE /api/v1/docker/containers/:id
func (a *DockerExtAPI) ContainerRemove(c *gin.Context) {
	if err := a.Ext.ContainerRemove(c.Request.Context(), c.Param("id"), c.Query("force") == "1"); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// RegistryList GET /api/v1/docker/registry（B10）
func (a *DockerExtAPI) RegistryList(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/registry")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// RegistrySet PUT /api/v1/docker/registry（B10）
func (a *DockerExtAPI) RegistrySet(c *gin.Context) {
	req, ok := bind[struct {
		Registry string `json:"registry" binding:"required"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Ext.PostJSON(c.Request.Context(), "/agent/v1/docker/registry", map[string]string{
		"registry": req.Registry, "username": req.Username, "password": req.Password,
	})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// RegistryRemove DELETE /api/v1/docker/registry?registry=（B10）
func (a *DockerExtAPI) RegistryRemove(c *gin.Context) {
	registry := c.Query("registry")
	if registry == "" {
		respErr(c, errBadRequest("registry 必填"))
		return
	}
	out, err := a.Ext.Delete(c.Request.Context(), "/agent/v1/docker/registry?registry="+url.QueryEscape(registry))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// DaemonConfig GET /api/v1/docker/daemon-config// DaemonConfig GET /api/v1/docker/daemon-config
func (a *DockerExtAPI) DaemonConfig(c *gin.Context) {
	out, err := a.Ext.Passthrough(c.Request.Context(), "/agent/v1/docker/daemon-config")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}

// UpdateDaemonConfig PUT /api/v1/docker/daemon-config {content}（JSON 校验 + 重启 docker）
func (a *DockerExtAPI) UpdateDaemonConfig(c *gin.Context) {
	req, ok := bind[struct {
		Content string `json:"content" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if !json.Valid([]byte(req.Content)) {
		respErr(c, errBadRequest("daemon.json 不是合法 JSON"))
		return
	}
	out, err := a.Ext.PostJSON(c.Request.Context(), "/agent/v1/docker/daemon-config", map[string]string{"content": req.Content})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, json2any(out))
}
