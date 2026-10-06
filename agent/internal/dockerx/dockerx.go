// Package dockerx 基于 Docker 官方 SDK 的容器管理。
// Docker 未安装/守护进程不可达时能力降级：Available()=false，接口返回能力错误。
package dockerx

import (
	"context"
	"strings"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/moby/moby/client"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Manager Docker 管理器。
type Manager struct {
	cli  *client.Client
	once struct {
		cli *client.Client
		err error
	}
}

// socketCandidates 常见 docker socket 路径。
var socketCandidates = []string{
	"/var/run/docker.sock",
	"/run/docker.sock",
}

// New 创建管理器；首次调用探测 socket。
func New() *Manager { return &Manager{} }

func (m *Manager) getClient() (*client.Client, error) {
	if m.once.cli != nil {
		return m.once.cli, nil
	}
	if m.once.err != nil {
		return nil, m.once.err
	}
	// B19：DOCKER_HOST 优先（tcp://host:port / unix:///path / ssh://user@host）
	if dh := strings.TrimSpace(os.Getenv("DOCKER_HOST")); dh != "" {
		cli, err := client.NewClientWithOpts(
			client.WithHost(dh),
			client.WithAPIVersionNegotiation(),
		)
		if err == nil {
			if _, perr := cli.Ping(context.Background(), client.PingOptions{}); perr == nil {
				m.once.cli = cli
				m.once.err = nil
				return cli, nil
			}
		}
	}
	for _, sock := range socketCandidates {
		if _, err := os.Stat(sock); err != nil {
			continue
		}
		host := "unix://" + filepath.Clean(sock)
		cli, err := client.NewClientWithOpts(
			client.WithHost(host),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			continue
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := cli.Ping(pingCtx, client.PingOptions{}); err != nil {
			_ = cli.Close()
			continue
		}
		m.once.cli = cli
		return cli, nil
	}
	m.once.err = errs.ErrAgentDisabled
	return nil, m.once.err
}

// Available 探测 Docker 是否可用（带 1s 缓存语义：进程内首次探测后固定）。
func (m *Manager) Available() bool {
	cli, err := m.getClient()
	return cli != nil && err == nil
}

// List 容器列表（含已停止）。
func (m *Manager) List(ctx context.Context) ([]dto.ContainerItem, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	list, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "docker list: "+err.Error())
	}
	out := make([]dto.ContainerItem, 0, len(list.Items))
	for _, c := range list.Items {
		item := dto.ContainerItem{
			ID:      shortID(c.ID),
			Name:    containerName(c.Names),
			Image:   c.Image,
			State:   string(c.State),
			Status:  c.Status,
			Command: c.Command,
			Created: time.Unix(c.Created, 0),
			Labels:  c.Labels,
		}
		for _, p := range c.Ports {
			hostPort := ""
			if p.PublicPort > 0 {
				hostPort = strconv.FormatUint(uint64(p.PublicPort), 10)
			}
			item.Ports = append(item.Ports, dto.PortBinding{
				HostIP:        p.IP.String(),
				HostPort:      hostPort,
				ContainerPort: strconv.FormatUint(uint64(p.PrivatePort), 10),
				Proto:         p.Type,
			})
		}
		out = append(out, item)
	}
	return out, nil
}

// Action 容器电源操作（start/stop/restart 白名单由入参校验层保证）。
func (m *Manager) Action(ctx context.Context, id, action string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	stopTimeoutSecs := 30
	switch action {
	case "start":
		_, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
		return err
	case "stop":
		_, err := cli.ContainerStop(ctx, id, client.ContainerStopOptions{Timeout: &stopTimeoutSecs})
		return err
	case "restart":
		_, err := cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{Timeout: &stopTimeoutSecs})
		return err
	default:
		return errs.ErrBadRequest
	}
}

// Logs 读取容器日志。follow 时持续写入 w 直到 ctx 取消。
func (m *Manager) Logs(ctx context.Context, id string, tail string, follow bool, w io.Writer) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	opts := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: true,
	}
	reader, err := cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "docker logs: "+err.Error())
	}
	defer func() { _ = reader.Close() }()

	// docker 多路复用流（非 TTY 容器）带 8 字节帧头，用官方 stdcopy 还原为纯文本流
	if _, err := stdcopy.StdCopy(w, w, reader); err != nil && ctx.Err() == nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "docker logs copy: "+err.Error())
	}
	return nil
}

// shortID 取 12 位短 ID。
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	// docker 返回形如 "/nginx"
	if names[0][0] == '/' {
		return names[0][1:]
	}
	return names[0]
}
