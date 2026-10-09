package dto

import "time"

// ComposeServiceState compose 项目内单个服务状态（由容器聚合）。
type ComposeServiceState struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	State   string `json:"state"`
}

// ComposeProject compose 项目（托管目录或外部识别）。
type ComposeProject struct {
	Name     string                `json:"name"`
	Dir      string                `json:"dir"`                // 项目工作目录
	Managed  bool                  `json:"managed"`            // 是否位于 YPanel 托管目录（可编辑 yaml）
	Running  int                   `json:"running"`            // 运行中服务数
	Total    int                   `json:"total"`              // 服务总数
	Services []ComposeServiceState `json:"services"`
}

// ComposeTopologyNode 拓扑节点：compose 服务定义 + 容器实时状态。
type ComposeTopologyNode struct {
	Name   string   `json:"name"`
	Image  string   `json:"image"`
	State  string   `json:"state,omitempty"`   // running/exited/...；未创建容器为空
	Health string   `json:"health,omitempty"`  // healthy/starting/unhealthy；无 healthcheck 为空
	Ports  []string `json:"ports,omitempty"`   // 编排定义的发布端口，如 "8080:80/tcp"
}

// ComposeTopologyEdge 依赖边：from 依赖 to（depends_on）。
type ComposeTopologyEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"` // service_started / service_healthy / service_completed_successfully
}

// ComposeTopology compose 项目服务拓扑（M26 P1：仅可视化，编排行为仍由 compose 自身依赖排序）。
type ComposeTopology struct {
	Name  string                `json:"name"`
	Dir   string                `json:"dir"`
	Nodes []ComposeTopologyNode `json:"nodes"`
	Edges []ComposeTopologyEdge `json:"edges"`
}

// ComposeConfigResp 读取 compose yaml 内容。
type ComposeConfigResp struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	File    string `json:"file"` // 实际配置文件路径
	Content string `json:"content"`
}

// ComposeWriteReq 写托管项目 yaml。
type ComposeWriteReq struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// ComposeActionReq compose 项目操作（up/down）。
type ComposeActionReq struct {
	Name string `json:"name" binding:"required"`
	Dir  string `json:"dir"` // 外部项目的工作目录；托管项目留空
}

// ComposeLogsReq compose 日志查询参数。
type ComposeLogsReq struct {
	Name    string `form:"name"`
	Dir     string `form:"dir"`
	Tail    string `form:"tail"`
	Follow  string `form:"follow"`
	Service string `form:"service"`
}

// ExecReq 受控命令执行（计划任务/脚本通道）。
// Env：注入进程环境变量（敏感值走 env 不进 argv，避免宿主进程列表泄露）。
type ExecReq struct {
	Command     string            `json:"command" binding:"required"`
	TimeoutSecs int               `json:"timeoutSecs"` // 0 = 默认 300
	Env         map[string]string `json:"env,omitempty"`
}

// ExecResp 执行结果。
type ExecResp struct {
	Output   string `json:"output"`
	TimedOut bool   `json:"timedOut"`
	ExitCode int    `json:"exitCode"`
}

// CronTask 计划任务。
type CronTask struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Cron        string    `json:"cron"`
	Command     string    `json:"command"`
	Type        string    `json:"type,omitempty"`
	Payload     string    `json:"payload,omitempty"`
	Enabled     bool      `json:"enabled"`
	TimeoutSecs int       `json:"timeoutSecs"`
	LastRunAt   *time.Time `json:"lastRunAt"`
	LastSuccess *bool     `json:"lastSuccess"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CronTaskCreateReq 创建任务。
type CronTaskCreateReq struct {
	Name        string `json:"name" binding:"required,max=64"`
	Cron        string `json:"cron" binding:"required"`
	Command     string `json:"command" binding:"max=8192"` // 类型化任务（备份/curl/清理等）可为空
	Type        string `json:"type"`                       // shell（默认）/ db_backup / site_backup / dir_backup / compose_backup / curl / cut_website_log / clean / cert_renew / container_op / script
	Payload     string `json:"payload"`                    // 类型参数 JSON
	TimeoutSecs int    `json:"timeoutSecs"`                // 0 = 300
}

// CronTaskUpdateReq 更新任务（零值字段不更新）。
type CronTaskUpdateReq struct {
	Name        *string `json:"name" binding:"omitempty,max=64"`
	Cron        *string `json:"cron" binding:"omitempty"`
	Command     *string `json:"command" binding:"omitempty,max=8192"`
	Type        *string `json:"type" binding:"omitempty,oneof=shell db_backup site_backup dir_backup compose_backup curl cut_website_log clean cert_renew container_op script"`
	Payload     *string `json:"payload"`
	TimeoutSecs *int    `json:"timeoutSecs" binding:"omitempty,min=1,max=86400"`
	Enabled     *bool   `json:"enabled"`
}

// CronTaskLogItem 执行记录条目。
type CronTaskLogItem struct {
	ID         uint      `json:"id"`
	TaskID     uint      `json:"taskId"`
	TaskName   string    `json:"taskName"`
	Trigger    string    `json:"trigger"` // cron / manual / running
	StartAt    time.Time `json:"startAt"`
	EndAt      *time.Time `json:"endAt"`
	DurationMs int64     `json:"durationMs"`
	Success    bool      `json:"success"`
	Output     string    `json:"output"`
}
