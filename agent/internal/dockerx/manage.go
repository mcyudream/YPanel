// Docker 管理扩展：镜像/网络/卷/容器详情与清理。
// 同包内复用 dockerx.go 的 Manager 与 getClient。
// 约束：M17 起 moby 模块拆分（github.com/moby/moby/{client,api}），勿回退 docker/docker 旧路径。
package dockerx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"os"
	"regexp"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/ypanel/shared/errs"
)

// ImageItem 镜像条目。
type ImageItem struct {
	ID        string   `json:"id"`
	Tags      []string `json:"tags"`
	SizeMB    float64  `json:"sizeMb"`
	CreatedAt int64    `json:"createdAt"`
}

// ImageList 镜像列表。
func (m *Manager) ImageList(ctx context.Context) ([]ImageItem, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	list, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "镜像列表失败: "+err.Error())
	}
	out := make([]ImageItem, 0, len(list.Items))
	for _, img := range list.Items {
		it := ImageItem{ID: strings.TrimPrefix(img.ID, "sha256:"), Tags: img.RepoTags, CreatedAt: img.Created}
		it.SizeMB = float64(img.Size) / 1024 / 1024
		out = append(out, it)
	}
	return out, nil
}

// pullLine 拉取流的一行输出。
type pullLine struct {
	Status   string `json:"status"`
	ID       string `json:"id"`
	Progress string `json:"progress"`
	Error    string `json:"error"`
}

// ImagePull 拉取镜像（阻塞至完成，返回聚合输出文本）。
func (m *Manager) ImagePull(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", errs.ErrBadRequest
	}
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	rd, err := cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "拉取失败: "+err.Error())
	}
	defer func() { _ = rd.Close() }()
	var sb strings.Builder
	dec := json.NewDecoder(rd)
	for {
		var line pullLine
		if err := dec.Decode(&line); err != nil {
			break
		}
		if line.Error != "" {
			return sb.String(), errs.Wrapc(errs.CodeFileOpFailed, "拉取失败: "+line.Error)
		}
		if line.Progress != "" {
			fmt.Fprintf(&sb, "%s %s\n", line.ID, line.Progress)
		} else if line.Status != "" {
			fmt.Fprintf(&sb, "%s\n", line.Status)
		}
		if sb.Len() > 256*1024 {
			sb.Reset()
		}
	}
	return sb.String(), nil
}

// ImageRemove 删除镜像。
func (m *Manager) ImageRemove(ctx context.Context, imageID string, force bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	_, err = cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{Force: force})
	return err
}

// ImagesPrune 清理悬空镜像（新 client 无 prune 端点：列出 dangling 逐个删除）。
func (m *Manager) ImagesPrune(ctx context.Context) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	list, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return "", err
	}
	removed, freed := 0, int64(0)
	for _, img := range list.Items {
		if len(img.RepoTags) != 0 {
			continue // 仅悬空（无 tag）
		}
		if _, err := cli.ImageRemove(ctx, img.ID, client.ImageRemoveOptions{}); err == nil {
			removed++
			freed += img.Size
		}
	}
	return fmt.Sprintf("清理 %d 个悬空镜像，释放 %.1f MB", removed, float64(freed)/1024/1024), nil
}

// NetworkItem 网络条目。
type NetworkItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
	Subnet string `json:"subnet"`
}

// NetworkList 网络列表。
func (m *Manager) NetworkList(ctx context.Context) ([]NetworkItem, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	list, err := cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "网络列表失败: "+err.Error())
	}
	out := make([]NetworkItem, 0, len(list.Items))
	for _, n := range list.Items {
		it := NetworkItem{ID: n.ID, Name: n.Name, Driver: n.Driver, Scope: n.Scope}
		for _, ipam := range n.IPAM.Config {
			it.Subnet = ipam.Subnet.String()
			break
		}
		out = append(out, it)
	}
	return out, nil
}

// NetworkCreate 创建网络。
func (m *Manager) NetworkCreate(ctx context.Context, name, driver string) error {
	if name == "" || len(name) > 64 {
		return errs.Wrap(errs.ErrBadRequest, "网络名不合法")
	}
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	_, err = cli.NetworkCreate(ctx, name, client.NetworkCreateOptions{Driver: driver})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return errs.New(errs.CodeConflict, "error.networkExists", "网络已存在")
	}
	return err
}

// NetworkRemove 删除网络。
func (m *Manager) NetworkRemove(ctx context.Context, name string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	if _, err := cli.NetworkRemove(ctx, name, client.NetworkRemoveOptions{}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "删除网络失败: "+err.Error())
	}
	return nil
}

// VolumeItem 卷条目。
type VolumeItem struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
}

// VolumeList 卷列表。
func (m *Manager) VolumeList(ctx context.Context) ([]VolumeItem, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	list, err := cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "卷列表失败: "+err.Error())
	}
	out := make([]VolumeItem, 0, len(list.Items))
	for _, v := range list.Items {
		out = append(out, VolumeItem{Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint})
	}
	return out, nil
}

// VolumeCreate 创建卷。
func (m *Manager) VolumeCreate(ctx context.Context, name string) error {
	if name == "" || len(name) > 128 {
		return errs.Wrap(errs.ErrBadRequest, "卷名不合法")
	}
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	_, err = cli.VolumeCreate(ctx, client.VolumeCreateOptions{Name: name})
	return err
}

// VolumeRemove 删除卷。
func (m *Manager) VolumeRemove(ctx context.Context, name string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	if _, err := cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "删除卷失败: "+err.Error())
	}
	return nil
}

// VolumesPrune 清理未使用卷（in-use 跳过）。
func (m *Manager) VolumesPrune(ctx context.Context) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	list, err := cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return "", err
	}
	removed := 0
	for _, v := range list.Items {
		if v.Name == "" {
			continue
		}
		if _, err := cli.VolumeRemove(ctx, v.Name, client.VolumeRemoveOptions{}); err == nil {
			removed++
		}
	}
	return fmt.Sprintf("清理 %d 个未使用卷", removed), nil
}

// ContainerInspectRaw 容器 inspect 原始 JSON。
func (m *Manager) ContainerInspectRaw(ctx context.Context, id string) (json.RawMessage, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeNotFound, "容器不存在: "+err.Error())
	}
	return res.Raw, nil
}

// ContainerRootfsDir 返回容器可写层在宿主机上的目录（overlay2 UpperDir）。
// 用途：容器无法启动（挂载损坏/文件损坏）时经宿主文件通道直读/修复可写层文件。
// 注意：仅含可写层（运行期写入/修改的文件）；仅对非运行容器操作才与容器视图一致。
func (m *Manager) ContainerRootfsDir(ctx context.Context, id string) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return "", errs.Wrapc(errs.CodeNotFound, "容器不存在: "+err.Error())
	}
	upper := ""
	if res.Container.GraphDriver != nil && res.Container.GraphDriver.Data != nil {
		upper = res.Container.GraphDriver.Data["UpperDir"]
	}
	if upper == "" {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "该容器的存储驱动不提供可写层目录（GraphDriver.Data.UpperDir 为空）")
	}
	if _, err := os.Stat(upper); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "可写层目录不可访问: "+err.Error())
	}
	return upper, nil
}

// ContainerStatsRaw 容器单次 stats 原始 JSON。
func (m *Manager) ContainerStatsRaw(ctx context.Context, id string) (json.RawMessage, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "stats 采集失败: "+err.Error())
	}
	defer func() { _ = res.Body.Close() }()
	var raw json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// ContainerRemove 删除容器（force 强制；volumes 同时删除匿名数据卷）。
func (m *Manager) ContainerRemove(ctx context.Context, id string, force bool, volumes bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	_, rmErr := cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: force, RemoveVolumes: volumes})
	return rmErr
}

// ContainersPrune 清理已停止容器（新 client 无 prune 端点：遍历逐个删除）。
func (m *Manager) ContainersPrune(ctx context.Context) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	list, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return "", err
	}
	removed := 0
	for _, c := range list.Items {
		if c.State != "exited" && c.State != "created" && c.State != "dead" {
			continue
		}
		if _, err := cli.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{}); err == nil {
			removed++
		}
	}
	return fmt.Sprintf("清理 %d 个已停止容器", removed), nil
}

// ContainerExecCreate 创建 exec 实例，返回 exec ID（供 Attach 流式使用）。
func (m *Manager) ExecCreate(ctx context.Context, id string, cmd []string) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	execID, err := cli.ExecCreate(ctx, id, client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
	})
	if err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "exec 创建失败: "+err.Error())
	}
	return execID.ID, nil
}

// ExecAttach 附着 exec 实例（TTY 双向流）。调用方负责关闭连接。
func (m *Manager) ExecAttach(ctx context.Context, execID string) (io.Reader, io.Writer, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, nil, err
	}
	resp, err := cli.ExecAttach(ctx, execID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, nil, errs.Wrapc(errs.CodeFileOpFailed, "exec attach 失败: "+err.Error())
	}
	return resp.Reader, resp.Conn, nil
}

// ExecResize 调整 exec TTY 尺寸（终端面板拖拽高度后由前端下发）。
func (m *Manager) ExecResize(ctx context.Context, execID string, cols, rows uint16) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	if cols == 0 || rows == 0 {
		return nil
	}
	if _, err := cli.ExecResize(ctx, execID, client.ExecResizeOptions{Width: uint(cols), Height: uint(rows)}); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "exec resize 失败: "+err.Error())
	}
	return nil
}

var containerNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// ContainerCreateReq 容器创建参数。
type ContainerCreateReq struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Cmd     []string `json:"cmd"`
	Env     []string `json:"env"`
	Ports   []PortMap `json:"ports"`
	Mounts  []string `json:"mounts"`
	Restart string   `json:"restart"`
	Network string   `json:"network"`
	// M23 扩展（结构化创建表单）
	Entrypoint []string          `json:"entrypoint"`
	Workdir    string            `json:"workdir"`
	Tty        bool              `json:"tty"`
	Labels     map[string]string `json:"labels"`
	Privileged bool              `json:"privileged"`
	MemoryMB   int64             `json:"memoryMB"` // 内存上限（MB），0=不限
	Cpus       float64           `json:"cpus"`     // CPU 核数上限，0=不限
}

// PortMap 端口映射。
type PortMap struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	Proto     string `json:"proto"`
}

// ContainerCreate 创建并启动容器。
func (m *Manager) ContainerCreate(ctx context.Context, r ContainerCreateReq) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	id, err := m.containerCreateOnly(ctx, r)
	if err != nil {
		return "", err
	}
	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "启动容器失败: "+err.Error())
	}
	return id, nil
}

// containerCreateOnly 仅创建不启动（recreate 需按原运行状态决定是否启动）。
func (m *Manager) containerCreateOnly(ctx context.Context, r ContainerCreateReq) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	if r.Name == "" || r.Image == "" {
		return "", errs.Wrap(errs.ErrBadRequest, "name/image 必填")
	}
	if !containerNamePattern.MatchString(r.Name) {
		return "", errs.Wrap(errs.ErrBadRequest, "容器名不合法（小写字母/数字/中划线）")
	}
	cc := container.Config{
		Image: r.Image, Cmd: r.Cmd, Env: r.Env,
		Entrypoint: r.Entrypoint, WorkingDir: r.Workdir,
		Tty: r.Tty, OpenStdin: r.Tty, Labels: r.Labels,
	}
	hc := container.HostConfig{PortBindings: network.PortMap{}, Privileged: r.Privileged}
	if r.Restart != "" {
		hc.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(r.Restart)}
	}
	if r.MemoryMB > 0 {
		hc.Memory = r.MemoryMB * 1024 * 1024
	}
	if r.Cpus > 0 {
		hc.NanoCPUs = int64(r.Cpus * 1e9)
	}
	for _, pm := range r.Ports {
		proto := pm.Proto
		if proto == "" {
			proto = "tcp"
		}
		if pm.Host == "" || pm.Container == "" {
			continue
		}
		portKey, perr := network.ParsePort(pm.Container + "/" + proto)
		if perr != nil {
			return "", errs.Wrap(errs.ErrBadRequest, "端口不合法: "+pm.Container+"/"+proto)
		}
		hostIP, iperr := netip.ParseAddr("0.0.0.0")
		if iperr != nil {
			return "", errs.Wrap(errs.ErrBadRequest, "HostIP 解析失败")
		}
		hc.PortBindings[network.Port(portKey)] = []network.PortBinding{{HostIP: hostIP, HostPort: pm.Host}}
	}
	hc.Binds = append(hc.Binds, r.Mounts...)
	networkingConfig := network.NetworkingConfig{}
	if r.Network != "" {
		networkingConfig.EndpointsConfig = map[string]*network.EndpointSettings{r.Network: {}}
	}
	resp, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           &cc,
		HostConfig:       &hc,
		NetworkingConfig: &networkingConfig,
		Name:             r.Name,
	})
	if err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "创建容器失败: "+err.Error())
	}
	return resp.ID, nil
}
