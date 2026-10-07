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
	Auth     *service.Auth
	Nodes    *service.NodeService
	Settings *service.SettingService
	Cron     *service.Cron
	Scripts  *service.ScriptService
	DBSvc    *service.DatabaseService
	AI       *service.AIService
	Acme     *service.AcmeService
	DBS      *service.DatabaseService
	Sites    *service.SiteService
	Certs    *service.CertificateService
	Groups   *service.SiteGroupService
	Store    *service.StoreService
	Tasks    *service.TaskService
	FW       *service.FirewallService
	NatF     *service.NatForwardService
	Alerts   *service.AlertService
	Notif    *service.NotificationService
	PanelBP  *service.PanelBackupService
	Hist     *service.HistoryRecorder
	F2B      *service.Fail2banService
	DBAdmin  *service.DBAdminService
	SU       *service.SelfUpdateService
	RT       *service.RuntimeService
	DockerExt *service.DockerExtService
	Rev       *service.RevisionService
	Sec      *service.SecuritySettingsService
	Version  string
}

// CronDB 计划任务审计库句柄。
func (d *Deps) CronDB() *gorm.DB { return d.Auth.DB() }

// Setup 装配全部路由。
func Setup(d *Deps) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.AccessLog(), middleware.CORS(), middleware.SecurityGate(d.Sec))

	authAPI := &api.AuthAPI{Auth: d.Auth, Sec: d.Sec, Version: d.Version}
	securityAPI := &api.SecurityAPI{Sec: d.Sec, Auth: d.Auth}
	userAPI := &api.UserAPI{DB: d.Auth.DB()}
	sysAPI := &api.SystemAPI{Nodes: d.Nodes, Notif: d.Notif}
	sysManageAPI := &api.SysManageAPI{Nodes: d.Nodes}
	fileAPI := &api.FileAPI{Nodes: d.Nodes, Rev: d.Rev}
	dockerAPI := &api.DockerAPI{Nodes: d.Nodes}
	termAPI := &api.TerminalAPI{Nodes: d.Nodes}
	setAPI := &api.SettingsAPI{Settings: d.Settings}
	composeAPI := &api.ComposeAPI{Nodes: d.Nodes}
	cronAPI := &api.CronAPI{DB: d.CronDB(), Cron: d.Cron}
	scriptAPI := &api.ScriptAPI{Scripts: d.Scripts}
	aiAPI := &api.AIAPI{AI: d.AI, Nodes: d.Nodes}
	dbAPI := &api.DatabaseAPI{DBS: d.DBS}
	siteAPI := &api.SiteAPI{Sites: d.Sites, Acme: d.Acme}
	siteConfAPI := &api.SiteConfAPI{Sites: d.Sites}
	certAPI := &api.CertAPI{Certs: d.Certs, Groups: d.Groups}
	nodeAPI := &api.NodeAPI{Nodes: d.Nodes}
	storeAPI := &api.StoreAPI{Store: d.Store}
	taskAPI := &api.TaskAPI{Tasks: d.Tasks}
	fwAPI := &api.FirewallAPI{FW: d.FW}
	natAPI := &api.NatForwardAPI{NF: d.NatF}
	alertAPI := &api.AlertAPI{Alerts: d.Alerts}
	f2bAPI := &api.Fail2banAPI{F2B: d.F2B}
	rtAPI := &api.RuntimeAPI{RT: d.RT}
	dbAdminAPI := &api.DBAdminAPI{Admin: d.DBAdmin}
	suAPI := &api.SelfUpdateAPI{SU: d.SU}
	notifAPI := &api.NotificationAPI{Notif: d.Notif, ParseToken: func(t string) error { _, err := d.Auth.ParseToken(t); return err }}
	procExecAPI := &api.NodeExecAPI{Nodes: d.Nodes}
	procProxy := &api.ProcProxy{Nodes: d.Nodes}
	dockerExtAPI := &api.DockerExtAPI{Ext: d.DockerExt}
	containerFileAPI := &api.ContainerFileAPI{Nodes: d.Nodes}
	revisionAPI := &api.RevisionAPI{Rev: d.Rev}
	pbAPI := &api.PanelBackupAPI{BP: d.PanelBP}
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
	v1.POST("/pair/heartbeat", nodeAPI.Heartbeat)

		authed := v1.Group("", middleware.Auth(d.Auth), middleware.Audit(d.CronDB()))
		{
			authed.GET("/auth/me", authAPI.Me)
			authed.POST("/auth/logout", authAPI.Logout)
			authed.PUT("/auth/password", authAPI.ChangePassword)

			authed.GET("/system/overview", sysAPI.Overview)
			authed.GET("/system/history", sysAPI.History)
			authed.POST("/system/manage", sysManageAPI.Manage)

			authed.GET("/files/list", fileAPI.List)
			authed.GET("/files/read", fileAPI.Read)
			authed.GET("/files/download", fileAPI.Download) // 下载走 ?token=
			authed.GET("/files/upload", fileAPI.Upload)
			authed.POST("/files/upload", fileAPI.Upload)
			authed.POST("/files/write", fileAPI.Write)
			authed.POST("/files/mkdir", fileAPI.Mkdir)
			authed.POST("/files/rename", fileAPI.Rename)
			authed.POST("/files/delete", fileAPI.Delete)
			authed.POST("/files/chmod", fileAPI.Chmod)
			authed.POST("/files/compress", fileAPI.Compress)
			authed.POST("/files/decompress", fileAPI.Decompress)
			authed.GET("/files/search", fileAPI.Search)

			authed.GET("/docker/containers", dockerAPI.List)
			authed.GET("/docker/containers/:id/logs", dockerAPI.Logs)
			authed.POST("/docker/containers/:id/:action", dockerAPI.Action)

			authed.GET("/terminal", termAPI.Handle) // WS 走 ?token=

			authed.GET("/settings", setAPI.Get)
			authed.PUT("/settings", setAPI.Put)

			authed.GET("/compose/scan", composeAPI.ScanCompose)
			authed.POST("/compose/adopt", composeAPI.AdoptCompose)
			authed.GET("/compose/projects", composeAPI.List)
			authed.GET("/compose/config", composeAPI.Config)
			authed.POST("/compose/config", composeAPI.Write)
			authed.POST("/compose/up", composeAPI.Action("up"))
			authed.POST("/compose/down", composeAPI.Action("down"))
			authed.POST("/compose/service-action", composeAPI.ServiceAction)
			authed.DELETE("/compose/projects/:name", composeAPI.ProjectDelete)
			authed.GET("/compose/logs", composeAPI.Logs)

			authed.GET("/ai/providers", aiAPI.ListProviders)
			authed.GET("/ai/presets", aiAPI.Presets)
			authed.POST("/ai/providers", aiAPI.SaveProvider)
			authed.DELETE("/ai/providers/:id", aiAPI.DeleteProvider)
			authed.POST("/ai/chat", aiAPI.Chat)
			authed.GET("/ai/conversations", aiAPI.ListConversations)
			authed.GET("/ai/conversations/:id", aiAPI.GetConversation)
			authed.POST("/ai/conversations", aiAPI.SaveConversation)
			authed.DELETE("/ai/conversations/:id", aiAPI.DeleteConversation)
			authed.GET("/ai/memories", aiAPI.ListMemories)
			authed.DELETE("/ai/memories", aiAPI.ClearMemories)
			authed.GET("/ai/skills", aiAPI.ListSkills)
			authed.POST("/ai/skills", aiAPI.SaveSkill)
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
			authed.GET("/ai/workspace", aiAPI.WorkspaceList)
			authed.POST("/ai/workspace/run", aiAPI.WorkspaceRun)
			authed.GET("/scripts", scriptAPI.List)
			authed.POST("/scripts", scriptAPI.Create)
			authed.PUT("/scripts/:id", scriptAPI.Update)
			authed.DELETE("/scripts/:id", scriptAPI.Delete)
			authed.GET("/cron/tasks", cronAPI.List)
			authed.POST("/cron/tasks", cronAPI.Create)
			authed.PUT("/cron/tasks/:id", cronAPI.Update)
			authed.DELETE("/cron/tasks/:id", cronAPI.Delete)
			authed.POST("/cron/tasks/:id/run", cronAPI.Run)
			authed.GET("/cron/logs", cronAPI.Logs)

			authed.GET("/database/instances", dbAPI.List)
			authed.POST("/database/instances", dbAPI.Create)
			authed.POST("/database/instances/external", dbAPI.CreateExternal)
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
			authed.POST("/database/instances/:id/remote", dbAPI.RemoteAccess)
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
				admin.DELETE("/nodes/:id", nodeAPI.Delete)

			admin.POST("/auth/2fa/setup", authAPI.TwoFASetup)
			admin.POST("/auth/2fa/disable", authAPI.TwoFADisable)
			admin.GET("/security/settings", securityAPI.Get)
			admin.PUT("/security/settings", securityAPI.Update)

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

			admin.GET("/system/update/status", suAPI.Status)
			admin.POST("/system/update/apply", suAPI.Apply)

			dbAdmin := authed.Group("/plugin/db-admin")
			{
				dbAdmin.GET("/instances", dbAdminAPI.Instances)
				dbAdmin.GET("/:id/databases", dbAdminAPI.Databases)
				dbAdmin.GET("/:id/tables", dbAdminAPI.Tables)
				dbAdmin.POST("/:id/query", dbAdminAPI.Query)
			}

			admin.GET("/alert/rules", alertAPI.ListRules)
			admin.POST("/alert/rules", alertAPI.CreateRule)
			admin.PUT("/alert/rules/:id", alertAPI.UpdateRule)
			admin.DELETE("/alert/rules/:id", alertAPI.DeleteRule)

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
			authed.DELETE("/docker/images/:id", dockerExtAPI.ImageRemove)
			authed.POST("/docker/images/prune", dockerExtAPI.ImagesPrune)
			authed.GET("/docker/networks", dockerExtAPI.Networks)
			authed.POST("/docker/networks", dockerExtAPI.NetworkCreate)
			authed.DELETE("/docker/networks/:name", dockerExtAPI.NetworkRemove)
			authed.GET("/docker/volumes", dockerExtAPI.Volumes)
			authed.POST("/docker/volumes", dockerExtAPI.VolumeCreate)
			authed.DELETE("/docker/volumes/:name", dockerExtAPI.VolumeRemove)
			authed.POST("/docker/volumes/prune", dockerExtAPI.VolumesPrune)
			authed.POST("/docker/containers/prune", dockerExtAPI.ContainersPrune)
			authed.GET("/docker/containers/:id/inspect", dockerExtAPI.ContainerInspect)
			authed.GET("/docker/containers/:id/stats", dockerExtAPI.ContainerStats)
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
			authed.POST("/docker/containers/:id/files/upload", containerFileAPI.Upload)
			authed.GET("/docker/registry", dockerExtAPI.RegistryList)
			authed.PUT("/docker/registry", dockerExtAPI.RegistrySet)
			authed.DELETE("/docker/registry", dockerExtAPI.RegistryRemove)
			authed.GET("/docker/daemon-config", dockerExtAPI.DaemonConfig)
			authed.PUT("/docker/daemon-config", dockerExtAPI.UpdateDaemonConfig)

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
			admin.GET("/panel/backups", pbAPI.List)
			admin.POST("/panel/backups", pbAPI.Create)
			admin.DELETE("/panel/backups", pbAPI.Delete)
			admin.GET("/panel/backups/restore-hint", pbAPI.RestoreHint)
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
