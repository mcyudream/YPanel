// Package procs 进程与 systemd 服务管理（gopsutil + systemctl）。
package procs

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// ListOpts 进程列表选项。
type ListOpts struct {
	Sort  string // cpu|mem|rss|pid|name，默认 cpu
	Order string // asc|desc，默认 desc
	Limit int    // 默认 500，上限 2000
}

// List 进程列表（按指定字段排序，截断 Limit 条；截断发生在排序之后，保证排序准确）。
func List(ctx context.Context, opts ListOpts) ([]dto.ProcessItem, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	out := make([]dto.ProcessItem, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		if name == "" {
			continue
		}
		item := dto.ProcessItem{Pid: p.Pid, Name: name}
		if cpu, err := p.CPUPercent(); err == nil {
			item.Cpu = cpu
		}
		if mem, err := p.MemoryPercent(); err == nil {
			item.Mem = float64(mem)
		}
		if mi, err := p.MemoryInfo(); err == nil {
			item.MemRSS = mi.RSS
		}
		if username, err := p.Username(); err == nil {
			item.User = username
		}
		if cl, err := p.Cmdline(); err == nil {
			item.Cmdline = cl
		}
		out = append(out, item)
	}
	sortBy(out, opts.Sort, opts.Order)
	limit := opts.Limit
	if limit <= 0 {
		limit = 500
	}
	if limit > 2000 {
		limit = 2000
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// sortBy 按字段排序；非法字段回退 cpu。
func sortBy(list []dto.ProcessItem, field, order string) {
	asc := order == "asc"
	less := func(a, b float64) bool {
		if asc {
			return a < b
		}
		return a > b
	}
	switch field {
	case "mem":
		sort.Slice(list, func(i, j int) bool { return less(list[i].Mem, list[j].Mem) })
	case "rss":
		sort.Slice(list, func(i, j int) bool { return less(float64(list[i].MemRSS), float64(list[j].MemRSS)) })
	case "pid":
		sort.Slice(list, func(i, j int) bool { return less(float64(list[i].Pid), float64(list[j].Pid)) })
	case "name":
		sort.Slice(list, func(i, j int) bool {
			if asc {
				return list[i].Name < list[j].Name
			}
			return list[i].Name > list[j].Name
		})
	default: // cpu
		sort.Slice(list, func(i, j int) bool { return less(list[i].Cpu, list[j].Cpu) })
	}
}

// Kill 结束进程（SIGKILL）。
func Kill(ctx context.Context, pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return errs.Wrap(errs.ErrNotFound, "进程不存在")
	}
	name, _ := p.Name()
	if pid <= 1 || name == "systemd" || name == "init" {
		return errs.Wrap(errs.ErrBadRequest, "拒绝结束系统关键进程")
	}
	if err := p.Kill(); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return nil
}

// ListServices systemd 服务列表。
func ListServices(ctx context.Context) ([]dto.ServiceItem, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain")
	out, err := cmd.Output()
	if err != nil {
		// 非 systemd 环境
		return nil, errs.Wrap(errs.ErrAgentDisabled, "systemctl 不可用: "+err.Error())
	}
	items := []dto.ServiceItem{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasSuffix(fields[0], ".service") {
			continue
		}
		it := dto.ServiceItem{Name: fields[0], Load: fields[1], Active: fields[2]}
		// 描述为第 4 列之后剩余
		if idx := strings.Index(line, fields[3]); idx >= 0 {
			it.Desc = strings.TrimSpace(line[idx:])
		}
		items = append(items, it)
	}
	return items, nil
}

// ServiceAction 服务电源操作。
func ServiceAction(ctx context.Context, name, action string) (string, error) {
	if !servicePattern.MatchString(name) {
		return "", errs.Wrap(errs.ErrBadRequest, "服务名不合法")
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return "", errs.ErrBadRequest
	}
	cmd := exec.CommandContext(ctx, "systemctl", action, name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("systemctl %s 失败: %v", action, err))
	}
	return string(out), nil
}

var servicePattern = regexp.MustCompile(`^[a-zA-Z0-9.@_\\-]+\.service$`)
