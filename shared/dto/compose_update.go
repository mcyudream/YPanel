package dto

import "time"

// ComposePullUpdateReq 编排镜像更新。NodeID 为空时由 API 的 node 查询参数补齐。
type ComposePullUpdateReq struct {
	NodeID            string `json:"nodeId"`
	Project           string `json:"project"`
	Name              string `json:"name,omitempty"` // 兼容原前端 name 入参
	Dir               string `json:"dir,omitempty"`
	PreHook           string `json:"preHook,omitempty"`
	PostHook          string `json:"postHook,omitempty"`
	HookFailurePolicy string `json:"hookFailurePolicy,omitempty"` // stop / continue
}

type ComposeAutoUpdateSaveReq struct {
	ComposePullUpdateReq
	Enabled     bool   `json:"enabled"`
	IntervalMin int    `json:"intervalMin"`
	Remark      string `json:"remark,omitempty"`
}

type ComposeUpdateTaskResp struct {
	TaskID  uint   `json:"taskId"`
	Project string `json:"project"`
}

// ComposeAutoUpdateResp 参数与运行状态分开返回；最近任务同时包括手动更新。
type ComposeAutoUpdateResp struct {
	NodeID            string     `json:"nodeId"`
	Project           string     `json:"project"`
	Dir               string     `json:"dir"`
	Enabled           bool       `json:"enabled"`
	IntervalMin       int        `json:"intervalMin"`
	PreHook           string     `json:"preHook"`
	PostHook          string     `json:"postHook"`
	HookFailurePolicy string     `json:"hookFailurePolicy"`
	Remark            string     `json:"remark"`
	LastCheckedAt     *time.Time `json:"lastCheckedAt,omitempty"`
	NextCheckAt       *time.Time `json:"nextCheckAt,omitempty"`
	LastResult        string     `json:"lastResult"`
	LastTaskID        uint       `json:"lastTaskId"`
	LastStatus        string     `json:"lastStatus"`
	Exists            bool       `json:"exists"`
}
