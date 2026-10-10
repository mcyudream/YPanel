// NatForwardService NAT 端口转发（iptables DNAT，M24，设计适配自 DPanel iptables-forward）。
// 执行通道：core → agent /agent/v1/exec（sh -c 语义），本机（内嵌 agent）与远程节点同构。
// 隔离：自建 nat 表链 YPANEL_FWD / YPANEL_FWD_POST，不冲刷 docker 等外部管理的链。
// 安全：入参严格校验（端口整数、IP 解析、网卡名白名单正则）后拼接进命令串。
package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// nat 链名与注释前缀（nat 表）。
const (
	natChainFwd      = "YPANEL_FWD"
	natChainPost     = "YPANEL_FWD_POST"
	natCommentPrefix = "ypanel-fwd-"
)

// natMaxRangeSize 端口范围尺寸上限（校验与占用检测共用）。
const natMaxRangeSize = 1000

// natIfacePattern 网卡名白名单。
var natIfacePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

// NatForwardService NAT 转发服务。
type NatForwardService struct {
	db    *gorm.DB
	nodes *NodeService
	mu    sync.Mutex // 串行化 apply，避免并发重建链相互冲刷
}

// NewNatForwardService 创建。
func NewNatForwardService(db *gorm.DB, nodes *NodeService) *NatForwardService {
	return &NatForwardService{db: db, nodes: nodes}
}

// exec 经 agent exec 通道执行命令，返回 (output, exitCode)。
func (s *NatForwardService) exec(ctx context.Context, nodeId, cmd string, timeoutSecs int) (string, int, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return "", 0, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](agentclient.New(node.BaseURL, node.Token), ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs})
	if err != nil {
		return "", 0, err
	}
	return out.Output, out.ExitCode, nil
}

// ---------- 校验 ----------

// natBadReq 参数类业务错误（直接给用户可读消息，不走 Wrap 以免追加通用后缀）。
func natBadReq(msg string) *errs.Error {
	return errs.New(errs.CodeBadRequest, "error.badRequest", msg)
}

// natValidateRule 入参语义校验（controller 层承担 required 等基础校验）。
func natValidateRule(r *model.NatForwardRule) error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || len(r.Name) > 64 {
		return natBadReq("规则名称必填且不超过 64 字符")
	}
	if r.Protocol != "tcp" && r.Protocol != "udp" {
		return natBadReq("协议仅支持 tcp / udp")
	}
	if r.IPFamily != 4 && r.IPFamily != 6 {
		return natBadReq("IP 族仅支持 4 / 6")
	}
	if err := natValidatePortRange(r.ListenPort, r.ListenPortEnd); err != nil {
		return err
	}
	if err := natValidatePortRange(r.TargetPort, r.TargetPortEnd); err != nil {
		return err
	}
	// 端口范围语义：仅允许「范围 → 同尺寸范围」或「单端口 → 单端口」
	listenSize := natRangeSize(r.ListenPort, r.ListenPortEnd)
	targetSize := natRangeSize(r.TargetPort, r.TargetPortEnd)
	if (listenSize == 1) != (targetSize == 1) || (listenSize > 1 && listenSize != targetSize) {
		return natBadReq("端口范围语义不合法：仅允许「范围→同尺寸范围」或「单端口→单端口」")
	}
	ip := net.ParseIP(strings.TrimSpace(r.TargetIP))
	if ip == nil {
		return natBadReq("目标 IP 不合法")
	}
	if r.IPFamily == 4 && ip.To4() == nil {
		return natBadReq("目标 IP 与 IP 族不匹配（应为 IPv4）")
	}
	if r.IPFamily == 6 && ip.To4() != nil {
		return natBadReq("目标 IP 与 IP 族不匹配（应为纯 IPv6 地址）")
	}
	r.TargetIP = ip.String()
	if r.Iface != "" && !natIfacePattern.MatchString(r.Iface) {
		return natBadReq("入站网卡名不合法")
	}
	// DestIP 目标地址匹配（内核 -d）：空 = 不限；仅单 IP 且与地址族匹配
	r.DestIP = strings.TrimSpace(r.DestIP)
	if r.DestIP != "" {
		dip := net.ParseIP(r.DestIP)
		if dip == nil {
			return natBadReq("目标地址匹配仅支持单个 IP")
		}
		if r.IPFamily == 4 && dip.To4() == nil {
			return natBadReq("目标地址匹配与 IP 族不匹配（应为 IPv4）")
		}
		if r.IPFamily == 6 && dip.To4() != nil {
			return natBadReq("目标地址匹配与 IP 族不匹配（应为纯 IPv6 地址）")
		}
		r.DestIP = dip.String()
	}
	return nil
}

func natValidatePortRange(start, end int) error {
	if start < 1 || start > 65535 {
		return natBadReq("端口须在 1-65535 内")
	}
	if end == 0 {
		return nil
	}
	if end < start || end > 65535 {
		return natBadReq("端口范围不合法（结束须 ≥ 起始且 ≤ 65535）")
	}
	if end-start+1 > natMaxRangeSize {
		return natBadReq(fmt.Sprintf("端口范围尺寸不得超过 %d", natMaxRangeSize))
	}
	return nil
}

func natPortEnd(start, end int) int {
	if end == 0 {
		return start
	}
	return end
}

func natRangeSize(start, end int) int {
	return natPortEnd(start, end) - start + 1
}

// ---------- 渲染（纯函数） ----------

// natRenderView 规则渲染投影。
type natRenderView struct {
	ID       uint
	Protocol string
	Iface    string
	DestIP   string // 内核 -d 目标地址匹配，空 = 不限
	Listen   [2]int // 起始/结束（结束 0 = 单端口）
	TargetIP string
	Target   [2]int
}

// natDportSpec 匹配端口规格：单端口 p；范围 a:b（冒号）。
func natDportSpec(r [2]int) string {
	if r[1] == 0 {
		return strconv.Itoa(r[0])
	}
	return strconv.Itoa(r[0]) + ":" + strconv.Itoa(r[1])
}

// natToDestPortSpec --to-destination 端口规格：单端口 p；范围 a-b（短横线，iptables 语法差异）。
func natToDestPortSpec(r [2]int) string {
	if r[1] == 0 {
		return strconv.Itoa(r[0])
	}
	return strconv.Itoa(r[0]) + "-" + strconv.Itoa(r[1])
}

// natTargetAddr --to-destination 目标：IPv6 用 [addr]:port。
func natTargetAddr(ip string, port [2]int, family int) string {
	p := natToDestPortSpec(port)
	if family == 6 {
		return "[" + ip + "]:" + p
	}
	return ip + ":" + p
}

func natIsLoopback(ipStr string, family int) bool {
	if family == 4 {
		return strings.HasPrefix(ipStr, "127.")
	}
	return ipStr == "::1"
}

func natInList(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// buildNatApplyScript 渲染单个地址族的整链重建脚本（幂等）。
// bin 为 iptables / ip6tables；localAddrs 是目标机本机地址集，用于回程 MASQUERADE 判定；
// removals 是接管规则的内核原规则摘除清单（M58），在全部渲染成功之后执行——渲染失败原规则分毫未动。
func buildNatApplyScript(bin string, family int, rules []natRenderView, localAddrs []string, removals []natRemovalSpec) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	fmt.Fprintf(&b, "command -v %s >/dev/null 2>&1 || { echo 'YPERR:%s 命令不存在或权限不足'; exit 64; }\n", bin, bin)
	b.WriteString("IPT=" + bin + "\n")
	fmt.Fprintf(&b, "$IPT -t nat -N %s 2>/dev/null || true\n", natChainFwd)
	fmt.Fprintf(&b, "$IPT -t nat -N %s 2>/dev/null || true\n", natChainPost)
	fmt.Fprintf(&b, "$IPT -t nat -C PREROUTING -j %s 2>/dev/null || $IPT -t nat -I PREROUTING 1 -j %s\n", natChainFwd, natChainFwd)
	fmt.Fprintf(&b, "$IPT -t nat -C OUTPUT -j %s 2>/dev/null || $IPT -t nat -I OUTPUT 1 -j %s\n", natChainFwd, natChainFwd)
	fmt.Fprintf(&b, "$IPT -t nat -C POSTROUTING -j %s 2>/dev/null || $IPT -t nat -I POSTROUTING 1 -j %s\n", natChainPost, natChainPost)
	fmt.Fprintf(&b, "$IPT -t nat -F %s\n", natChainFwd)
	fmt.Fprintf(&b, "$IPT -t nat -F %s\n", natChainPost)
	for _, r := range rules {
		comment := natCommentPrefix + strconv.FormatUint(uint64(r.ID), 10)
		var dn strings.Builder
		dn.WriteString(fmt.Sprintf("$IPT -t nat -A %s -p %s", natChainFwd, r.Protocol))
		if r.Iface != "" {
			dn.WriteString(fmt.Sprintf(" -i '%s'", r.Iface))
		}
		if r.DestIP != "" {
			dn.WriteString(fmt.Sprintf(" -d '%s'", r.DestIP))
		}
		dn.WriteString(fmt.Sprintf(" --dport %s -m comment --comment '%s' -j DNAT --to-destination '%s'",
			natDportSpec(r.Listen), comment, natTargetAddr(r.TargetIP, r.Target, family)))
		fmt.Fprintf(&b, "if ! _o=$(%s 2>&1); then echo 'YPERR:规则 #%d 应用失败: '$_o; exit 65; fi\n", dn.String(), r.ID)
		if !natIsLoopback(r.TargetIP, family) && !natInList(localAddrs, r.TargetIP) {
			fmt.Fprintf(&b, "if ! _o=$($IPT -t nat -A %s -p %s -d '%s' --dport %s -m comment --comment '%s' -j MASQUERADE 2>&1); then echo 'YPERR:规则 #%d 回程 MASQUERADE 失败: '$_o; exit 65; fi\n",
				natChainPost, r.Protocol, r.TargetIP, natDportSpec(r.Target), comment, r.ID)
		}
	}
	b.WriteString(buildNatRemovalSection(removals))
	// 转发开关：为 0 时运行时开启并告警（不持久化，持久化属宿主机运维范畴）
	if family == 4 {
		b.WriteString("ipf=$(cat /proc/sys/net/ipv4/ip_forward 2>/dev/null || echo 1)\n")
		b.WriteString("if [ \"$ipf\" != '1' ]; then sysctl -w net.ipv4.ip_forward=1 >/dev/null && echo 'YPWARN:net.ipv4.ip_forward 已运行时开启（重启后失效，请在宿主机持久化）' || { echo 'YPERR:无法开启 net.ipv4.ip_forward'; exit 66; }; fi\n")
	} else {
		b.WriteString("ipf=$(cat /proc/sys/net/ipv6/conf/all/forwarding 2>/dev/null || echo 1)\n")
		b.WriteString("if [ \"$ipf\" != '1' ]; then sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null && echo 'YPWARN:net.ipv6.conf.all.forwarding 已运行时开启（重启后失效，请在宿主机持久化）' || { echo 'YPERR:无法开启 net.ipv6.conf.all.forwarding'; exit 66; }; fi\n")
	}
	b.WriteString("echo YPOK\n")
	return b.String()
}

// natRemovalSpec 接管规则的原内核规则摘除项（spec 为 -S 输出原文，已过白名单）。
type natRemovalSpec struct {
	Chain  string
	Spec   string
	RuleID uint
}

// natSpecPattern 外部 spec 白名单：仅允许出现在 iptables -S 输出中的安全字符，
// 杜绝 $ ` " ' ; & | ( ) < > 等可被 shell 解释的字符混进摘除命令（引号经 natShellSpec 单独处理）。
var natSpecPattern = regexp.MustCompile(`^[a-zA-Z0-9 !:.,/=\[\]@%+_-]+$`)

// natChainPattern 内核链名白名单（iptables 链名 ≤28 字符）。
var natChainPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,28}$`)

// natCommentSafePattern --comment 内容安全字符：禁 shell 元字符与一切引号（将以单引号嵌入命令）。
var natCommentSafePattern = regexp.MustCompile(`^[^"'` + "`" + `$;&|<>()\\\n\r]*$`)

// natShellSpec 将 -S 输出的 spec 转为可安全嵌入 sh -c 的参数串：
// --comment "x y" 段单独校验内容后重写为单引号形式，其余文本过严格白名单。
// 任一段不安全返回 false（该规则不可导入/不可摘除）。
func natShellSpec(spec string) (string, bool) {
	var b strings.Builder
	rest := spec
	for {
		loc := natCommentRe.FindStringIndex(rest)
		if loc == nil {
			break
		}
		if !natSpecPattern.MatchString(rest[:loc[0]]) {
			return "", false
		}
		b.WriteString(rest[:loc[0]])
		m := natCommentRe.FindStringSubmatch(rest)
		content := ""
		for _, g := range m[1:] {
			if g != "" {
				content = g
				break
			}
		}
		if !natCommentSafePattern.MatchString(content) {
			return "", false
		}
		b.WriteString("--comment '" + content + "'")
		rest = rest[loc[1]:]
	}
	if !natSpecPattern.MatchString(rest) {
		return "", false
	}
	b.WriteString(rest)
	return strings.Join(strings.Fields(b.String()), " "), true
}

// natSplitSrcSpec 拆分接管来源 "<chain>|<spec>"：链名校验 + 结构校验；
// spec 文本的安全化在命令生成处经 natShellSpec 单独执行。
func natSplitSrcSpec(src string) (chain, spec string, ok bool) {
	i := strings.Index(src, "|")
	if i <= 0 || i == len(src)-1 {
		return "", "", false
	}
	chain, spec = src[:i], src[i+1:]
	if !natChainPattern.MatchString(chain) {
		return "", "", false
	}
	return chain, spec, true
}

// buildNatRemovalSection 接管摘除段：逐条 -C 守卫的幂等 -D（重复规则上限 5 份，防死循环）。
// 原规则已不存在时 -C 失败整段跳过，天然幂等；不论面板规则启用与否都摘除（停用 ≠ 还原原规则）。
func buildNatRemovalSection(removals []natRemovalSpec) string {
	var b strings.Builder
	for _, rm := range removals {
		fmt.Fprintf(&b, "_n=0; while [ $_n -lt 5 ] && $IPT -t nat -C %s %s 2>/dev/null; do $IPT -t nat -D %s %s || { echo 'YPERR:规则 #%d 原规则摘除失败'; exit 67; }; _n=$((_n+1)); done\n",
			rm.Chain, rm.Spec, rm.Chain, rm.Spec, rm.RuleID)
	}
	return b.String()
}

// buildNatRestoreScript 导入失败回滚用的原规则还原脚本（尽力而为，逐条 ! -C → -A，失败仅告警不中断）。
func buildNatRestoreScript(bin string, family int, rules []*model.NatForwardRule) string {
	var b strings.Builder
	fmt.Fprintf(&b, "command -v %s >/dev/null 2>&1 || exit 0\n", bin)
	b.WriteString("IPT=" + bin + "\n")
	for _, r := range rules {
		if r.IPFamily != family || r.SrcSpec == "" {
			continue
		}
		chain, spec, ok := natSplitSrcSpec(r.SrcSpec)
		if !ok {
			continue
		}
		ss, ok := natShellSpec(spec)
		if !ok {
			slog.Warn("NAT 接管来源 spec 含不安全字符，跳过还原", "rule", r.ID)
			continue
		}
		fmt.Fprintf(&b, "if ! $IPT -t nat -C %s %s 2>/dev/null; then if ! _o=$($IPT -t nat -A %s %s 2>&1); then echo 'YPWARN:原规则还原失败: '$_o; fi; fi\n",
			chain, ss, chain, ss)
	}
	b.WriteString("echo YPOK\n")
	return b.String()
}

// buildNatFlushScript 容忍式冲刷脚本：该地址族无启用规则时清除可能残留的链内容。
// 命令缺失或链不存在均静默成功（exit 0），不新建链、不挂载、不触碰 sysctl。
func buildNatFlushScript(bin string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "command -v %s >/dev/null 2>&1 || exit 0\n", bin)
	fmt.Fprintf(&b, "%s -t nat -L %s -n >/dev/null 2>&1 && %s -t nat -F %s\n", bin, natChainFwd, bin, natChainFwd)
	fmt.Fprintf(&b, "%s -t nat -L %s -n >/dev/null 2>&1 && %s -t nat -F %s\n", bin, natChainPost, bin, natChainPost)
	b.WriteString("echo YPOK\n")
	return b.String()
}

// natEnabledViews 提取某地址族下启用态规则的渲染投影。
func natEnabledViews(rules []model.NatForwardRule, family int) []natRenderView {
	views := []natRenderView{}
	for _, r := range rules {
		if r.IPFamily != family || !r.Enabled {
			continue
		}
		views = append(views, natRenderView{
			ID: r.ID, Protocol: r.Protocol, Iface: r.Iface, DestIP: r.DestIP,
			Listen:   [2]int{r.ListenPort, r.ListenPortEnd},
			TargetIP: r.TargetIP,
			Target:   [2]int{r.TargetPort, r.TargetPortEnd},
		})
	}
	return views
}

// natParseWarnings 从脚本输出解析 YPWARN 告警行。
func natParseWarnings(out string) []string {
	ws := []string{}
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "YPWARN:"); i >= 0 {
			ws = append(ws, strings.TrimSpace(line[i+len("YPWARN:"):]))
		}
	}
	return ws
}

// natApplyFailMsg 从失败输出提取 YPERR 短消息，回退为输出尾部。
func natApplyFailMsg(bin, out string) string {
	msg := strings.TrimSpace(out)
	if i := strings.LastIndex(msg, "YPERR:"); i >= 0 {
		msg = strings.TrimSpace(msg[i+len("YPERR:"):])
	} else if len(msg) > 400 {
		msg = msg[len(msg)-400:]
	}
	return bin + " 应用失败: " + msg
}

// ---------- 探测 ----------

// NatInterface 网卡地址条目。
type NatInterface struct {
	Name     string `json:"name"`
	Family   int    `json:"family"` // 4 / 6
	Addr     string `json:"addr"`
	Internal bool   `json:"internal"` // lo / docker 自动网卡（默认展示置后/弱化）
}

func natIsInternalIface(name string) bool {
	return name == "lo" || strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "br-")
}

// GetInterfaces 查询节点网卡与 IP（入站网卡下拉 / 一键填写本机 / MASQ 判定共用）。
func (s *NatForwardService) GetInterfaces(ctx context.Context, nodeId string, family int) ([]NatInterface, error) {
	out, code, err := s.exec(ctx, nodeId, "ip -o addr show", 30)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "ip 命令执行失败（目标机需安装 iproute2）: "+strings.TrimSpace(out))
	}
	ifaces := []NatInterface{}
	for _, it := range parseNatInterfaces(out) {
		if family == 4 || family == 6 {
			if it.Family != family {
				continue
			}
		}
		ifaces = append(ifaces, it)
	}
	return ifaces, nil
}

// parseNatInterfaces 解析 ip -o addr show 输出（纯函数）。
// 行样例：`2: eth0    inet 192.168.1.10/24 brd 192.168.1.255 scope global eth0\ ...`
func parseNatInterfaces(out string) []NatInterface {
	res := []NatInterface{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 4 || !strings.HasSuffix(f[0], ":") {
			continue
		}
		var family int
		switch f[2] {
		case "inet":
			family = 4
		case "inet6":
			family = 6
		default:
			continue
		}
		addr := strings.SplitN(f[3], "/", 2)[0]
		if net.ParseIP(addr) == nil {
			continue
		}
		res = append(res, NatInterface{Name: f[1], Family: family, Addr: addr, Internal: natIsInternalIface(f[1])})
	}
	return res
}

// NatPortOccupy 端口占用明细。
type NatPortOccupy struct {
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
	Process string `json:"process"`
}

// CheckPort 检测映射端口区间在目标机上的监听占用（逐端口判定）。
func (s *NatForwardService) CheckPort(ctx context.Context, nodeId, protocol string, listenPort, listenPortEnd int) ([]NatPortOccupy, error) {
	if protocol != "tcp" && protocol != "udp" {
		return nil, natBadReq("协议仅支持 tcp / udp")
	}
	if err := natValidatePortRange(listenPort, listenPortEnd); err != nil {
		return nil, err
	}
	out, code, err := s.exec(ctx, nodeId, "echo ===TCP===; ss -H -lntp 2>/dev/null; echo ===UDP===; ss -H -lnup 2>/dev/null", 30)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "ss 命令执行失败（目标机需安装 iproute2）: "+strings.TrimSpace(out))
	}
	listening := parseNatListeningPorts(out)
	occupied := []NatPortOccupy{}
	for p := listenPort; p <= natPortEnd(listenPort, listenPortEnd); p++ {
		// 同端口 v4/v6 多套接字去重：按 端口+协议 合并，保留首个非空进程信息
		seen := map[string]bool{}
		for _, o := range listening[p] {
			key := o.Proto
			if seen[key] {
				continue
			}
			seen[key] = true
			occupied = append(occupied, o)
		}
	}
	return occupied, nil
}

// parseNatListeningPorts 解析 ss -H -lntp/-lnup 输出为 端口→占用列表（纯函数）。
func parseNatListeningPorts(out string) map[int][]NatPortOccupy {
	res := map[int][]NatPortOccupy{}
	proto := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "===TCP===":
			proto = "tcp"
			continue
		case "===UDP===":
			proto = "udp"
			continue
		}
		if proto == "" || line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		// Local Address:Port 为第 4 列（IPv6 形如 [::]:22，取最后一个冒号之后）
		idx := strings.LastIndex(f[3], ":")
		if idx < 0 {
			continue
		}
		port, err := strconv.Atoi(f[3][idx+1:])
		if err != nil {
			continue
		}
		proc := ""
		if rest := strings.Join(f[5:], " "); strings.Contains(rest, "users:((") {
			proc = rest
		}
		res[port] = append(res[port], NatPortOccupy{Port: port, Proto: proto, Process: proc})
	}
	return res
}

func natPortInUseErr(occupied []NatPortOccupy) error {
	parts := []string{}
	for _, o := range occupied {
		p := strconv.Itoa(o.Port) + "/" + o.Proto
		if o.Process != "" {
			p += "（" + o.Process + "）"
		}
		parts = append(parts, p)
	}
	return natBadReq("映射端口在目标机上已被监听占用: " + strings.Join(parts, "、"))
}

// ---------- 应用 ----------

// ApplyNode 整链重建指定节点的转发规则（幂等），返回 warnings（如 sysctl 运行时开启）。
func (s *NatForwardService) ApplyNode(ctx context.Context, nodeId string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyNodeLocked(ctx, nodeId)
}

func (s *NatForwardService) applyNodeLocked(ctx context.Context, nodeId string) ([]string, error) {
	var rules []model.NatForwardRule
	if err := s.db.Where("node_id = ?", nodeId).Order("sort ASC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	// 仅当存在启用的非回环目标时才需要本机地址集判定 MASQUERADE
	needAddrs := false
	for _, r := range rules {
		if r.Enabled && !natIsLoopback(r.TargetIP, r.IPFamily) {
			needAddrs = true
			break
		}
	}
	localAddrs := []string{}
	if needAddrs {
		ifaces, err := s.GetInterfaces(ctx, nodeId, 0)
		if err != nil {
			return nil, natBadReq("获取本机地址失败（回程 MASQUERADE 判定需要）: " + err.Error())
		}
		for _, it := range ifaces {
			localAddrs = append(localAddrs, it.Addr)
		}
	}
	warnings := []string{}
	for _, fb := range []struct {
		family int
		bin    string
	}{{4, "iptables"}, {6, "ip6tables"}} {
		// 仅启用态规则渲染进内核链；停用/全删后走容忍式冲刷清除残留
		views := natEnabledViews(rules, fb.family)
		// 接管摘除清单：本节点该地址族全部 SrcSpec 记录（不论启用状态）；spec 过 shell 化校验
		removals := []natRemovalSpec{}
		for _, r := range rules {
			if r.IPFamily != fb.family || r.SrcSpec == "" {
				continue
			}
			ch, sp, ok := natSplitSrcSpec(r.SrcSpec)
			if !ok {
				slog.Warn("NAT 接管来源 spec 结构非法，跳过摘除", "rule", r.ID)
				continue
			}
			ss, ok := natShellSpec(sp)
			if !ok {
				slog.Warn("NAT 接管来源 spec 含不安全字符，跳过摘除", "rule", r.ID)
				continue
			}
			removals = append(removals, natRemovalSpec{Chain: ch, Spec: ss, RuleID: r.ID})
		}
		script := buildNatApplyScript(fb.bin, fb.family, views, localAddrs, removals)
		if len(views) == 0 && len(removals) == 0 {
			// 该族无启用规则且无待摘除接管规则：容忍式冲刷（命令缺失/链不存在均静默跳过，不建链不探测硬失败）
			script = buildNatFlushScript(fb.bin)
		}
		out, code, err := s.exec(ctx, nodeId, script, 60)
		if err != nil {
			return nil, err
		}
		if code != 0 {
			return nil, errs.Wrapc(errs.CodeFileOpFailed, natApplyFailMsg(fb.bin, out))
		}
		warnings = append(warnings, natParseWarnings(out)...)
	}
	return warnings, nil
}

// ApplyAll 启动重放：逐个拥有规则的节点整链重建。返回各节点失败（第二个 error 为规则清单读取失败）。
func (s *NatForwardService) ApplyAll(ctx context.Context) (map[string]error, error) {
	var nodeIds []string
	if err := s.db.Model(&model.NatForwardRule{}).Distinct().Pluck("node_id", &nodeIds).Error; err != nil {
		return nil, err
	}
	fails := map[string]error{}
	for _, id := range nodeIds {
		if _, err := s.ApplyNode(ctx, id); err != nil {
			fails[id] = err
		}
	}
	return fails, nil
}

// ---------- CRUD ----------

// List 节点规则列表（sort,id 升序）。
func (s *NatForwardService) List(nodeId string) ([]model.NatForwardRule, error) {
	rows := []model.NatForwardRule{}
	if err := s.db.Where("node_id = ?", nodeId).Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Save 新建（ID=0）/更新。写库成功后应用；应用失败回滚 DB（尽力还原旧链）并报错。
func (s *NatForwardService) Save(ctx context.Context, r *model.NatForwardRule) (*model.NatForwardRule, []string, error) {
	if err := natValidateRule(r); err != nil {
		return nil, nil, err
	}
	if _, err := s.nodes.ByID(r.NodeID); err != nil {
		return nil, nil, err
	}
	isUpdate := r.ID != 0
	var before model.NatForwardRule
	if isUpdate {
		if err := s.db.First(&before, r.ID).Error; err != nil {
			return nil, nil, errs.Wrap(errs.ErrNotFound, "规则不存在")
		}
	}
	// 启用状态下创建/更新：前置占用检测
	if r.Enabled {
		occupied, err := s.CheckPort(ctx, r.NodeID, r.Protocol, r.ListenPort, r.ListenPortEnd)
		if err != nil {
			return nil, nil, err
		}
		if len(occupied) > 0 {
			return nil, nil, natPortInUseErr(occupied)
		}
	}
	if isUpdate {
		if err := s.db.Model(&before).Updates(map[string]any{
			"node_id": r.NodeID, "name": r.Name, "protocol": r.Protocol, "ip_family": r.IPFamily,
			"listen_port": r.ListenPort, "listen_port_end": r.ListenPortEnd,
			"target_ip": r.TargetIP, "target_port": r.TargetPort, "target_port_end": r.TargetPortEnd,
			"iface": r.Iface, "dest_ip": r.DestIP, "enabled": r.Enabled, "sort": r.Sort,
		}).Error; err != nil {
			return nil, nil, err
		}
	} else {
		if err := s.db.Create(r).Error; err != nil {
			return nil, nil, err
		}
	}
	warnings, err := s.ApplyNode(ctx, r.NodeID)
	if err == nil && isUpdate && before.NodeID != r.NodeID {
		// 节点迁移：旧节点链也要移除该规则
		_, err = s.ApplyNode(ctx, before.NodeID)
	}
	if err != nil {
		s.rollbackSave(&before, r, isUpdate)
		return nil, nil, natBadReq("规则已回滚（应用失败）: " + err.Error())
	}
	return r, warnings, nil
}

// rollbackSave 应用失败后的 DB 回滚 + 尽力还原旧链（还原失败仅记日志，不吞主错误）。
func (s *NatForwardService) rollbackSave(before *model.NatForwardRule, cur *model.NatForwardRule, isUpdate bool) {
	if isUpdate {
		if err := s.db.Save(before).Error; err != nil {
			slog.Error("NAT 规则回滚失败", "id", before.ID, "err", err.Error())
		}
		// cur 的规则可能已部分进链：从 DB（已还原）重建各相关节点
		for _, nid := range []string{before.NodeID, cur.NodeID} {
			if _, err := s.ApplyNode(context.Background(), nid); err != nil {
				slog.Warn("NAT 规则回滚后还原链失败", "node", nid, "err", err.Error())
			}
			if before.NodeID == cur.NodeID {
				break
			}
		}
		return
	}
	if err := s.db.Delete(cur).Error; err != nil {
		slog.Error("NAT 规则回滚失败", "id", cur.ID, "err", err.Error())
	}
	if _, err := s.ApplyNode(context.Background(), cur.NodeID); err != nil {
		slog.Warn("NAT 规则回滚后还原链失败", "node", cur.NodeID, "err", err.Error())
	}
}

// Delete 删除规则；应用失败时回滚（还原记录并重建链）。
func (s *NatForwardService) Delete(ctx context.Context, id uint) ([]string, error) {
	var row model.NatForwardRule
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "规则不存在")
	}
	if err := s.db.Delete(&row).Error; err != nil {
		return nil, err
	}
	warnings, err := s.ApplyNode(ctx, row.NodeID)
	if err != nil {
		if e := s.db.Create(&row).Error; e != nil {
			slog.Error("NAT 规则删除回滚失败", "id", row.ID, "err", e.Error())
		}
		_, _ = s.ApplyNode(ctx, row.NodeID)
		return nil, natBadReq("删除已回滚（应用失败）: " + err.Error())
	}
	return warnings, nil
}

// SetEnabled 启停规则。启用方向先做占用检测；变更后应用，失败还原状态。
func (s *NatForwardService) SetEnabled(ctx context.Context, id uint, enabled bool) ([]string, error) {
	var row model.NatForwardRule
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "规则不存在")
	}
	if row.Enabled == enabled {
		return []string{}, nil
	}
	if enabled {
		occupied, err := s.CheckPort(ctx, row.NodeID, row.Protocol, row.ListenPort, row.ListenPortEnd)
		if err != nil {
			return nil, err
		}
		if len(occupied) > 0 {
			return nil, natPortInUseErr(occupied)
		}
	}
	if err := s.db.Model(&row).Update("enabled", enabled).Error; err != nil {
		return nil, err
	}
	warnings, err := s.ApplyNode(ctx, row.NodeID)
	if err != nil {
		if e := s.db.Model(&row).Update("enabled", !enabled).Error; e != nil {
			slog.Error("NAT 规则状态还原失败", "id", row.ID, "err", e.Error())
		}
		_, _ = s.ApplyNode(ctx, row.NodeID)
		return nil, natBadReq("已还原规则状态（应用失败）: " + err.Error())
	}
	return warnings, nil
}
