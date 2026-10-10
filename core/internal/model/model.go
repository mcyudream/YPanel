// Package model 面板自身数据实体（GORM）。
package model

import "time"

// User 面板用户。
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"uniqueIndex;size:32;not null" json:"username"`
	Password     string     `gorm:"size:72;not null" json:"-"` // bcrypt
	Nickname     string     `gorm:"size:64" json:"nickname"`
	Role         string     `gorm:"size:16;not null;default:user" json:"role"` // 旧二值角色：M54 起冻结只读（冗余 role key），权限以 RoleID 为准
	RoleID       uint       `gorm:"index;not null;default:0" json:"roleId"`    // 关联 Role；0 = 按 Role 旧值回退映射
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

// MetricRecord 历史监控（60s 聚合，保留 30 天，全节点）。
type MetricRecord struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	NodeId  string    `gorm:"size:32;index:idx_metric_node_at;default:''" json:"nodeId"` // 空串视为 local（多节点化前的历史数据）
	At      time.Time `gorm:"index:idx_metric_node_at,priority:2" json:"at"`
	Cpu     float64   `json:"cpu"`
	Mem     float64   `json:"mem"`
	Swap    float64   `json:"swap"` // swap 使用率 %（无 swap 为 0）
	RxSpeed float64   `json:"rxSpeed"`
	TxSpeed float64   `json:"txSpeed"`
	Load1   float64   `json:"load1"`
}

// Runtime 运行环境（容器化运行时或接管本机 fastcgi）。
type Runtime struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"uniqueIndex;size:32;not null" json:"name"`
	Type           string    `gorm:"size:16;not null;default:php;index" json:"type"`   // php / node / python / java / go
	Version        string    `gorm:"size:16;not null" json:"version"`                  // 如 8.2 / 20 / 3.12
	Origin         string    `gorm:"size:16;not null;default:container" json:"origin"` // container / external
	FCGIAddr       string    `gorm:"size:128" json:"fcgiAddr"`                         // external: host:port 或 unix:/path/s.sock
	Image          string    `gorm:"size:128" json:"image"`                            // 构建产物镜像名（ypanel-rt/php-<name>:<ver>）
	CodeDir        string    `gorm:"size:255" json:"codeDir"`                          // 代码运行时：宿主代码目录
	ContainerName  string    `gorm:"size:64" json:"containerName"`
	ComposeProject string    `gorm:"size:64;not null" json:"composeProject"`
	NodeID         string    `gorm:"size:32;not null;default:''" json:"nodeId"`      // M57 归属节点（空=local）
	Status         string    `gorm:"size:16;not null;default:running" json:"status"` // running / stopped / building / creating / error
	Message        string    `gorm:"type:text" json:"message"`                       // 最近一次失败原因
	EnvJSON        string    `gorm:"type:text" json:"envJson"`                       // 运行参数 JSON（extensions / 快捷设置 / 启动命令等）
	Port           string    `gorm:"size:64" json:"port"`                            // 代码运行时宿主端口（逗号分隔）
	Remark         string    `gorm:"size:255" json:"remark"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AppTask 统一任务记录（商店安装/卸载、镜像拉取等耗时操作）。
type AppTask struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:32;index;not null" json:"type"` // store-install / store-uninstall / image-pull / ...
	Title     string    `gorm:"size:255;not null" json:"title"`
	Ref       string    `gorm:"size:255;index" json:"ref"`                            // 业务引用（compose 项目名 / 镜像名）
	Status    string    `gorm:"size:16;index;not null;default:running" json:"status"` // running / success / failed
	LogText   string    `gorm:"type:text" json:"logText"`
	Error     string    `gorm:"type:text" json:"error"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AppStoreSource 应用源（onepanel zip / yp-url index.json / yp-git 仓库）。
type AppStoreSource struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Type       string     `gorm:"size:16;not null" json:"type"` // onepanel / yp-url / yp-git
	URL        string     `gorm:"size:512;not null" json:"url"` // onepanel: 1panel.json.zip 完整地址；yp-url: index.json 地址；yp-git: 仓库地址
	Branch     string     `gorm:"size:64" json:"branch"`        // yp-git 分支（空=远端默认）
	AuthToken  string     `gorm:"size:512" json:"-"`            // git 访问 token（可选）
	Enabled    bool       `gorm:"not null;default:true" json:"enabled"`
	Builtin    bool       `gorm:"not null;default:false" json:"builtin"`
	Remark     string     `gorm:"size:255" json:"remark"`
	Status     string     `gorm:"size:16;not null;default:pending" json:"status"` // pending / ok / error
	Message    string     `gorm:"size:512" json:"message"`
	AppCount   int        `json:"appCount"`
	LastSyncAt *time.Time `json:"lastSyncAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// AppStoreApp 应用商店应用（多源同步；(source_id,key) 唯一）。
type AppStoreApp struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SourceID      uint      `gorm:"uniqueIndex:idx_app_source_key,priority:1;not null;default:0" json:"sourceId"`
	Key           string    `gorm:"uniqueIndex:idx_app_source_key,priority:2;size:64;not null" json:"key"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	Title         string    `gorm:"size:255" json:"title"`
	Description   string    `gorm:"type:text" json:"description"`
	ReadMe        string    `gorm:"type:text" json:"readMe"`
	IconURL       string    `gorm:"size:512" json:"iconUrl"`
	Tags          string    `gorm:"size:255" json:"tags"`
	Kind          string    `gorm:"size:16;not null;default:app" json:"kind"` // app / service / middleware
	Author        string    `gorm:"size:128" json:"author"`
	Arch          string    `gorm:"size:64" json:"arch"` // 逗号分隔 amd64,arm64
	VersionsJSON  string    `gorm:"type:text" json:"versionsJson"`
	ReverseProxy  string    `gorm:"size:255" json:"reverseProxy"` // 一键反代声明的端口 env key（空=不支持）
	AdminUIJSON   string    `gorm:"type:text" json:"adminUI"`     // 内网管理界面声明 JSON（port envKey/path/name；安装后注册到桌面，经内网浏览器打开）
	Website       string    `gorm:"size:512" json:"website"`      // 官网地址
	SourceURL     string    `gorm:"size:512" json:"sourceUrl"`    // 开源社区地址（github 等）
	Document      string    `gorm:"size:512" json:"document"`     // 文档地址
	LatestVersion string    `gorm:"size:64" json:"latestVersion"` // 最新版本号（同步时冗余，升级判定用）
	LastModified  int64     `json:"lastModified"`
	SyncedAt      time.Time `json:"syncedAt"`
}

// AppStoreInstall 已安装的商店应用。
type AppStoreInstall struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SourceID       uint      `gorm:"index;not null;default:0" json:"sourceId"`
	Key            string    `gorm:"index;size:64;not null" json:"key"`
	Name           string    `gorm:"size:64;not null" json:"name"`
	Version        string    `gorm:"size:64;not null" json:"version"`
	ComposeProject string    `gorm:"size:64;not null;index" json:"composeProject"` // M55 多实例：同名应用可装多节点（项目名带节点后缀）
	Remark         string    `gorm:"size:255" json:"remark"`
	ParamsJSON     string    `gorm:"type:text" json:"paramsJson"` // 安装参数（含密码明文，仅 admin 视图返回）
	OwnerID        uint      `gorm:"index;not null;default:0" json:"ownerId"` // M54-P3 数据范围属主（0=公共）
	NodeID         string    `gorm:"size:32;not null;default:'';index" json:"nodeId"` // M55 目标节点（空=local）
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
	// M43 服务器资产
	ExpireDate  *time.Time `json:"expireDate"`              // 到期日
	MonthlyPrice string    `gorm:"size:40" json:"monthlyPrice"` // 价格描述（如 ¥45/月）
	TrafficQuotaGB int     `gorm:"not null;default:0" json:"trafficQuotaGB"` // 月流量额度（0=不限）
	AssetRemark  string    `gorm:"size:255" json:"assetRemark"`
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
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"uniqueIndex;size:32;not null" json:"name"`
	Type             string    `gorm:"size:8;not null" json:"type"` // static / proxy
	Domain           string    `gorm:"uniqueIndex:idx_sites_domain_port;size:255;not null" json:"domain"`
	Domains          string    `gorm:"type:text" json:"domains"` // 附加域名 JSON 数组（M13 多域名）
	Port             int       `gorm:"not null;default:80;uniqueIndex:idx_sites_domain_port" json:"port"`
	ProxyPass        string    `gorm:"size:255" json:"proxyPass"`   // 默认反代规则（"/"）
	ProxyRules       string    `gorm:"type:text" json:"proxyRules"` // 反代规则 JSON 数组 [{prefix,target,ws}]
	IndexFiles       string    `gorm:"size:255" json:"indexFiles"`  // 默认文档
	LogsEnabled      bool      `gorm:"not null;default:true" json:"logsEnabled"`
	RuntimeContainer string    `gorm:"size:64" json:"runtimeContainer"`  // php-fpm 容器名
	RewriteName      string    `gorm:"size:32" json:"rewriteName"`       // 伪静态模板名（空=无）
	RewriteContent   string    `gorm:"type:text" json:"rewriteContent"`  // 自定义伪静态文本
	CustomLocations  string    `gorm:"type:text" json:"customLocations"` // 自定义 location JSON 数组 [{comment,content}]
	ErrorPage404     string    `gorm:"size:255" json:"errorPage404"`     // 自定义 404 路径（相对站点 root）
	RuntimeID        uint      `json:"runtimeId"`                        // php 类型绑定的运行环境
	CacheEnable      bool      `gorm:"not null;default:false" json:"cacheEnable"`
	CacheDuration    string    `gorm:"size:16" json:"cacheDuration"`      // 如 12h / 1d
	CertDomain       string    `gorm:"size:255" json:"certDomain"`        // 非空 = 已启用 SSL
	NodeID           string    `gorm:"size:32;not null;default:'';index" json:"nodeId"` // M57 站点归属节点（空=local，nginx 所在机）
	CertID           uint      `gorm:"not null;default:0" json:"certId"`  // 绑定证书库条目（B23）
	RunDir           string    `gorm:"size:128" json:"runDir"`            // 运行目录（相对 root 的二级目录，空=根）
	GroupID          uint      `gorm:"not null;default:0" json:"groupId"` // 分组（0=默认分组）
	Remark           string    `gorm:"size:255" json:"remark"`            // 备注
	OwnerID          uint      `gorm:"index;not null;default:0" json:"ownerId"` // M54-P3 数据范围属主（0=公共）
	HTTPSJSON        string    `gorm:"type:text" json:"httpsJson"`        // HTTPS 高级设置 JSON（HTTP 模式/HSTS/TLS 版本/加密算法）
	OriginFile       string    `gorm:"size:255" json:"originFile"`        // 接管来源 conf（站点识别）
	WafJSON          string    `gorm:"type:text" json:"wafJson"`          // WAF 配置（service.SiteWaf 序列化）
	ConfJSON         string     `gorm:"type:text" json:"confJson"` // 扩展配置域 JSON（防盗链/Basic认证/CORS/重定向/真实IP/限连/负载均衡）
	IsDefault        bool       `gorm:"not null;default:false" json:"isDefault"` // 默认站点（nginx default_server，M39）
	ExpireAt         *time.Time `json:"expireAt"`                                 // 网站到期时间（M39）
	Enabled          bool       `gorm:"not null;default:true" json:"enabled"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// SiteGroup 站点分组（对齐 1Panel：创建/删除/设为默认，默认分组不可删）。
type SiteGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:32;not null" json:"name"`
	IsDefault bool      `gorm:"not null;default:false" json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
}

// SitePortLease 站点附加端口登记（M50）：非标 listen 端口的系统侧落点（容器映射 / 防火墙放行），
// 由站点表对账推导维护；防火墙页据此展示"站点托管"放行项。
type SitePortLease struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Port      int       `gorm:"uniqueIndex;not null" json:"port"`
	Proto     string    `gorm:"size:8;not null;default:tcp" json:"proto"`
	Sites     string    `gorm:"size:255" json:"sites"` // 来源站点名（逗号分隔，对账时刷新）
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Certificate 证书库（独立于站点的证书全生命周期管理，对齐 1Panel 证书页）。
type Certificate struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CertName      string     `gorm:"uniqueIndex;size:255;not null" json:"certName"` // 文件名基准（nginx 引用）
	Domain        string     `gorm:"size:255;not null" json:"domain"`               // 主域名
	AltDomains    string     `gorm:"type:text" json:"altDomains"`                   // 其他域名 JSON 数组
	Provider      string     `gorm:"size:16;not null" json:"provider"`              // acme / selfsigned / upload
	Issuer        string     `gorm:"size:128" json:"issuer"`                        // 颁发组织（Let's Encrypt / ZeroSSL…）
	Remark        string     `gorm:"size:255" json:"remark"`
	AutoRenew     bool       `gorm:"not null;default:true" json:"autoRenew"`
	NotAfter      *time.Time `json:"notAfter"`                                  // 过期时间（签发/上传后探测落库）
	Status        string     `gorm:"size:16;not null;default:ok" json:"status"` // ok / expiring / expired / error
	AcmeAccountID uint       `json:"acmeAccountId"`
	DnsAccountID  uint       `json:"dnsAccountId"`
	IssueLog      string     `gorm:"type:text" json:"issueLog"` // 最近一次申请日志
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// DnsAccount DNS 服务商 API 账户（ACME DNS 挑战用；密钥 AES-GCM 加密存储）。
type DnsAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Provider  string    `gorm:"size:16;not null" json:"provider"` // aliyun / dnspod / cloudflare
	AccessKey string    `gorm:"size:255;not null" json:"accessKey"`
	SecretEnc string    `gorm:"type:text;not null" json:"-"` // AES-GCM
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AcmeAccount ACME 账户（邮箱 + CA + 密钥算法）。
type AcmeAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"size:128;not null" json:"email"`
	CAType    string    `gorm:"size:24;not null" json:"caType"` // letsencrypt / zerossl / buypass
	KeyType   string    `gorm:"size:16;not null;default:ec-256" json:"keyType"`
	CreatedAt time.Time `json:"createdAt"`
}

// CronTask 计划任务（shell 类型）。
type CronTask struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:64;not null" json:"name"`
	Cron        string     `gorm:"size:32;not null" json:"cron"`
	Command     string     `gorm:"type:text;not null" json:"command"`
	Type        string     `gorm:"size:16;not null;default:shell" json:"type"` // shell / db_backup / site_backup / container_op / script
	Payload     string     `gorm:"type:text" json:"payload"`                   // 类型参数 JSON（B4：{dbId|siteName|container|action|scriptId}）
	NodeID      string     `gorm:"size:32;not null;default:''" json:"nodeId"` // M55 目标节点（空=local；shell/script 类在目标节点执行）
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
	Type           string    `gorm:"size:16;not null" json:"type"`                     // mysql / postgres / redis / mongo
	Origin         string    `gorm:"size:16;not null;default:container" json:"origin"` // container / external
	Host           string    `gorm:"size:255;not null;default:127.0.0.1" json:"host"`  // external: 远端主机地址
	Port           int       `gorm:"not null" json:"port"`
	RootUser       string    `gorm:"size:32" json:"rootUser"`
	PasswordEnc    string    `gorm:"type:text;not null" json:"-"`
	Remark         string    `gorm:"size:255" json:"remark"`
	ComposeProject string    `gorm:"size:64;not null" json:"composeProject"`
	OwnerID        uint      `gorm:"index;not null;default:0" json:"ownerId"` // M54-P3 数据范围属主（0=公共）
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// DatabaseAuditLog 数据库管理台操作审计（写路径全埋点）。
type DatabaseAuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:32;index" json:"username"`
	InstanceID   uint      `gorm:"index" json:"instanceId"`
	InstanceName string    `gorm:"size:64" json:"instanceName"`
	DbType       string    `gorm:"size:16" json:"dbType"`
	Kind         string    `gorm:"size:32" json:"kind"`     // row_insert/row_update/row_delete/index_create/index_drop/redis_write/redis_delete/redis_ttl/redis_exec/mongo_* 等
	Target       string    `gorm:"size:255" json:"target"`  // 库.表 / 集合 / key 定位
	Detail       string    `gorm:"type:text" json:"detail"` // 语句/命令摘要（截断，不含敏感值原文可含参数结构）
	Success      bool      `json:"success"`
	CreatedAt    time.Time `gorm:"index" json:"createdAt"`
}

// AlertRule 告警规则。
type AlertRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Metric      string    `gorm:"size:16;not null" json:"metric"` // cpu / memory / disk / load / network / cert_expiry（M37）
	Threshold   int       `gorm:"not null" json:"threshold"`      // 百分比 / load 值 / MB/s / 天数
	WebhookURL  string    `gorm:"size:512;not null" json:"webhookUrl"`
	WebhookType string    `gorm:"size:16;not null;default:generic" json:"webhookType"` // generic/feishu/dingtalk/wecom/telegram/bark/email（M37）
	SilentStart string    `gorm:"size:5" json:"silentStart"`                           // 静默时段 HH:MM（空=不静默，M37）
	SilentEnd   string    `gorm:"size:5" json:"silentEnd"`
	Query       string    `gorm:"size:512" json:"query"`                 // metric=log：LogsQL 查询（P3 日志量告警）
	WindowSec   int       `gorm:"not null;default:300" json:"windowSec"` // metric=log：统计窗口秒数
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// MetricHourly 历史监控小时聚合（M37）：原始明细保留 30 天，聚合保留 365 天。
type MetricHourly struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	NodeId  string    `gorm:"size:32;uniqueIndex:idx_metric_hourly,priority:1;default:''" json:"nodeId"`
	Hour    time.Time `gorm:"uniqueIndex:idx_metric_hourly,priority:2" json:"hour"` // 整点（UTC 截断）
	CpuAvg  float64   `json:"cpuAvg"`
	CpuMax  float64   `json:"cpuMax"`
	MemAvg  float64   `json:"memAvg"`
	MemMax  float64   `json:"memMax"`
	RxMB    float64   `json:"rxMb"` // 小时内累计流量（按 60s 采样速率×周期累加，MB）
	TxMB    float64   `json:"txMb"`
	LoadMax float64   `json:"loadMax"`
}

// AIProvider AI 供应商配置（B18）：支持多供应商与自定义接入。
type AIProvider struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:32;not null" json:"name"`
	APIType   string    `gorm:"size:16;not null" json:"apiType"` // openai / anthropic / response
	BaseURL   string    `gorm:"size:255;not null" json:"baseURL"`
	APIKey    string    `gorm:"type:text;not null" json:"apiKey"`
	Model     string    `gorm:"size:64;not null" json:"model"`
	Models    string    `gorm:"type:text" json:"models"` // 可用模型列表（逗号分隔）；空 = 仅 Model 一个
	IsDefault bool      `gorm:"not null;default:false" json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIMemory AI 长期记忆（B18：AI 自动沉淀的运维经验/用户偏好）。
type AIMemory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// AIToolFlag AI 内置系统工具启用开关（无记录 = 启用）。
type AIToolFlag struct {
	Name      string    `gorm:"primaryKey;size:64" json:"name"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIOperationLog AI 工具操作审计（M31）：write/danger 级工具的执行与 ask 确认结果全留痕。
type AIOperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Tool      string    `gorm:"size:64;index;not null" json:"tool"`
	Module    string    `gorm:"size:32;not null" json:"module"`
	Risk      string    `gorm:"size:8;not null" json:"risk"`    // write / danger
	Action    string    `gorm:"size:16;not null" json:"action"` // executed / approved / denied / timeout
	Args      string    `gorm:"type:text" json:"args"`          // 入参 JSON（截断 2KB）
	Result    string    `gorm:"size:512" json:"result"`         // 结果摘要（截断）
	Success   bool      `gorm:"not null;default:true" json:"success"`
}

// AIConversation AI 会话（B18：多会话持久化）。
type AIConversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:128" json:"title"`
	Messages  string    `gorm:"type:text" json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIKnowledge AI 知识库条目（B18：对话时关键词检索注入）。
type AIKnowledge struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:128;not null" json:"title"`
	Body      string    `gorm:"type:text;not null" json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIKnowledgeDoc 用户注入的知识文档（md/txt），保存时切为分块供检索。
type AIKnowledgeDoc struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:128;not null" json:"title"`
	Filename  string    `gorm:"size:255;not null" json:"filename"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIKnowledgeChunk 知识文档分块（检索粒度；标题=所在章节路径）。
type AIKnowledgeChunk struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	DocID   uint   `gorm:"index;not null" json:"docId"`
	Idx     int    `gorm:"not null" json:"idx"`
	Heading string `gorm:"size:255" json:"heading"`
	Body    string `gorm:"type:text;not null" json:"body"`
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
	NodeID     string     `gorm:"size:32;not null;default:''" json:"nodeId"`
}

// ConfigRevision 受管配置版本快照（M23）：面板写盘前自动快照旧内容。
// Scope 形如 "local:/opt/ypanel/compose/app/docker-compose.yml"（node:path）。
type ConfigRevision struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Scope     string    `gorm:"size:255;index:idx_rev_scope,priority:1;not null" json:"scope"`
	CreatedAt time.Time `gorm:"index:idx_rev_scope,priority:2" json:"createdAt"`
	Content   string    `gorm:"type:text" json:"content"`
	Trigger   string    `gorm:"size:16;not null;default:save" json:"trigger"` // save / rollback
	Note      string    `gorm:"size:255" json:"note"`
	Author    string    `gorm:"size:32" json:"author"`
}

// NatForwardRule NAT 端口转发规则（iptables DNAT，M24）。
// 渲染语义：映射端口与目标端口仅允许「范围→同尺寸范围」或「单端口→单端口」。
type NatForwardRule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	NodeID        string    `gorm:"index;size:32;not null;default:local" json:"nodeId"` // local 或远程节点 ID
	Name          string    `gorm:"size:64;not null" json:"name"`
	Protocol      string    `gorm:"size:8;not null;default:tcp" json:"protocol"` // tcp / udp
	IPFamily      int       `gorm:"not null;default:4" json:"ipFamily"`          // 4 / 6
	ListenPort    int       `gorm:"not null" json:"listenPort"`
	ListenPortEnd int       `gorm:"not null;default:0" json:"listenPortEnd"` // 0 = 单端口
	TargetIP      string    `gorm:"size:64;not null" json:"targetIp"`
	TargetPort    int       `gorm:"not null" json:"targetPort"`
	TargetPortEnd int       `gorm:"not null;default:0" json:"targetPortEnd"` // 0 = 单端口
	Iface         string    `gorm:"size:32" json:"iface"`                    // 空 = 所有网卡
	DestIP        string    `gorm:"size:64" json:"destIp"`                   // 目标地址匹配（内核 -d，M58）；空 = 不限，仅单 IP
	SrcSpec       string    `gorm:"size:255" json:"srcSpec"`                 // 接管来源原文 "<chain>|<spec>"（M58）；空 = 面板原生，非空时 apply 追加幂等摘除段
	Enabled       bool      `gorm:"not null;default:true" json:"enabled"`
	Sort          int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// GitCredential 私有仓库凭据（M26 P2）：https 账密/token 或 ssh 私钥，克隆时按 host 自动匹配。
type GitCredential struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Type      string    `gorm:"size:8;not null;default:token" json:"type"` // token(https 账密/token) / ssh(私钥)
	Host      string    `gorm:"size:128;not null;index" json:"host"`       // 匹配主机，如 github.com；* 兜底
	Username  string    `gorm:"size:128" json:"username"`                  // https 用户名（token 类型）
	Secret    string    `gorm:"type:text" json:"-"`                        // 密码/token/私钥（不回传）
	Remark    string    `gorm:"size:255" json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DnsRecord 内网 DNS 解析记录（M27 dnsmasq）：全局记录集，渲染进 dnsmasq.conf 下发部署节点。
type DnsRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:8;not null;default:address" json:"type"` // address / cname / txt
	Domain    string    `gorm:"size:253;not null" json:"domain"`
	Target    string    `gorm:"size:255;not null" json:"target"` // address=IP；cname=主机名；txt=文本
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	Comment   string    `gorm:"size:128" json:"comment"`
	Sort      int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// HostRecord hosts 解析记录（M28）：全局记录集，渲染为托管块分发到托管节点 /etc/hosts。
type HostRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IP        string    `gorm:"size:64;not null" json:"ip"`
	Hostnames string    `gorm:"size:512;not null" json:"hostnames"` // 空格分隔多值
	Comment   string    `gorm:"size:128" json:"comment"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	Sort      int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// HostTarget hosts 托管节点（M28）：记录已下发托管块的节点，记录变更自动重放。
type HostTarget struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NodeID    string    `gorm:"uniqueIndex;size:32;not null" json:"nodeId"`
	CreatedAt time.Time `json:"createdAt"`
}

// StorageAccount 远程备份存储账号（M34）：s3 兼容（S3/MinIO/OSS/COS）/ webdav / sftp。
// 凭据 AES-GCM 加密存储不回显；备份产物经统一流式接口上传，不经临时盘。
type StorageAccount struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Type       string    `gorm:"size:16;not null" json:"type"`                               // s3 / webdav / sftp
	Endpoint   string    `gorm:"size:255;not null" json:"endpoint"`                          // s3: host[:port]（配合 UseSSL）；webdav: https://host/dav；sftp: host[:port]
	Region     string    `gorm:"size:64" json:"region"`                                      // s3 region（OSS/COS 填地域，MinIO 可空）
	Bucket     string    `gorm:"size:128" json:"bucket"`                                     // s3 专用
	AccessKey  string    `gorm:"size:255" json:"accessKey"`                                  // s3 AK / webdav 用户名 / sftp 用户名
	SecretEnc  string    `gorm:"type:text" json:"-"`                                         // AES-GCM（s3 SK / webdav 密码 / sftp 密码或 PEM 私钥）
	BackupPath string    `gorm:"size:255;not null;default:ypanel-backups" json:"backupPath"` // 远端备份根路径（对象 key / 目录前缀）
	UseSSL     bool      `gorm:"not null;default:true" json:"useSSL"`                        // s3 是否 https
	Remark     string    `gorm:"size:255" json:"remark"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// MonitorProbe 监控探针（M29）：站点/容器/HTTP/TCP 可达性探测，状态翻转推送通知。
type MonitorProbe struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"size:64;not null" json:"name"`
	TargetType    string     `gorm:"size:16;not null;default:site" json:"targetType"` // site / container / http / tcp
	TargetRef     string     `gorm:"size:255;not null" json:"targetRef"`              // site=站点名；container=容器名；http=URL；tcp=host:port
	Method        string     `gorm:"size:8;not null;default:GET" json:"method"`       // http 用
	ExpectStatus  string     `gorm:"size:64;not null;default:2xx" json:"expectStatus"`
	Keyword       string     `gorm:"size:128" json:"keyword"` // http 响应体包含（可选）
	IntervalSec   int        `gorm:"not null;default:60" json:"intervalSec"`
	TimeoutSec    int        `gorm:"not null;default:5" json:"timeoutSec"`
	Retries       int        `gorm:"not null;default:3" json:"retries"`
	WebhookURL    string     `gorm:"size:512" json:"webhookUrl"`
	WebhookType   string     `gorm:"size:16;not null;default:generic" json:"webhookType"`
	Enabled       bool       `gorm:"not null;default:false" json:"enabled"` // 默认不开启
	LastStatus    string     `gorm:"size:8" json:"lastStatus"`              // up / down / ""（未检测）
	LastCheckedAt *time.Time `json:"lastCheckedAt"`
	LastDownAt    *time.Time `json:"lastDownAt"`
	LastError     string     `gorm:"size:512" json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// FileFavorite 文件收藏（M38）。
type FileFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Path      string    `gorm:"uniqueIndex;size:512;not null" json:"path"`
	Name      string    `gorm:"size:255" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// FileShare 文件分享链接（M38）：URL 携带原始 token，库存 SHA256；过期/禁用即 404。
type FileShare struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	TokenHash string     `gorm:"uniqueIndex;size:64;not null" json:"-"`
	Path      string     `gorm:"size:512;not null" json:"path"`
	Name      string     `gorm:"size:255" json:"name"`
	ExpireAt  *time.Time `json:"expireAt"`
	Enabled   bool       `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time  `json:"createdAt"`
}

// MCPOperation MCP 写操作审批（M45）：pending→approved→executing→succeeded/failed；10 分钟过期。
type MCPOperation struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Token      string     `gorm:"uniqueIndex;size:64;not null" json:"token"`
	Tool       string     `gorm:"size:64;not null" json:"tool"`
	Args       string     `gorm:"type:text" json:"args"`
	State      string     `gorm:"size:16;not null;default:pending;index" json:"state"`
	Result     string     `gorm:"type:text" json:"result"`
	ApprovedBy string     `gorm:"size:32" json:"approvedBy"`
	ApprovedAt *time.Time `json:"approvedAt"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// DockerEnvironment 多 Docker 环境（M35）：local=本机 agent dockerx 全量能力；tcp=core 直连远程 dockerd（核心子集）。
type DockerEnvironment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Type      string    `gorm:"size:8;not null;default:tcp" json:"type"` // local / tcp
	Endpoint  string    `gorm:"size:255;not null" json:"endpoint"`       // tcp: host:port（local 忽略）
	TLSEnable bool      `gorm:"not null;default:false" json:"tlsEnable"`
	TLSCA     string    `gorm:"type:text" json:"-"` // CA PEM（服务端校验）
	TLSCert   string    `gorm:"type:text" json:"-"` // 客户端证书 PEM
	TLSKeyEnc string    `gorm:"type:text" json:"-"` // 客户端私钥 PEM（AES-GCM）
	Remark    string    `gorm:"size:255" json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LogSearchQuery 日志中心查询历史/常用查询（pinned=true 为常用查询，M33 P2.5）。
type LogSearchQuery struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	QueryText      string     `gorm:"size:512;not null" json:"queryText"` // LogsQL 语句
	ContainersJSON string     `gorm:"type:text" json:"-"`                 // 容器过滤数组（JSON）
	Keyword        string     `gorm:"size:256" json:"keyword"`
	IsRegex        bool       `gorm:"not null;default:false" json:"isRegex"`
	RangeType      string     `gorm:"size:8;not null;default:1h" json:"rangeType"` // 15m/1h/6h/24h/7d/custom
	StartAt        *time.Time `json:"startAt"`                                     // custom 起止（UTC）
	EndAt          *time.Time `json:"endAt"`
	Pinned         bool       `gorm:"not null;default:false;index" json:"pinned"`
	Remark         string     `gorm:"size:64" json:"remark"` // 常用查询命名
	CreatedAt      time.Time  `json:"createdAt"`
}

// DiskGuardEvent 磁盘空间保护触发事件（M55）。
// 存在 RestoredAt 为空的事件即该节点处于触发态：容器被压制停机，需管理员一键恢复。
type DiskGuardEvent struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	NodeID         string     `gorm:"size:32;not null;index" json:"nodeId"`
	NodeName       string     `gorm:"size:64" json:"nodeName"`
	TriggeredAt    time.Time  `json:"triggeredAt"`
	RestoredAt     *time.Time `json:"restoredAt"`
	FreeBytes      uint64     `json:"freeBytes"`      // 触发时观测到的最低剩余
	ThresholdBytes uint64     `json:"thresholdBytes"` // 触发时阈值
	ContainersJSON string     `gorm:"type:text" json:"-"`
	Remark         string     `gorm:"size:512" json:"remark"` // 压制追停/恢复结果等备注
	CreatedAt      time.Time  `json:"createdAt"`
}
