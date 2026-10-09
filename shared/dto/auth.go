package dto

import "time"

// LoginReq 登录请求。
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	OtpCode  string `json:"otpCode"` // 启用 2FA 时必填
}

// LoginResp 登录响应。
type LoginResp struct {
	Token    string     `json:"token"`
	ExpireAt time.Time  `json:"expireAt"`
	User     UserInfo   `json:"user"`
}

// UserInfo 用户信息。
type UserInfo struct {
	ID            uint       `json:"id"`
	Username      string     `json:"username"`
	Nickname      string     `json:"nickname"`
	Role          string     `json:"role"` // 旧二值角色（冻结兼容字段）
	RoleID        uint       `json:"roleId"`                     // 角色 ID
	RoleKey       string     `json:"roleKey"`                    // M54 RBAC 角色 key
	Permissions   []string   `json:"permissions"`                // 权限点列表（含 "*" 通配）
	DataScope     string     `json:"dataScope"`                  // all / assigned（P3 生效）
	ScopeAllNodes bool       `json:"scopeAllNodes"`              // true=不限节点（P2 生效）
	Nodes         []string   `json:"nodes"`                      // 允许的节点 ID（P2 生效；不限时为空）
	LastLoginAt   *time.Time `json:"lastLoginAt"`
}

// ChangePasswordReq 修改自身密码。
type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=64"`
}

// UserCreateReq 创建用户（user:manage）。
// RoleID 优先；为 0 时兼容旧 Role 字符串（admin/user）。
type UserCreateReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Nickname string `json:"nickname"`
	RoleID   uint   `json:"roleId"`
	Role     string `json:"role"`
}

// UserUpdateReq 更新用户（user:manage）。零值字段不更新。
type UserUpdateReq struct {
	Password *string `json:"password" binding:"omitempty,min=6,max=64"`
	Nickname *string `json:"nickname"`
	RoleID   *uint   `json:"roleId"`
	Role     *string `json:"role"` // 旧字段，RoleID 缺省时兼容
	Status   *int    `json:"status" binding:"omitempty,oneof=0,1"` // 1 启用 0 禁用
}

// RoleCreateReq 创建角色。
type RoleCreateReq struct {
	Key           string   `json:"key" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	Remark        string   `json:"remark"`
	DataScope     string   `json:"dataScope"`
	Perms         []string `json:"perms"`
	ScopeAllNodes *bool    `json:"scopeAllNodes"` // 缺省 true=不限节点
	NodeIDs       []string `json:"nodeIds"`       // ScopeAllNodes=false 时生效（local / 远程节点 ID）
}

// RoleUpdateReq 更新角色（零值字段不更新；perms/nodeIds 为 nil 不动对应集合）。
type RoleUpdateReq struct {
	Name          *string   `json:"name"`
	Remark        *string   `json:"remark"`
	DataScope     *string   `json:"dataScope"`
	Perms         *[]string `json:"perms"`
	ScopeAllNodes *bool     `json:"scopeAllNodes"`
	NodeIDs       *[]string `json:"nodeIds"`
}

// RoleInfo 角色信息。
type RoleInfo struct {
	ID            uint     `json:"id"`
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Builtin       bool     `json:"builtin"`
	ScopeAllNodes bool     `json:"scopeAllNodes"`
	DataScope     string   `json:"dataScope"`
	Remark        string   `json:"remark"`
	NodeIDs       []string `json:"nodeIds"` // 节点范围（ScopeAllNodes=false 时有值）
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
