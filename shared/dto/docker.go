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
