package dto

import "time"

// SystemOverview 主机概览（agent /sysinfo/overview）。
type SystemOverview struct {
	Hostname      string      `json:"hostname"`
	OS            string      `json:"os"`            // linux
	Platform      string      `json:"platform"`      // ubuntu 22.04
	KernelVersion string      `json:"kernelVersion"`
	Arch          string      `json:"arch"`
	Uptime        uint64      `json:"uptime"` // 秒
	CPU           CPUSummary  `json:"cpu"`
	Memory        MemSummary  `json:"memory"`
	Swap          MemSummary  `json:"swap"`
	Disks         []DiskUsage `json:"disks"`
	Network       NetSummary  `json:"network"`
	Load          LoadAvg     `json:"load"`
	CollectedAt   time.Time   `json:"collectedAt"`
}

// CPUSummary CPU 汇总。
type CPUSummary struct {
	LogicalCount int      `json:"logicalCount"`
	PhysicalCount int     `json:"physicalCount"`
	ModelName    string   `json:"modelName"`
	UsagePercent float64  `json:"usagePercent"`
	PerCore      []float64 `json:"perCore"`
}

// MemSummary 内存/交换分区汇总，字节。
type MemSummary struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Available    uint64  `json:"available"`
	UsagePercent float64 `json:"usagePercent"`
}

// DiskUsage 单个挂载点用量。
type DiskUsage struct {
	Mountpoint   string  `json:"mountpoint"`
	FSType       string  `json:"fsType"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usagePercent"`
}

// NetSummary 网卡汇总流量（累计字节 + 速率采样窗口）。
type NetSummary struct {
	RxTotal    uint64  `json:"rxTotal"`
	TxTotal    uint64  `json:"txTotal"`
	RxSpeedBps float64 `json:"rxSpeedBps"` // 本采样窗口速率 B/s
	TxSpeedBps float64 `json:"txSpeedBps"`
}

// LoadAvg 负载均值。
type LoadAvg struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// MetricSample 单个监控采样点（core 端历史环形缓冲存储）。
type MetricSample struct {
	At           time.Time `json:"at"`
	CPUPercent   float64   `json:"cpuPercent"`
	MemPercent   float64   `json:"memPercent"`
	RxSpeedBps   float64   `json:"rxSpeedBps"`
	TxSpeedBps   float64   `json:"txSpeedBps"`
	Load1        float64   `json:"load1"`
}

// ProcessItem 进程列表条目。
type ProcessItem struct {
	Pid     int32   `json:"pid"`
	Name    string  `json:"name"`
	Cpu     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	MemRSS  uint64  `json:"memRss"`
	User    string  `json:"user"`
	Cmdline string  `json:"cmdline"`
}

// ServiceItem systemd 服务条目。
type ServiceItem struct {
	Name   string `json:"name"`
	Load   string `json:"load"`
	Active string `json:"active"`
	Desc   string `json:"desc"`
}

// HealthResp agent 健康检查。
type HealthResp struct {
	Status  string `json:"status"` // ok
	Version string `json:"version"`
}
