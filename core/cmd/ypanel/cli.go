// CLI 子命令：服务管理 / 信息查询 / 在线升级 / 卸载 / 安全救急。
// 服务器上经 /usr/local/bin/ypanel 软链使用；服务类命令依赖 systemd 且需 root。
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ypanel/core/internal/config"
	"github.com/ypanel/core/internal/db"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/release"
)

const (
	installDir     = "/opt/ypanel"
	installDataDir = "/opt/ypanel/data"
	installEnvFile = "/opt/ypanel/ypanel.env"
	panelUnit      = "ypanel"
	agentUnit      = "ypagent"
)

const cliUsageText = `YPanel CLI
用法:
  ypanel                            启动面板（默认；flags: --port --data --tls-cert --tls-key --admin-password）
  ypanel version                    查看版本
  ypanel user-info                  查看面板访问信息（地址/安全入口/端口/服务状态）
  ypanel entry                      查看安全入口
  ypanel status                     查看服务状态
  ypanel start|stop|restart         服务启停/重启（需 root；ypagent 单元存在时同步操作）
  ypanel update [--source gitee|github] [--force]
                                    在线升级（下载 + sha256 校验 + 备份替换 + 重启）
  ypanel backup                     面板备份
  ypanel clean-cache                清理 30 天前历史监控
  ypanel reset mfa                  关闭面板 2FA（验证器丢失救急）
  ypanel -reset-admin <user>        重置用户密码（交互输入，可选同时关闭 2FA）
  ypanel uninstall [--yes]          卸载（删除 /opt/ypanel，含数据）
`

// runSubcommand 处理 CLI 子命令，返回是否已处理（false = 未知命令）。
func runSubcommand(cfg *config.Config, cmd string, rest []string) bool {
	switch cmd {
	case "version":
		fmt.Println("YPanel", cfg.Version)
	case "user-info":
		cliUserInfo(cfg)
	case "entry":
		cliEntry(cfg)
	case "status":
		cliServiceStatus()
	case "start", "stop", "restart":
		cliServiceAction(cmd)
	case "update":
		cliUpdate(cfg, rest)
	case "backup":
		cliBackup(cfg)
	case "clean-cache":
		cliCleanCache(cfg)
	case "reset":
		cliReset(cfg, rest)
	case "uninstall":
		cliUninstall(rest)
	default:
		return false
	}
	return true
}

// ---- 通用助手 ----

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "[ypanel] "+msg)
	os.Exit(1)
}

func requireRoot(action string) {
	if os.Geteuid() != 0 {
		fail(action + " 需要 root 权限，请用 sudo 执行")
	}
}

func requireLinux(action string) {
	if runtime.GOOS != "linux" {
		fail(action + " 仅支持 Linux（systemd）环境")
	}
}

func argsHave(rest []string, name string) bool {
	for _, a := range rest {
		if a == name {
			return true
		}
	}
	return false
}

func argValue(rest []string, name string) string {
	for i, a := range rest {
		if a == name && i+1 < len(rest) {
			return rest[i+1]
		}
	}
	return ""
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func dbFileExists(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, "ypanel.db"))
	return err == nil
}

// readInstallEnv 读安装时落盘的 env 文件（KEY=VALUE），CLI 进程没有 systemd EnvironmentFile，用它兜底端口/数据目录。
func readInstallEnv() map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(installEnvFile)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

// resolveTarget 校正 CLI 视角的数据目录与端口：显式 flag > 安装 env 文件 > 默认值。
func resolveTarget(cfg *config.Config) (dataDir string, port int) {
	explicitData, explicitPort := false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "data":
			explicitData = true
		case "port":
			explicitPort = true
		}
	})
	env := readInstallEnv()
	dataDir = cfg.DataDir
	if !explicitData && !dirExists(dataDir) {
		switch {
		case env["YPANEL_DATA_DIR"] != "" && dirExists(env["YPANEL_DATA_DIR"]):
			dataDir = env["YPANEL_DATA_DIR"]
		case dirExists(installDataDir):
			dataDir = installDataDir
		}
	}
	port = cfg.Port
	if !explicitPort {
		if n, err := strconv.Atoi(env["YPANEL_PORT"]); err == nil && n > 0 {
			port = n
		}
	}
	return dataDir, port
}

// lanIP 取本机局域网 IP（UDP connect 只选路由不发包）。
func lanIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer func() { _ = conn.Close() }()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return "127.0.0.1"
}

// safeEntryFromDB 读安全入口；数据目录不存在（未安装）时返回空串而非建库。
func safeEntryFromDB(cfg *config.Config) string {
	dataDir, _ := resolveTarget(cfg)
	if !dbFileExists(dataDir) {
		return ""
	}
	gdb, err := db.Open(dataDir)
	if err != nil {
		return ""
	}
	return strings.Trim(service.NewSettingService(gdb).Get(service.KeySafeEntry, ""), "/ ")
}

// ---- 信息查询 ----

func cliUserInfo(cfg *config.Config) {
	dataDir, port := resolveTarget(cfg)
	env := readInstallEnv()
	scheme := "http"
	if cfg.TLSCert != "" || cfg.TLSKey != "" || env["YPANEL_TLS_CERT"] != "" {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d", scheme, lanIP(), port)
	entry := safeEntryFromDB(cfg)
	if entry != "" {
		url += "/" + entry
	}
	state, enabled := unitState(panelUnit)
	fmt.Println("YPanel", cfg.Version)
	fmt.Printf("面板端口:   %d\n", port)
	fmt.Printf("访问地址:   %s\n", url)
	if entry == "" {
		fmt.Println("安全入口:   （数据目录未找到，无法读取；确认面板已安装）")
	} else {
		fmt.Printf("安全入口:   /%s\n", entry)
	}
	fmt.Printf("数据目录:   %s\n", dataDir)
	fmt.Printf("服务状态:   %s (%s)\n", state, enabled)
	if unitExists(agentUnit) {
		aState, aEnabled := unitState(agentUnit)
		fmt.Printf("节点 agent: %s (%s)\n", aState, aEnabled)
	}
}

func cliEntry(cfg *config.Config) {
	entry := safeEntryFromDB(cfg)
	if entry == "" {
		fail("安全入口未设置或数据目录不可读（确认面板已安装）")
	}
	fmt.Println("/" + entry)
}

// ---- 服务管理（systemd 封装） ----

func systemctl(args ...string) (string, error) {
	out, err := exec.Command("systemctl", args...).Output()
	return strings.TrimSpace(string(out)), err
}

func unitExists(unit string) bool {
	return exec.Command("systemctl", "cat", unit).Run() == nil
}

func unitState(unit string) (state, enabled string) {
	state, _ = systemctl("is-active", unit)
	enabled, _ = systemctl("is-enabled", unit)
	if state == "" {
		state = "unknown"
	}
	if enabled == "" {
		enabled = "unknown"
	}
	return state, enabled
}

func cliServiceStatus() {
	requireLinux("ypanel status")
	state, enabled := unitState(panelUnit)
	fmt.Printf("面板服务 %s:    %s (%s)\n", panelUnit, state, enabled)
	if unitExists(agentUnit) {
		aState, aEnabled := unitState(agentUnit)
		fmt.Printf("节点 agent %s: %s (%s)\n", agentUnit, aState, aEnabled)
	} else {
		fmt.Printf("节点 agent %s: 未安装（本机节点为内嵌 agent，随面板服务运行）\n", agentUnit)
	}
	fmt.Println("日志: journalctl -u " + panelUnit + " -f")
}

func cliServiceAction(action string) {
	requireLinux("ypanel " + action)
	requireRoot("ypanel " + action)
	units := []string{panelUnit}
	if unitExists(agentUnit) {
		units = append(units, agentUnit)
	}
	for _, u := range units {
		if out, err := systemctl(action, u); err != nil {
			fail(fmt.Sprintf("%s %s 失败: %s %v", action, u, out, err))
		}
	}
	switch action {
	case "start":
		fmt.Println("已启动，稍候可用 ypanel status 确认就绪")
	case "stop":
		fmt.Println("已停止")
	case "restart":
		fmt.Println("已重启，服务就绪约需数秒，可用 ypanel status 确认")
	}
}

// ---- 在线升级（复用面板自更新的下载/校验/apply 链路） ----

func cliUpdate(cfg *config.Config, rest []string) {
	requireLinux("ypanel update")
	requireRoot("ypanel update")
	source := argValue(rest, "--source")
	force := argsHave(rest, "--force")
	if source == "" {
		source = "gitee"
	}
	if source != "gitee" && source != "github" {
		fail("未知更新源（gitee|github）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	fmt.Printf("检查最新版本（%s）…\n", source)
	rel := release.FetchLatestRelease(ctx, source)
	if !rel.Reachable && source == "gitee" {
		fmt.Println("Gitee 不可达，回落 GitHub…")
		source = "github"
		rel = release.FetchLatestRelease(ctx, source)
	}
	if !rel.Reachable {
		fail(source + " 不可达: " + rel.Error)
	}
	if rel.AssetURL == "" {
		fail(rel.Error)
	}
	if !force && !release.IsDevVersion(cfg.Version) && release.CompareVersions(cfg.Version, rel.Version) >= 0 {
		fmt.Println("已是最新版本", cfg.Version)
		return
	}
	if err := release.ValidateAssetURL(rel.AssetURL); err != nil {
		fail(err.Error())
	}
	fmt.Printf("最新版本 %s，开始下载并校验…\n", rel.Version)
	if err := os.MkdirAll(service.UpdateChannelDir, 0o755); err != nil {
		fail(err.Error())
	}
	dl := exec.CommandContext(ctx, "sh", "-c", release.DownloadScript(service.UpdateChannelDir, runtime.GOARCH, rel.AssetURL, rel.SumURL))
	dl.Stdout = os.Stdout
	dl.Stderr = os.Stderr
	if err := dl.Run(); err != nil {
		fail("下载/校验失败，请稍后重试或用 --source github 换源: " + err.Error())
	}
	src := filepath.Join(service.UpdateChannelDir, "ypanel")
	if _, err := os.Stat(src); err != nil {
		fail("解包产物中未找到 ypanel 二进制")
	}
	applyScript := filepath.Join(service.UpdateChannelDir, "apply-update.sh")
	if err := os.WriteFile(applyScript, []byte(release.ApplyScript(src, "/opt/ypanel/ypanel", "ypanel")), 0o700); err != nil {
		fail(err.Error())
	}
	if out, err := exec.Command("sh", "-c", release.ApplyDetachedCommand(applyScript, "/tmp/ypanel-apply.log")).CombinedOutput(); err != nil {
		fail("apply 启动失败: " + string(out))
	}
	fmt.Println("更新已启动：约 2 秒后替换二进制并重启服务")
	fmt.Println("稍后用 ypanel version 确认（当前版本 " + cfg.Version + "）")
}

// ---- 安全救急 ----

func cliReset(cfg *config.Config, rest []string) {
	if len(rest) == 0 {
		fail("用法: ypanel reset mfa")
	}
	switch rest[0] {
	case "mfa":
		dataDir, _ := resolveTarget(cfg)
		if !dbFileExists(dataDir) {
			fail("数据目录未找到（确认面板已安装或用 --data 指定）")
		}
		gdb, err := db.Open(dataDir)
		if err != nil {
			fail("打开数据库失败: " + err.Error())
		}
		settings := service.NewSettingService(gdb)
		if _, err := settings.GetOrCreate("jwt_secret", ""); err != nil {
			fmt.Fprintln(os.Stderr, "[ypanel] 读取设置失败（可忽略）")
		}
		if err := service.NewSecuritySettingsService(settings).Disable2FA(context.Background()); err != nil {
			fail("关闭 2FA 失败: " + err.Error())
		}
		fmt.Println("2FA 已关闭，请用用户名密码重新登录；如需重新绑定，到面板「安全设置」开启")
	default:
		fail("未知 reset 项: " + rest[0] + "（当前支持: mfa）")
	}
}

// ---- 备份与清理 ----

func cliBackup(cfg *config.Config) {
	dataDir, _ := resolveTarget(cfg)
	gdb, err := db.Open(dataDir)
	if err != nil {
		fail("打开数据库失败")
	}
	settings := service.NewSettingService(gdb)
	if _, err := settings.GetOrCreate("jwt_secret", ""); err != nil {
		fmt.Fprintln(os.Stderr, "[ypanel] 读取设置失败（可忽略）")
	}
	res, err := service.NewPanelBackupService(nil).Create(context.Background())
	if err != nil {
		fail("备份失败: " + err.Error())
	}
	fmt.Printf("备份完成: %v\n", res["file"])
}

func cliCleanCache(cfg *config.Config) {
	dataDir, _ := resolveTarget(cfg)
	gdb, err := db.Open(dataDir)
	if err != nil {
		fail("打开数据库失败")
	}
	if err := gdb.Exec("DELETE FROM metric_records WHERE at < datetime('now', '-30 days')").Error; err != nil {
		fail("清理失败: " + err.Error())
	}
	fmt.Println("历史监控缓存已清理（保留 30 天）")
}

// ---- 卸载 ----

func cliUninstall(rest []string) {
	requireLinux("ypanel uninstall")
	requireRoot("ypanel uninstall")
	fmt.Println("即将卸载 YPanel：")
	fmt.Println("  - 停止并移除 systemd 服务 " + panelUnit + "（ypagent 单元存在时一并移除）")
	fmt.Println("  - 删除安装目录 " + installDir + "（含 SQLite 数据、配置、更新通道）")
	fmt.Println("如需保留数据，请先执行 ypanel backup 并把备份文件移出数据目录")
	if !argsHave(rest, "--yes") {
		fmt.Print("数据删除后不可恢复，确认卸载？输入 yes 确认: ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(line) != "yes" {
			fmt.Println("已取消")
			return
		}
	}
	units := []string{panelUnit}
	if unitExists(agentUnit) {
		units = append(units, agentUnit)
	}
	for _, u := range units {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
		_ = os.Remove("/etc/systemd/system/" + u + ".service")
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	if err := os.RemoveAll(installDir); err != nil {
		fail("删除 " + installDir + " 失败: " + err.Error())
	}
	_ = os.Remove("/usr/local/bin/ypanel")
	fmt.Println("YPanel 已卸载")
}
