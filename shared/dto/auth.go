package dto

import "time"

// LoginReq 登录请求。
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResp 登录响应。
type LoginResp struct {
	Token    string     `json:"token"`
	ExpireAt time.Time  `json:"expireAt"`
	User     UserInfo   `json:"user"`
}

// UserInfo 用户信息。
type UserInfo struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	Role        string     `json:"role"` // admin / user
	LastLoginAt *time.Time `json:"lastLoginAt"`
}

// ChangePasswordReq 修改自身密码。
type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=64"`
}

// UserCreateReq 创建用户（admin）。
type UserCreateReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Nickname string `json:"nickname"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
}

// UserUpdateReq 更新用户（admin）。零值字段不更新。
type UserUpdateReq struct {
	Password *string `json:"password" binding:"omitempty,min=6,max=64"`
	Nickname *string `json:"nickname"`
	Role     *string `json:"role" binding:"omitempty,oneof=admin user"`
	Status   *int    `json:"status" binding:"omitempty,oneof=0,1"` // 1 启用 0 禁用
}

// LoginLogItem 登录审计条目。
type LoginLogItem struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}
