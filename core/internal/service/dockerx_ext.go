// DockerService 扩展服务（镜像/网络/卷/容器详情/stats/清理），代理到 agent dockerx 能力。
// 与 M4 dbadmin 模式一致：core 不直连 docker，统一经 agent 通道。
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// DockerExtService Docker 管理扩展服务。
type DockerExtService struct {
	nodes *NodeService
}

// NewDockerExtService 创建。
func NewDockerExtService(nodes *NodeService) *DockerExtService {
	return &DockerExtService{nodes: nodes}
}

// Client 暴露 agent 客户端（WS 代理等直连场景使用）。
func (s *DockerExtService) Client() (*agentclient.Client, error) {
	return s.client()
}

func (s *DockerExtService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// passthrough 透传 GET 请求（返回原始 JSON）。
func (s *DockerExtService) Passthrough(ctx context.Context, path string) (json.RawMessage, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	req, err := ac.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := ac.HTTP.Do(req)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errs.Wrap(errs.ErrAgentUnreach, fmt.Sprintf("agent HTTP %d", resp.StatusCode))
	}
	var env dto.Resp[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, err
	}
	if env.Code != 0 {
		return nil, &errs.Error{Code: env.Code, Message: env.Message}
	}
	return env.Data, nil
}

// postJSON 透传 POST JSON。
func (s *DockerExtService) PostJSON(ctx context.Context, path string, body any) (json.RawMessage, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, err := ac.NewRequest(ctx, http.MethodPost, path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := ac.HTTP.Do(req)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	var env dto.Resp[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, err
	}
	if env.Code != 0 {
		return nil, &errs.Error{Code: env.Code, Message: env.Message}
	}
	return env.Data, nil
}

// delete 透传 DELETE。
func (s *DockerExtService) Delete(ctx context.Context, path string) (json.RawMessage, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	req, err := ac.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := ac.HTTP.Do(req)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errs.Wrap(errs.ErrAgentUnreach, fmt.Sprintf("agent HTTP %d", resp.StatusCode))
	}
	var raw json.RawMessage
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	return raw, nil
}


// ImagePull 镜像拉取（阻塞至完成，走 agent 专用端点）。
func (s *DockerExtService) ImagePull(ctx context.Context, ref string) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[struct {
		Ref string `json:"ref"`
	}, struct {
		Output string `json:"output"`
	}](ac, ctx, http.MethodPost, "/agent/v1/docker/images/pull", &struct {
		Ref string `json:"ref"`
	}{Ref: ref})
	if err != nil {
		return "", err
	}
	return out.Output, nil
}

// ImageRemove 删除镜像。
func (s *DockerExtService) ImageRemove(ctx context.Context, imageID string, force bool) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[struct{}, struct{}](ac, ctx, "DELETE",
		"/agent/v1/docker/images/"+imageID+"?force="+boolStr(force), nil)
	return err
}

// NetworkCreate 创建网络。
func (s *DockerExtService) NetworkCreate(ctx context.Context, name, driver string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[struct {
		Driver string `json:"driver"`
	}, struct{}](ac, ctx, http.MethodPost,
		"/agent/v1/docker/networks/"+name, &struct {
			Driver string `json:"driver"`
		}{Driver: driver})
	return err
}

// NetworkRemove 删除网络。
func (s *DockerExtService) NetworkRemove(ctx context.Context, name string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[struct{}, struct{}](ac, ctx, "DELETE",
		"/agent/v1/docker/networks/"+name, nil)
	return err
}

// VolumeCreate 创建卷。
func (s *DockerExtService) VolumeCreate(ctx context.Context, name string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[struct{}, struct{}](ac, ctx, "POST",
		"/agent/v1/docker/volumes/"+name, nil)
	return err
}

// VolumeRemove 删除卷。
func (s *DockerExtService) VolumeRemove(ctx context.Context, name string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[struct{}, struct{}](ac, ctx, "DELETE",
		"/agent/v1/docker/volumes/"+name, nil)
	return err
}

// ContainersPrune 清理已停止容器。
func (s *DockerExtService) ContainersPrune(ctx context.Context) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST",
		"/agent/v1/docker/containers/prune", &dto.ExecReq{Command: "prune", TimeoutSecs: 300})
	if err != nil {
		return "", err
	}
	return out.Output, nil
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// ExtPortMap 端口映射（与 agent dockerx.PortMap 字段一致）。
type ExtPortMap struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	Proto     string `json:"proto"`
}

// ExtContainerCreateReq 容器创建请求（与 agent dockerx.ContainerCreateReq 字段一致）。
type ExtContainerCreateReq struct {
	Name    string       `json:"name" binding:"required"`
	Image   string       `json:"image" binding:"required"`
	Cmd     []string     `json:"cmd"`
	Env     []string     `json:"env"`
	Ports   []ExtPortMap `json:"ports"`
	Mounts  []string     `json:"mounts"`
	Restart string       `json:"restart"`
	Network string       `json:"network"`
	// M23 扩展（结构化创建表单）
	Entrypoint []string          `json:"entrypoint"`
	Workdir    string            `json:"workdir"`
	Tty        bool              `json:"tty"`
	Labels     map[string]string `json:"labels"`
	Privileged bool              `json:"privileged"`
	MemoryMB   int64             `json:"memoryMB"`
	Cpus       float64           `json:"cpus"`
}

// ContainerCreate 创建并启动容器（返回容器 ID）。
func (s *DockerExtService) ContainerCreate(ctx context.Context, req ExtContainerCreateReq) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[ExtContainerCreateReq, struct {
		ID string `json:"id"`
	}](ac, ctx, http.MethodPost, "/agent/v1/docker/containers", &req)
	if err != nil {
		return "", err
	}
	return out.ID, nil
}

// ContainerRemove 删除容器。
func (s *DockerExtService) ContainerRemove(ctx context.Context, id string, force bool) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[struct{}, struct{}](ac, ctx, http.MethodDelete,
		"/agent/v1/docker/containers/"+id+"?force="+boolStr(force), nil)
	return err
}
