// CLI 子命令：节点信息 / 服务管理 / 自升级 / 卸载。
// 服务器上经 /usr/local/bin/ypagent 软链使用；无参与 flags 启动行为不变（配对/续启/手工 PSK）。
package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ypanel/agent/internal/pair"
	"github.com/ypanel/shared/release"
)

const (
	agentInstallDir = "/opt/ypagent"
	agentBin        = "/opt/ypagent/ypagent"
	agentUnit       = "ypagent"
	agentUpdatesDir = "/opt/ypagent/updates"
	agentSymlink    = "/usr/local/bin/ypagent"
	agentApplyLog   = "/tmp/ypagent-apply.log"
)

const agentUsageText = `YPanel Agent CLI
用法:
  ypagent                            启动 agent（默认；配对: -core -code [-name] [-pair-only]，手工 PSK: -token）
  ypagent version                    查看版本
  ypagent info                       节点信息：版本/节点名/core 地址/凭据/服务状态/core 可达性
  ypagent status                     查看服务状态
  ypagent start|stop|restart         服务启停/重启（需 root）
  ypagent update [--source gitee|github] [--force]
                                     在线自升级（下载 Release → sha256 校验 → 备份替换 → 重启）
  ypagent uninstall [--yes]          卸载（删除 /opt/ypagent 与 /etc/ypanel 凭据）
`

// runSubcommand 处理 CLI 子命令，返回是否已处理（false = 未知命令）。
// args 为 flag.Parse() 之后的全部位置参数（args[0] 是子命令）。
func runSubcommand(version string, args []string) bool {
	dataDir := defaultDataDir()
	if v := os.Getenv("YPANEL_AGENT_DATA"); v != "" {
		dataDir = v
	}
	switch args[0] {
	case "version":
		fmt.Println("YPanel Agent", version)
	case "info":
		agentInfo(version, dataDir)
	case "status":
		agentServiceStatus()
	case "start", "stop", "restart":
		agentServiceAction(args[0])
	case "update":
		agentUpdate(version, args[1:])
	case "uninstall":
		agentUninstall(args[1:])
	default:
		return false
	}
	return true
}

// ---- 通用助手 ----

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "[ypagent] "+msg)
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

// ---- 节点信息 ----

func agentInfo(version, dataDir string) {
	fmt.Println("YPanel Agent", version)
	hostname, _ := os.Hostname()
	fmt.Printf("本机节点名: %s\n", lower(hostname))
	cred, err := pair.Load(filepath.Join(dataDir, "agent.json"))
	if err != nil {
		fmt.Println("配对凭据:   未找到（未配对或 -data 指向他处）")
	} else {
		fmt.Printf("节点名:     %s\n", cred.Name)
		fmt.Printf("core 地址:  %s\n", cred.CoreURL)
		if cred.CoreURL != "" {
			fmt.Printf("core 状态:  %s\n", probeCore(cred.CoreURL))
		}
	}
	if runtime.GOOS == "linux" {
		state, enabled := unitState(agentUnit)
		fmt.Printf("服务状态:   %s (%s)\n", state, enabled)
	}
}

// probeCore 探测 core 可达性（一次性 GET /health，3s 超时；地址来自节点自身配对凭据）。
func probeCore(coreURL string) string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(strings.TrimRight(coreURL, "/") + "/health")
	if err != nil {
		return "不可达（" + err.Error() + "）"
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return "可达"
	}
	return fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// ---- 服务管理（systemd 封装） ----

func agentServiceStatus() {
	requireLinux("ypagent status")
	state, enabled := unitState(agentUnit)
	fmt.Printf("节点 agent %s: %s (%s)\n", agentUnit, state, enabled)
	fmt.Println("日志: journalctl -u " + agentUnit + " -f")
}

func agentServiceAction(action string) {
	requireLinux("ypagent " + action)
	requireRoot("ypagent " + action)
	if out, err := systemctl(action, agentUnit); err != nil {
		fail(fmt.Sprintf("%s %s 失败: %s %v", action, agentUnit, out, err))
	}
	switch action {
	case "start":
		fmt.Println("已启动，稍候可用 ypagent status 确认就绪")
	case "stop":
		fmt.Println("已停止（面板「节点管理」将显示本节点离线）")
	case "restart":
		fmt.Println("已重启，服务就绪约需数秒")
	}
}

// ---- 在线自升级（与面板 CLI 同一 shared/release 链路；面板不可达时的兜底通道） ----

func agentUpdate(version string, rest []string) {
	requireLinux("ypagent update")
	requireRoot("ypagent update")
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
	if !force && !release.IsDevVersion(version) && release.CompareVersions(version, rel.Version) >= 0 {
		fmt.Println("已是最新版本", version)
		return
	}
	if err := release.ValidateAssetURL(rel.AssetURL); err != nil {
		fail(err.Error())
	}
	fmt.Printf("最新版本 %s，开始下载并校验…\n", rel.Version)
	if err := os.MkdirAll(agentUpdatesDir, 0o755); err != nil {
		fail(err.Error())
	}
	dl := exec.CommandContext(ctx, "sh", "-c", release.DownloadScript(agentUpdatesDir, runtime.GOARCH, rel.AssetURL, rel.SumURL))
	dl.Stdout = os.Stdout
	dl.Stderr = os.Stderr
	if err := dl.Run(); err != nil {
		fail("下载/校验失败，请稍后重试或用 --source github 换源: " + err.Error())
	}
	src := filepath.Join(agentUpdatesDir, "ypagent")
	if _, err := os.Stat(src); err != nil {
		fail("解包产物中未找到 ypagent 二进制")
	}
	applyScript := filepath.Join(agentUpdatesDir, "apply-update.sh")
	if err := os.WriteFile(applyScript, []byte(release.ApplyScript(src, agentBin, agentUnit)), 0o700); err != nil {
		fail(err.Error())
	}
	if out, err := exec.Command("sh", "-c", release.ApplyDetachedCommand(applyScript, agentApplyLog)).CombinedOutput(); err != nil {
		fail("apply 启动失败: " + string(out))
	}
	fmt.Println("更新已启动：约 2 秒后替换二进制并重启服务")
	fmt.Printf("稍后用 ypagent version 确认（当前版本 %s）\n", version)
}

// ---- 卸载 ----

func agentUninstall(rest []string) {
	requireLinux("ypagent uninstall")
	requireRoot("ypagent uninstall")
	fmt.Println("即将卸载 YPanel 节点 agent：")
	fmt.Println("  - 停止并移除 systemd 服务 " + agentUnit)
	fmt.Println("  - 删除 " + agentInstallDir + " 与配对凭据 /etc/ypanel")
	fmt.Println("  - 卸载后面板「节点管理」将显示本节点离线，可在面板侧移除节点记录")
	if !argsHave(rest, "--yes") {
		fmt.Print("确认卸载？输入 yes 确认: ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(line) != "yes" {
			fmt.Println("已取消")
			return
		}
	}
	_ = exec.Command("systemctl", "disable", "--now", agentUnit).Run()
	_ = os.Remove("/etc/systemd/system/" + agentUnit + ".service")
	_ = exec.Command("systemctl", "daemon-reload").Run()
	if err := os.RemoveAll(agentInstallDir); err != nil {
		fail("删除 " + agentInstallDir + " 失败: " + err.Error())
	}
	_ = os.RemoveAll("/etc/ypanel")
	_ = os.Remove(agentSymlink)
	fmt.Println("节点 agent 已卸载")
}
