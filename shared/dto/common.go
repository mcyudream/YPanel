// Package dto 定义 core / agent / 前端三方共享的协议结构。
// 本包只放纯数据与常量，禁止引入任何运行时依赖。
package dto

// Resp 统一响应体。
// 约定：HTTP 状态码恒为 200（网关友好），业务成败以 code 区分；
// code=0 成功，非 0 见 errs 包错误码表。
type Resp[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// PageReq 通用分页请求。
type PageReq struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// PageResp 通用分页响应。
type PageResp[T any] struct {
	Total int64 `json:"total"`
	Items []T   `json:"items"`
}

// NewPageResp 构造分页响应，空列表返回空切片而非 nil。
func NewPageResp[T any](total int64, items []T) PageResp[T] {
	if items == nil {
		items = make([]T, 0)
	}
	return PageResp[T]{Total: total, Items: items}
}
