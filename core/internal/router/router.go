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
)

// Deps 路由依赖。
type Deps struct {
	Auth     *service.Auth
	Nodes    *service.NodeService
	Settings *service.SettingService
	Cron     *service.Cron
	DBS      *service.DatabaseService
	Sites    *service.SiteService
	Market   *service.MarketService
	FW       *service.FirewallService
	Alerts   *service.AlertService
	Notif    *service.NotificationService
	PanelBP  *service.PanelBackupService
	Hist     *service.HistoryRecorder
	F2B      *service.Fail2banService
	DBAdmin  *service.DBAdminService
	SU       *service.SelfUpdateService
	MarketStore *service.MarketStoreService
	RT       *service.RuntimeService
	Version  string
}

// CronDB 计划任务审计库句柄。
func (d *Deps) CronDB() *gorm.DB { return d.Auth.DB() }

// Setup 装配全部路由。
func Setup(d *Deps) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.AccessLog(), middleware.CORS())

	authAPI := &api.AuthAPI{Auth: d.Auth, Version: d.Version}
	userAPI := &api.UserAPI{DB: d.Auth.DB()}
	sysAPI := &api.SystemAPI{Nodes: d.Nodes}
	fileAPI := &api.FileAPI{Nodes: d.Nodes}
	dockerAPI := &api.DockerAPI{Nodes: d.Nodes}
	termAPI := &api.TerminalAPI{Nodes: d.Nodes}
	setAPI := &api.SettingsAPI{Settings: d.Settings}
	composeAPI := &api.ComposeAPI{Nodes: d.Nodes}
	cronAPI := &api.CronAPI{DB: d.CronDB(), Cron: d.Cron}
	dbAPI := &api.DatabaseAPI{DBS: d.DBS}
	siteAPI := &api.SiteAPI{Sites: d.Sites}
	nodeAPI := &api.NodeAPI{Nodes: d.Nodes}
	marketAPI := &api.MarketAPI{Market: d.Market}
	fwAPI := &api.FirewallAPI{FW: d.FW}
	alertAPI := &api.AlertAPI{Alerts: d.Alerts}
	f2bAPI := &api.Fail2banAPI{F2B: d.F2B}
	rtAPI := &api.RuntimeAPI{RT: d.RT}
	dbAdminAPI := &api.DBAdminAPI{Admin: d.DBAdmin}
	suAPI := &api.SelfUpdateAPI{SU: d.SU}
	storeAPI := &api.MarketStoreAPI{Store: d.MarketStore}
	notifAPI := &api.NotificationAPI{Notif: d.Notif}
	procExecAPI := &api.NodeExecAPI{Nodes: d.Nodes}
	procProxy := &api.ProcProxy{Nodes: d.Nodes}
	pbAPI := &api.PanelBackupAPI{BP: d.PanelBP}
	auditAPI := &api.AuditAPI{DB: d.CronDB(), Hist: d.Hist}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": d.Version})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authAPI.Login)
	v1.POST("/pair", nodeAPI.Pair)
	v1.POST("/pair/heartbeat", nodeAPI.Heartbeat)

		authed := v1.Group("", middleware.Auth(d.Auth), middleware.Audit(d.CronDB()))
		{
			authed.GET("/auth/me", authAPI.Me)
			authed.POST("/auth/logout", authAPI.Logout)
			authed.PUT("/auth/password", authAPI.ChangePassword)

			authed.GET("/system/overview", sysAPI.Overview)
			authed.GET("/system/history", sysAPI.History)

			authed.GET("/files/list", fileAPI.List)
			authed.GET("/files/read", fileAPI.Read)
			authed.GET("/files/download", fileAPI.Download) // 下载走 ?token=
			authed.GET("/files/upload", fileAPI.Upload)
			authed.POST("/files/upload", fileAPI.Upload)
			authed.POST("/files/write", fileAPI.Write)
			authed.POST("/files/mkdir", fileAPI.Mkdir)
			authed.POST("/files/rename", fileAPI.Rename)
			authed.POST("/files/delete", fileAPI.Delete)

			authed.GET("/docker/containers", dockerAPI.List)
			authed.GET("/docker/containers/:id/logs", dockerAPI.Logs)
			authed.POST("/docker/containers/:id/:action", dockerAPI.Action)

			authed.GET("/terminal", termAPI.Handle) // WS 走 ?token=

			authed.GET("/settings", setAPI.Get)
			authed.PUT("/settings", setAPI.Put)

			authed.GET("/compose/projects", composeAPI.List)
			authed.GET("/compose/config", composeAPI.Config)
			authed.POST("/compose/config", composeAPI.Write)
			authed.POST("/compose/up", composeAPI.Action("up"))
			authed.POST("/compose/down", composeAPI.Action("down"))
			authed.GET("/compose/logs", composeAPI.Logs)

			authed.GET("/cron/tasks", cronAPI.List)
			authed.POST("/cron/tasks", cronAPI.Create)
			authed.PUT("/cron/tasks/:id", cronAPI.Update)
			authed.DELETE("/cron/tasks/:id", cronAPI.Delete)
			authed.POST("/cron/tasks/:id/run", cronAPI.Run)
			authed.GET("/cron/logs", cronAPI.Logs)

			authed.GET("/database/instances", dbAPI.List)
			authed.POST("/database/instances", dbAPI.Create)
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
			authed.GET("/database/instances/:id/backups", dbAPI.Backups)
			authed.POST("/database/instances/:id/backups", dbAPI.CreateBackup)
			authed.DELETE("/database/instances/:id/backups", dbAPI.DeleteBackup)
			authed.POST("/database/instances/:id/backups/restore", dbAPI.RestoreBackup)

			authed.GET("/nginx/status", siteAPI.Status)
			authed.POST("/nginx/install", siteAPI.Install)
			authed.GET("/sites", siteAPI.List)
			authed.GET("/sites/scan", siteAPI.Scan)
			authed.GET("/sites/rewrite-templates", siteAPI.RewriteTemplates)
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
			authed.GET("/sites/:id/waf", siteAPI.GetWaf)
			authed.PUT("/sites/:id/waf", siteAPI.UpdateWaf)
			authed.POST("/sites/:id/cert/selfsigned", siteAPI.IssueSelfSigned)

			admin := authed.Group("", middleware.Admin())
			{
				admin.GET("/nodes", nodeAPI.List)
				admin.POST("/nodes/pairing-code", nodeAPI.PairingCode)
				admin.DELETE("/nodes/:id", nodeAPI.Delete)

			authed.GET("/store/apps", storeAPI.List)
			authed.GET("/store/apps/:key", storeAPI.Get)
			authed.POST("/store/sync", storeAPI.Sync)
			authed.GET("/store/installed", storeAPI.Installed)
			authed.POST("/store/install", storeAPI.Install)
			authed.DELETE("/store/install/:project", storeAPI.Uninstall)

			admin.GET("/market/apps", marketAPI.List)
			admin.GET("/market/installed", marketAPI.Installed)
			admin.POST("/market/install", marketAPI.Install)

			authed.GET("/firewall/status", fwAPI.Status)
			authed.POST("/firewall/allow", fwAPI.Allow)
			authed.DELETE("/firewall/rules/:number", fwAPI.DeleteRule)
			authed.POST("/firewall/enable", fwAPI.SetEnabled(true))
			authed.POST("/firewall/disable", fwAPI.SetEnabled(false))

			admin.GET("/fail2ban/status", f2bAPI.Status)
			admin.POST("/fail2ban/unban", f2bAPI.Unban)
			admin.POST("/fail2ban/ban", f2bAPI.Ban)

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
			authed.POST("/runtimes", rtAPI.Create)
			authed.DELETE("/runtimes/:id", rtAPI.Delete)
			authed.POST("/runtimes/:id/start", rtAPI.SetEnabled(true))
			authed.POST("/runtimes/:id/stop", rtAPI.SetEnabled(false))

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
