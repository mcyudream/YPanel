// 容器编辑保存：删除重建（recreate）与资源热更新（docker update）。
// Docker 容器创建后 env/端口/网络/挂载等不可变，编辑的本质即按新参数重建同名容器。
package dockerx

import (
	"context"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/ypanel/shared/errs"
)

// LabelComposeProject compose 项目 label；带此 label 的容器由编排管理，禁止面板直接重建
//（重建后脱离 compose 追踪，下次编排部署会还原或冲突）。
const LabelComposeProject = "com.docker.compose.project"

// ContainerUpdateReq 资源限制/重启策略热更新参数（docker update，免重建）。
type ContainerUpdateReq struct {
	MemoryMB int64   `json:"memoryMB"` // 0=不限（-1 传给 daemon）
	Cpus     float64 `json:"cpus"`     // 0=不限（NanoCPUs=0）
	Restart  string  `json:"restart"`  // no/always/unless-stopped/on-failure，空=不修改
}

var restartModes = map[string]bool{
	"no": true, "always": true, "unless-stopped": true, "on-failure": true,
}

// composeManaged 容器是否由 compose 编排管理。
func composeManaged(c container.InspectResponse) bool {
	return c.Config != nil && c.Config.Labels[LabelComposeProject] != ""
}

// ContainerRecreate 删除并按新参数重建同名容器（编辑保存）。返回新容器 ID。
// 流程：inspect 记录原状态 → 停止并删除 → 新参数重建（失败用原参数回建兜底）→ 原运行中则启动。
func (m *Manager) ContainerRecreate(ctx context.Context, id string, r ContainerCreateReq) (string, error) {
	cli, err := m.getClient()
	if err != nil {
		return "", err
	}
	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return "", errs.Wrapc(errs.CodeNotFound, "容器不存在: "+err.Error())
	}
	old := res.Container
	if composeManaged(old) {
		return "", errs.New(errs.CodeConflict, "error.containerManagedByCompose", "容器由 Compose 编排管理，请在编排/应用处修改")
	}
	name := strings.TrimPrefix(old.Name, "/")
	if r.Name != "" && r.Name != name {
		return "", errs.Wrap(errs.ErrBadRequest, "编辑不允许修改容器名")
	}
	r.Name = name
	if r.Image == "" && old.Config != nil {
		r.Image = old.Config.Image
	}
	wasRunning := old.State != nil && old.State.Running
	oldReq := inspectToReq(old)

	stopTimeout := 15
	if wasRunning {
		if _, err := cli.ContainerStop(ctx, id, client.ContainerStopOptions{Timeout: &stopTimeout}); err != nil {
			return "", errs.Wrapc(errs.CodeFileOpFailed, "停止原容器失败: "+err.Error())
		}
	}
	if _, err := cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true}); err != nil {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "删除原容器失败: "+err.Error())
	}

	newID, cerr := m.containerCreateOnly(ctx, r)
	if cerr != nil {
		// 兜底：按原参数回建，尽量恢复原容器（失败则如实告知，需人工处理）
		if rbID, rbErr := m.containerCreateOnly(ctx, oldReq); rbErr == nil {
			if wasRunning {
				_, _ = cli.ContainerStart(ctx, rbID, client.ContainerStartOptions{})
			}
			return "", errs.Wrapc(errs.CodeFileOpFailed, "重建失败（已按原配置回建原容器）: "+cerr.Error())
		}
		return "", errs.Wrapc(errs.CodeFileOpFailed, "重建失败且原容器恢复失败，请检查 docker 日志后手动处理: "+cerr.Error())
	}

	if wasRunning {
		if _, err := cli.ContainerStart(ctx, newID, client.ContainerStartOptions{}); err != nil {
			return newID, errs.Wrapc(errs.CodeFileOpFailed, "容器已重建但启动失败: "+err.Error())
		}
	}
	return newID, nil
}

// inspectToReq 从 inspect 结果提取创建参数（重建失败时按原样回建兜底）。
// 覆盖面与 ContainerCreateReq 一致；多网络容器取第一个网络，每端口取第一条绑定。
func inspectToReq(c container.InspectResponse) ContainerCreateReq {
	r := ContainerCreateReq{Name: strings.TrimPrefix(c.Name, "/")}
	if c.Config != nil {
		r.Image = c.Config.Image
		r.Cmd = c.Config.Cmd
		r.Env = c.Config.Env
		r.Entrypoint = c.Config.Entrypoint
		r.Workdir = c.Config.WorkingDir
		r.Tty = c.Config.Tty
		if len(c.Config.Labels) > 0 {
			r.Labels = make(map[string]string, len(c.Config.Labels))
			for k, v := range c.Config.Labels {
				r.Labels[k] = v
			}
		}
	}
	if c.HostConfig != nil {
		r.Privileged = c.HostConfig.Privileged
		if c.HostConfig.Memory > 0 {
			r.MemoryMB = c.HostConfig.Memory / 1024 / 1024
		}
		if c.HostConfig.NanoCPUs > 0 {
			r.Cpus = float64(c.HostConfig.NanoCPUs) / 1e9
		}
		if p := c.HostConfig.RestartPolicy.Name; p != "" && p != container.RestartPolicyDisabled {
			r.Restart = string(p)
		}
		for portKey, binds := range c.HostConfig.PortBindings {
			if len(binds) == 0 {
				continue
			}
			proto := string(portKey.Proto())
			r.Ports = append(r.Ports, PortMap{
				Host:      binds[0].HostPort,
				Container: portKey.Port(),
				Proto:     proto,
			})
		}
		r.Mounts = append(r.Mounts, c.HostConfig.Binds...)
	}
	if c.NetworkSettings != nil {
		for netName := range c.NetworkSettings.Networks {
			r.Network = netName
			break
		}
	}
	return r
}

// ContainerUpdateResources 资源限制/重启策略热更新（docker update，免重建）。返回 daemon 警告。
func (m *Manager) ContainerUpdateResources(ctx context.Context, id string, r ContainerUpdateReq) ([]string, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeNotFound, "容器不存在: "+err.Error())
	}
	if composeManaged(res.Container) {
		return nil, errs.New(errs.CodeConflict, "error.containerManagedByCompose", "容器由 Compose 编排管理，请在编排/应用处修改")
	}
	opts := client.ContainerUpdateOptions{}
	if r.Restart != "" {
		if !restartModes[r.Restart] {
			return nil, errs.Wrap(errs.ErrBadRequest, "重启策略不合法")
		}
		opts.RestartPolicy = &container.RestartPolicy{Name: container.RestartPolicyMode(r.Restart)}
	}
	if r.MemoryMB > 0 || r.Cpus > 0 {
		res0 := container.Resources{Memory: -1} // -1 = daemon 语义的不限
		if r.MemoryMB > 0 {
			res0.Memory = r.MemoryMB * 1024 * 1024
		}
		if r.Cpus > 0 {
			res0.NanoCPUs = int64(r.Cpus * 1e9)
		}
		opts.Resources = &res0
	}
	out, err := cli.ContainerUpdate(ctx, id, opts)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "更新容器配置失败: "+err.Error())
	}
	return out.Warnings, nil
}
