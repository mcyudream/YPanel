// Package sysinfo 基于 gopsutil 的主机采集：概览快照 + 周期采样环形缓冲。
package sysinfo

import (
	"context"
	"log/slog"
	"sync"
	"time"

	gopscpu "github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/ypanel/shared/dto"
)

// Collector 采集器。零值不可用，须经 New 创建并 Start。
type Collector struct {
	cancel context.CancelFunc

	mu       sync.Mutex
	lastNet  gopsnet.IOCountersStat
	lastNetT time.Time
	samples  []dto.MetricSample // 环形：满后覆盖头部
	max      int
}

// New 创建采集器。sampleEvery 为采样周期，history 为历史保留条数。
func New(sampleEvery time.Duration, history int) *Collector {
	if sampleEvery <= 0 {
		sampleEvery = 2 * time.Second
	}
	if history <= 0 {
		history = 900 // 2s * 900 = 30 分钟
	}
	return &Collector{samples: make([]dto.MetricSample, 0, history), max: history}
}

// Start 启动周期采样，直到 ctx 取消。重复调用无效果。
func (c *Collector) Start(ctx context.Context) {
	if c.cancel != nil {
		return
	}
	ctx, c.cancel = context.WithCancel(ctx)
	// 立即采一次，避免首个查询窗口无数据
	c.sampleOnce()
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.sampleOnce()
			}
		}
	}()
}

// Stop 停止采样。
func (c *Collector) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Overview 返回即时主机概览。
func (c *Collector) Overview() (*dto.SystemOverview, error) {
	info, err := host.Info()
	if err != nil {
		return nil, err
	}
	logical, _ := gopscpu.Counts(true)
	physical, _ := gopscpu.Counts(false)
	perCore, _ := gopscpu.Percent(0, true)
	// CPU 使用率优先取采样器最新值：Percent(0) 的全局态与采样器并发调用会互相吞差值
	cpuPercent := []float64{}
	if latest := c.Latest(); latest != nil {
		cpuPercent = []float64{latest.CPUPercent}
	} else {
		cpuPercent, err = gopscpu.Percent(0, false)
		if err != nil {
			return nil, err
		}
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	sw, _ := mem.SwapMemory()
	net := c.netSample()
	avg, _ := load.Avg()

	ov := &dto.SystemOverview{
		Hostname:      info.Hostname,
		OS:            info.OS,
		Platform:      info.Platform + " " + info.PlatformVersion,
		KernelVersion: info.KernelArch,
		Arch:          info.KernelArch,
		Uptime:        info.Uptime,
		CPU: dto.CPUSummary{
			LogicalCount:  logical,
			PhysicalCount: physical,
			ModelName:     cpuModel(),
			UsagePercent:  round1(firstOr(cpuPercent)),
			PerCore:       roundAll(perCore),
		},
		Memory: memOf(vm.Total, vm.Used, vm.Available, vm.UsedPercent),
		Disks:  diskUsage(),
		Network: dto.NetSummary{
			RxTotal:    net.rxTotal,
			TxTotal:    net.txTotal,
			RxSpeedBps: net.rxSpeed,
			TxSpeedBps: net.txSpeed,
		},
		CollectedAt: time.Now(),
	}
	if sw != nil {
		ov.Swap = memOf(sw.Total, sw.Used, sw.Free, sw.UsedPercent)
	}
	if avg != nil {
		ov.Load = dto.LoadAvg{Load1: avg.Load1, Load5: avg.Load5, Load15: avg.Load15}
	}
	return ov, nil
}

// History 返回最近 seconds 秒的采样序列。
func (c *Collector) History(seconds int) []dto.MetricSample {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.samples) == 0 {
		return []dto.MetricSample{}
	}
	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second)
	out := make([]dto.MetricSample, 0, len(c.samples))
	for _, s := range c.samples {
		if s.At.After(cutoff) {
			out = append(out, s)
		}
	}
	return out
}

// Latest 返回最近一个采样点（无数据时返回 nil）。
func (c *Collector) Latest() *dto.MetricSample {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.samples) == 0 {
		return nil
	}
	s := c.samples[len(c.samples)-1]
	return &s
}

func (c *Collector) sampleOnce() {
	vm, err := mem.VirtualMemory()
	if err != nil {
		slog.Warn("sysinfo: mem sample failed", "err", err)
		return
	}
	cpuPercent, err := gopscpu.Percent(0, false)
	if err != nil {
		slog.Warn("sysinfo: cpu sample failed", "err", err)
		return
	}
	net := c.netSample()
	s := dto.MetricSample{
		At:         time.Now(),
		CPUPercent: round1(firstOr(cpuPercent)),
		MemPercent: round1(vm.UsedPercent),
		RxSpeedBps: net.rxSpeed,
		TxSpeedBps: net.txSpeed,
	}
	if avg, err := load.Avg(); err == nil {
		s.Load1 = avg.Load1
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.samples) >= c.max {
		copy(c.samples, c.samples[1:])
		c.samples[len(c.samples)-1] = s
		return
	}
	c.samples = append(c.samples, s)
}

type netSample struct {
	rxTotal, txTotal   uint64
	rxSpeed, txSpeed   float64
}

func (c *Collector) netSample() netSample {
	io, err := gopsnet.IOCounters(false)
	if err != nil || len(io) == 0 {
		return netSample{}
	}
	now := time.Now()
	cur := io[0]

	c.mu.Lock()
	defer c.mu.Unlock()
	dt := now.Sub(c.lastNetT).Seconds()
	s := netSample{rxTotal: cur.BytesRecv, txTotal: cur.BytesSent}
	if dt > 0.2 && c.lastNetT.Unix() > 0 {
		s.rxSpeed = round1(float64(cur.BytesRecv-c.lastNet.BytesRecv) / dt)
		s.txSpeed = round1(float64(cur.BytesSent-c.lastNet.BytesSent) / dt)
		if s.rxSpeed < 0 {
			s.rxSpeed = 0 // 计数器重置（重启）
		}
		if s.txSpeed < 0 {
			s.txSpeed = 0
		}
	}
	c.lastNet = cur
	c.lastNetT = now
	return s
}

// pseudoFS 伪文件系统黑名单：不作为磁盘用量展示。
var pseudoFS = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "sysfs": true, "proc": true,
	"devfs": true, "cgroupfs": true, "cgroup": true, "overlay": false, // overlay 保留（容器根文件系统有观测价值）
	"squashfs": true, "squash": true, "ramfs": true, "mqueue": true,
	"nsfs": true, "tracefs": true, "debugfs": true, "bpf": true, "configfs": true,
	"fusectl": true, "fuseblk": false, "hugetlbfs": true, "efivarfs": true,
	"rpc_pipefs": true, "autofs": true, "binfmt_misc": true, "pstore": true,
	"bdev": true, "securityfs": true, "devpts": true, "mtd0": true,
}

var pseudoMount = map[string]bool{
	"/proc": true, "/sys": true, "/dev": true, "/run": true, "/boot/efi": false,
}

func diskUsage() []dto.DiskUsage {
	parts, err := disk.Partitions(false)
	if err != nil {
		return []dto.DiskUsage{}
	}
	seen := map[string]bool{}
	out := make([]dto.DiskUsage, 0, len(parts))
	for _, p := range parts {
		if pseudoFS[p.Fstype] || pseudoMount[p.Mountpoint] || seen[p.Mountpoint] {
			continue
		}
		du, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		seen[p.Mountpoint] = true
		out = append(out, dto.DiskUsage{
			Mountpoint:   p.Mountpoint,
			FSType:       p.Fstype,
			Total:        du.Total,
			Used:         du.Used,
			Free:         du.Free,
			UsagePercent: round1(du.UsedPercent),
		})
	}
	return out
}

// memOf 由内存统计字段构造汇总（兼容 VirtualMemoryStat / SwapMemoryStat）。
func memOf(total, used, available uint64, usedPercent float64) dto.MemSummary {
	return dto.MemSummary{
		Total:        total,
		Used:         used,
		Available:    available,
		UsagePercent: round1(usedPercent),
	}
}

func cpuModel() string {
	infos, err := gopscpu.Info()
	if err != nil || len(infos) == 0 {
		return ""
	}
	return infos[0].ModelName
}

func firstOr(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	return s[0]
}

func round1(v float64) float64 {
	if v < 0 {
		return 0
	}
	return float64(int(v*10+0.5)) / 10
}

func roundAll(s []float64) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = round1(v)
	}
	return out
}
