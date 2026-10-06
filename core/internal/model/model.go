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

// Notification 站内通知。
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Level     string    `gorm:"size:8;not null;default:info" json:"level"`
	Title     string    `gorm:"size:128;not null" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Read      bool      `gorm:"not null;default:false" json:"read"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

// AuditLog 操作审计（写操作留痕）。
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:32;index" json:"username"`
	Method    string    `gorm:"size:8" json:"method"`
	Path      string    `gorm:"size:255" json:"path"`
	Detail    string    `gorm:"type:text" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	Success   bool      `json:"success"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

// MetricRecord 历史监控（60s 聚合，保留 30 天）。
type MetricRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	At        time.Time `gorm:"index" json:"at"`
	Cpu       float64   `json:"cpu"`
	Mem       float64   `json:"mem"`
	RxSpeed   float64   `json:"rxSpeed"`
	TxSpeed   float64   `json:"txSpeed"`
	Load1     float64   `json:"load1"`
}

// Runtime PHP 运行环境（php-fpm 容器化）。
type Runtime struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"uniqueIndex;size:32;not null" json:"name"`
	Version        string    `gorm:"size:16;not null" json:"version"` // 8.2 / 8.3
	ComposeProject string    `gorm:"size:64;not null" json:"composeProject"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AppStoreApp 应用商店应用（1Panel 默认源同步）。
type AppStoreApp struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Key          string    `gorm:"uniqueIndex;size:64;not null" json:"key"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Title        string    `gorm:"size:255" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	ReadMe       string    `gorm:"type:text" json:"readMe"`
	IconURL      string    `gorm:"size:512" json:"iconUrl"`
	Tags         string    `gorm:"size:255" json:"tags"`
	VersionsJSON string    `gorm:"type:text" json:"versionsJson"`
	LastModified int64     `json:"lastModified"`
	SyncedAt     time.Time `json:"syncedAt"`
}

// AppStoreInstall 已安装的商店应用。
type AppStoreInstall struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Key            string    `gorm:"size:64;not null" json:"key"`
	Name           string    `gorm:"size:64;not null" json:"name"`
	Version        string    `gorm:"size:32;not null" json:"version"`
	ComposeProject string    `gorm:"size:64;not null;uniqueIndex" json:"composeProject"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Node 远程受管节点（local 本机节点不入库，进程内嵌）。
type Node struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Addr       string    `gorm:"size:255;not null" json:"addr"` // agent 基址 http://host:9527
	Token      string    `gorm:"size:128;not null" json:"-"`
	Hostname   string    `gorm:"size:255" json:"hostname"`
	OS         string    `gorm:"size:64" json:"os"`
	Arch       string    `gorm:"size:32" json:"arch"`
	Version    string    `gorm:"size:32" json:"version"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// PairingCode 一次性配对码。
type PairingCode struct {
	Code      string    `gorm:"primaryKey;size:16" json:"code"`
	ExpiredAt time.Time `json:"expiredAt"`
	Used      bool      `gorm:"not null;default:false" json:"used"`
	CreatedAt time.Time `json:"createdAt"`
}

// Site 站点（容器化 nginx vhost）。
type Site struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:32;not null" json:"name"`
	Type        string    `gorm:"size:8;not null" json:"type"` // static / proxy
	Domain      string    `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	Domains     string    `gorm:"type:text" json:"domains"`   // 附加域名 JSON 数组（M13 多域名）
	Port        int       `gorm:"not null;default:80" json:"port"`
	ProxyPass   string    `gorm:"size:255" json:"proxyPass"`  // 默认反代规则（"/"）
	ProxyRules  string    `gorm:"type:text" json:"proxyRules"` // 反代规则 JSON 数组 [{prefix,target,ws}]
	IndexFiles  string    `gorm:"size:255" json:"indexFiles"`  // 默认文档
	LogsEnabled bool      `gorm:"not null;default:true" json:"logsEnabled"`
	RuntimeContainer string `gorm:"size:64" json:"runtimeContainer"` // php-fpm 容器名
	RewriteName string    `gorm:"size:32" json:"rewriteName"`         // 伪静态模板名（空=无）
	RewriteContent string `gorm:"type:text" json:"rewriteContent"`   // 自定义伪静态文本
	CustomLocations string `gorm:"type:text" json:"customLocations"` // 自定义 location JSON 数组 [{comment,content}]
	ErrorPage404   string `gorm:"size:255" json:"errorPage404"`      // 自定义 404 路径（相对站点 root）
	RuntimeID      uint   `json:"runtimeId"`                          // php 类型绑定的运行环境
	CacheEnable    bool   `gorm:"not null;default:false" json:"cacheEnable"`
	CacheDuration  string `gorm:"size:16" json:"cacheDuration"`      // 如 12h / 1d
	CertDomain  string    `gorm:"size:255" json:"certDomain"` // 非空 = 已启用 SSL
	OriginFile  string    `gorm:"size:255" json:"originFile"` // 接管来源 conf（站点识别）
	WafJSON     string    `gorm:"type:text" json:"wafJson"`   // WAF 配置（service.SiteWaf 序列化）
	ConfJSON    string    `gorm:"type:text" json:"confJson"`  // 扩展配置域 JSON（防盗链/Basic认证/CORS/重定向/真实IP/限连/负载均衡）
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CronTask 计划任务（shell 类型）。
type CronTask struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:64;not null" json:"name"`
	Cron        string     `gorm:"size:32;not null" json:"cron"`
	Command     string     `gorm:"type:text;not null" json:"command"`
	Type        string     `gorm:"size:16;not null;default:shell" json:"type"`  // shell / db_backup / site_backup / container_op / script
	Payload     string     `gorm:"type:text" json:"payload"`                    // 类型参数 JSON（B4：{dbId|siteName|container|action|scriptId}）
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

// AlertRule 告警规则。
type AlertRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Metric      string    `gorm:"size:16;not null" json:"metric"` // cpu / memory / disk
	Threshold   int       `gorm:"not null" json:"threshold"`
	WebhookURL  string    `gorm:"size:512;not null" json:"webhookUrl"`
	WebhookType string    `gorm:"size:16;not null;default:generic" json:"webhookType"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AIProvider AI 供应商配置（B18）：支持多供应商与自定义接入。
type AIProvider struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:32;not null" json:"name"`
	APIType   string    `gorm:"size:16;not null" json:"apiType"` // openai / anthropic / response
	BaseURL   string    `gorm:"size:255;not null" json:"baseURL"`
	APIKey    string    `gorm:"type:text;not null" json:"apiKey"`
	Model     string    `gorm:"size:64;not null" json:"model"`
	IsDefault bool      `gorm:"not null;default:false" json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Script 脚本库（B13：计划任务可引用）。
type Script struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
