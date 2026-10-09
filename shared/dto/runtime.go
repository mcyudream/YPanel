package dto

// FpmStatusItem FPM 状态页键值。
type FpmStatusItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// FpmStatusResp FPM 状态查询响应（agent /agent/v1/runtime/php/fpm-status）。
type FpmStatusResp struct {
	Items []FpmStatusItem `json:"items"`
}
