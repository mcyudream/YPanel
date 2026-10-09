package dto

// M33 P2.6：日志中心 VictoriaLogs 自动发现与多节点聚合（core→agent→本机 VL 代理）。

// VLDiscovery 节点上的 VictoriaLogs 自动发现结果。
type VLDiscovery struct {
	Found     bool   `json:"found"`
	Container string `json:"container"`
	Port      string `json:"port"` // 宿主映射端口（agent 侧 127.0.0.1 可达）
	Image     string `json:"image"`
}

// VLProxyReq VL 查询代理请求（path 白名单校验在 agent 侧）。
type VLProxyReq struct {
	Path   string            `json:"path"`   // /select/logsql/query | hits | stream_field_values
	Params map[string]string `json:"params"` // URL 查询参数
}

// VLProxyResp 上游响应透传。
type VLProxyResp struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}
