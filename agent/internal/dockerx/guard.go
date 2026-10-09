// 磁盘空间保护（Disk Guard）容器侧执行器。
// StopAllForGuard：停止全部 running/restarting/paused 容器（豁免名单除外），并把重启策略改写为 no——
// 防止 always/unless-stopped 容器在 daemon 或主机重启后自动拉起、击穿"腾出写入压力"的防线。
// RestoreFromGuard：按快照还原重启策略并拉起；停机期间容器被重建/删除时按名字兜底、只启动不改策略。
// 有意不复用 ContainerUpdateResources：其中有 compose 托管拦截，而保护动作必须覆盖 compose 容器。
package dockerx

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// StopAllForGuard 停止全部活动容器（豁免名单除外）。返回快照（供 core 持久化）与豁免/失败明细。
func (m *Manager) StopAllForGuard(ctx context.Context, exclude []string) (dto.GuardStopAllResp, error) {
	resp := dto.GuardStopAllResp{Stopped: []dto.GuardStoppedItem{}, Skipped: []string{}, Failed: []string{}}
	cli, err := m.getClient()
	if err != nil {
		return resp, err
	}
	exc := map[string]bool{}
	for _, n := range exclude {
		if n = strings.TrimSpace(n); n != "" {
			exc[n] = true
		}
	}
	list, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return resp, errs.Wrapc(errs.CodeFileOpFailed, "docker list: "+err.Error())
	}
	stopTimeout := 30
	for _, c := range list.Items {
		switch c.State {
		case "running", "restarting", "paused":
		default:
			continue
		}
		name := containerName(c.Names)
		if name == "" {
			name = shortID(c.ID)
		}
		if exc[name] {
			resp.Skipped = append(resp.Skipped, name)
			continue
		}
		// 先读原策略并改写为 no，再 stop：即使 stop 超时/失败，重启防线也已生效
		policy := ""
		if res, ierr := cli.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{}); ierr == nil {
			if res.Container.HostConfig != nil {
				policy = string(res.Container.HostConfig.RestartPolicy.Name)
			}
		} else {
			slog.Warn("diskguard: inspect 失败，跳过策略改写", "container", name, "err", ierr)
		}
		if _, uerr := cli.ContainerUpdate(ctx, c.ID, client.ContainerUpdateOptions{
			RestartPolicy: &container.RestartPolicy{Name: container.RestartPolicyDisabled},
		}); uerr != nil {
			slog.Warn("diskguard: 重启策略改写失败", "container", name, "err", uerr)
		}
		if c.State == "paused" {
			if _, perr := cli.ContainerUnpause(ctx, c.ID, client.ContainerUnpauseOptions{}); perr != nil {
				slog.Warn("diskguard: unpause 失败", "container", name, "err", perr)
			}
		}
		if _, serr := cli.ContainerStop(ctx, c.ID, client.ContainerStopOptions{Timeout: &stopTimeout}); serr != nil {
			resp.Failed = append(resp.Failed, fmt.Sprintf("%s: %v", name, serr))
			continue
		}
		resp.Stopped = append(resp.Stopped, dto.GuardStoppedItem{ID: shortID(c.ID), Name: name, RestartPolicy: policy})
	}
	return resp, nil
}

// RestoreFromGuard 按快照还原：改回原重启策略（仅 ID 命中时）并启动。
// 停机期间容器可能被删除或经 compose/面板重建（ID 变化）——按名字兜底命中时只启动、不回写旧策略。
func (m *Manager) RestoreFromGuard(ctx context.Context, items []dto.GuardStoppedItem) (dto.GuardRestoreResp, error) {
	resp := dto.GuardRestoreResp{Started: []string{}, Failed: []string{}}
	cli, err := m.getClient()
	if err != nil {
		return resp, err
	}
	list, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return resp, errs.Wrapc(errs.CodeFileOpFailed, "docker list: "+err.Error())
	}
	for _, it := range items {
		idMatch, nameMatch := "", ""
		for _, c := range list.Items {
			if idMatch == "" && strings.HasPrefix(c.ID, it.ID) {
				idMatch = c.ID
			}
			if nameMatch == "" && containerName(c.Names) == it.Name {
				nameMatch = c.ID
			}
		}
		id, exact := idMatch, idMatch != ""
		if id == "" {
			id, exact = nameMatch, false
		}
		if id == "" {
			resp.Failed = append(resp.Failed, it.Name+": 容器不存在（停机期间可能已被删除）")
			continue
		}
		if p := it.RestartPolicy; exact && p != "" && p != string(container.RestartPolicyDisabled) {
			if _, uerr := cli.ContainerUpdate(ctx, id, client.ContainerUpdateOptions{
				RestartPolicy: &container.RestartPolicy{Name: container.RestartPolicyMode(p)},
			}); uerr != nil {
				resp.Failed = append(resp.Failed, fmt.Sprintf("%s: 恢复重启策略失败(%v)", it.Name, uerr))
			}
		}
		var cur container.Summary
		for _, c := range list.Items {
			if c.ID == id {
				cur = c
				break
			}
		}
		if string(cur.State) == "running" {
			resp.Started = append(resp.Started, it.Name)
			continue
		}
		if _, serr := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); serr != nil {
			resp.Failed = append(resp.Failed, fmt.Sprintf("%s: 启动失败(%v)", it.Name, serr))
			continue
		}
		resp.Started = append(resp.Started, it.Name)
	}
	return resp, nil
}
