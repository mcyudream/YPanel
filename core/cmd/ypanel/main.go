// YPanel core 入口：单机合并部署（core + 内嵌 agent）。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/config"
	"github.com/ypanel/core/internal/db"
	"github.com/ypanel/core/internal/router"
	"github.com/ypanel/core/internal/service"
)

// version 由构建注入：-ldflags "-X main.version=xxx"
var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load(version)

	switch {
	case cfg.ResetAdmin != "":
		resetAdmin(cfg)
		return
	case flag.NArg() > 0:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n可用: ypanel [--port] [--data] | ypanel -reset-admin <user>\n", flag.Arg(0))
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("YPanel 启动失败", "err", err)
		os.Exit(1)
	}
	slog.Info("YPanel 已停止")
}

func run(ctx context.Context, cfg *config.Config) error {
	gdb, err := db.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	settings := service.NewSettingService(gdb)
	users := service.NewUserService(gdb)
	if _, err := users.EnsureBootstrap(cfg.DataDir, cfg.AdminPassword); err != nil {
		return fmt.Errorf("初始化管理员失败: %w", err)
	}
	auth, err := service.NewAuth(gdb, settings)
	if err != nil {
		return err
	}

	// 本机节点：进程内嵌 agent（loopback）
	nodes, err := service.NewNodeService(ctx, gdb)
	if err != nil {
		return fmt.Errorf("启动内嵌 agent 失败: %w", err)
	}

	// 计划任务调度器
	cronSvc := service.NewCron(gdb, nodes)
	if err := cronSvc.Start(); err != nil {
		return fmt.Errorf("启动计划任务调度失败: %w", err)
	}
	defer cronSvc.Stop()

	// 数据库实例管理（凭据加密密钥由 JWT 密钥派生）
	dbSvc := service.NewDatabaseService(gdb, nodes, string(auth.Secret()))
	siteSvc := service.NewSiteService(gdb, nodes)
	marketSvc := service.NewMarketService(gdb, nodes)
	fwSvc := service.NewFirewallService(nodes, cfg.Port)
	f2bSvc := service.NewFail2banService(nodes)
	dbAdminSvc := service.NewDBAdminService(gdb, dbSvc)
	notifSvc := service.NewNotificationService(gdb)
	alertSvc := service.NewAlertService(gdb, nodes, notifSvc)
	alertSvc.Start(ctx)
	histSvc := service.NewHistoryRecorder(gdb, nodes)
	histSvc.Start(ctx)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r, err := router.Setup(&router.Deps{
		Auth: auth, Nodes: nodes, Settings: settings, Cron: cronSvc, DBS: dbSvc, Sites: siteSvc,
		Market: marketSvc, FW: fwSvc, Alerts: alertSvc,
		Notif: notifSvc, PanelBP: service.NewPanelBackupService(nodes), Hist: histSvc,
		F2B: f2bSvc, DBAdmin: dbAdminSvc, Version: version,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{Addr: cfg.Addr(), Handler: r}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	slog.Info("YPanel 启动完成", "version", version, "addr", cfg.Addr(), "data", cfg.DataDir)
	slog.Info("默认入口", "url", fmt.Sprintf("http://127.0.0.1:%d", cfg.Port))
	err = srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func resetAdmin(cfg *config.Config) {
	gdb, err := db.Open(cfg.DataDir)
	if err != nil {
		slog.Error("打开数据库失败", "err", err)
		os.Exit(1)
	}
	settings := service.NewSettingService(gdb)
	if _, err := settings.GetOrCreate("jwt_secret", ""); err != nil {
		slog.Warn("读取设置失败（可忽略）", "err", err)
	}
	fmt.Print("请输入新密码: ")
	var pwd string
	if _, err := fmt.Scanln(&pwd); err != nil {
		slog.Error("读取密码失败", "err", err)
		os.Exit(1)
	}
	if err := service.NewUserService(gdb).ResetPassword(cfg.ResetAdmin, pwd); err != nil {
		slog.Error("重置失败", "err", err)
		os.Exit(1)
	}
	slog.Info("密码已重置", "user", cfg.ResetAdmin)
}
