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
	"github.com/ypanel/core/internal/gwserver"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/router"
	"github.com/ypanel/core/internal/service"
)

// version 由构建注入：-ldflags "-X main.version=xxx"
var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load(version)

	if cfg.ResetAdmin != "" {
		resetAdmin(cfg)
		return
	}
	if flag.NArg() > 0 {
		if !runSubcommand(cfg, flag.Arg(0), flag.Args()[1:]) {
			fmt.Fprintf(os.Stderr, "未知命令: %s\n\n%s", flag.Arg(0), cliUsageText)
			os.Exit(2)
		}
		return
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
	rbacSvc := rbac.New(gdb)

	// 本机节点：进程内嵌 agent（loopback）
	nodes, err := service.NewNodeService(ctx, gdb)
	if err != nil {
		return fmt.Errorf("启动内嵌 agent 失败: %w", err)
	}

	// 任务中心（迁移等长跑任务）
	taskSvc := service.NewTaskService(gdb)

	// 数据库实例管理（凭据加密密钥由 JWT 密钥派生）
	dbSvc := service.NewDatabaseService(gdb, nodes, taskSvc, string(auth.Secret()))
	// 商店已装的数据库应用自动接管（启动即扫 + 30s 周期，接管失败自动重试）
	dbSvc.StartAutoAdopt(ctx)
	siteSvc := service.NewSiteService(gdb, nodes)
	certSvc := service.NewCertificateService(gdb, nodes, siteSvc, string(auth.Secret()))
	certSvc.SetEnvGetter(os.Getenv)
	groupSvc := service.NewSiteGroupService(gdb)
	acmeSvc := service.NewAcmeService(siteSvc, certSvc)

	// 计划任务调度器（B4：依赖数据库备份/站点备份服务）
	cronSvc := service.NewCron(gdb, nodes)
	// 远程备份存储（M34）：账号管理 + 备份产物上传/拉回
	storageSvc := service.NewStorageService(gdb, nodes, string(auth.Secret()))
	backupSvc := service.NewBackupService(nodes, storageSvc)
	dockerEnvSvc := service.NewDockerEnvService(gdb, nodes, string(auth.Secret()))
	dockerImgSvc := service.NewDockerImgService(nodes)
	fileExtSvc := service.NewFileExtService(gdb, nodes)
	sysToolSvc := service.NewSystemToolService(nodes)
	panelBkSvc := service.NewPanelBackupService(nodes)
	panelBkSvc.SetStorage(storageSvc)
	cronSvc.DBSvc = dbSvc
	cronSvc.SiteBk = service.NewSiteBackupService(nodes)
	cronSvc.SiteBk.SetStorage(storageSvc)
	cronSvc.BackupSrv = backupSvc
	cronSvc.Certs = certSvc
	dbSvc.SetStorage(storageSvc)
	scriptSvc := service.NewScriptService(gdb, nodes)
	if err := cronSvc.Start(); err != nil {
		return fmt.Errorf("启动计划任务调度失败: %w", err)
	}
	defer cronSvc.Stop()
	storeSvc := service.NewStoreService(gdb, nodes, siteSvc, taskSvc, dbSvc, cfg.DataDir)
	vpnSvc := service.NewVPNService(gdb, nodes, settings, taskSvc, storeSvc)
	fwSvc := service.NewFirewallService(gdb, nodes, cfg.Port)
	f2bSvc := service.NewFail2banService(nodes)
	dbAdminSvc := service.NewDBAdminService(gdb, dbSvc)
	rtSvc := service.NewRuntimeService(gdb, nodes, taskSvc)
	dockerExtSvc := service.NewDockerExtService(nodes, taskSvc)
	dockerInstallSvc := service.NewDockerInstallService(nodes, taskSvc)
	suSvc := service.NewSelfUpdateService(nodes, taskSvc, version)
	notifSvc := service.NewNotificationService(gdb)
	cronSvc.Notif = notifSvc
	dashboardSvc := service.NewDashboardService(gdb, nodes, dockerExtSvc, notifSvc)
	secSvc := service.NewSecuritySettingsService(settings)
	// M49：安全入口强制开启——存量空入口自动生成随机 8 位
	if entry, generated, err := secSvc.EnsureSafeEntry(); err == nil && generated {
		slog.Info("安全入口已自动生成（强制策略）", "entry", "/"+entry)
	}
	alertSvc := service.NewAlertService(gdb, nodes, notifSvc, settings)
	alertSvc.Start(ctx)
	probeSvc := service.NewProbeService(gdb, nodes, notifSvc)
	probeSvc.Start(ctx)
	histSvc := service.NewHistoryRecorder(gdb, nodes)
	histSvc.Start(ctx)
	// M55 磁盘空间保护：低水位自动停容器强制清理（配置走 SettingService，事件落库）
	diskGuardSvc := service.NewDiskGuardService(gdb, nodes, settings, notifSvc)
	diskGuardSvc.Start(ctx)
	revSvc := service.NewRevisionService(gdb, nodes)
	natSvc := service.NewNatForwardService(gdb, nodes)
	siteSvc.SetPortDeps(fwSvc, natSvc) // M50 站点端口对账：防火墙放行 + 端口占用检测
	dnsSvc := service.NewDnsService(gdb, nodes, settings)
	hostsSvc := service.NewHostsService(gdb, nodes)
	credSvc := service.NewGitCredService(gdb)
	src2Svc := &service.Src2ComposeService{Nodes: nodes, Tasks: taskSvc, Creds: credSvc}
	logCentralSvc := service.NewLogCentralService(gdb, settings, nodes)
	alertSvc.SetLogCentral(logCentralSvc) // P3 日志量告警：VL 聚合计数（metric=log 规则）
	// M51：桌面工作台内网浏览器——会话式反代网关（第二端口）
	webgwSvc := service.NewWebGwService(ctx, settings)
	webgwSvc.SetSelfEntry(cfg.Port, secSvc.SafeEntry) // 经网关浏览面板自身时补安全入口
	webgwSvc.SetNodes(nodes)
	// M31：AI 工具治理升级——全量服务依赖装配（原 126 行 aiSvc 创建移至此处，确保全部依赖就绪）
	aiSvc := service.NewAIService(gdb, service.AIDeps{
		Nodes: nodes, DBSvc: dbSvc, DBAdmin: dbAdminSvc, Settings: settings,
		Skills: service.NewSkillsManager(nodes, settings),
		Sites:  siteSvc, Certs: certSvc, Runtimes: rtSvc, DockerX: dockerExtSvc,
		FW: fwSvc, NAT: natSvc, Hosts: hostsSvc, DNS: dnsSvc, Cron: cronSvc,
		Store: storeSvc, F2B: f2bSvc, Src2: src2Svc,
	})
	// M41：MCP 对外开放（复用 AI 注册表 read 工具）
	mcpSvc := service.NewMCPService(gdb, aiSvc, settings)
	snapSvc := service.NewSnapshotService(gdb, nodes, panelBkSvc, settings)
	sysSnapSvc := service.NewSystemSnapshotService(nodes, taskSvc, dbSvc, gdb, panelBkSvc)
	ftpSvc := service.NewFtpService(nodes)
	sshGSvc := service.NewSshGuardService(nodes)
	panelBkSvc.SetPruneFn(func(keep int) int { n, _ := snapSvc.Prune(context.Background(), "local", keep); return n })
	mcpOpSvc := service.NewMCPOperationService(gdb)
	mcpSvc.RegisterWriteDeps(nodes, panelBkSvc, mcpOpSvc)
	// M24 NAT 转发启动重放：失败仅告警不阻断启动（agent 不可达 / iptables 缺失属预期场景）
	go func() {
		fails, err := natSvc.ApplyAll(ctx)
		if err != nil {
			slog.Warn("NAT 转发启动重放：规则清单读取失败", "err", err.Error())
			return
		}
		for nodeId, applyErr := range fails {
			slog.Warn("NAT 转发启动重放失败", "node", nodeId, "err", applyErr.Error())
		}
	}()
	// M50 站点端口启动重放：compose 映射 / 防火墙放行与站点表对账，自愈漂移（失败仅告警）
	go func() {
		if err := siteSvc.ReconcilePorts(ctx); err != nil {
			slog.Warn("站点端口对账失败", "err", err.Error())
		}
	}()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r, err := router.Setup(&router.Deps{
		Auth: auth, RBAC: rbacSvc, Nodes: nodes, Settings: settings, Cron: cronSvc, DBS: dbSvc, Sites: siteSvc, Certs: certSvc, Groups: groupSvc,
		Scripts: scriptSvc, DBSvc: dbSvc, Acme: acmeSvc, AI: aiSvc,
		FW: fwSvc, Alerts: alertSvc,
		Notif: notifSvc, PanelBP: panelBkSvc, Hist: histSvc,
		Storage: storageSvc, BackupSrv: backupSvc,
		DockerEnv: dockerEnvSvc, DockerImg: dockerImgSvc,
		FileExt: fileExtSvc,
		SysTool: sysToolSvc,
		MCP:     mcpSvc, OpsM: mcpOpSvc,
		Snap:    snapSvc,
		SysSnap: sysSnapSvc,
		Ftp:     ftpSvc,
		SshG:    sshGSvc,
		DiskGuard: diskGuardSvc,
		F2B:     f2bSvc, DBAdmin: dbAdminSvc, SU: suSvc, Store: storeSvc, Tasks: taskSvc, RT: rtSvc,
		Vpn: vpnSvc, Probes: probeSvc,
		DockerExt: dockerExtSvc, Sec: secSvc, Rev: revSvc, NatF: natSvc, DNS: dnsSvc, HS: hostsSvc, Version: version,
		DockerInstall: dockerInstallSvc,
		Src2: src2Svc, Creds: credSvc, LogCentral: logCentralSvc, WebGW: webgwSvc, Dashboard: dashboardSvc,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{Addr: cfg.Addr(), Handler: r}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	// M51 内网浏览器网关（第二端口；TLS 与面板共用证书）
	gwSrv := gwserver.New(webgwSvc, cfg.GwAddr(), cfg.TLSCert, cfg.TLSKey)
	go func() {
		if err := gwSrv.ListenAndServe(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("内网浏览器网关启动失败", "err", err)
		}
	}()

	slog.Info("YPanel 启动完成", "version", version, "addr", cfg.Addr(), "gw", cfg.GwAddr(), "data", cfg.DataDir)
	scheme := "http"
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		scheme = "https"
		slog.Info("面板 HTTPS 已启用", "cert", cfg.TLSCert)
	}
	slog.Info("默认入口", "url", fmt.Sprintf("%s://127.0.0.1:%d", scheme, cfg.Port))
	if scheme == "https" {
		err = srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
	} else {
		err = srv.ListenAndServe()
	}
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
	// 交互确认是否同时关闭面板级 2FA（验证器丢失时的救急通道；默认不动）
	fmt.Print("是否同时关闭面板 2FA（验证器丢失时选 y）[y/N]: ")
	var ans string
	if _, err := fmt.Scanln(&ans); err == nil && (ans == "y" || ans == "Y" || ans == "yes") {
		if err := service.NewSecuritySettingsService(settings).Disable2FA(context.Background()); err != nil {
			slog.Warn("关闭 2FA 失败（可稍后用 ypanel reset mfa）", "err", err)
		} else {
			slog.Info("面板 2FA 已关闭")
		}
	}
}
