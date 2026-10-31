package dto

// ComposeServiceActionReq 编排单服务操作；Project 为标准字段，Name 兼容旧页面。
type ComposeServiceActionReq struct {
	Project string `json:"project"`
	Name    string `json:"name,omitempty"`
	Service string `json:"service"`
	Action  string `json:"action"`
	Dir     string `json:"dir,omitempty"` // 外部编排工作目录，托管项目可省略
}

type ComposeServiceActionResp struct {
	Output string `json:"output"`
}
