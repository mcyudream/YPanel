// Package router 路由装配。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/api"
	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/core/internal/web"
	"github.com/ypanel/core/internal/wsbus"
)

// Deps 路由依赖。
type Deps struct {
	Auth       *service.Auth
	Nodes      *service.NodeService
	Settings   *service.SettingService
	Cron       *service.Cron
	Scripts    *service.ScriptService
	DBSvc      *service.DatabaseService
	AI         *service.AIService
	Acme       *service.AcmeService
	DBS        *service.DatabaseService
	Sites      *service.SiteService
	Certs      *service.CertificateService
	Groups     *service.SiteGroupService
	Store      *service.StoreService
	Vpn        *service.VPNService
	Tasks      *service.TaskService
	Src2       *service.Src2ComposeService
	Creds      *service.GitCredService
	FW         *service.FirewallService
	NatF       *service.NatForwardService
	DNS        *service.DnsService
	HS         *service.HostsService
	Alerts     *service.AlertService
	Notif      *service.NotificationService
	Probes     *service.ProbeService
	PanelBP    *service.PanelBackupService
	Hist       *service.HistoryRecorder
	F2B        *service.Fail2banService
	DBAdmin    *service.DBAdminService
	SU         *service.SelfUpdateService
	RT         *service.RuntimeService
	LogCentral *service.LogCentralService
	WebGW      *service.WebGwService
	DockerExt  *service.DockerExtService
	DockerInstall *service.DockerInstallService
	Rev        *service.RevisionService
	Sec        *service.SecuritySettingsService
	Storage    *service.StorageService
	BackupSrv  *service.BackupService
	DockerEnv  *service.DockerEnvService
	DockerImg  *service.DockerImgService
	FileExt    *service.FileExtService
	FileCross  *service.FileCrossService
	Nginx      *service.NginxService
	SysTool    *service.SystemToolService
	MCP        *service.MCPService
	OpsM       *service.MCPOperationService
	Snap       *service.SnapshotService
	SysSnap    *service.SystemSnapshotService
	Ftp        *service.FtpService
	SshG       *service.SshGuardService
	RBAC       *rbac.Service
	DiskGuard  *service.DiskGuardService
	Dashboard  *service.DashboardService
	Version    string
}

// CronDB 计划任务审计库句柄。
func (d *Deps) CronDB() *gorm.DB { return d.Auth.DB() }

// Setup 装配全部路由。
func Setup(d *Deps) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.EntryGate(d.Sec), middleware.AccessLog(), middleware.CORS(), middleware.SecurityGate(d.Sec))

	authAPI := &api.AuthAPI{Auth: d.Auth, Sec: d.Sec, RBAC: d.RBAC, Version: d.Version}
	securityAPI := &api.SecurityAPI{Sec: d.Sec, Auth: d.Auth}
	userAPI := &api.UserAPI{DB: d.Auth.DB(), RBAC: d.RBAC}
	roleAPI := &api.RoleAPI{RBAC: d.RBAC}
	sysAPI := &api.SystemAPI{Nodes: d.Nodes, Notif: d.Notif}
	dashboardAPI := &api.DashboardAPI{Svc: d.Dashboard}
	sysManageAPI := &api.SysManageAPI{Nodes: d.Nodes}
	fileAPI := &api.FileAPI{Nodes: d.Nodes, Rev: d.Rev, Cross: d.FileCross}
	nginxAPI := &api.NginxAPI{Ng: d.Nginx}
	dockerAPI := &api.DockerAPI{Nodes: d.Nodes}
	termAPI := &api.TerminalAPI{Nodes: d.Nodes}
	setAPI := &api.SettingsAPI{Settings: d.Settings}
	composeAPI := &api.ComposeAPI{Nodes: d.Nodes, Src2: d.Src2, Creds: d.Creds}
	credAPI := &api.GitCredAPI{Creds: d.Creds}
	cronAPI := &api.CronAPI{DB: d.CronDB(), Cron: d.Cron}
	scriptAPI := &api.ScriptAPI{Scripts: d.Scripts}
	aiAPI := &api.AIAPI{AI: d.AI, Nodes: d.Nodes}
	dbAPI := &api.DatabaseAPI{DBS: d.DBS}
	siteAPI := &api.SiteAPI{Sites: d.Sites, Acme: d.Acme, DNS: d.DNS}
	siteConfAPI := &api.SiteConfAPI{Sites: d.Sites, DNS: d.DNS}
	certAPI := &api.CertAPI{Certs: d.Certs, Groups: d.Groups}
	nodeAPI := &api.NodeAPI{Nodes: d.Nodes, SU: d.SU}
	storeAPI := &api.StoreAPI{Store: d.Store}
	vpnAPI := &api.VpnAPI{Vpn: d.Vpn}
	taskAPI := &api.TaskAPI{Tasks: d.Tasks}
	fwAPI := &api.FirewallAPI{FW: d.FW}
	natAPI := &api.NatForwardAPI{NF: d.NatF}
	dnsAPI := &api.DnsAPI{DNS: d.DNS, NF: d.NatF}
	hostsAPI := &api.HostsAPI{HS: d.HS}
	alertAPI := &api.AlertAPI{Alerts: d.Alerts, Settings: d.Settings}
	probeAPI := &api.ProbeAPI{Probes: d.Probes}
	f2bAPI := &api.Fail2banAPI{F2B: d.F2B}
	rtAPI := &api.RuntimeAPI{RT: d.RT}
	dbAdminAPI := &api.DBAdminAPI{Admin: d.DBAdmin}
	suAPI := &api.SelfUpdateAPI{SU: d.SU}
	notifAPI := &api.NotificationAPI{Notif: d.Notif, ParseToken: func(t string) error { _, err := d.Auth.ParseToken(t); return err }}
	procExecAPI := &api.NodeExecAPI{Nodes: d.Nodes}
	procProxy := &api.ProcProxy{Nodes: d.Nodes}
	dockerExtAPI := &api.DockerExtAPI{Ext: d.DockerExt}
	dockerInstallAPI := &api.DockerInstallAPI{Svc: d.DockerInstall}
	containerFileAPI := &api.ContainerFileAPI{Nodes: d.Nodes}
	logsAPI := &api.LogsAPI{Nodes: d.Nodes}
	logCentralAPI := &api.LogCentralAPI{LC: d.LogCentral}
	webgwAPI := &api.WebGwAPI{GW: d.WebGW}
	revisionAPI := &api.RevisionAPI{Rev: d.Rev}
	pbAPI := &api.PanelBackupAPI{BP: d.PanelBP}
	storageAPI := &api.StorageAPI{Svc: d.Storage, BK: d.BackupSrv}
	dockerEnvAPI := &api.DockerEnvAPI{Env: d.DockerEnv, Img: d.DockerImg, Nodes: d.Nodes}
	fileExtAPI := &api.FileExtAPI{Ext: d.FileExt, Nodes: d.Nodes}
	sysToolAPI := &api.SystemToolAPI{Tools: d.SysTool, Sec: d.Sec}
	mcpSrvAPI := &api.MCPServerAPI{MCP: d.MCP, Auth: d.Auth, Ops: d.OpsM, RBAC: d.RBAC}
	snapAPI := &api.SnapshotAPI{Snap: d.Snap}
	sysSnapAPI := &api.SystemSnapshotAPI{Snap: d.SysSnap}
	ftpAPI := &api.FtpAPI{Ftp: d.Ftp}
	sshAPI := &api.SshAPI{Ssh: d.SshG, F2B: d.F2B}
	diskGuardAPI := &api.DiskGuardAPI{G: d.DiskGuard}
	auditAPI := &api.AuditAPI{DB: d.CronDB(), Hist: d.Hist}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": d.Version})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authAPI.Login)
		v1.GET("/auth/2fa/status", authAPI.TwoFAStatus)
		v1.GET("/ws", wsbus.HandleGET(func(t string) error {
			_, err := d.Auth.ParseToken(t)
			return err
		}))
		v1.GET("/notifications/stream", notifAPI.Stream) // SSE：query token 自校验（EventSource 无法带 header）
		v1.POST("/pair", nodeAPI.Pair)
		v1.GET("/s/:token", fileExtAPI.ShareDownload) // M38 公开分享（无鉴权，token 即凭证）
		v1.POST("/pair/heartbeat", nodeAPI.Heartbeat)

		authed := v1.Group("", middleware.Auth(d.Auth, d.RBAC), middleware.NodeScope(), middleware.DangerLock(d.Sec), middleware.Audit(d.CronDB()))
		{
			pm := middleware.Perm // M54 RBAC：路由级权限点标注
			authed.GET("/auth/me", authAPI.Me)
			authed.GET("/app/permission", authAPI.AppPermission)
			authed.POST("/auth/logout", authAPI.Logout)
			authed.PUT("/auth/password", authAPI.ChangePassword)

			authed.GET("/system/overview", pm("dashboard:read"), sysAPI.Overview)
			authed.GET("/system/dashboard", pm("dashboard:read"), dashboardAPI.Dashboard)
			authed.GET("/system/hosts", pm("dashboard:read"), sysAPI.HostEntries)
			authed.GET("/system/history", pm("monitor:read"), sysAPI.History)
			authed.GET("/disk/usage", pm("dashboard:read"), sysAPI.DiskUsageTree)
			authed.POST("/system/manage", pm("host:manage"), sysManageAPI.Manage)

			authed.GET("/files/list", pm("file:read"), fileAPI.List)
			authed.GET("/files/read", pm("file:read"), fileAPI.Read)
			authed.GET("/files/download", pm("file:read"), fileAPI.Download) // 下载走 ?token=
			authed.GET("/files/upload", pm("file:read"), fileAPI.Upload)
			authed.POST("/files/upload", pm("file:write"), fileAPI.Upload)
				authed.POST("/files/copy-across", pm("file:write"), fileAPI.CopyAcross)
				// 节点 nginx 管理（M57 节点化基座）——/host 前缀避开站点域 /nginx/*（同组同路径 gin 启动 panic）
				authed.GET("/host/nginx/status", pm("host:read"), nginxAPI.Status)
				authed.POST("/host/nginx/install", pm("host:write"), nginxAPI.Install)
				authed.POST("/host/nginx/power", pm("host:write"), nginxAPI.Power)
				authed.POST("/host/nginx/reload", pm("host:write"), nginxAPI.Reload)
			authed.POST("/files/write", pm("file:write"), fileAPI.Write)
			authed.POST("/files/mkdir", pm("file:write"), fileAPI.Mkdir)
			authed.POST("/files/rename", pm("file:write"), fileAPI.Rename)
			authed.POST("/files/copy", pm("file:write"), fileAPI.Copy)
			authed.POST("/files/delete", pm("file:write"), fileAPI.Delete)
			authed.POST("/files/chmod", pm("file:write"), fileAPI.Chmod)
			authed.POST("/files/chown", pm("file:write"), fileAPI.Chown)
			authed.GET("/files/owners", pm("file:read"), fileAPI.Owners)
			authed.POST("/files/compress", pm("file:write"), fileAPI.Compress)
			authed.POST("/files/decompress", pm("file:write"), fileAPI.Decompress)
			authed.GET("/files/search", pm("file:read"), fileAPI.Search)

			authed.GET("/docker/containers", pm("docker:read"), dockerAPI.List)
			authed.GET("/docker/containers/:id/logs", pm("docker:read"), dockerAPI.Logs)
			authed.POST("/docker/containers/:id/:action", pm("docker:write"), dockerAPI.Action)
			authed.POST("/logs/search", pm("log:read"), logsAPI.Search)
			authed.GET("/logs/central/status", pm("log:read"), logCentralAPI.GetStatus)
			authed.GET("/logs/central/retention", pm("log:read"), logCentralAPI.GetRetention)
			authed.PUT("/logs/central/retention", pm("log:write"), logCentralAPI.SetRetention)
			authed.POST("/logs/central/query", pm("log:read"), logCentralAPI.Query)
			authed.POST("/logs/central/streams", pm("log:read"), logCentralAPI.StreamValues)
			authed.POST("/logs/central/hits", pm("log:read"), logCentralAPI.Hits)
			authed.GET("/logs/central/history", pm("log:read"), logCentralAPI.History)
			authed.POST("/logs/central/history", pm("log:write"), logCentralAPI.HistoryRecord)
			authed.PUT("/logs/central/history/:id/pin", pm("log:write"), logCentralAPI.HistoryPin)
			authed.DELETE("/logs/central/history/:id", pm("log:write"), logCentralAPI.HistoryDelete)

			authed.GET("/terminal", pm("terminal:access"), termAPI.Handle) // WS 走 ?token=

			authed.GET("/webgw/targets", pm("webgw:use"), webgwAPI.NodeTargets)    // M50：节点目标发现
			authed.POST("/webgw/session", pm("webgw:use"), webgwAPI.Create)        // M51 内网浏览器：创建代理会话（签发 gw Cookie）
			authed.GET("/webgw/session/:sid", pm("webgw:use"), webgwAPI.Get)       // 窗口重挂载复用校验
			authed.DELETE("/webgw/session/:sid", pm("webgw:use"), webgwAPI.Delete) // 关窗销毁（best effort）

			authed.GET("/settings", setAPI.Get)
			authed.PUT("/settings", pm("setting:write"), setAPI.Put)

			authed.GET("/compose/scan", pm("compose:read"), composeAPI.ScanCompose)
			authed.POST("/compose/adopt", pm("compose:write"), composeAPI.AdoptCompose)
			authed.GET("/compose/projects", pm("compose:read"), composeAPI.List)
			authed.GET("/compose/topology", pm("compose:read"), composeAPI.Topology)
			authed.GET("/compose/src2compose/templates", pm("compose:read"), composeAPI.Src2Templates)
			authed.GET("/compose/src2compose/preview/stream", pm("compose:read"), composeAPI.Src2PreviewStream)
			authed.POST("/compose/src2compose", pm("compose:write"), composeAPI.Src2Create)
			authed.GET("/compose/config", pm("compose:read"), composeAPI.Config)
			authed.POST("/compose/config", pm("compose:write"), composeAPI.Write)
			authed.POST("/compose/up", pm("compose:write"), composeAPI.Action("up"))
			authed.POST("/compose/down", pm("compose:write"), composeAPI.Action("down"))
			authed.POST("/compose/service-action", pm("compose:write"), composeAPI.ServiceAction)
			authed.DELETE("/compose/projects/:name", pm("compose:write"), composeAPI.ProjectDelete)
			authed.GET("/compose/logs", pm("compose:read"), composeAPI.Logs)

			authed.GET("/git/credentials", pm("compose:read"), credAPI.List)
			authed.POST("/git/credentials", pm("compose:write"), credAPI.Create)
			authed.PUT("/git/credentials/:id", pm("compose:write"), credAPI.Update)
			authed.DELETE("/git/credentials/:id", pm("compose:write"), credAPI.Delete)
			authed.GET("/git/credentials/match", pm("compose:read"), credAPI.Match)

			authed.GET("/ai/providers", pm("ai:use"), aiAPI.ListProviders)
			authed.GET("/ai/presets", pm("ai:use"), aiAPI.Presets)
			authed.POST("/ai/providers", pm("ai:admin"), aiAPI.SaveProvider)
			authed.GET("/ai/providers/:id/models", pm("ai:use"), aiAPI.ProviderModels)
			authed.DELETE("/ai/providers/:id", pm("ai:admin"), aiAPI.DeleteProvider)
			authed.POST("/ai/chat", pm("ai:use"), aiAPI.Chat)
			authed.POST("/ai/ask/:id", pm("ai:use"), aiAPI.ResolveAsk)
			authed.POST("/ai/ask-question/:id", pm("ai:use"), aiAPI.ResolveAskQuestion)
			authed.GET("/ai/oplogs", pm("ai:use"), aiAPI.ListOperationLogs)
			authed.GET("/ai/tools/ask-mode", pm("ai:use"), aiAPI.GetAskMode)
			authed.POST("/ai/tools/ask-mode", pm("ai:use"), aiAPI.SetAskMode)
			authed.GET("/ai/conversations", pm("ai:use"), aiAPI.ListConversations)
			authed.GET("/ai/conversations/:id", pm("ai:use"), aiAPI.GetConversation)
			authed.POST("/ai/conversations", pm("ai:use"), aiAPI.SaveConversation)
			authed.DELETE("/ai/conversations/:id", pm("ai:use"), aiAPI.DeleteConversation)
			authed.GET("/ai/memories", pm("ai:use"), aiAPI.ListMemories)
			authed.DELETE("/ai/memories", pm("ai:use"), aiAPI.ClearMemories)
			authed.GET("/ai/skills", pm("ai:use"), aiAPI.ListSkills)
			authed.POST("/ai/skills", pm("ai:use"), aiAPI.SaveSkill)
			authed.POST("/ai/skills/upload", pm("ai:use"), aiAPI.UploadSkillZip)
			authed.POST("/ai/skills/:name/enable", pm("ai:use"), aiAPI.SetSkillEnabled)
			authed.DELETE("/ai/skills/:name", pm("ai:use"), aiAPI.RemoveSkill)
			authed.GET("/ai/tools", pm("ai:use"), aiAPI.ListTools)
			authed.POST("/ai/tools/flag", pm("ai:admin"), aiAPI.SetToolFlag)
			authed.GET("/ai/mcp/servers", pm("ai:admin"), aiAPI.ListMCPServers)
			authed.PUT("/ai/mcp/servers", pm("ai:admin"), aiAPI.SaveMCPServers)
			authed.POST("/ai/mcp/test", pm("ai:admin"), aiAPI.TestMCP)
			authed.DELETE("/ai/mcp/:name", pm("ai:admin"), aiAPI.CloseMCP)
			authed.GET("/ai/knowledge", pm("ai:use"), aiAPI.ListKnowledge)
			authed.POST("/ai/knowledge", pm("ai:use"), aiAPI.SaveKnowledge)
			authed.DELETE("/ai/knowledge/:id", pm("ai:use"), aiAPI.DeleteKnowledge)
			authed.GET("/ai/knowledge/docs", pm("ai:use"), aiAPI.ListKnowledgeDocs)
			authed.POST("/ai/knowledge/doc", pm("ai:use"), aiAPI.SaveKnowledgeDoc)
			authed.GET("/ai/knowledge/doc/:id", pm("ai:use"), aiAPI.GetKnowledgeDoc)
			authed.DELETE("/ai/knowledge/doc/:id", pm("ai:use"), aiAPI.DeleteKnowledgeDoc)
			authed.GET("/ai/workspace", pm("ai:use"), aiAPI.WorkspaceList)
			authed.POST("/ai/workspace/run", pm("ai:use"), aiAPI.WorkspaceRun)
			authed.GET("/scripts", pm("script:read"), scriptAPI.List)
			authed.POST("/scripts", pm("script:write"), scriptAPI.Create)
			authed.PUT("/scripts/:id", pm("script:write"), scriptAPI.Update)
			authed.DELETE("/scripts/:id", pm("script:write"), scriptAPI.Delete)
			authed.POST("/scripts/:id/run", pm("script:write"), scriptAPI.Run)

			// M38：文件管理扩展
			authed.POST("/sites/batch", pm("site:write"), siteAPI.BatchOperate)
			authed.POST("/sites/:id/default", pm("site:write"), siteAPI.SetDefault)
			authed.PUT("/sites/:id/expire", pm("site:write"), siteAPI.SetExpire)
			authed.GET("/database/instances/:id/privileges", pm("db:read"), dbAPI.GrantMatrix)
			authed.PUT("/database/instances/:id/privileges", pm("db:write"), dbAPI.SetPrivileges)
			authed.GET("/database/instances/:id/variables", pm("db:read"), dbAPI.Variables)
			authed.PUT("/database/instances/:id/variables", pm("db:write"), dbAPI.SetVariable)
			authed.GET("/database/instances/:id/status", pm("db:read"), dbAPI.DBStatus)

			// M41：MCP 对外开放
			authed.GET("/mcp/status", pm("mcp:manage"), mcpSrvAPI.Status)
			authed.GET("/mcp/operations", pm("mcp:manage"), mcpSrvAPI.OpList)
			authed.PUT("/mcp/operations/:id", pm("mcp:manage"), mcpSrvAPI.OpApprove)
			v1.Any("/mcp", mcpSrvAPI.Handler) // MCP 客户端自带 Bearer 鉴权（Bearer/​?token=）

			authed.GET("/system/swap", pm("host:manage"), sysToolAPI.SwapStatus)
			authed.POST("/system/swap", pm("host:manage"), sysToolAPI.SwapApply)
			authed.GET("/system/bbr", pm("host:manage"), sysToolAPI.BBRStatus)
			authed.POST("/system/bbr", pm("host:manage"), sysToolAPI.BBRApply)
			authed.POST("/system/clean", pm("host:manage"), sysToolAPI.Clean)
			authed.GET("/security/password-policy", sysToolAPI.GetPasswordPolicy)
			authed.PUT("/security/password-policy", pm("setting:write"), sysToolAPI.PutPasswordPolicy)

			authed.POST("/files/trash", pm("file:write"), fileExtAPI.Trash)
			authed.GET("/files/trash/list", pm("file:read"), fileExtAPI.TrashList)
			authed.POST("/files/trash/restore", pm("file:write"), fileExtAPI.TrashRestore)
			authed.POST("/files/trash/purge", pm("file:write"), fileExtAPI.TrashPurge)
			authed.POST("/files/trash/clear", pm("file:write"), fileExtAPI.TrashClear)
			authed.GET("/files/favorites", pm("file:read"), fileExtAPI.Favorites)
			authed.POST("/files/favorites", pm("file:write"), fileExtAPI.FavoriteAdd)
			authed.DELETE("/files/favorites/:id", pm("file:write"), fileExtAPI.FavoriteRemove)
			authed.GET("/files/shares", pm("file:read"), fileExtAPI.Shares)
			authed.POST("/files/shares", pm("file:share"), fileExtAPI.ShareCreate)
			authed.DELETE("/files/shares/:id", pm("file:share"), fileExtAPI.ShareRevoke)
			authed.POST("/files/remote-download", pm("file:write"), fileExtAPI.RemoteDownload)
			authed.GET("/cron/tasks", pm("cron:read"), cronAPI.List)
			authed.POST("/cron/tasks", pm("cron:write"), cronAPI.Create)
			authed.PUT("/cron/tasks/:id", pm("cron:write"), cronAPI.Update)
			authed.DELETE("/cron/tasks/:id", pm("cron:write"), cronAPI.Delete)
			authed.POST("/cron/tasks/:id/run", pm("cron:write"), cronAPI.Run)
			authed.GET("/cron/logs", pm("cron:read"), cronAPI.Logs)

			authed.GET("/database/instances", pm("db:read"), dbAPI.List)
			authed.POST("/database/instances", pm("db:write"), dbAPI.Create)
			authed.POST("/database/instances/external", pm("db:write"), dbAPI.CreateExternal)
			authed.POST("/database/instances/adopt", pm("db:write"), dbAPI.Adopt)
			authed.DELETE("/database/instances/:id", pm("db:write"), dbAPI.Delete)
			authed.POST("/database/instances/:id/start", pm("db:write"), dbAPI.StartStop(true))
			authed.POST("/database/instances/:id/stop", pm("db:write"), dbAPI.StartStop(false))
			authed.GET("/database/instances/:id/reveal", pm("db:write"), dbAPI.Reveal)
			authed.PUT("/database/instances/:id/owner", pm("db:write"), dbAPI.SetOwner)
			authed.GET("/database/instances/:id/databases", pm("db:read"), dbAPI.Databases)
			authed.POST("/database/instances/:id/databases", pm("db:write"), dbAPI.CreateDatabase)
			authed.DELETE("/database/instances/:id/databases/:name", pm("db:write"), dbAPI.DropDatabase)
			authed.GET("/database/instances/:id/users", pm("db:read"), dbAPI.Users)
			authed.POST("/database/instances/:id/users", pm("db:write"), dbAPI.CreateUser)
			authed.DELETE("/database/instances/:id/users/:name", pm("db:write"), dbAPI.DropUser)
			authed.PUT("/database/instances/:id/users/:name/password", pm("db:write"), dbAPI.ChangeUserPassword)
			authed.GET("/database/instances/:id/remote", pm("db:read"), dbAPI.RemoteAccessStatus)
			authed.POST("/database/instances/:id/remote", pm("db:write"), dbAPI.RemoteAccess)
			authed.POST("/database/instances/:id/backups/import", pm("db:write"), dbAPI.BackupImport)
			authed.GET("/database/instances/:id/backups", pm("db:read"), dbAPI.Backups)
			authed.POST("/database/instances/:id/backups", pm("db:write"), dbAPI.CreateBackup)
			authed.DELETE("/database/instances/:id/backups", pm("db:write"), dbAPI.DeleteBackup)
			authed.POST("/database/instances/:id/backups/restore", pm("db:write"), dbAPI.RestoreBackup)

			authed.GET("/nginx/status", pm("site:read"), siteAPI.Status)
			authed.POST("/nginx/install", pm("site:write"), siteAPI.Install)
			authed.POST("/nginx/adopt-host", pm("site:write"), siteAPI.AdoptHost)
			authed.PUT("/nginx/mode", pm("site:write"), siteAPI.SetMode)
			authed.GET("/sites", pm("site:read"), siteAPI.List)
			authed.GET("/sites/scan", pm("site:read"), siteAPI.Scan)
			authed.GET("/sites/rewrite-templates", pm("site:read"), siteAPI.RewriteTemplates)
			authed.GET("/sites/:id/conf/domain", pm("site:read"), siteConfAPI.GetDomain)
			authed.PUT("/sites/:id/conf/domain", pm("site:write"), siteConfAPI.UpdateDomain)
			authed.GET("/sites/:id/conf/defaults", pm("site:read"), siteConfAPI.GetDefaults)
			authed.PUT("/sites/:id/conf/defaults", pm("site:write"), siteConfAPI.UpdateDefaults)
			authed.GET("/sites/:id/conf/proxy", pm("site:read"), siteConfAPI.GetProxy)
			authed.PUT("/sites/:id/conf/proxy", pm("site:write"), siteConfAPI.UpdateProxy)
			authed.GET("/sites/:id/conf/rewrite", pm("site:read"), siteConfAPI.GetRewrite)
			authed.PUT("/sites/:id/conf/rewrite", pm("site:write"), siteConfAPI.UpdateRewrite)
			authed.GET("/sites/:id/conf/https", pm("site:read"), siteConfAPI.GetHTTPS)
			authed.POST("/sites/:id/conf/https", pm("site:write"), siteConfAPI.EnableHTTPS)
			authed.PUT("/sites/:id/conf/https", pm("site:write"), siteConfAPI.UpdateHTTPS)
			authed.DELETE("/sites/:id/conf/https", pm("site:write"), siteConfAPI.DisableHTTPS)
			authed.GET("/sites/:id/conf/rundir", pm("site:read"), siteAPI.GetRunDir)
			authed.PUT("/sites/:id/conf/rundir", pm("site:write"), siteAPI.UpdateRunDir)
			authed.GET("/sites/:id/conf/antileech", pm("site:read"), siteConfAPI.GetAntiLeech)
			authed.PUT("/sites/:id/conf/antileech", pm("site:write"), siteConfAPI.UpdateAntiLeech)
			authed.GET("/sites/:id/conf/authbasic", pm("site:read"), siteConfAPI.GetAuthBasic)
			authed.PUT("/sites/:id/conf/authbasic", pm("site:write"), siteConfAPI.UpdateAuthBasic)
			authed.GET("/sites/:id/conf/cors", pm("site:read"), siteConfAPI.GetCORS)
			authed.PUT("/sites/:id/conf/cors", pm("site:write"), siteConfAPI.UpdateCORS)
			authed.GET("/sites/:id/conf/redirect", pm("site:read"), siteConfAPI.GetRedirect)
			authed.PUT("/sites/:id/conf/redirect", pm("site:write"), siteConfAPI.UpdateRedirect)
			authed.GET("/sites/:id/conf/realip", pm("site:read"), siteConfAPI.GetRealIP)
			authed.PUT("/sites/:id/conf/realip", pm("site:write"), siteConfAPI.UpdateRealIP)
			authed.GET("/sites/:id/conf/limitconn", pm("site:read"), siteConfAPI.GetLimitConn)
			authed.PUT("/sites/:id/conf/limitconn", pm("site:write"), siteConfAPI.UpdateLimitConn)
			authed.GET("/sites/:id/conf/loadbalance", pm("site:read"), siteConfAPI.GetLoadBalance)
			authed.PUT("/sites/:id/conf/loadbalance", pm("site:write"), siteConfAPI.UpdateLoadBalance)
			authed.GET("/sites/:id/conf/port", pm("site:read"), siteConfAPI.GetPort)
			authed.PUT("/sites/:id/conf/port", pm("site:write"), siteConfAPI.UpdatePort)
			authed.GET("/sites/:id/detail", pm("site:read"), siteAPI.GetSite)
			authed.GET("/sites/:id/ext", pm("site:read"), siteAPI.GetExt)
			authed.PUT("/sites/:id/ext", pm("site:write"), siteAPI.UpdateExt)
			authed.POST("/sites/adopt", pm("site:write"), siteAPI.Adopt)
			authed.POST("/sites", pm("site:write"), siteAPI.Create)
			authed.DELETE("/sites/:id", pm("site:write"), siteAPI.Delete)
			authed.POST("/sites/:id/enable", pm("site:write"), siteAPI.SetEnabled(true))
			authed.POST("/sites/:id/disable", pm("site:write"), siteAPI.SetEnabled(false))
			authed.GET("/sites/:id/config", pm("site:read"), siteAPI.Config)
			authed.GET("/sites/:id/logs", pm("site:read"), siteAPI.SiteLogs)
			authed.PUT("/sites/:id/config", pm("site:write"), siteAPI.UpdateConfig)
			authed.PUT("/sites/:id/owner", pm("site:write"), siteAPI.SetOwner)
			authed.PUT("/sites/:id/meta", pm("site:write"), siteAPI.UpdateMeta)
			authed.GET("/sites/:id/waf", pm("site:read"), siteAPI.GetWaf)
			authed.PUT("/sites/:id/waf", pm("site:write"), siteAPI.UpdateWaf)
			authed.POST("/sites/:id/cert/acme", pm("site:write"), siteAPI.IssueACME)
			authed.POST("/sites/:id/cert/selfsigned", pm("site:write"), siteAPI.IssueSelfSigned)

			// B23 证书库 / 站点分组
			authed.GET("/certs", pm("cert:read"), certAPI.List)
			authed.POST("/certs/issue", pm("cert:write"), certAPI.Issue)
			authed.POST("/certs/upload", pm("cert:write"), certAPI.Upload)
			authed.POST("/certs/selfsigned", pm("cert:write"), certAPI.SelfSigned)
			authed.GET("/certs/dns-accounts", pm("cert:read"), certAPI.ListDnsAccounts)
			authed.POST("/certs/dns-accounts", pm("cert:write"), certAPI.CreateDnsAccount)
			authed.PUT("/certs/dns-accounts/:id", pm("cert:write"), certAPI.UpdateDnsAccount)
			authed.DELETE("/certs/dns-accounts/:id", pm("cert:write"), certAPI.DeleteDnsAccount)
			authed.GET("/certs/acme-accounts", pm("cert:read"), certAPI.ListAcmeAccounts)
			authed.POST("/certs/acme-accounts", pm("cert:write"), certAPI.CreateAcmeAccount)
			authed.DELETE("/certs/acme-accounts/:id", pm("cert:write"), certAPI.DeleteAcmeAccount)
			authed.GET("/certs/:id", pm("cert:read"), certAPI.Detail)
			authed.PUT("/certs/:id", pm("cert:write"), certAPI.Update)
			authed.POST("/certs/:id/renew", pm("cert:write"), certAPI.Renew)
			authed.DELETE("/certs/:id", pm("cert:write"), certAPI.Remove)
			authed.GET("/site-groups", pm("cert:read"), certAPI.ListGroups)
			authed.POST("/site-groups", pm("cert:write"), certAPI.CreateGroup)
			authed.PUT("/site-groups/:id", pm("cert:write"), certAPI.UpdateGroup)
			authed.DELETE("/site-groups/:id", pm("cert:write"), certAPI.DeleteGroup)
			authed.POST("/site-groups/:id/default", pm("cert:write"), certAPI.SetDefaultGroup)

			// 原 Admin() 组已拆除：M54 起全部路由逐条标注权限点（super-admin 持 "*" 等价旧 admin 全量）
			admin := authed.Group("")
			{
				authed.GET("/nodes/metrics", pm("monitor:read"), nodeAPI.AggregateMetrics)
				admin.GET("/nodes", pm("node:read"), nodeAPI.List)
				admin.POST("/nodes/pairing-code", pm("node:manage"), nodeAPI.PairingCode)
				admin.GET("/nodes/agent-update/check", pm("node:manage"), nodeAPI.CheckAgentUpdate)
				admin.POST("/nodes/:id/upgrade-agent", pm("node:manage"), nodeAPI.UpgradeAgent)
				admin.DELETE("/nodes/:id", pm("node:manage"), nodeAPI.Delete)
				admin.PUT("/nodes/:id/asset", pm("node:manage"), nodeAPI.UpdateAsset)

				admin.POST("/auth/2fa/setup", pm("setting:write"), authAPI.TwoFASetup)
				admin.POST("/auth/2fa/disable", pm("setting:write"), authAPI.TwoFADisable)
				admin.GET("/security/settings", pm("setting:write"), securityAPI.Get)
				admin.PUT("/security/settings", pm("setting:write"), securityAPI.Update)

				// M55 磁盘空间保护
				admin.GET("/diskguard/status", pm("host:manage"), diskGuardAPI.Status)
				admin.PUT("/diskguard/config", pm("host:manage"), diskGuardAPI.UpdateConfig)
				admin.POST("/diskguard/restore", pm("host:manage"), diskGuardAPI.Restore)

				authed.GET("/probes", pm("monitor:read"), probeAPI.List)
				authed.POST("/probes", pm("monitor:write"), probeAPI.Create)
				authed.POST("/probes/test", pm("monitor:read"), probeAPI.Test)
				authed.PUT("/probes/:id", pm("monitor:write"), probeAPI.Update)
				authed.DELETE("/probes/:id", pm("monitor:write"), probeAPI.Delete)
				authed.POST("/probes/:id/enable", pm("monitor:write"), probeAPI.SetEnabled(true))
				authed.POST("/probes/:id/disable", pm("monitor:write"), probeAPI.SetEnabled(false))

				authed.GET("/vpn/easytier/status", pm("tool:vpn"), vpnAPI.Status)
				authed.GET("/vpn/easytier/config", pm("tool:vpn"), vpnAPI.GetConfig)
				authed.PUT("/vpn/easytier/config", pm("tool:vpn"), vpnAPI.SaveConfig)
				authed.POST("/vpn/easytier/install", pm("tool:vpn"), vpnAPI.Install)
				authed.POST("/vpn/easytier/apply", pm("tool:vpn"), vpnAPI.Apply)
				authed.GET("/vpn/easytier/peers", pm("tool:vpn"), vpnAPI.Peers)
				authed.GET("/vpn/easytier/routes", pm("tool:vpn"), vpnAPI.Routes)

				authed.GET("/store/sources", pm("store:read"), storeAPI.ListSources)
				authed.POST("/store/sources", pm("store:write"), storeAPI.CreateSource)
				authed.PUT("/store/sources/:id", pm("store:write"), storeAPI.UpdateSource)
				authed.POST("/store/sources/:id/enable", pm("store:write"), storeAPI.SetSourceEnabled(true))
				authed.POST("/store/sources/:id/disable", pm("store:write"), storeAPI.SetSourceEnabled(false))
				authed.DELETE("/store/sources/:id", pm("store:write"), storeAPI.DeleteSource)
				authed.POST("/store/sources/:id/sync", pm("store:write"), storeAPI.SyncSource)
				authed.POST("/store/sync", pm("store:write"), storeAPI.Sync)
				authed.GET("/store/apps", pm("store:read"), storeAPI.Apps)
				authed.GET("/store/tags", pm("store:read"), storeAPI.Tags)
				authed.GET("/store/apps/:sourceId/:key", pm("store:read"), storeAPI.Get)
				authed.GET("/store/apps/:sourceId/:key/icon", pm("store:read"), storeAPI.Icon)
				authed.GET("/store/installed", pm("store:read"), storeAPI.Installed)
				authed.POST("/store/install", pm("store:write"), storeAPI.Install)
				authed.DELETE("/store/install/:project", pm("store:write"), storeAPI.Uninstall)
				authed.POST("/store/installed/:project/:action", pm("store:write"), storeAPI.InstalledAction)
				authed.POST("/store/installed/:project/upgrade", pm("store:write"), storeAPI.UpgradeApp)
			authed.POST("/store/installed/:project/owner", pm("store:write"), storeAPI.SetOwner)
				authed.GET("/store/upgrades/check", pm("store:read"), storeAPI.CheckUpgrades)
				authed.POST("/store/local/scan", pm("store:write"), storeAPI.ScanLocalStore)
				authed.GET("/store/installed/:project/env", pm("store:read"), storeAPI.InstallEnv)
				authed.PUT("/store/installed/:project/env", pm("store:write"), storeAPI.SaveInstallEnv)

				authed.GET("/tasks", taskAPI.List)
				authed.GET("/tasks/:id", taskAPI.Get)
				authed.DELETE("/tasks/:id", taskAPI.Delete)
				authed.DELETE("/tasks", taskAPI.Clear)

				authed.GET("/firewall/status", pm("tool:firewall"), fwAPI.Status)
				authed.POST("/firewall/allow", pm("tool:firewall"), fwAPI.Allow)
				authed.DELETE("/firewall/rules/:number", pm("tool:firewall"), fwAPI.DeleteRule)
				authed.POST("/firewall/enable", pm("tool:firewall"), fwAPI.SetEnabled(true))
				authed.POST("/firewall/disable", pm("tool:firewall"), fwAPI.SetEnabled(false))

				admin.GET("/fail2ban/status", pm("tool:bruteforce"), f2bAPI.Status)
				admin.POST("/fail2ban/install", pm("tool:bruteforce"), f2bAPI.Install)
				admin.POST("/fail2ban/unban", pm("tool:bruteforce"), f2bAPI.Unban)
				admin.POST("/fail2ban/ban", pm("tool:bruteforce"), f2bAPI.Ban)

				// M24 NAT 端口转发（iptables DNAT）
				admin.GET("/nat/forwards", pm("tool:nat"), natAPI.List)
				admin.POST("/nat/forwards", pm("tool:nat"), natAPI.Save)
				admin.DELETE("/nat/forwards/:id", pm("tool:nat"), natAPI.Delete)
				admin.POST("/nat/forwards/:id/enable", pm("tool:nat"), natAPI.SetEnabled(true))
				admin.POST("/nat/forwards/:id/disable", pm("tool:nat"), natAPI.SetEnabled(false))
				admin.GET("/nat/interfaces", pm("tool:nat"), natAPI.Interfaces)
				admin.POST("/nat/check-port", pm("tool:nat"), natAPI.CheckPort)
				admin.POST("/nat/apply", pm("tool:nat"), natAPI.Apply)

				// M27 内网 DNS（dnsmasq 页面自管部署）
				admin.GET("/dns/overview", pm("tool:dns"), dnsAPI.Overview)
				admin.GET("/dns/records", pm("tool:dns"), dnsAPI.ListRecords)
				admin.POST("/dns/records", pm("tool:dns"), dnsAPI.SaveRecord)
				admin.DELETE("/dns/records/:id", pm("tool:dns"), dnsAPI.DeleteRecord)
				admin.POST("/dns/records/:id/enable", pm("tool:dns"), dnsAPI.SetRecordEnabled(true))
				admin.POST("/dns/records/:id/disable", pm("tool:dns"), dnsAPI.SetRecordEnabled(false))
				admin.GET("/dns/settings", pm("tool:dns"), dnsAPI.GetSettings)
				admin.PUT("/dns/settings", pm("tool:dns"), dnsAPI.SaveSettings)
				admin.POST("/dns/deploy", pm("tool:dns"), dnsAPI.Deploy)
				admin.POST("/dns/undeploy", pm("tool:dns"), dnsAPI.Undeploy)
				admin.POST("/dns/apply", pm("tool:dns"), dnsAPI.Apply)
				admin.POST("/dns/check", pm("tool:dns"), dnsAPI.CheckResolve)
				admin.GET("/dns/interfaces", pm("tool:dns"), dnsAPI.Interfaces)

				// M28 Hosts 可视化编辑（托管块分发）
				admin.GET("/hosts/records", pm("tool:hosts"), hostsAPI.ListRecords)
				admin.POST("/hosts/records", pm("tool:hosts"), hostsAPI.SaveRecord)
				admin.DELETE("/hosts/records/:id", pm("tool:hosts"), hostsAPI.DeleteRecord)
				admin.POST("/hosts/records/:id/enable", pm("tool:hosts"), hostsAPI.SetRecordEnabled(true))
				admin.POST("/hosts/records/:id/disable", pm("tool:hosts"), hostsAPI.SetRecordEnabled(false))
				admin.GET("/hosts/status", pm("tool:hosts"), hostsAPI.Status)
				admin.POST("/hosts/apply", pm("tool:hosts"), hostsAPI.Apply)
				admin.POST("/hosts/remove", pm("tool:hosts"), hostsAPI.Remove)

				admin.GET("/system/update/status", pm("update:manage"), suAPI.Status)
				admin.POST("/system/update/apply", pm("update:manage"), suAPI.Apply)
				admin.GET("/system/update/check", pm("update:manage"), suAPI.CheckOnline)
				admin.POST("/system/update/upgrade", pm("update:manage"), suAPI.Upgrade)

				dbAdmin := authed.Group("/plugin/db-admin")
				{
					dbAdmin.GET("/instances", pm("plugin.db-admin:use"), dbAdminAPI.Instances)
					dbAdmin.GET("/audits", pm("plugin.db-admin:use"), dbAdminAPI.Audits)
					dbAdmin.GET("/:id/ping", pm("plugin.db-admin:use"), dbAdminAPI.Ping)
					dbAdmin.GET("/:id/databases", pm("plugin.db-admin:use"), dbAdminAPI.Databases)
					dbAdmin.GET("/:id/schemas", pm("plugin.db-admin:use"), dbAdminAPI.Schemas)
					dbAdmin.GET("/:id/tables", pm("plugin.db-admin:use"), dbAdminAPI.Tables)
					dbAdmin.GET("/:id/columns", pm("plugin.db-admin:use"), dbAdminAPI.Columns)
					dbAdmin.GET("/:id/ddl", pm("plugin.db-admin:use"), dbAdminAPI.TableDDL)
					dbAdmin.GET("/:id/indexes", pm("plugin.db-admin:use"), dbAdminAPI.Indexes)
					dbAdmin.GET("/:id/pk", pm("plugin.db-admin:use"), dbAdminAPI.PrimaryKey)
					dbAdmin.POST("/:id/query", pm("plugin.db-admin:use"), dbAdminAPI.Query)
					dbAdmin.POST("/:id/browse", pm("plugin.db-admin:use"), dbAdminAPI.Browse)
					dbAdmin.POST("/:id/rows/insert", pm("plugin.db-admin:use"), dbAdminAPI.RowInsert)
					dbAdmin.POST("/:id/rows/update", pm("plugin.db-admin:use"), dbAdminAPI.RowUpdate)
					dbAdmin.POST("/:id/rows/delete", pm("plugin.db-admin:use"), dbAdminAPI.RowDelete)
					dbAdmin.POST("/:id/indexes/create", pm("plugin.db-admin:use"), dbAdminAPI.IndexCreate)
					dbAdmin.POST("/:id/indexes/drop", pm("plugin.db-admin:use"), dbAdminAPI.IndexDrop)
					dbAdmin.GET("/:id/redis/keys", pm("plugin.db-admin:use"), dbAdminAPI.RedisKeys)
					dbAdmin.GET("/:id/redis/key", pm("plugin.db-admin:use"), dbAdminAPI.RedisKey)
					dbAdmin.POST("/:id/redis/key", pm("plugin.db-admin:use"), dbAdminAPI.RedisKeyWrite)
					dbAdmin.POST("/:id/redis/key/delete", pm("plugin.db-admin:use"), dbAdminAPI.RedisKeyDelete)
					dbAdmin.POST("/:id/redis/key/ttl", pm("plugin.db-admin:use"), dbAdminAPI.RedisKeyTTL)
					dbAdmin.POST("/:id/redis/exec", pm("plugin.db-admin:use"), dbAdminAPI.RedisExec)
					dbAdmin.POST("/:id/mongo/docs", pm("plugin.db-admin:use"), dbAdminAPI.MongoDocs)
					dbAdmin.POST("/:id/mongo/aggregate", pm("plugin.db-admin:use"), dbAdminAPI.MongoAggregate)
					dbAdmin.GET("/:id/mongo/doc", pm("plugin.db-admin:use"), dbAdminAPI.MongoDoc)
					dbAdmin.POST("/:id/mongo/doc", pm("plugin.db-admin:use"), dbAdminAPI.MongoDocInsert)
					dbAdmin.POST("/:id/mongo/doc/update", pm("plugin.db-admin:use"), dbAdminAPI.MongoDocUpdate)
					dbAdmin.POST("/:id/mongo/doc/delete", pm("plugin.db-admin:use"), dbAdminAPI.MongoDocDelete)
					dbAdmin.GET("/:id/mongo/indexes", pm("plugin.db-admin:use"), dbAdminAPI.MongoIndexes)
					dbAdmin.POST("/:id/mongo/indexes/create", pm("plugin.db-admin:use"), dbAdminAPI.MongoIndexCreate)
					dbAdmin.POST("/:id/mongo/indexes/drop", pm("plugin.db-admin:use"), dbAdminAPI.MongoIndexDrop)
					dbAdmin.POST("/:id/mongo/collection", pm("plugin.db-admin:use"), dbAdminAPI.MongoCollection)
					dbAdmin.POST("/:id/sql/import", pm("plugin.db-admin:use"), dbAdminAPI.ImportSQL)
					dbAdmin.POST("/:id/mongo/import", pm("plugin.db-admin:use"), dbAdminAPI.MongoImport)
					dbAdmin.GET("/migrate/preview", pm("plugin.db-admin:use"), dbAdminAPI.MigratePreview)
					dbAdmin.POST("/migrate/start", pm("plugin.db-admin:use"), dbAdminAPI.MigrateStart)
				}

				admin.GET("/alert/rules", pm("alert:read"), alertAPI.ListRules)
				admin.POST("/alert/rules", pm("alert:write"), alertAPI.CreateRule)
				admin.PUT("/alert/rules/:id", pm("alert:write"), alertAPI.UpdateRule)
				admin.DELETE("/alert/rules/:id", pm("alert:write"), alertAPI.DeleteRule)
				admin.GET("/alert/smtp", pm("alert:read"), alertAPI.GetSMTPSettings)
				admin.PUT("/alert/smtp", pm("alert:write"), alertAPI.PutSMTPSettings)
				admin.POST("/alert/smtp/test", pm("alert:write"), alertAPI.TestSMTP)

				authed.GET("/notifications", notifAPI.List)
				authed.GET("/notifications/unread", notifAPI.Unread)
				authed.POST("/notifications/read", notifAPI.MarkRead)
				authed.GET("/system/history/persisted", pm("audit:read"), auditAPI.History)

				authed.GET("/runtimes", pm("runtime:read"), rtAPI.List)
				authed.GET("/runtimes/:id", pm("runtime:read"), rtAPI.Detail)
				authed.POST("/runtimes", pm("runtime:write"), rtAPI.Create)
				authed.POST("/runtimes/external", pm("runtime:write"), rtAPI.AttachExternal)
				authed.DELETE("/runtimes/:id", pm("runtime:write"), rtAPI.Delete)
				authed.POST("/runtimes/:id/start", pm("runtime:write"), rtAPI.Operate("start"))
				authed.POST("/runtimes/:id/stop", pm("runtime:write"), rtAPI.Operate("stop"))
				authed.POST("/runtimes/:id/restart", pm("runtime:write"), rtAPI.Operate("restart"))
				authed.POST("/runtimes/:id/rebuild", pm("runtime:write"), rtAPI.Rebuild)
				authed.GET("/runtimes/:id/backups", pm("runtime:read"), rtAPI.BackupList)
				authed.POST("/runtimes/:id/backups", pm("runtime:write"), rtAPI.BackupCreate)
				authed.POST("/runtimes/:id/backups/restore", pm("runtime:write"), rtAPI.BackupRestore)
				authed.DELETE("/runtimes/:id/backups", pm("runtime:write"), rtAPI.BackupDelete)

				authed.GET("/runtimes/php/catalog", pm("runtime:read"), rtAPI.PHPExtensionCatalog)
				authed.GET("/runtimes/php/extensions", pm("runtime:read"), rtAPI.PHPExtensions)
				authed.POST("/runtimes/php/extensions/install", pm("runtime:write"), rtAPI.PHPExtensionInstall)
				authed.POST("/runtimes/php/extensions/uninstall", pm("runtime:write"), rtAPI.PHPExtensionUninstall)
				authed.GET("/runtimes/php/config", pm("runtime:read"), rtAPI.GetPHPConfig)
				authed.POST("/runtimes/php/config", pm("runtime:write"), rtAPI.UpdatePHPConfig)
				authed.GET("/runtimes/php/fpm-config", pm("runtime:read"), rtAPI.GetFPMConfig)
				authed.POST("/runtimes/php/fpm-config", pm("runtime:write"), rtAPI.UpdateFPMConfig)
				authed.GET("/runtimes/php/fpm-status", pm("runtime:read"), rtAPI.FPMStatus)
				authed.GET("/runtimes/php/supervisor", pm("runtime:read"), rtAPI.SupervisorList)
				authed.POST("/runtimes/php/supervisor", pm("runtime:write"), rtAPI.SupervisorUpsert)
				authed.POST("/runtimes/php/supervisor/operate", pm("runtime:write"), rtAPI.SupervisorOperate)
				authed.DELETE("/runtimes/php/supervisor", pm("runtime:write"), rtAPI.SupervisorDelete)
				authed.GET("/runtimes/php/supervisor/log", pm("runtime:read"), rtAPI.SupervisorLog)
				authed.GET("/runtimes/php/slow-log", pm("runtime:read"), rtAPI.SlowLog)
				authed.POST("/runtimes/php/slow-log/clear", pm("runtime:write"), rtAPI.SlowLogClear)

				authed.GET("/runtimes/node/modules", pm("runtime:read"), rtAPI.NodeModules)
				authed.POST("/runtimes/node/modules/operate", pm("runtime:write"), rtAPI.OperateNodeModule)

				authed.GET("/docker/images", pm("docker:read"), dockerExtAPI.Images)
				authed.POST("/docker/images/pull", pm("docker:write"), dockerExtAPI.ImagePull)
				// *id 通配：镜像引用含斜杠（registry/name:tag），单段 :id 会路由 miss 报 404
				authed.DELETE("/docker/images/*id", pm("docker:write"), dockerExtAPI.ImageRemove)
				authed.POST("/docker/images/prune", pm("docker:write"), dockerExtAPI.ImagesPrune)
				authed.GET("/docker/networks", pm("docker:read"), dockerExtAPI.Networks)
				authed.POST("/docker/networks", pm("docker:write"), dockerExtAPI.NetworkCreate)
				authed.DELETE("/docker/networks/:name", pm("docker:write"), dockerExtAPI.NetworkRemove)
				authed.GET("/docker/volumes", pm("docker:read"), dockerExtAPI.Volumes)
				authed.POST("/docker/volumes", pm("docker:write"), dockerExtAPI.VolumeCreate)
				authed.DELETE("/docker/volumes/:name", pm("docker:write"), dockerExtAPI.VolumeRemove)
				authed.POST("/docker/volumes/prune", pm("docker:write"), dockerExtAPI.VolumesPrune)
				authed.GET("/docker/usage", pm("docker:read"), dockerExtAPI.Usage)
				authed.POST("/docker/buildcache/prune", pm("docker:write"), dockerExtAPI.BuildCachePrune)
				authed.POST("/docker/containers/prune", pm("docker:write"), dockerExtAPI.ContainersPrune)
				authed.GET("/docker/containers/:id/inspect", pm("docker:read"), dockerExtAPI.ContainerInspect)
				authed.GET("/docker/containers/:id/rootfs", pm("docker:read"), dockerExtAPI.ContainerRootfs)
				authed.GET("/docker/containers/:id/stats", pm("docker:read"), dockerExtAPI.ContainerStats)
				authed.GET("/docker/containers/:id/last-op-err", pm("docker:read"), dockerAPI.ContainerLastOpErr)
				authed.GET("/docker/containers/:id/exec", pm("docker:terminal"), dockerExtAPI.ContainerExecWS)
				authed.POST("/docker/containers", pm("docker:write"), dockerExtAPI.ContainerCreate)
				authed.POST("/docker/containers/:id/recreate", pm("docker:write"), dockerExtAPI.ContainerRecreate)
				authed.POST("/docker/containers/:id/update", pm("docker:write"), dockerExtAPI.ContainerUpdate)
				authed.DELETE("/docker/containers/:id", pm("docker:write"), dockerExtAPI.ContainerRemove)
				// 容器内文件管理（tar 归档 + 容器内 exec）
				authed.GET("/docker/containers/:id/files/list", pm("docker:read"), containerFileAPI.List)
				authed.GET("/docker/containers/:id/files/read", pm("docker:read"), containerFileAPI.Read)
				authed.GET("/docker/containers/:id/files/download", pm("docker:read"), containerFileAPI.Download)
				authed.POST("/docker/containers/:id/files/write", pm("docker:write"), containerFileAPI.Write)
				authed.POST("/docker/containers/:id/files/mkdir", pm("docker:write"), containerFileAPI.Mkdir)
				authed.POST("/docker/containers/:id/files/rename", pm("docker:write"), containerFileAPI.Rename)
				authed.POST("/docker/containers/:id/files/delete", pm("docker:write"), containerFileAPI.Delete)
				authed.POST("/docker/containers/:id/files/chmod", pm("docker:write"), containerFileAPI.Chmod)
				authed.POST("/docker/containers/:id/files/chown", pm("docker:write"), containerFileAPI.Chown)
				authed.POST("/docker/containers/:id/files/upload", pm("docker:write"), containerFileAPI.Upload)
				authed.GET("/docker/registry", pm("docker:read"), dockerExtAPI.RegistryList)
				authed.PUT("/docker/registry", pm("docker:write"), dockerExtAPI.RegistrySet)
				authed.DELETE("/docker/registry", pm("docker:write"), dockerExtAPI.RegistryRemove)
				authed.GET("/docker/daemon-config", pm("docker:read"), dockerExtAPI.DaemonConfig)
				authed.PUT("/docker/daemon-config", pm("docker:write"), dockerExtAPI.UpdateDaemonConfig)

				// M52：Docker/Compose 一键安装 + 镜像加速器
				authed.POST("/docker/install/precheck", pm("docker:read"), dockerInstallAPI.Precheck)
				admin.POST("/docker/install", pm("docker:write"), dockerInstallAPI.Install)
				authed.GET("/docker/registry-mirrors", pm("docker:read"), dockerInstallAPI.GetMirrors)
				authed.POST("/docker/registry-mirrors", pm("docker:write"), dockerInstallAPI.SetMirrors)

				// M35：镜像生命周期 + 容器增强 + Swarm 检测
				authed.POST("/docker/images/build", pm("docker:write"), dockerEnvAPI.ImageBuild)
				authed.POST("/docker/images/save", pm("docker:write"), dockerEnvAPI.ImageSave)
				authed.POST("/docker/images/load", pm("docker:write"), dockerEnvAPI.ImageLoad)
				authed.POST("/docker/images/tag", pm("docker:write"), dockerEnvAPI.ImageTag)
				authed.POST("/docker/images/check-updates", pm("docker:read"), dockerEnvAPI.ImageCheckUpdates)
				authed.POST("/docker/containers/:id/commit", pm("docker:write"), dockerEnvAPI.ContainerCommit)
				authed.POST("/docker/containers/:id/backup", pm("docker:write"), dockerEnvAPI.ContainerBackup)
				authed.GET("/docker/swarm-status", pm("docker:read"), dockerEnvAPI.SwarmStatus)

				// M35：多 Docker 环境（CRUD 挂 admin，资源查看经 authed）
				admin.GET("/docker/environments", pm("node:manage"), dockerEnvAPI.ListEnvs)
				admin.POST("/docker/environments", pm("node:manage"), dockerEnvAPI.CreateEnv)
				admin.PUT("/docker/environments/:id", pm("node:manage"), dockerEnvAPI.UpdateEnv)
				admin.DELETE("/docker/environments/:id", pm("node:manage"), dockerEnvAPI.DeleteEnv)
				admin.POST("/docker/environments/:id/test", pm("node:manage"), dockerEnvAPI.TestEnv)
				admin.GET("/docker/environments/:id/containers", pm("docker:read"), dockerEnvAPI.EnvContainers)
				admin.GET("/docker/environments/:id/images", pm("docker:read"), dockerEnvAPI.EnvImages)
				admin.POST("/docker/environments/:id/containers/:name/:action", pm("docker:write"), dockerEnvAPI.EnvContainerAction)
				admin.DELETE("/docker/environments/:id/images/:image", pm("docker:write"), dockerEnvAPI.EnvImageRemove)

				// 受管配置版本快照（M23）
				authed.GET("/config-revisions", pm("file:read"), revisionAPI.List)
				authed.GET("/config-revisions/:id", pm("file:read"), revisionAPI.Get)
				authed.POST("/config-revisions/:id/restore", pm("file:write"), revisionAPI.Restore)

				admin.POST("/nodes/exec", pm("node:exec"), procExecAPI.Exec)
				authed.GET("/processes", pm("monitor:read"), procProxy.Processes)
				authed.POST("/processes/kill", pm("host:manage"), procProxy.KillProcess)
				authed.GET("/services", pm("monitor:read"), procProxy.Services)
				authed.POST("/services/:name/:action", pm("host:manage"), procProxy.ServiceAction)

				admin.GET("/audit/ops", pm("audit:read"), auditAPI.List)
				// M47：系统级快照（对齐 1Panel）
				// M49：工具箱扩展（FTP / SSH 管理 / 暴力破解防护）
				admin.GET("/ftp/status", pm("tool:ftp"), ftpAPI.Status)
				admin.POST("/ftp/install", pm("tool:ftp"), ftpAPI.Install)
				admin.POST("/ftp/power", pm("tool:ftp"), ftpAPI.Power)
				admin.POST("/ftp/port", pm("tool:ftp"), ftpAPI.SetPort)
				admin.GET("/ftp/config", pm("tool:ftp"), ftpAPI.GetConfig)
				admin.PUT("/ftp/config", pm("tool:ftp"), ftpAPI.PutConfig)
				admin.GET("/ssh/config", pm("tool:ssh"), sshAPI.GetConfig)
				admin.PUT("/ssh/config", pm("tool:ssh"), sshAPI.SetConfig)
				admin.GET("/ssh/keys", pm("tool:ssh"), sshAPI.Keys)
				admin.POST("/ssh/keys/generate", pm("tool:ssh"), sshAPI.GenerateKey)
				admin.POST("/ssh/keys/import", pm("tool:ssh"), sshAPI.ImportKey)
				admin.POST("/ssh/keys/delete", pm("tool:ssh"), sshAPI.DeleteKey)
				admin.POST("/ssh/keys/deploy", pm("tool:ssh"), sshAPI.DeployKey)
				admin.POST("/ssh/keys/undeploy", pm("tool:ssh"), sshAPI.UndeployKey)
				admin.GET("/ssh/attempts", pm("tool:ssh"), sshAPI.FailedAttempts)

				admin.GET("/system/snapshots", pm("backup:manage"), sysSnapAPI.List)
				admin.POST("/system/snapshots", pm("backup:manage"), sysSnapAPI.Create)
				admin.DELETE("/system/snapshots", pm("backup:manage"), sysSnapAPI.Delete)
				admin.POST("/system/snapshots/restore", pm("backup:manage"), sysSnapAPI.Restore)
				admin.GET("/system/snapshots/keep", pm("backup:manage"), sysSnapAPI.GetKeep)
				admin.PUT("/system/snapshots/keep", pm("backup:manage"), sysSnapAPI.SetKeep)

				admin.GET("/snapshots", pm("backup:manage"), snapAPI.List)
				admin.GET("/snapshots/plan", pm("backup:manage"), snapAPI.Plan)
				admin.POST("/snapshots/execute", pm("backup:manage"), snapAPI.Execute)
				admin.POST("/snapshots/import", pm("backup:manage"), snapAPI.Import)
				admin.GET("/snapshots/keep", pm("backup:manage"), snapAPI.GetKeep)
				admin.PUT("/snapshots/keep", pm("backup:manage"), snapAPI.SetKeep)
				admin.POST("/snapshots/prune", pm("backup:manage"), snapAPI.Prune)
				admin.GET("/panel/backups", pm("backup:manage"), pbAPI.List)
				admin.POST("/panel/backups", pm("backup:manage"), pbAPI.Create)
				admin.DELETE("/panel/backups", pm("backup:manage"), pbAPI.Delete)
				admin.GET("/panel/backups/restore-hint", pm("backup:manage"), pbAPI.RestoreHint)

				// 远程备份存储 + 目录/编排备份（M34）
				admin.GET("/storage-accounts", pm("backup:manage"), storageAPI.List)
				admin.POST("/storage-accounts", pm("backup:manage"), storageAPI.Create)
				admin.PUT("/storage-accounts/:id", pm("backup:manage"), storageAPI.Update)
				admin.DELETE("/storage-accounts/:id", pm("backup:manage"), storageAPI.Delete)
				admin.POST("/storage-accounts/:id/test", pm("backup:manage"), storageAPI.Test)
				admin.GET("/storage-accounts/:id/objects", pm("backup:manage"), storageAPI.Objects)
				admin.DELETE("/storage-accounts/:id/object", pm("backup:manage"), storageAPI.DeleteObject)
				admin.POST("/storage-accounts/:id/fetch", pm("backup:manage"), storageAPI.Fetch)
				admin.POST("/backups/dir", pm("backup:manage"), storageAPI.DirBackup)
				admin.PUT("/mcp/status", pm("backup:manage"), mcpSrvAPI.SetEnabled)
				admin.POST("/backups/compose", pm("backup:manage"), storageAPI.ComposeBackup)

				admin.GET("/users", pm("user:manage"), userAPI.List)
				admin.POST("/users", pm("user:manage"), userAPI.Create)
				admin.PUT("/users/:id", pm("user:manage"), userAPI.Update)
				admin.DELETE("/users/:id", pm("user:manage"), userAPI.Delete)
				admin.GET("/audit/logins", pm("audit:read"), userAPI.LoginLogs)

				// M54 RBAC：角色管理与权限目录
				authed.GET("/rbac/catalog", pm("user:manage"), roleAPI.Catalog)
				authed.GET("/rbac/roles", pm("user:manage"), roleAPI.List)
				authed.POST("/rbac/roles", pm("user:manage"), roleAPI.Create)
				authed.PUT("/rbac/roles/:id", pm("user:manage"), roleAPI.Update)
				authed.DELETE("/rbac/roles/:id", pm("user:manage"), roleAPI.Delete)
			}
		}
	}

	if err := web.Register(r); err != nil {
		return nil, err
	}
	return r, nil
}
