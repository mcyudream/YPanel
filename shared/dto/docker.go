package dto

import "time"

// ContainerItem 容器列表条目。
type ContainerItem struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`   // running / exited / paused / created / restarting / dead
	Status  string            `json:"status"`  // 人类可读，如 "Up 3 hours"
	Command string            `json:"command"`
	Created time.Time         `json:"created"`
	Ports   []PortBinding     `json:"ports"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// PortBinding 端口映射。
type PortBinding struct {
	HostIP      string `json:"hostIp"`
	HostPort    string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Proto       string `json:"proto"` // tcp / udp
}

// ContainerActionReq 容器操作请求。
type ContainerActionReq struct {
	Action string `json:"action" binding:"required,oneof=start stop restart"` // 语义白名单，防任意动词
}

// DockerUsage Docker 资源用量统计（来源 system df；verbose 实算 size，调用方懒加载）。
type DockerUsage struct {
	ImagesTotalSize  int64 `json:"imagesTotalSize"`  // 镜像总大小（去共享，含中间镜像）
	ImagesCount      int   `json:"imagesCount"`
	ContainersRWSize int64 `json:"containersRwSize"` // 容器可写层总量（根目录及写入数据）
	ContainersCount  int   `json:"containersCount"`
	VolumesTotalSize int64 `json:"volumesTotalSize"`
	VolumesCount     int   `json:"volumesCount"`
	BuildCacheSize   int64 `json:"buildCacheSize"`
	BuildCacheCount  int   `json:"buildCacheCount"`
	NetworksCount    int   `json:"networksCount"`
	HostPortsCount   int   `json:"hostPortsCount"` // 去重后的主机端口数（tcp/udp 分开计）
	ContainerItems   []DockerUsageItem `json:"containerItems"`
	ImageItems       []DockerUsageItem `json:"imageItems"`
	VolumeItems      []DockerUsageItem `json:"volumeItems"`
	CollectedAt      time.Time         `json:"collectedAt"`
}

// DockerUsageItem 用量明细条目（treemap 展示用）。
type DockerUsageItem struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Sub  string `json:"sub,omitempty"` // 备注：容器状态 / 镜像短 ID 等
}

// LogsSearchReq 多容器日志搜索请求（M33 日志中心 P1）。
type LogsSearchReq struct {
	Containers []string `json:"containers"` // 容器短 ID
	Since      string   `json:"since"`      // RFC3339 / unix 秒 / 相对时长（如 15m），透传 docker
	Until      string   `json:"until"`      // 空 = 至今
	Pattern    string   `json:"pattern"`    // 正则（RE2），空 = 不过滤
	Negate     bool     `json:"negate"`     // true = 排除命中行
	PerLimit   int      `json:"perLimit"`   // 单容器保留条数（最后 N 条命中）
	TotalLimit int      `json:"totalLimit"` // 归并后总量上限（保最新）
}

// LogSearchItem 归并后的一条命中行。
type LogSearchItem struct {
	Ts        time.Time `json:"ts"`
	Container string    `json:"container"`
	ID        string    `json:"id"`
	Level     string    `json:"level,omitempty"`
	Line      string    `json:"line"`
	Node      string    `json:"node,omitempty"` // 跨节点查询时由 core 归并侧标注
}

// LogsSearchResp 搜索结果（有界，服务端完成过滤与限额）。
type LogsSearchResp struct {
	Items     []LogSearchItem `json:"items"`
	Truncated bool            `json:"truncated"`
	Scanned   int64           `json:"scanned"` // 各容器扫描行数合计
	Errors    []string        `json:"errors,omitempty"`
}
