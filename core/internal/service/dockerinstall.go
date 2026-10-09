// DockerInstallService Docker/Compose 一键安装（M52）：预检 + 多源 repo 安装 + 镜像加速器配置。
// 零 agent 改动：全部经 execOnNode 分步执行（本机内嵌 agent 与远程节点同构）；任务中心承载进度日志。
// 源清单为服务端常量，命令模板不拼接用户输入；镜像自定义域名经字符白名单校验。
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/errs"
)

// 安装源标识（枚举，进入命令前经 map 白名单映射为服务端常量）。
const (
	SourceOfficial = "official"
	SourceAliyun   = "aliyun"
	SourceTuna     = "tuna"
	SourceUstc     = "ustc"
)

// aptSourceBases apt 系（Debian/Ubuntu）docker-ce 仓库 base，实际地址 = <base>/<distro>。
var aptSourceBases = map[string]string{
	SourceOfficial: "https://download.docker.com/linux",
	SourceAliyun:   "https://mirrors.aliyun.com/docker-ce/linux",
	SourceTuna:     "https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux",
	SourceUstc:     "https://mirrors.ustc.edu.cn/docker-ce/linux",
}

// yumRepoURLs yum 系（RHEL 兼容）docker-ce.repo 文件地址。
var yumRepoURLs = map[string]string{
	SourceOfficial: "https://download.docker.com/linux/centos/docker-ce.repo",
	SourceAliyun:   "https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo",
	SourceTuna:     "https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/centos/docker-ce.repo",
	SourceUstc:     "https://mirrors.ustc.edu.cn/docker-ce/linux/centos/docker-ce.repo",
}

// yumOfficialBase 官方 yum base（repo 文件内 baseurl/gpgkey 前缀，换源时整段替换）。
const yumOfficialBase = "https://download.docker.com/linux/centos"

// yumMirrorBases 镜像源对应的 yum base（与官方同路径结构）。
var yumMirrorBases = map[string]string{
	SourceAliyun: "https://mirrors.aliyun.com/docker-ce/linux/centos",
	SourceTuna:   "https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/centos",
	SourceUstc:   "https://mirrors.ustc.edu.cn/docker-ce/linux/centos",
}

// aptDebFamilies apt 通道发行版（os-release ID）。
var aptDebFamilies = map[string]bool{"debian": true, "ubuntu": true}

// yumRhelFamilies yum 通道发行版（一律复用 centos 仓库按主版本映射）。
var yumRhelFamilies = map[string]bool{
	"centos": true, "rhel": true, "rocky": true, "almalinux": true,
	"anolis": true, "alinux": true, "opencloudos": true, "tencentos": true,
}

// dockerPackages 完整安装包组（apt 与 yum 同名）。
const dockerPackages = "docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"

// mirrorURLOnly 自定义加速器地址白名单（scheme+host[:port][/path]，杜绝 shell 元字符混入）。
var mirrorURLOnly = regexp.MustCompile(`^https?://[A-Za-z0-9.-]+(:\d{1,5})?(/[A-Za-z0-9._/-]*)?$`)

// DockerInstallService Docker 安装服务。
type DockerInstallService struct {
	nodes *NodeService
	tasks *TaskService
}

// NewDockerInstallService 创建。
func NewDockerInstallService(nodes *NodeService, tasks *TaskService) *DockerInstallService {
	return &DockerInstallService{nodes: nodes, tasks: tasks}
}

// DockerInstallPrecheck 预检结果。
type DockerInstallPrecheck struct {
	NodeID           string `json:"nodeId"`
	Supported        bool   `json:"supported"`
	Reason           string `json:"reason,omitempty"`
	Family           string `json:"family"`             // debian|rhel|unknown
	Distro           string `json:"distro"`             // os-release ID
	Version          string `json:"version"`            // os-release VERSION_ID
	Codename         string `json:"codename,omitempty"` // os-release VERSION_CODENAME
	PrettyName       string `json:"prettyName,omitempty"`
	Arch             string `json:"arch,omitempty"`
	Systemd          bool   `json:"systemd"`
	DockerInstalled  bool   `json:"dockerInstalled"`
	DockerVersion    string `json:"dockerVersion,omitempty"`
	DockerRunning    bool   `json:"dockerRunning"`
	ComposeInstalled bool   `json:"composeInstalled"`
	ComposeVersion   string `json:"composeVersion,omitempty"`
	Action           string `json:"action"` // full|compose-only|none
	Note             string `json:"note,omitempty"`
}

// detectScript 预检探测脚本（只读，POSIX sh）。
const detectScript = `. /etc/os-release 2>/dev/null
echo "ID=${ID:-}"
echo "VERSION_ID=${VERSION_ID:-}"
echo "CODENAME=${VERSION_CODENAME:-$(lsb_release -cs 2>/dev/null)}"
echo "PRETTY=${PRETTY_NAME:-}"
echo "ARCH=$(uname -m 2>/dev/null)"
echo "INIT=$(ps -p 1 -o comm= 2>/dev/null | tr -d ' ')"
echo "DBIN=$(command -v docker 2>/dev/null)"
echo "DVER=$(docker --version 2>/dev/null)"
echo "DACT=$(systemctl is-active docker 2>/dev/null)"
echo "CVER=$(docker compose version 2>/dev/null)"`

// Precheck 在目标节点执行只读环境探测与支持性判定。
func (s *DockerInstallService) Precheck(ctx context.Context, nodeId string) (*DockerInstallPrecheck, error) {
	nodeId = normalizeNodeID(nodeId)
	out, code, err := execOnNode(ctx, s.nodes, nodeId, detectScript, 30)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, svcBadReq("环境探测失败（exit " + fmt.Sprintf("%d", code) + "）：" + tailStr(out, 500))
	}
	pk := parseDetect(out)
	pk.NodeID = nodeId
	classify(pk)
	return pk, nil
}

// parseDetect 解析 key=value 探测输出。
func parseDetect(out string) *DockerInstallPrecheck {
	pk := &DockerInstallPrecheck{}
	kv := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		if i := strings.IndexByte(line, '='); i > 0 {
			kv[line[:i]] = strings.TrimSpace(line[i+1:])
		}
	}
	pk.Distro = kv["ID"]
	pk.Version = kv["VERSION_ID"]
	pk.Codename = kv["CODENAME"]
	pk.PrettyName = kv["PRETTY"]
	pk.Arch = kv["ARCH"]
	pk.Systemd = kv["INIT"] == "systemd"
	pk.DockerInstalled = kv["DBIN"] != ""
	pk.DockerVersion = kv["DVER"]
	pk.DockerRunning = kv["DACT"] == "active"
	pk.ComposeInstalled = kv["CVER"] != ""
	pk.ComposeVersion = kv["CVER"]
	return pk
}

// classify 支持性判定与安装路径决策（原地修改 pk）。
func classify(pk *DockerInstallPrecheck) {
	if !pk.Systemd {
		pk.Supported, pk.Reason, pk.Family = false, "目标机未使用 systemd（PID 1 非 systemd），无法托管 docker 服务", "unknown"
		return
	}
	if pk.Arch != "" && pk.Arch != "x86_64" && pk.Arch != "aarch64" {
		pk.Supported, pk.Reason, pk.Family = false, "暂不支持的架构: "+pk.Arch, "unknown"
		return
	}
	switch {
	case aptDebFamilies[pk.Distro]:
		pk.Family = "debian"
		if pk.Codename == "" {
			pk.Supported, pk.Reason = false, "无法确定发行版代号（VERSION_CODENAME），docker 仓库需要该标识"
			return
		}
	case yumRhelFamilies[pk.Distro]:
		pk.Family = "rhel"
		major, ok := yumMajor(pk)
		if !ok {
			pk.Supported, pk.Reason = false, "docker-ce 仓库不支持该版本: "+pk.Distro+" "+pk.Version
			return
		}
		if pk.Distro == "centos" && major == "8" {
			pk.Supported, pk.Reason = false, "CentOS 8 已停止维护，docker-ce 镜像源已移除该版本；请升级到 Rocky/AlmaLinux 9 或 CentOS Stream 9"
			return
		}
		if pk.Distro == "centos" && major == "7" {
			pk.Note = joinNote(pk.Note, "CentOS 7 已 EOL，docker-ce 仓库仍可用，建议尽快升级系统")
		}
	default:
		name := orDefaultStr(pk.PrettyName, pk.Distro)
		pk.Supported, pk.Reason, pk.Family = false, "暂不支持的发行版: "+name+"（当前支持 Debian/Ubuntu 及 RHEL 兼容系）", "unknown"
		return
	}
	pk.Supported = true
	switch {
	case pk.DockerInstalled && pk.ComposeInstalled:
		pk.Action = "none"
		pk.Note = joinNote(pk.Note, "Docker 与 Compose 插件均已安装，无需安装")
	case pk.DockerInstalled:
		pk.Action = "compose-only"
	default:
		pk.Action = "full"
	}
}

// yumMajor yum 系主版本映射（显式主版本，不依赖 $releasever——Alma/Rocky 的 releasever 为 9.x 形式，仓库路径会 404）。
func yumMajor(pk *DockerInstallPrecheck) (string, bool) {
	major := pk.Version
	if i := strings.IndexByte(major, '.'); i > 0 {
		major = major[:i]
	}
	// Alibaba Cloud Linux 3 兼容 RHEL8、2 兼容 RHEL7
	if pk.Distro == "alinux" {
		switch major {
		case "3":
			return "8", true
		case "2":
			return "7", true
		}
	}
	switch major {
	case "7", "8", "9":
		return major, true
	}
	return major, false
}

// ---- 安装步骤 ----

// InstallStep 单个安装步骤。
type InstallStep struct {
	Name    string `json:"name"`
	Cmd     string `json:"cmd"`
	Timeout int    `json:"timeout"`
}

// buildSteps 依据预检结果与安装源生成步骤序列（全部服务端常量，无用户输入拼接）。
func buildSteps(pk *DockerInstallPrecheck, source string) ([]InstallStep, error) {
	if !pk.Supported || pk.Action == "none" {
		return nil, svcBadReq("当前环境无需或不可安装")
	}
	if _, ok := aptSourceBases[source]; !ok {
		return nil, svcBadReq("未知安装源: " + source)
	}
	var steps []InstallStep
	add := func(name, cmd string, timeout int) {
		steps = append(steps, InstallStep{Name: name, Cmd: cmd, Timeout: timeout})
	}
	pkg := dockerPackages
	if pk.Action == "compose-only" {
		pkg = "docker-compose-plugin"
	}
	verify := `docker version --format 'Docker Server {{.Server.Version}}' && docker compose version`

	if pk.Family == "debian" {
		repoBase := aptSourceBases[source] + "/" + pk.Distro
		// apt 架构名与 uname -m 不同：x86_64→amd64、aarch64→arm64（143 真机实锤：
		// arch=x86_64 时 apt 找不到任何 Packages 索引，报 Unable to locate package）
		aptArch := pk.Arch
		switch aptArch {
		case "x86_64":
			aptArch = "amd64"
		case "aarch64":
			aptArch = "arm64"
		}
		// keyring 用 .gpg 扩展：apt 2.4 对 signed-by 按 .asc/.gpg 扩展名分流解析，
		// dearmor 输出是二进制，存 .asc 会被当 armored 解析导致 NO_PUBKEY（143 真机实锤）
		line := fmt.Sprintf("deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] %s %s stable", aptArch, repoBase, pk.Codename)
		// 写源必须最前：换源重装/重试场景下，旧源残留的脏索引会毒死后续任何 apt-get update
		// （143 真机实锤：任务在依赖步骤的 update 撞旧源 hash mismatch，根本走不到写源步骤）
		add("写入 apt 仓库配置（/etc/apt/sources.list.d/docker.list）",
			buildB64WriteScript("/etc/apt/sources.list.d/docker.list", []byte(line)), 30)
		add("检查依赖（curl/gnupg，缺失才安装）",
			"command -v curl >/dev/null 2>&1 && command -v gpg >/dev/null 2>&1 || { export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get install -y curl ca-certificates gnupg; }", 600)
		add("导入 Docker 仓库 GPG 密钥",
			"install -m 0755 -d /etc/apt/keyrings && curl -fsSL "+repoBase+"/gpg | gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg && chmod 0644 /etc/apt/keyrings/docker.gpg", 60)
		// 公共镜像站存在同步瞬态（Packages 与 InRelease 短暂不一致），自动重试一次（143 真机实锤）
		add("刷新 apt 索引", "apt-get update || (sleep 3 && apt-get update)", 600)
		add("安装软件包（"+pkg+"）",
			"export DEBIAN_FRONTEND=noninteractive; apt-get install -y "+pkg, 600)
		add("启用并启动 docker 服务", "systemctl enable --now docker", 60)
		add("验证安装", verify, 30)
		return steps, nil
	}

	// rhel 系：下载 docker-ce.repo 后做域名替换（换源时）与主版本显式化（必做）
	major, _ := yumMajor(pk)
	sedExpr := ""
	if target, ok := yumMirrorBases[source]; ok {
		sedExpr = fmt.Sprintf("s#%s#%s#g; ", yumOfficialBase, target)
	}
	sedExpr += fmt.Sprintf(`s#\$releasever#%s#g`, major)
	add("确保 curl 可用", "command -v curl >/dev/null 2>&1 || yum install -y curl", 300)
	add("下载 docker-ce 仓库配置（/etc/yum.repos.d/docker-ce.repo）",
		"curl -fsSL -o /etc/yum.repos.d/docker-ce.repo "+yumRepoURLs[source], 60)
	add("仓库路径显式化（主版本 "+major+"）",
		"sed -i '"+sedExpr+"' /etc/yum.repos.d/docker-ce.repo", 30)
	add("刷新软件源缓存", "yum -y makecache || (sleep 3 && yum -y makecache)", 600)
	add("安装软件包（"+pkg+"）", "yum install -y "+pkg, 900)
	add("启用并启动 docker 服务", "systemctl enable --now docker", 60)
	add("验证安装", verify, 30)
	return steps, nil
}

// InstallReq 安装请求。
type InstallReq struct {
	NodeID          string   `json:"nodeId,omitempty"`
	Source          string   `json:"source" binding:"required"`
	DryRun          bool     `json:"dryRun"`
	ConfigureMirror bool     `json:"configureMirror"`
	Mirrors         []string `json:"mirrors,omitempty"`
}

// Install 创建安装任务（dryRun=true 时只生成步骤并在目标节点做 bash -n 语法校验，不执行安装）。
func (s *DockerInstallService) Install(ctx context.Context, req InstallReq) (map[string]any, error) {
	if err := validateMirrors(req.Mirrors); err != nil {
		return nil, err
	}
	nodeId := normalizeNodeID(req.NodeID)

	if req.DryRun {
		pk, err := s.Precheck(ctx, nodeId)
		if err != nil {
			return nil, err
		}
		steps, err := buildSteps(pk, req.Source)
		if err != nil {
			return nil, err
		}
		syntaxOk, syntaxOut := s.checkSyntax(ctx, nodeId, steps)
		return map[string]any{"dryRun": true, "precheck": pk, "steps": steps, "syntaxOk": syntaxOk, "syntaxOut": syntaxOut}, nil
	}

	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	task, err := s.tasks.StartTask(TaskDockerInstall, "安装 Docker/Compose（"+sourceLabel(req.Source)+"）", nodeId+":"+req.Source, 30*time.Minute,
		func(ctx context.Context, logf TaskLogf) error {
			return s.runInstall(ctx, logf, nodeId, req)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// runInstall 任务体：预检复核（幂等）→ 分步执行 → 可选加速器配置。
func (s *DockerInstallService) runInstall(ctx context.Context, logf TaskLogf, nodeId string, req InstallReq) error {
	logf("info", "[预检] 正在复核目标环境…")
	pk, err := s.Precheck(ctx, nodeId)
	if err != nil {
		return fmt.Errorf("预检失败: %w", err)
	}
	logf("info", "[预检] %s / %s / %s", orDefaultStr(pk.PrettyName, pk.Distro), pk.Arch, mapStr(pk.Systemd, "systemd", "无 systemd"))
	if !pk.Supported {
		return fmt.Errorf("环境不支持安装: %s", pk.Reason)
	}
	if pk.Action == "none" {
		logf("info", "[预检] Docker 与 Compose 插件均已就绪，无需安装")
		return nil
	}
	logf("info", "[预检] 安装路径: %s；安装源: %s", mapStr(pk.Action == "full", "完整安装", "仅补装 Compose 插件"), sourceLabel(req.Source))

	steps, err := buildSteps(pk, req.Source)
	if err != nil {
		return err
	}
	for i, st := range steps {
		logf("info", "[%d/%d] %s", i+1, len(steps), st.Name)
		out, code, err := execOnNode(ctx, s.nodes, nodeId, st.Cmd, st.Timeout)
		if err != nil {
			return fmt.Errorf("步骤「%s」执行失败: %w", st.Name, err)
		}
		if code != 0 {
			logf("error", "步骤「%s」失败（exit %d），输出尾部：\n%s", st.Name, code, tailStr(out, 4096))
			if strings.Contains(out, "dpkg was interrupted") || strings.Contains(out, "dpkg --configure -a") {
				logf("info", "检测到 dpkg 半装状态，可在节点终端执行 `dpkg --configure -a` 后重试")
			}
			if strings.Contains(out, "unexpected size") || strings.Contains(out, "MD5Sum mismatch") || strings.Contains(out, "hash sum mismatch") {
				logf("info", "检测到镜像站索引同步不一致（站方瞬态），可稍后重试或更换安装源")
			}
			return fmt.Errorf("步骤「%s」失败（exit %d）", st.Name, code)
		}
		if lines := tailStr(out, 2048); lines != "" {
			logf("info", "输出：\n%s", lines)
		}
	}
	logf("info", "Docker/Compose 安装完成")

	if req.ConfigureMirror && len(req.Mirrors) > 0 {
		logf("info", "[加速器] 正在配置 registry-mirrors（%d 个）…", len(req.Mirrors))
		if _, err := s.SetRegistryMirrors(ctx, nodeId, req.Mirrors); err != nil {
			logf("error", "[加速器] 配置失败（不影响安装结果）：%s", err.Error())
		} else {
			logf("info", "[加速器] 已写入 daemon.json 并重启 docker 生效")
		}
	}
	return nil
}

// checkSyntax dryRun：合并步骤脚本在目标节点做 bash -n 语法校验。
func (s *DockerInstallService) checkSyntax(ctx context.Context, nodeId string, steps []InstallStep) (bool, string) {
	var sb strings.Builder
	for _, st := range steps {
		sb.WriteString("## step: " + st.Name + "\n" + st.Cmd + "\n")
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(sb.String()))
	out, code, err := execOnNode(ctx, s.nodes, nodeId, "printf %s "+b64+" | base64 -d | bash -n && echo YPSYNTAXOK", 30)
	if err != nil {
		return false, err.Error()
	}
	return code == 0 && strings.Contains(out, "YPSYNTAXOK"), tailStr(out, 500)
}

// TestSource 源连通性测试（status 为 HTTP 状态码，"000"=不可达）。
// URL 按目标机家族构造：apt 系测 dists/<codename>/Release（官方与镜像站均按代号组织，无 stable 套件路径）；
// yum 系测 <yumBase>/<主版本>/repodata/repomd.xml。
func (s *DockerInstallService) TestSource(ctx context.Context, nodeId, source string) (map[string]any, error) {
	if _, ok := aptSourceBases[source]; !ok {
		return nil, svcBadReq("未知安装源: " + source)
	}
	nodeId = normalizeNodeID(nodeId)
	pk, err := s.Precheck(ctx, nodeId)
	if err != nil {
		return nil, err
	}
	if !pk.Supported {
		return nil, svcBadReq("目标环境不支持安装: " + pk.Reason)
	}
	var url string
	switch pk.Family {
	case "debian":
		if code := orDefaultStr(pk.Codename, ""); code != "" {
			url = aptSourceBases[source] + "/" + pk.Distro + "/dists/" + code + "/Release"
		} else {
			url = aptSourceBases[source] + "/" + pk.Distro + "/dists/"
		}
	case "rhel":
		base := yumOfficialBase
		if m, ok := yumMirrorBases[source]; ok {
			base = m
		}
		major, _ := yumMajor(pk)
		url = base + "/" + major + "/repodata/repomd.xml"
	default:
		return nil, svcBadReq("暂不支持的发行版: " + orDefaultStr(pk.PrettyName, pk.Distro))
	}
	cmd := "curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 '" + url + "'"
	out, _, err := execOnNode(ctx, s.nodes, nodeId, cmd, 20)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(out)
	detail := ""
	// curl 层错误（如 SSL reset）不是三位状态码，归一为 000 并保留原始错误供前端展示
	if len(status) != 3 || status[0] < '0' || status[0] > '9' {
		detail = status
		status = "000"
	}
	res := map[string]any{"source": source, "url": url, "status": status, "ok": status == "200"}
	if detail != "" {
		res["detail"] = detail
	}
	return res, nil
}

// ---- 镜像加速器 ----

// GetRegistryMirrors 读取 daemon.json 当前 registry-mirrors。
func (s *DockerInstallService) GetRegistryMirrors(ctx context.Context, nodeId string) ([]string, error) {
	nodeId = normalizeNodeID(nodeId)
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	out, err := agentclient.DoJSON[struct{}, struct {
		Content string `json:"content"`
	}](ac, ctx, "GET", "/agent/v1/docker/daemon-config", nil)
	if err != nil {
		return nil, err
	}
	return parseMirrors(out.Content), nil
}

// SetRegistryMirrors 读 daemon-config → 合并 registry-mirrors → 写回（agent 自动重启 docker 生效）。空列表 = 移除该字段。
func (s *DockerInstallService) SetRegistryMirrors(ctx context.Context, nodeId string, mirrors []string) (map[string]any, error) {
	if err := validateMirrors(mirrors); err != nil {
		return nil, err
	}
	nodeId = normalizeNodeID(nodeId)
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	got, err := agentclient.DoJSON[struct{}, struct {
		Content string `json:"content"`
	}](ac, ctx, "GET", "/agent/v1/docker/daemon-config", nil)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{}
	if strings.TrimSpace(got.Content) != "" {
		if err := json.Unmarshal([]byte(got.Content), &cfg); err != nil {
			return nil, svcBadReq("现有 daemon.json 不是合法 JSON，请先在 Docker 配置页修正: " + err.Error())
		}
	}
	if len(mirrors) == 0 {
		delete(cfg, "registry-mirrors")
	} else {
		cfg["registry-mirrors"] = mirrors
	}
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	put, err := agentclient.DoJSON[struct {
		Content string `json:"content"`
	}, struct {
		Message string `json:"message"`
	}](ac, ctx, "PUT", "/agent/v1/docker/daemon-config", &struct {
		Content string `json:"content"`
	}{Content: string(content)})
	if err != nil {
		return nil, err
	}
	return map[string]any{"mirrors": mirrors, "message": put.Message}, nil
}

// validateMirrors 校验加速器地址白名单。
func validateMirrors(mirrors []string) error {
	for _, m := range mirrors {
		if !mirrorURLOnly.MatchString(m) {
			return svcBadReq("加速器地址不合法（仅支持 http(s)://域名[:端口][/路径]）: " + m)
		}
	}
	return nil
}

func parseMirrors(content string) []string {
	out := []string{}
	if strings.TrimSpace(content) == "" {
		return out
	}
	var cfg struct {
		RegistryMirrors []string `json:"registry-mirrors"`
	}
	if json.Unmarshal([]byte(content), &cfg) == nil && len(cfg.RegistryMirrors) > 0 {
		out = append(out, cfg.RegistryMirrors...)
	}
	return out
}

// ---- 小工具 ----
// normalizeNodeID / tailStr / orDefaultStr 复用包内既有实现（history.go / src2compose.go）。

func sourceLabel(source string) string {
	switch source {
	case SourceOfficial:
		return "官方源"
	case SourceAliyun:
		return "阿里云镜像"
	case SourceTuna:
		return "清华 TUNA 镜像"
	case SourceUstc:
		return "中科大镜像"
	}
	return source
}

func joinNote(a, b string) string {
	if a == "" {
		return b
	}
	return a + "；" + b
}

func mapStr(cond bool, t, f string) string {
	if cond {
		return t
	}
	return f
}
