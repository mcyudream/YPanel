// Package model 面板自身数据实体（GORM）。
package model

import "time"

// User 面板用户。
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"uniqueIndex;size:32;not null" json:"username"`
	Password     string     `gorm:"size:72;not null" json:"-"` // bcrypt
	Nickname     string     `gorm:"size:64" json:"nickname"`
	Role         string     `gorm:"size:16;not null;default:user" json:"role"` // admin / user
	Status       int        `gorm:"not null;default:1" json:"status"`          // 1 启用 0 禁用
	TokenVersion int        `gorm:"not null;default:1" json:"-"`               // 改密/强制下线时递增
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// LoginLog 登录审计。
type LoginLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:32;index" json:"username"`
	IP        string    `gorm:"size:64" json:"ip"`
	UserAgent string    `gorm:"size:255" json:"userAgent"`
	Success   bool      `json:"success"`
	Message   string    `gorm:"size:255" json:"message"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

// Setting 键值设置。
type Setting struct {
	Key       string    `gorm:"primaryKey;size:64" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
