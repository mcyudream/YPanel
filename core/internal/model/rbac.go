package model

import "time"

// Role 角色（M54 RBAC）：权限点集合与节点/数据范围挂角色，用户经 role_id 关联。
type Role struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Key           string    `gorm:"uniqueIndex;size:32;not null" json:"key"` // super-admin / viewer / operator / 自定义
	Name          string    `gorm:"size:32;not null" json:"name"`
	Builtin       bool      `gorm:"not null;default:false" json:"builtin"`    // 内置角色不可删改，仅可复制派生
	ScopeAllNodes bool      `gorm:"not null;default:true" json:"scopeAllNodes"` // true=不限节点；false 时以 role_nodes 为准（M54-P2 生效）
	DataScope     string    `gorm:"size:16;not null;default:all" json:"dataScope"` // all / assigned（M54-P3 生效）
	Remark        string    `gorm:"size:255" json:"remark"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// RolePermission 角色—权限点关联（(role_id, perm_key) 唯一）。
// 权限点目录本身静态定义在 rbac 包，不入库。
type RolePermission struct {
	RoleID  uint   `gorm:"uniqueIndex:idx_role_perm,priority:1;not null" json:"roleId"`
	PermKey string `gorm:"uniqueIndex:idx_role_perm,priority:2;size:64;not null" json:"permKey"`
}

// RoleNode 角色—节点范围关联（Role.ScopeAllNodes=false 时生效；M54-P2）。
type RoleNode struct {
	RoleID uint   `gorm:"uniqueIndex:idx_role_node,priority:1;not null" json:"roleId"`
	NodeID string `gorm:"uniqueIndex:idx_role_node,priority:2;size:32;not null" json:"nodeId"` // local 或远程节点 ID
}
