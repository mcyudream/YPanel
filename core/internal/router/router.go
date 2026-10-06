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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": d.Version})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authAPI.Login)

		authed := v1.Group("", middleware.Auth(d.Auth))
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

			admin := authed.Group("", middleware.Admin())
			{
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
