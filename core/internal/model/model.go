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

// CronTask 计划任务（shell 类型）。
type CronTask struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:64;not null" json:"name"`
	Cron        string     `gorm:"size:32;not null" json:"cron"`
	Command     string     `gorm:"type:text;not null" json:"command"`
	Enabled     bool       `gorm:"not null;default:true" json:"enabled"`
	TimeoutSecs int        `gorm:"not null;default:300" json:"timeoutSecs"`
	LastRunAt   *time.Time `json:"lastRunAt"`
	LastSuccess *bool      `json:"lastSuccess"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// DatabaseInstance 数据库实例元数据（密码 AES-GCM 加密存储）。
type DatabaseInstance struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"uniqueIndex;size:32;not null" json:"name"`
	Type           string    `gorm:"size:16;not null" json:"type"` // mysql / postgres / redis / mongo
	Port           int       `gorm:"not null" json:"port"`
	RootUser       string    `gorm:"size:32" json:"rootUser"`
	PasswordEnc    string    `gorm:"type:text;not null" json:"-"`
	ComposeProject string    `gorm:"size:64;not null" json:"composeProject"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// CronTaskLog 计划任务执行记录。
type CronTaskLog struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	TaskID     uint       `gorm:"index;not null" json:"taskId"`
	TaskName   string     `gorm:"size:64" json:"taskName"`
	Trigger    string     `gorm:"size:8;not null" json:"trigger"` // cron / manual
	StartAt    time.Time  `json:"startAt"`
	EndAt      *time.Time `json:"endAt"`
	DurationMs int64      `json:"durationMs"`
	Success    bool       `json:"success"`
	Output     string     `gorm:"type:text" json:"output"`
}
