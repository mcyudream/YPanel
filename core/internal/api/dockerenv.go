// DockerEnvAPI 多 Docker 环境 + 镜像生命周期接口（M35）。
package api

import (
	"path"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
)

// DockerEnvAPI 环境与镜像扩展。
type DockerEnvAPI struct {
	Env   *service.DockerEnvService
	Img   *service.DockerImgService
	Nodes *service.NodeService
}

// ListEnvs GET /api/v1/docker/environments
func (a *DockerEnvAPI) ListEnvs(c *gin.Context) {
	out, err := a.Env.List()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// CreateEnv POST /api/v1/docker/environments
func (a *DockerEnvAPI) CreateEnv(c *gin.Context) {
	req, ok := bind[struct {
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required,oneof=local tcp"`
		Endpoint  string `json:"endpoint"`
		TLSEnable bool   `json:"tlsEnable"`
		TLSCA     string `json:"tlsCa"`
		TLSCert   string `json:"tlsCert"`
		TLSKey    string `json:"tlsKey"`
		Remark    string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Env.Create(c.Request.Context(), service.EnvInput{
		Name: req.Name, Type: req.Type, Endpoint: req.Endpoint, TLSEnable: req.TLSEnable,
		TLSCA: req.TLSCA, TLSCert: req.TLSCert, TLSKey: req.TLSKey, Remark: req.Remark,
	})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": out.ID, "name": out.Name})
}

// UpdateEnv PUT /api/v1/docker/environments/:id
func (a *DockerEnvAPI) UpdateEnv(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required,oneof=local tcp"`
		Endpoint  string `json:"endpoint"`
		TLSEnable bool   `json:"tlsEnable"`
		TLSCA     string `json:"tlsCa"`
		TLSCert   string `json:"tlsCert"`
		TLSKey    string `json:"tlsKey"`
		Remark    string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	if err := a.Env.Update(c.Request.Context(), id, service.EnvInput{
		Name: req.Name, Type: req.Type, Endpoint: req.Endpoint, TLSEnable: req.TLSEnable,
		TLSCA: req.TLSCA, TLSCert: req.TLSCert, TLSKey: req.TLSKey, Remark: req.Remark,
	}); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// DeleteEnv DELETE /api/v1/docker/environments/:id
func (a *DockerEnvAPI) DeleteEnv(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Env.Delete(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// TestEnv POST /api/v1/docker/environments/:id/test
func (a *DockerEnvAPI) TestEnv(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("id 不合法"))
		return
	}
	out, err := a.Env.Test(c.Request.Context(), uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// EnvContainers GET /api/v1/docker/environments/:id/containers
func (a *DockerEnvAPI) EnvContainers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("id 不合法"))
		return
	}
	out, err := a.Env.EnvContainers(c.Request.Context(), uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// EnvImages GET /api/v1/docker/environments/:id/images
func (a *DockerEnvAPI) EnvImages(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("id 不合法"))
		return
	}
	out, err := a.Env.EnvImages(c.Request.Context(), uint(id))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// EnvContainerAction POST /api/v1/docker/environments/:id/containers/:name/:action
func (a *DockerEnvAPI) EnvContainerAction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("id 不合法"))
		return
	}
	if err := a.Env.EnvContainerAction(c.Request.Context(), uint(id), c.Param("name"), c.Param("action")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// EnvImageRemove DELETE /api/v1/docker/environments/:id/images/:image（?force=）
func (a *DockerEnvAPI) EnvImageRemove(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errBadRequest("id 不合法"))
		return
	}
	if err := a.Env.EnvImageRemove(c.Request.Context(), uint(id), c.Param("image"), c.Query("force") == "true"); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ImageBuild POST /api/v1/docker/images/build {contextDir,dockerfile,tag}
func (a *DockerEnvAPI) ImageBuild(c *gin.Context) {
	req, ok := bind[struct {
		ContextDir string `json:"contextDir" binding:"required"`
		Dockerfile string `json:"dockerfile"`
		Tag        string `json:"tag" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Img.Build(c.Request.Context(), req.ContextDir, req.Dockerfile, req.Tag)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ImageSave POST /api/v1/docker/images/save {images[],name}
func (a *DockerEnvAPI) ImageSave(c *gin.Context) {
	req, ok := bind[struct {
		Images []string `json:"images" binding:"required,min=1"`
		Name   string   `json:"name"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Img.Save(c.Request.Context(), req.Images, req.Name)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ImageLoad POST /api/v1/docker/images/load（multipart file 上传，直灌 agent tmp 后 docker load）
func (a *DockerEnvAPI) ImageLoad(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		respErr(c, errBadRequest("缺少上传文件"))
		return
	}
	src, err := fh.Open()
	if err != nil {
		respErr(c, err)
		return
	}
	defer func() { _ = src.Close() }()
	node, nerr := a.Nodes.ByID("local")
	if nerr != nil {
		respErr(c, nerr)
		return
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	target := path.Join("/opt/ypanel/tmp", fh.Filename)
	if err := service.UploadMultipartToAgent(ac, c.Request.Context(), "/opt/ypanel/tmp", fh.Filename, src); err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Img.Load(c.Request.Context(), target)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ImageTag POST /api/v1/docker/images/tag {src,dst}
func (a *DockerEnvAPI) ImageTag(c *gin.Context) {
	req, ok := bind[struct {
		Src string `json:"src" binding:"required"`
		Dst string `json:"dst" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Img.Tag(c.Request.Context(), req.Src, req.Dst); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ImageCheckUpdates POST /api/v1/docker/images/check-updates {images[]}
func (a *DockerEnvAPI) ImageCheckUpdates(c *gin.Context) {
	req, ok := bind[struct {
		Images []string `json:"images" binding:"required,min=1"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Img.CheckUpdates(c.Request.Context(), req.Images)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ContainerCommit POST /api/v1/docker/containers/:id/commit {image}（:id=容器名，与既有 :id/:action 同位）
func (a *DockerEnvAPI) ContainerCommit(c *gin.Context) {
	req, ok := bind[struct {
		Image string `json:"image" binding:"required"`
	}](c)
	if !ok {
		return
	}
	if err := a.Img.Commit(c.Request.Context(), c.Param("id"), req.Image); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// ContainerBackup POST /api/v1/docker/containers/:id/backup（named volumes 导出 + inspect 落盘；:id=容器名）
func (a *DockerEnvAPI) ContainerBackup(c *gin.Context) {
	out, err := a.Img.ContainerVolumes(c.Request.Context(), c.Param("id"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// SwarmStatus GET /api/v1/docker/swarm-status
func (a *DockerEnvAPI) SwarmStatus(c *gin.Context) {
	out, err := a.Img.SwarmStatus(c.Request.Context())
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
