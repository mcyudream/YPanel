// Package router 路由装配。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/api"
	"github.com/ypanel/core/internal/middleware"
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
	SysTool    *service.SystemToolService
	MCP        *service.MCPService
	OpsM       *service.MCPOperationService
	Snap       *service.SnapshotService
	SysSnap    *service.SystemSnapshotService
	Ftp        *service.FtpService
	SshG       *service.SshGuardService
	Dashboard  *service.DashboardService
	Version    string
}

// CronDB 计划任务审计库句柄。
func (d *Deps) CronDB() *gorm.DB { return d.Auth.DB() }

// Setup 装配全部路由。
func Setup(d *Deps) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.EntryGate(d.Sec), middleware.AccessLog(), middleware.CORS(), middleware.SecurityGate(d.Sec))

	authAPI := &api.AuthAPI{Auth: d.Auth, Sec: d.Sec, Version: d.Version}
	securityAPI := &api.SecurityAPI{Sec: d.Sec, Auth: d.Auth}
	userAPI := &api.UserAPI{DB: d.Auth.DB()}
	sysAPI := &api.SystemAPI{Nodes: d.Nodes, Notif: d.Notif}
	dashboardAPI := &api.DashboardAPI{Svc: d.Dashboard}
	sysManageAPI := &api.SysManageAPI{Nodes: d.Nodes}
	fileAPI := &api.FileAPI{Nodes: d.Nodes, Rev: d.Rev}
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
	mcpSrvAPI := &api.MCPServerAPI{MCP: d.MCP, Auth: d.Auth, Ops: d.OpsM}
	snapAPI := &api.SnapshotAPI{Snap: d.Snap}
	sysSnapAPI := &api.SystemSnapshotAPI{Snap: d.SysSnap}
	ftpAPI := &api.FtpAPI{Ftp: d.Ftp}
	sshAPI := &api.SshAPI{Ssh: d.SshG, F2B: d.F2B}
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

		authed := v1.Group("", middleware.Auth(d.Auth), middleware.DangerLock(d.Sec), middleware.Audit(d.CronDB()))
		{
			authed.GET("/auth/me", authAPI.Me)
			authed.POST("/auth/logout", authAPI.Logout)
			authed.PUT("/auth/password", authAPI.ChangePassword)

			authed.GET("/system/overview", sysAPI.Overview)
			authed.GET("/system/dashboard", dashboardAPI.Dashboard)
			authed.GET("/system/hosts", sysAPI.HostEntries)
			authed.GET("/system/history", sysAPI.History)
			authed.GET("/disk/usage", sysAPI.DiskUsageTree)
			authed.POST("/system/manage", sysManageAPI.Manage)

			authed.GET("/files/list", fileAPI.List)
			authed.GET("/files/read", fileAPI.Read)
			authed.GET("/files/download", fileAPI.Download) // 下载走 ?token=
			authed.GET("/files/upload", fileAPI.Upload)
			authed.POST("/files/upload", fileAPI.Upload)
			authed.POST("/files/write", fileAPI.Write)
			authed.POST("/files/mkdir", fileAPI.Mkdir)
			authed.POST("/files/rename", fileAPI.Rename)
			authed.POST("/files/copy", fileAPI.Copy)
			authed.POST("/files/delete", fileAPI.Delete)
			authed.POST("/files/chmod", fileAPI.Chmod)
			authed.POST("/files/chown", fileAPI.Chown)
			authed.GET("/files/owners", fileAPI.Owners)
			authed.POST("/files/compress", fileAPI.Compress)
			authed.POST("/files/decompress", fileAPI.Decompress)
			authed.GET("/files/search", fileAPI.Search)

			authed.GET("/docker/containers", dockerAPI.List)
			authed.GET("/docker/containers/:id/logs", dockerAPI.Logs)
			authed.POST("/docker/containers/:id/:action", dockerAPI.Action)
			authed.POST("/logs/search", logsAPI.Search)
			authed.GET("/logs/central/status", logCentralAPI.GetStatus)
			authed.GET("/logs/central/retention", logCentralAPI.GetRetention)
			authed.PUT("/logs/central/retention", logCentralAPI.SetRetention)
			authed.POST("/logs/central/query", logCentralAPI.Query)
			authed.POST("/logs/central/streams", logCentralAPI.StreamValues)
			authed.POST("/logs/central/hits", logCentralAPI.Hits)
			authed.GET("/logs/central/history", logCentralAPI.History)
			authed.POST("/logs/central/history", logCentralAPI.HistoryRecord)
			authed.PUT("/logs/central/history/:id/pin", logCentralAPI.HistoryPin)
			authed.DELETE("/logs/central/history/:id", logCentralAPI.HistoryDelete)

			authed.GET("/terminal", termAPI.Handle) // WS 走 ?token=

			authed.GET("/webgw/targets", webgwAPI.NodeTargets)    // M50：节点目标发现
			authed.POST("/webgw/session", webgwAPI.Create)        // M51 内网浏览器：创建代理会话（签发 gw Cookie）
			authed.GET("/webgw/session/:sid", webgwAPI.Get)       // 窗口重挂载复用校验
			authed.DELETE("/webgw/session/:sid", webgwAPI.Delete) // 关窗销毁（best effort）

			authed.GET("/settings", setAPI.Get)
			authed.PUT("/settings", setAPI.Put)

			authed.GET("/compose/scan", composeAPI.ScanCompose)
			authed.POST("/compose/adopt", composeAPI.AdoptCompose)
			authed.GET("/compose/projects", composeAPI.List)
			authed.GET("/compose/topology", composeAPI.Topology)
			authed.GET("/compose/src2compose/templates", composeAPI.Src2Templates)
			authed.GET("/compose/src2compose/preview/stream", composeAPI.Src2PreviewStream)
			authed.POST("/compose/src2compose", composeAPI.Src2Create)
			authed.GET("/compose/config", composeAPI.Config)
			authed.POST("/compose/config", composeAPI.Write)
			authed.POST("/compose/up", composeAPI.Action("up"))
			authed.POST("/compose/down", composeAPI.Action("down"))
			authed.POST("/compose/service-action", composeAPI.ServiceAction)
			authed.DELETE("/compose/projects/:name", composeAPI.ProjectDelete)
			authed.GET("/compose/logs", composeAPI.Logs)

			authed.GET("/git/credentials", credAPI.List)
			authed.POST("/git/credentials", credAPI.Create)
			authed.PUT("/git/credentials/:id", credAPI.Update)
			authed.DELETE("/git/credentials/:id", credAPI.Delete)
			authed.GET("/git/credentials/match", credAPI.Match)

			authed.GET("/ai/providers", aiAPI.ListProviders)
			authed.GET("/ai/presets", aiAPI.Presets)
			authed.POST("/ai/providers", aiAPI.SaveProvider)
			authed.GET("/ai/providers/:id/models", aiAPI.ProviderModels)
			authed.DELETE("/ai/providers/:id", aiAPI.DeleteProvider)
			authed.POST("/ai/chat", aiAPI.Chat)
			authed.POST("/ai/ask/:id", aiAPI.ResolveAsk)
			authed.POST("/ai/ask-question/:id", aiAPI.ResolveAskQuestion)
			authed.GET("/ai/oplogs", aiAPI.ListOperationLogs)
			authed.GET("/ai/tools/ask-mode", aiAPI.GetAskMode)
			authed.POST("/ai/tools/ask-mode", aiAPI.SetAskMode)
			authed.GET("/ai/conversations", aiAPI.ListConversations)
			authed.GET("/ai/conversations/:id", aiAPI.GetConversation)
			authed.POST("/ai/conversations", aiAPI.SaveConversation)
			authed.DELETE("/ai/conversations/:id", aiAPI.DeleteConversation)
			authed.GET("/ai/memories", aiAPI.ListMemories)
			authed.DELETE("/ai/memories", aiAPI.ClearMemories)
			authed.GET("/ai/skills", aiAPI.ListSkills)
			authed.POST("/ai/skills", aiAPI.SaveSkill)
			authed.POST("/ai/skills/upload", aiAPI.UploadSkillZip)
			authed.POST("/ai/skills/:name/enable", aiAPI.SetSkillEnabled)
			authed.DELETE("/ai/skills/:name", aiAPI.RemoveSkill)
			authed.GET("/ai/tools", aiAPI.ListTools)
			authed.POST("/ai/tools/flag", aiAPI.SetToolFlag)
			authed.GET("/ai/mcp/servers", aiAPI.ListMCPServers)
			authed.PUT("/ai/mcp/servers", aiAPI.SaveMCPServers)
			authed.POST("/ai/mcp/test", aiAPI.TestMCP)
			authed.DELETE("/ai/mcp/:name", aiAPI.CloseMCP)
			authed.GET("/ai/knowledge", aiAPI.ListKnowledge)
			authed.POST("/ai/knowledge", aiAPI.SaveKnowledge)
			authed.DELETE("/ai/knowledge/:id", aiAPI.DeleteKnowledge)
			authed.GET("/ai/knowledge/docs", aiAPI.ListKnowledgeDocs)
			authed.POST("/ai/knowledge/doc", aiAPI.SaveKnowledgeDoc)
			authed.GET("/ai/knowledge/doc/:id", aiAPI.GetKnowledgeDoc)
			authed.DELETE("/ai/knowledge/doc/:id", aiAPI.DeleteKnowledgeDoc)
			authed.GET("/ai/workspace", aiAPI.WorkspaceList)
			authed.POST("/ai/workspace/run", aiAPI.WorkspaceRun)
			authed.GET("/scripts", scriptAPI.List)
			authed.POST("/scripts", scriptAPI.Create)
			authed.PUT("/scripts/:id", scriptAPI.Update)
			authed.DELETE("/scripts/:id", scriptAPI.Delete)
			authed.POST("/scripts/:id/run", scriptAPI.Run)

			// M38：文件管理扩展
			authed.POST("/sites/batch", siteAPI.BatchOperate)
			authed.POST("/sites/:id/default", siteAPI.SetDefault)
			authed.PUT("/sites/:id/expire", siteAPI.SetExpire)
			authed.GET("/database/instances/:id/privileges", dbAPI.GrantMatrix)
			authed.PUT("/database/instances/:id/privileges", dbAPI.SetPrivileges)
			authed.GET("/database/instances/:id/variables", dbAPI.Variables)
			authed.PUT("/database/instances/:id/variables", dbAPI.SetVariable)
			authed.GET("/database/instances/:id/status", dbAPI.DBStatus)

			// M41：MCP 对外开放
			authed.GET("/mcp/status", mcpSrvAPI.Status)
			authed.GET("/mcp/operations", mcpSrvAPI.OpList)
			authed.PUT("/mcp/operations/:id", mcpSrvAPI.OpApprove)
			v1.Any("/mcp", mcpSrvAPI.Handler) // MCP 客户端自带 Bearer 鉴权（Bearer/​?token=）

			authed.GET("/system/swap", sysToolAPI.SwapStatus)
			authed.POST("/system/swap", sysToolAPI.SwapApply)
			authed.GET("/system/bbr", sysToolAPI.BBRStatus)
			authed.POST("/system/bbr", sysToolAPI.BBRApply)
			authed.POST("/system/clean", sysToolAPI.Clean)
			authed.GET("/security/password-policy", sysToolAPI.GetPasswordPolicy)
			authed.PUT("/security/password-policy", sysToolAPI.PutPasswordPolicy)

			authed.POST("/files/trash", fileExtAPI.Trash)
			authed.GET("/files/trash/list", fileExtAPI.TrashList)
			authed.POST("/files/trash/restore", fileExtAPI.TrashRestore)
			authed.POST("/files/trash/purge", fileExtAPI.TrashPurge)
			authed.POST("/files/trash/clear", fileExtAPI.TrashClear)
			authed.GET("/files/favorites", fileExtAPI.Favorites)
			authed.POST("/files/favorites", fileExtAPI.FavoriteAdd)
			authed.DELETE("/files/favorites/:id", fileExtAPI.FavoriteRemove)
			authed.GET("/files/shares", fileExtAPI.Shares)
			authed.POST("/files/shares", fileExtAPI.ShareCreate)
			authed.DELETE("/files/shares/:id", fileExtAPI.ShareRevoke)
			authed.POST("/files/remote-download", fileExtAPI.RemoteDownload)
			authed.GET("/cron/tasks", cronAPI.List)
			authed.POST("/cron/tasks", cronAPI.Create)
			authed.PUT("/cron/tasks/:id", cronAPI.Update)
			authed.DELETE("/cron/tasks/:id", cronAPI.Delete)
			authed.POST("/cron/tasks/:id/run", cronAPI.Run)
			authed.GET("/cron/logs", cronAPI.Logs)

			authed.GET("/database/instances", dbAPI.List)
			authed.POST("/database/instances", dbAPI.Create)
			authed.POST("/database/instances/external", dbAPI.CreateExternal)
			authed.POST("/database/instances/adopt", dbAPI.Adopt)
			authed.DELETE("/database/instances/:id", dbAPI.Delete)
			authed.POST("/database/instances/:id/start", dbAPI.StartStop(true))
			authed.POST("/database/instances/:id/stop", dbAPI.StartStop(false))
			authed.GET("/database/instances/:id/reveal", dbAPI.Reveal)
			authed.GET("/database/instances/:id/databases", dbAPI.Databases)
			authed.POST("/database/instances/:id/databases", dbAPI.CreateDatabase)
			authed.DELETE("/database/instances/:id/databases/:name", dbAPI.DropDatabase)
			authed.GET("/database/instances/:id/users", dbAPI.Users)
			authed.POST("/database/instances/:id/users", dbAPI.CreateUser)
			authed.DELETE("/database/instances/:id/users/:name", dbAPI.DropUser)
			authed.PUT("/database/instances/:id/users/:name/password", dbAPI.ChangeUserPassword)
			authed.GET("/database/instances/:id/remote", dbAPI.RemoteAccessStatus)
			authed.POST("/database/instances/:id/remote", dbAPI.RemoteAccess)
			authed.POST("/database/instances/:id/backups/import", dbAPI.BackupImport)
			authed.GET("/database/instances/:id/backups", dbAPI.Backups)
			authed.POST("/database/instances/:id/backups", dbAPI.CreateBackup)
			authed.DELETE("/database/instances/:id/backups", dbAPI.DeleteBackup)
			authed.POST("/database/instances/:id/backups/restore", dbAPI.RestoreBackup)

			authed.GET("/nginx/status", siteAPI.Status)
			authed.POST("/nginx/install", siteAPI.Install)
			authed.POST("/nginx/adopt-host", siteAPI.AdoptHost)
			authed.PUT("/nginx/mode", siteAPI.SetMode)
			authed.GET("/sites", siteAPI.List)
			authed.GET("/sites/scan", siteAPI.Scan)
			authed.GET("/sites/rewrite-templates", siteAPI.RewriteTemplates)
			authed.GET("/sites/:id/conf/domain", siteConfAPI.GetDomain)
			authed.PUT("/sites/:id/conf/domain", siteConfAPI.UpdateDomain)
			authed.GET("/sites/:id/conf/defaults", siteConfAPI.GetDefaults)
			authed.PUT("/sites/:id/conf/defaults", siteConfAPI.UpdateDefaults)
			authed.GET("/sites/:id/conf/proxy", siteConfAPI.GetProxy)
			authed.PUT("/sites/:id/conf/proxy", siteConfAPI.UpdateProxy)
			authed.GET("/sites/:id/conf/rewrite", siteConfAPI.GetRewrite)
			authed.PUT("/sites/:id/conf/rewrite", siteConfAPI.UpdateRewrite)
			authed.GET("/sites/:id/conf/https", siteConfAPI.GetHTTPS)
			authed.POST("/sites/:id/conf/https", siteConfAPI.EnableHTTPS)
			authed.PUT("/sites/:id/conf/https", siteConfAPI.UpdateHTTPS)
			authed.DELETE("/sites/:id/conf/https", siteConfAPI.DisableHTTPS)
			authed.GET("/sites/:id/conf/rundir", siteAPI.GetRunDir)
			authed.PUT("/sites/:id/conf/rundir", siteAPI.UpdateRunDir)
			authed.GET("/sites/:id/conf/antileech", siteConfAPI.GetAntiLeech)
			authed.PUT("/sites/:id/conf/antileech", siteConfAPI.UpdateAntiLeech)
			authed.GET("/sites/:id/conf/authbasic", siteConfAPI.GetAuthBasic)
			authed.PUT("/sites/:id/conf/authbasic", siteConfAPI.UpdateAuthBasic)
			authed.GET("/sites/:id/conf/cors", siteConfAPI.GetCORS)
			authed.PUT("/sites/:id/conf/cors", siteConfAPI.UpdateCORS)
			authed.GET("/sites/:id/conf/redirect", siteConfAPI.GetRedirect)
			authed.PUT("/sites/:id/conf/redirect", siteConfAPI.UpdateRedirect)
			authed.GET("/sites/:id/conf/realip", siteConfAPI.GetRealIP)
			authed.PUT("/sites/:id/conf/realip", siteConfAPI.UpdateRealIP)
			authed.GET("/sites/:id/conf/limitconn", siteConfAPI.GetLimitConn)
			authed.PUT("/sites/:id/conf/limitconn", siteConfAPI.UpdateLimitConn)
			authed.GET("/sites/:id/conf/loadbalance", siteConfAPI.GetLoadBalance)
			authed.PUT("/sites/:id/conf/loadbalance", siteConfAPI.UpdateLoadBalance)
			authed.GET("/sites/:id/conf/port", siteConfAPI.GetPort)
			authed.PUT("/sites/:id/conf/port", siteConfAPI.UpdatePort)
			authed.GET("/sites/:id/detail", siteAPI.GetSite)
			authed.GET("/sites/:id/ext", siteAPI.GetExt)
			authed.PUT("/sites/:id/ext", siteAPI.UpdateExt)
			authed.POST("/sites/adopt", siteAPI.Adopt)
			authed.POST("/sites", siteAPI.Create)
			authed.DELETE("/sites/:id", siteAPI.Delete)
			authed.POST("/sites/:id/enable", siteAPI.SetEnabled(true))
			authed.POST("/sites/:id/disable", siteAPI.SetEnabled(false))
			authed.GET("/sites/:id/config", siteAPI.Config)
			authed.GET("/sites/:id/logs", siteAPI.SiteLogs)
			authed.PUT("/sites/:id/config", siteAPI.UpdateConfig)
			authed.PUT("/sites/:id/meta", siteAPI.UpdateMeta)
			authed.GET("/sites/:id/waf", siteAPI.GetWaf)
			authed.PUT("/sites/:id/waf", siteAPI.UpdateWaf)
			authed.POST("/sites/:id/cert/acme", siteAPI.IssueACME)
			authed.POST("/sites/:id/cert/selfsigned", siteAPI.IssueSelfSigned)

			// B23 证书库 / 站点分组
			authed.GET("/certs", certAPI.List)
			authed.POST("/certs/issue", certAPI.Issue)
			authed.POST("/certs/upload", certAPI.Upload)
			authed.POST("/certs/selfsigned", certAPI.SelfSigned)
			authed.GET("/certs/dns-accounts", certAPI.ListDnsAccounts)
			authed.POST("/certs/dns-accounts", certAPI.CreateDnsAccount)
			authed.PUT("/certs/dns-accounts/:id", certAPI.UpdateDnsAccount)
			authed.DELETE("/certs/dns-accounts/:id", certAPI.DeleteDnsAccount)
			authed.GET("/certs/acme-accounts", certAPI.ListAcmeAccounts)
			authed.POST("/certs/acme-accounts", certAPI.CreateAcmeAccount)
			authed.DELETE("/certs/acme-accounts/:id", certAPI.DeleteAcmeAccount)
			authed.GET("/certs/:id", certAPI.Detail)
			authed.PUT("/certs/:id", certAPI.Update)
			authed.POST("/certs/:id/renew", certAPI.Renew)
			authed.DELETE("/certs/:id", certAPI.Remove)
			authed.GET("/site-groups", certAPI.ListGroups)
			authed.POST("/site-groups", certAPI.CreateGroup)
			authed.PUT("/site-groups/:id", certAPI.UpdateGroup)
			authed.DELETE("/site-groups/:id", certAPI.DeleteGroup)
			authed.POST("/site-groups/:id/default", certAPI.SetDefaultGroup)

			admin := authed.Group("", middleware.Admin())
			{
				authed.GET("/nodes/metrics", nodeAPI.AggregateMetrics)
				admin.GET("/nodes", nodeAPI.List)
				admin.POST("/nodes/pairing-code", nodeAPI.PairingCode)
				admin.GET("/nodes/agent-update/check", nodeAPI.CheckAgentUpdate)
				admin.POST("/nodes/:id/upgrade-agent", nodeAPI.UpgradeAgent)
				admin.DELETE("/nodes/:id", nodeAPI.Delete)
				admin.PUT("/nodes/:id/asset", nodeAPI.UpdateAsset)

				admin.POST("/auth/2fa/setup", authAPI.TwoFASetup)
				admin.POST("/auth/2fa/disable", authAPI.TwoFADisable)
				admin.GET("/security/settings", securityAPI.Get)
				admin.PUT("/security/settings", securityAPI.Update)

				authed.GET("/probes", probeAPI.List)
				authed.POST("/probes", probeAPI.Create)
				authed.POST("/probes/test", probeAPI.Test)
				authed.PUT("/probes/:id", probeAPI.Update)
				authed.DELETE("/probes/:id", probeAPI.Delete)
				authed.POST("/probes/:id/enable", probeAPI.SetEnabled(true))
				authed.POST("/probes/:id/disable", probeAPI.SetEnabled(false))

				authed.GET("/vpn/easytier/status", vpnAPI.Status)
				authed.GET("/vpn/easytier/config", vpnAPI.GetConfig)
				authed.PUT("/vpn/easytier/config", vpnAPI.SaveConfig)
				authed.POST("/vpn/easytier/install", vpnAPI.Install)
				authed.POST("/vpn/easytier/apply", vpnAPI.Apply)
				authed.GET("/vpn/easytier/peers", vpnAPI.Peers)
				authed.GET("/vpn/easytier/routes", vpnAPI.Routes)

				authed.GET("/store/sources", storeAPI.ListSources)
				authed.POST("/store/sources", storeAPI.CreateSource)
				authed.PUT("/store/sources/:id", storeAPI.UpdateSource)
				authed.POST("/store/sources/:id/enable", storeAPI.SetSourceEnabled(true))
				authed.POST("/store/sources/:id/disable", storeAPI.SetSourceEnabled(false))
				authed.DELETE("/store/sources/:id", storeAPI.DeleteSource)
				authed.POST("/store/sources/:id/sync", storeAPI.SyncSource)
				authed.POST("/store/sync", storeAPI.Sync)
				authed.GET("/store/apps", storeAPI.Apps)
				authed.GET("/store/tags", storeAPI.Tags)
				authed.GET("/store/apps/:sourceId/:key", storeAPI.Get)
				authed.GET("/store/apps/:sourceId/:key/icon", storeAPI.Icon)
				authed.GET("/store/installed", storeAPI.Installed)
				authed.POST("/store/install", storeAPI.Install)
				authed.DELETE("/store/install/:project", storeAPI.Uninstall)
				authed.POST("/store/installed/:project/:action", storeAPI.InstalledAction)
				authed.POST("/store/installed/:project/upgrade", storeAPI.UpgradeApp)
				authed.GET("/store/upgrades/check", storeAPI.CheckUpgrades)
				authed.POST("/store/local/scan", storeAPI.ScanLocalStore)
				authed.GET("/store/installed/:project/env", storeAPI.InstallEnv)
				authed.PUT("/store/installed/:project/env", storeAPI.SaveInstallEnv)

				authed.GET("/tasks", taskAPI.List)
				authed.GET("/tasks/:id", taskAPI.Get)
				authed.DELETE("/tasks/:id", taskAPI.Delete)
				authed.DELETE("/tasks", taskAPI.Clear)

				authed.GET("/firewall/status", fwAPI.Status)
				authed.POST("/firewall/allow", fwAPI.Allow)
				authed.DELETE("/firewall/rules/:number", fwAPI.DeleteRule)
				authed.POST("/firewall/enable", fwAPI.SetEnabled(true))
				authed.POST("/firewall/disable", fwAPI.SetEnabled(false))

				admin.GET("/fail2ban/status", f2bAPI.Status)
				admin.POST("/fail2ban/unban", f2bAPI.Unban)
				admin.POST("/fail2ban/ban", f2bAPI.Ban)

				// M24 NAT 端口转发（iptables DNAT）
				admin.GET("/nat/forwards", natAPI.List)
				admin.POST("/nat/forwards", natAPI.Save)
				admin.DELETE("/nat/forwards/:id", natAPI.Delete)
				admin.POST("/nat/forwards/:id/enable", natAPI.SetEnabled(true))
				admin.POST("/nat/forwards/:id/disable", natAPI.SetEnabled(false))
				admin.GET("/nat/interfaces", natAPI.Interfaces)
				admin.POST("/nat/check-port", natAPI.CheckPort)
				admin.POST("/nat/apply", natAPI.Apply)

				// M27 内网 DNS（dnsmasq 页面自管部署）
				admin.GET("/dns/overview", dnsAPI.Overview)
				admin.GET("/dns/records", dnsAPI.ListRecords)
				admin.POST("/dns/records", dnsAPI.SaveRecord)
				admin.DELETE("/dns/records/:id", dnsAPI.DeleteRecord)
				admin.POST("/dns/records/:id/enable", dnsAPI.SetRecordEnabled(true))
				admin.POST("/dns/records/:id/disable", dnsAPI.SetRecordEnabled(false))
				admin.GET("/dns/settings", dnsAPI.GetSettings)
				admin.PUT("/dns/settings", dnsAPI.SaveSettings)
				admin.POST("/dns/deploy", dnsAPI.Deploy)
				admin.POST("/dns/undeploy", dnsAPI.Undeploy)
				admin.POST("/dns/apply", dnsAPI.Apply)
				admin.POST("/dns/check", dnsAPI.CheckResolve)
				admin.GET("/dns/interfaces", dnsAPI.Interfaces)

				// M28 Hosts 可视化编辑（托管块分发）
				admin.GET("/hosts/records", hostsAPI.ListRecords)
				admin.POST("/hosts/records", hostsAPI.SaveRecord)
				admin.DELETE("/hosts/records/:id", hostsAPI.DeleteRecord)
				admin.POST("/hosts/records/:id/enable", hostsAPI.SetRecordEnabled(true))
				admin.POST("/hosts/records/:id/disable", hostsAPI.SetRecordEnabled(false))
				admin.GET("/hosts/status", hostsAPI.Status)
				admin.POST("/hosts/apply", hostsAPI.Apply)
				admin.POST("/hosts/remove", hostsAPI.Remove)

				admin.GET("/system/update/status", suAPI.Status)
				admin.POST("/system/update/apply", suAPI.Apply)
				admin.GET("/system/update/check", suAPI.CheckOnline)
				admin.POST("/system/update/upgrade", suAPI.Upgrade)

				dbAdmin := authed.Group("/plugin/db-admin")
				{
					dbAdmin.GET("/instances", dbAdminAPI.Instances)
					dbAdmin.GET("/audits", dbAdminAPI.Audits)
					dbAdmin.GET("/:id/ping", dbAdminAPI.Ping)
					dbAdmin.GET("/:id/databases", dbAdminAPI.Databases)
					dbAdmin.GET("/:id/schemas", dbAdminAPI.Schemas)
					dbAdmin.GET("/:id/tables", dbAdminAPI.Tables)
					dbAdmin.GET("/:id/columns", dbAdminAPI.Columns)
					dbAdmin.GET("/:id/ddl", dbAdminAPI.TableDDL)
					dbAdmin.GET("/:id/indexes", dbAdminAPI.Indexes)
					dbAdmin.GET("/:id/pk", dbAdminAPI.PrimaryKey)
					dbAdmin.POST("/:id/query", dbAdminAPI.Query)
					dbAdmin.POST("/:id/browse", dbAdminAPI.Browse)
					dbAdmin.POST("/:id/rows/insert", dbAdminAPI.RowInsert)
					dbAdmin.POST("/:id/rows/update", dbAdminAPI.RowUpdate)
					dbAdmin.POST("/:id/rows/delete", dbAdminAPI.RowDelete)
					dbAdmin.POST("/:id/indexes/create", dbAdminAPI.IndexCreate)
					dbAdmin.POST("/:id/indexes/drop", dbAdminAPI.IndexDrop)
					dbAdmin.GET("/:id/redis/keys", dbAdminAPI.RedisKeys)
					dbAdmin.GET("/:id/redis/key", dbAdminAPI.RedisKey)
					dbAdmin.POST("/:id/redis/key", dbAdminAPI.RedisKeyWrite)
					dbAdmin.POST("/:id/redis/key/delete", dbAdminAPI.RedisKeyDelete)
					dbAdmin.POST("/:id/redis/key/ttl", dbAdminAPI.RedisKeyTTL)
					dbAdmin.POST("/:id/redis/exec", dbAdminAPI.RedisExec)
					dbAdmin.POST("/:id/mongo/docs", dbAdminAPI.MongoDocs)
					dbAdmin.POST("/:id/mongo/aggregate", dbAdminAPI.MongoAggregate)
					dbAdmin.GET("/:id/mongo/doc", dbAdminAPI.MongoDoc)
					dbAdmin.POST("/:id/mongo/doc", dbAdminAPI.MongoDocInsert)
					dbAdmin.POST("/:id/mongo/doc/update", dbAdminAPI.MongoDocUpdate)
					dbAdmin.POST("/:id/mongo/doc/delete", dbAdminAPI.MongoDocDelete)
					dbAdmin.GET("/:id/mongo/indexes", dbAdminAPI.MongoIndexes)
					dbAdmin.POST("/:id/mongo/indexes/create", dbAdminAPI.MongoIndexCreate)
					dbAdmin.POST("/:id/mongo/indexes/drop", dbAdminAPI.MongoIndexDrop)
					dbAdmin.POST("/:id/mongo/collection", dbAdminAPI.MongoCollection)
					dbAdmin.POST("/:id/sql/import", dbAdminAPI.ImportSQL)
					dbAdmin.POST("/:id/mongo/import", dbAdminAPI.MongoImport)
					dbAdmin.GET("/migrate/preview", dbAdminAPI.MigratePreview)
					dbAdmin.POST("/migrate/start", dbAdminAPI.MigrateStart)
				}

				admin.GET("/alert/rules", alertAPI.ListRules)
				admin.POST("/alert/rules", alertAPI.CreateRule)
				admin.PUT("/alert/rules/:id", alertAPI.UpdateRule)
				admin.DELETE("/alert/rules/:id", alertAPI.DeleteRule)
				admin.GET("/alert/smtp", alertAPI.GetSMTPSettings)
				admin.PUT("/alert/smtp", alertAPI.PutSMTPSettings)
				admin.POST("/alert/smtp/test", alertAPI.TestSMTP)

				authed.GET("/notifications", notifAPI.List)
				authed.GET("/notifications/unread", notifAPI.Unread)
				authed.POST("/notifications/read", notifAPI.MarkRead)
				authed.GET("/system/history/persisted", auditAPI.History)

				authed.GET("/runtimes", rtAPI.List)
				authed.GET("/runtimes/:id", rtAPI.Detail)
				authed.POST("/runtimes", rtAPI.Create)
				authed.POST("/runtimes/external", rtAPI.AttachExternal)
				authed.DELETE("/runtimes/:id", rtAPI.Delete)
				authed.POST("/runtimes/:id/start", rtAPI.Operate("start"))
				authed.POST("/runtimes/:id/stop", rtAPI.Operate("stop"))
				authed.POST("/runtimes/:id/restart", rtAPI.Operate("restart"))
				authed.POST("/runtimes/:id/rebuild", rtAPI.Rebuild)
				authed.GET("/runtimes/:id/backups", rtAPI.BackupList)
				authed.POST("/runtimes/:id/backups", rtAPI.BackupCreate)
				authed.POST("/runtimes/:id/backups/restore", rtAPI.BackupRestore)
				authed.DELETE("/runtimes/:id/backups", rtAPI.BackupDelete)

				authed.GET("/runtimes/php/catalog", rtAPI.PHPExtensionCatalog)
				authed.GET("/runtimes/php/extensions", rtAPI.PHPExtensions)
				authed.POST("/runtimes/php/extensions/install", rtAPI.PHPExtensionInstall)
				authed.POST("/runtimes/php/extensions/uninstall", rtAPI.PHPExtensionUninstall)
				authed.GET("/runtimes/php/config", rtAPI.GetPHPConfig)
				authed.POST("/runtimes/php/config", rtAPI.UpdatePHPConfig)
				authed.GET("/runtimes/php/fpm-config", rtAPI.GetFPMConfig)
				authed.POST("/runtimes/php/fpm-config", rtAPI.UpdateFPMConfig)
				authed.GET("/runtimes/php/fpm-status", rtAPI.FPMStatus)
				authed.GET("/runtimes/php/supervisor", rtAPI.SupervisorList)
				authed.POST("/runtimes/php/supervisor", rtAPI.SupervisorUpsert)
				authed.POST("/runtimes/php/supervisor/operate", rtAPI.SupervisorOperate)
				authed.DELETE("/runtimes/php/supervisor", rtAPI.SupervisorDelete)
				authed.GET("/runtimes/php/supervisor/log", rtAPI.SupervisorLog)
				authed.GET("/runtimes/php/slow-log", rtAPI.SlowLog)
				authed.POST("/runtimes/php/slow-log/clear", rtAPI.SlowLogClear)

				authed.GET("/runtimes/node/modules", rtAPI.NodeModules)
				authed.POST("/runtimes/node/modules/operate", rtAPI.OperateNodeModule)

				authed.GET("/docker/images", dockerExtAPI.Images)
				authed.POST("/docker/images/pull", dockerExtAPI.ImagePull)
				// *id 通配：镜像引用含斜杠（registry/name:tag），单段 :id 会路由 miss 报 404
				authed.DELETE("/docker/images/*id", dockerExtAPI.ImageRemove)
				authed.POST("/docker/images/prune", dockerExtAPI.ImagesPrune)
				authed.GET("/docker/networks", dockerExtAPI.Networks)
				authed.POST("/docker/networks", dockerExtAPI.NetworkCreate)
				authed.DELETE("/docker/networks/:name", dockerExtAPI.NetworkRemove)
				authed.GET("/docker/volumes", dockerExtAPI.Volumes)
				authed.POST("/docker/volumes", dockerExtAPI.VolumeCreate)
				authed.DELETE("/docker/volumes/:name", dockerExtAPI.VolumeRemove)
				authed.POST("/docker/volumes/prune", dockerExtAPI.VolumesPrune)
				authed.GET("/docker/usage", dockerExtAPI.Usage)
				authed.POST("/docker/buildcache/prune", dockerExtAPI.BuildCachePrune)
				authed.POST("/docker/containers/prune", dockerExtAPI.ContainersPrune)
				authed.GET("/docker/containers/:id/inspect", dockerExtAPI.ContainerInspect)
				authed.GET("/docker/containers/:id/rootfs", dockerExtAPI.ContainerRootfs)
				authed.GET("/docker/containers/:id/stats", dockerExtAPI.ContainerStats)
				authed.GET("/docker/containers/:id/last-op-err", dockerAPI.ContainerLastOpErr)
				authed.GET("/docker/containers/:id/exec", dockerExtAPI.ContainerExecWS)
				authed.POST("/docker/containers", dockerExtAPI.ContainerCreate)
				authed.POST("/docker/containers/:id/recreate", dockerExtAPI.ContainerRecreate)
				authed.POST("/docker/containers/:id/update", dockerExtAPI.ContainerUpdate)
				authed.DELETE("/docker/containers/:id", dockerExtAPI.ContainerRemove)
				// 容器内文件管理（tar 归档 + 容器内 exec）
				authed.GET("/docker/containers/:id/files/list", containerFileAPI.List)
				authed.GET("/docker/containers/:id/files/read", containerFileAPI.Read)
				authed.GET("/docker/containers/:id/files/download", containerFileAPI.Download)
				authed.POST("/docker/containers/:id/files/write", containerFileAPI.Write)
				authed.POST("/docker/containers/:id/files/mkdir", containerFileAPI.Mkdir)
				authed.POST("/docker/containers/:id/files/rename", containerFileAPI.Rename)
				authed.POST("/docker/containers/:id/files/delete", containerFileAPI.Delete)
				authed.POST("/docker/containers/:id/files/chmod", containerFileAPI.Chmod)
				authed.POST("/docker/containers/:id/files/chown", containerFileAPI.Chown)
				authed.POST("/docker/containers/:id/files/upload", containerFileAPI.Upload)
				authed.GET("/docker/registry", dockerExtAPI.RegistryList)
				authed.PUT("/docker/registry", dockerExtAPI.RegistrySet)
				authed.DELETE("/docker/registry", dockerExtAPI.RegistryRemove)
				authed.GET("/docker/daemon-config", dockerExtAPI.DaemonConfig)
				authed.PUT("/docker/daemon-config", dockerExtAPI.UpdateDaemonConfig)

				// M52：Docker/Compose 一键安装 + 镜像加速器
				authed.POST("/docker/install/precheck", dockerInstallAPI.Precheck)
				admin.POST("/docker/install", dockerInstallAPI.Install)
				authed.GET("/docker/registry-mirrors", dockerInstallAPI.GetMirrors)
				authed.POST("/docker/registry-mirrors", dockerInstallAPI.SetMirrors)

				// M35：镜像生命周期 + 容器增强 + Swarm 检测
				authed.POST("/docker/images/build", dockerEnvAPI.ImageBuild)
				authed.POST("/docker/images/save", dockerEnvAPI.ImageSave)
				authed.POST("/docker/images/load", dockerEnvAPI.ImageLoad)
				authed.POST("/docker/images/tag", dockerEnvAPI.ImageTag)
				authed.POST("/docker/images/check-updates", dockerEnvAPI.ImageCheckUpdates)
				authed.POST("/docker/containers/:id/commit", dockerEnvAPI.ContainerCommit)
				authed.POST("/docker/containers/:id/backup", dockerEnvAPI.ContainerBackup)
				authed.GET("/docker/swarm-status", dockerEnvAPI.SwarmStatus)

				// M35：多 Docker 环境（CRUD 挂 admin，资源查看经 authed）
				admin.GET("/docker/environments", dockerEnvAPI.ListEnvs)
				admin.POST("/docker/environments", dockerEnvAPI.CreateEnv)
				admin.PUT("/docker/environments/:id", dockerEnvAPI.UpdateEnv)
				admin.DELETE("/docker/environments/:id", dockerEnvAPI.DeleteEnv)
				admin.POST("/docker/environments/:id/test", dockerEnvAPI.TestEnv)
				admin.GET("/docker/environments/:id/containers", dockerEnvAPI.EnvContainers)
				admin.GET("/docker/environments/:id/images", dockerEnvAPI.EnvImages)
				admin.POST("/docker/environments/:id/containers/:name/:action", dockerEnvAPI.EnvContainerAction)
				admin.DELETE("/docker/environments/:id/images/:image", dockerEnvAPI.EnvImageRemove)

				// 受管配置版本快照（M23）
				authed.GET("/config-revisions", revisionAPI.List)
				authed.GET("/config-revisions/:id", revisionAPI.Get)
				authed.POST("/config-revisions/:id/restore", revisionAPI.Restore)

				admin.POST("/nodes/exec", procExecAPI.Exec)
				authed.GET("/processes", procProxy.Processes)
				authed.POST("/processes/kill", procProxy.KillProcess)
				authed.GET("/services", procProxy.Services)
				authed.POST("/services/:name/:action", procProxy.ServiceAction)

				admin.GET("/audit/ops", auditAPI.List)
				// M47：系统级快照（对齐 1Panel）
				// M49：工具箱扩展（FTP / SSH 管理 / 暴力破解防护）
				admin.GET("/ftp/status", ftpAPI.Status)
				admin.POST("/ftp/install", ftpAPI.Install)
				admin.POST("/ftp/power", ftpAPI.Power)
				admin.POST("/ftp/port", ftpAPI.SetPort)
				admin.GET("/ssh/config", sshAPI.GetConfig)
				admin.PUT("/ssh/config", sshAPI.SetConfig)
				admin.GET("/ssh/keys", sshAPI.Keys)
				admin.POST("/ssh/keys/generate", sshAPI.GenerateKey)
				admin.POST("/ssh/keys/import", sshAPI.ImportKey)
				admin.POST("/ssh/keys/delete", sshAPI.DeleteKey)
				admin.POST("/ssh/keys/deploy", sshAPI.DeployKey)
				admin.GET("/ssh/attempts", sshAPI.FailedAttempts)

				admin.GET("/system/snapshots", sysSnapAPI.List)
				admin.POST("/system/snapshots", sysSnapAPI.Create)
				admin.DELETE("/system/snapshots", sysSnapAPI.Delete)
				admin.POST("/system/snapshots/restore", sysSnapAPI.Restore)
				admin.GET("/system/snapshots/keep", sysSnapAPI.GetKeep)
				admin.PUT("/system/snapshots/keep", sysSnapAPI.SetKeep)

				admin.GET("/snapshots", snapAPI.List)
				admin.GET("/snapshots/plan", snapAPI.Plan)
				admin.POST("/snapshots/execute", snapAPI.Execute)
				admin.POST("/snapshots/import", snapAPI.Import)
				admin.GET("/snapshots/keep", snapAPI.GetKeep)
				admin.PUT("/snapshots/keep", snapAPI.SetKeep)
				admin.POST("/snapshots/prune", snapAPI.Prune)
				admin.GET("/panel/backups", pbAPI.List)
				admin.POST("/panel/backups", pbAPI.Create)
				admin.DELETE("/panel/backups", pbAPI.Delete)
				admin.GET("/panel/backups/restore-hint", pbAPI.RestoreHint)

				// 远程备份存储 + 目录/编排备份（M34）
				admin.GET("/storage-accounts", storageAPI.List)
				admin.POST("/storage-accounts", storageAPI.Create)
				admin.PUT("/storage-accounts/:id", storageAPI.Update)
				admin.DELETE("/storage-accounts/:id", storageAPI.Delete)
				admin.POST("/storage-accounts/:id/test", storageAPI.Test)
				admin.GET("/storage-accounts/:id/objects", storageAPI.Objects)
				admin.DELETE("/storage-accounts/:id/object", storageAPI.DeleteObject)
				admin.POST("/storage-accounts/:id/fetch", storageAPI.Fetch)
				admin.POST("/backups/dir", storageAPI.DirBackup)
				admin.PUT("/mcp/status", mcpSrvAPI.SetEnabled)
				admin.POST("/backups/compose", storageAPI.ComposeBackup)

				admin.GET("/users", userAPI.List)
				admin.POST("/users", userAPI.Create)
				admin.PUT("/users/:id", userAPI.Update)
				admin.DELETE("/users/:id", userAPI.Delete)
				admin.GET("/audit/logins", userAPI.LoginLogs)
			}
		}
	}

	if err := web.Register(r); err != nil {
		return nil, err
	}
	return r, nil
}
