// 外部 NAT 规则探测与接管（M58）。
// 只读展示全部外部 nat 表规则；仅 nat 表主链（PREROUTING/OUTPUT）手工 DNAT 可导入接管——
// 导入即转面板记录 + apply 摘除原规则（见 natforward.go 摘除段）；管理链（DOCKER/firewalld/
// ufw/libvirt/k8s）永久只读，绝不改动。
package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/ypanel/shared/errs"

	"github.com/ypanel/core/internal/model"
)

// NatExternalRule 外部（非面板管理）nat 表规则条目。
type NatExternalRule struct {
	ID          string `json:"id"`          // sha1(chain+" "+spec) 前 16 位
	Chain       string `json:"chain"`       // 所在链
	Family      int    `json:"family"`      // 4 / 6
	Spec        string `json:"spec"`        // -A <chain> 之后的原文（空白归一），用于展示与摘除
	Source      string `json:"source"`      // manual/docker/firewalld/ufw/libvirt/k8s/custom
	Proto       string `json:"proto"`       // tcp/udp，空 = 无 -p
	Iface       string `json:"iface"`       // -i
	DestIP      string `json:"destIp"`      // -d（已归一为裸 IP）
	DportStart  int    `json:"dportStart"`  // 0 = 无 --dport
	DportEnd    int    `json:"dportEnd"`    // 0 = 单端口
	ToIP        string `json:"toIp"`        // --to-destination 目标 IP（v6 已剥方括号）
	ToPortStart int    `json:"toPortStart"` // 0 = 无目标端口
	ToPortEnd   int    `json:"toPortEnd"`
	Comment     string `json:"comment"` // --comment
	Importable  bool   `json:"importable"`
	Reason      string `json:"reason"` // 不可导入原因（importable=false 时）
}

// NatExternalList 外部规则清单。
type NatExternalList struct {
	Available bool              `json:"available"` // iptables 可用
	Rules     []NatExternalRule `json:"rules"`
}

// natProbeScript 探测脚本：nat 表 -S 两地址族 + 持久化文件 DNAT/MASQ 行。
const natProbeScript = `echo ===V4===
command -v iptables >/dev/null 2>&1 && iptables -t nat -S 2>&1 || true
echo ===V6===
command -v ip6tables >/dev/null 2>&1 && ip6tables -t nat -S 2>&1 || true
echo ===PERSIST===
for f in /etc/iptables/rules.v4 /etc/sysconfig/iptables; do
  if [ -r "$f" ]; then echo "---$f"; grep -E 'DNAT|MASQUERADE|SNAT' "$f" 2>/dev/null || true; fi
done
echo YPOK
`

// natCommentRe 提取并摘除 --comment 值（带引号/裸单词），避免引号内空格干扰分词。
var natCommentRe = regexp.MustCompile(`--comment\s+(?:"([^"]*)"|'([^']*)'|(\S+))`)

// GetExternal 查询节点外部 nat 规则（只读）。
func (s *NatForwardService) GetExternal(ctx context.Context, nodeId string) (*NatExternalList, error) {
	if _, err := s.nodes.ByID(nodeId); err != nil {
		return nil, err
	}
	available, rules, _, err := s.probeExternal(ctx, nodeId)
	if err != nil {
		return nil, err
	}
	return &NatExternalList{Available: available, Rules: rules}, nil
}

// probeExternal 执行探测脚本并解析，persist 为持久化文件段原文（导入告警用）。
func (s *NatForwardService) probeExternal(ctx context.Context, nodeId string) (available bool, rules []NatExternalRule, persist string, err error) {
	out, code, err := s.exec(ctx, nodeId, natProbeScript, 30)
	if err != nil {
		return false, nil, "", err
	}
	if code != 0 {
		return false, nil, "", errs.Wrapc(errs.CodeFileOpFailed, "iptables 探测执行失败: "+strings.TrimSpace(out))
	}
	available, rules = parseNatExternal(out)
	if i := strings.Index(out, "===PERSIST==="); i >= 0 {
		seg := out[i+len("===PERSIST==="):]
		if j := strings.Index(seg, "YPOK"); j >= 0 {
			seg = seg[:j]
		}
		persist = strings.TrimSpace(seg)
	}
	return available, rules, persist, nil
}

// parseNatExternal 解析探测输出（纯函数）：-P/-N/-A 行按地址族归集，其余（告警/错误文本）跳过；
// 面板自建链 YPANEL_* 排除。
func parseNatExternal(out string) (available bool, rules []NatExternalRule) {
	rules = []NatExternalRule{}
	family := 0
	persist := false
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch strings.TrimSpace(line) {
		case "===V4===":
			family, persist = 4, false
			continue
		case "===V6===":
			family, persist = 6, false
			continue
		case "===PERSIST===":
			family, persist = 0, true
			continue
		case "YPOK", "":
			continue
		}
		if persist || family == 0 || !strings.HasPrefix(line, "-") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "-A "):
			if family == 4 {
				available = true
			}
			if r, ok := parseNatExternalRule(family, line[3:]); ok && r.Chain != natChainFwd && r.Chain != natChainPost {
				rules = append(rules, r)
			}
		case strings.HasPrefix(line, "-P "), strings.HasPrefix(line, "-N "):
			if family == 4 {
				available = true
			}
		}
	}
	return available, rules
}

// parseNatExternalRule 解析单条 "-A <chain> <spec>" 行（纯函数）。
func parseNatExternalRule(family int, body string) (NatExternalRule, bool) {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return NatExternalRule{}, false
	}
	r := NatExternalRule{
		Family: family,
		Chain:  fields[0],
		Spec:   strings.Join(fields[1:], " "),
	}
	sum := sha1.Sum([]byte(r.Chain + " " + r.Spec))
	r.ID = hex.EncodeToString(sum[:])[:16]

	// 注释先摘除（值可能带空格与引号），再分词解析其余字段
	work := r.Spec
	if m := natCommentRe.FindStringSubmatch(work); m != nil {
		for _, g := range m[1:] {
			if g != "" {
				r.Comment = g
				break
			}
		}
		work = strings.TrimSpace(strings.Replace(work, m[0], " ", 1))
	}

	// 字段先全部解析（不可导入的规则也照常展示匹配/目标），再按序判定可导入性
	target := ""
	unsupported := ""
	destRaw := ""
	dportRaw := ""
	toRaw := ""
	f := strings.Fields(work)
	for i := 0; i < len(f); i++ {
		t := f[i]
		next := func() string {
			if i+1 < len(f) {
				i++
				return f[i]
			}
			unsupported = "选项 " + t + " 缺少参数"
			return ""
		}
		switch t {
		case "-p", "--protocol":
			r.Proto = next()
		case "-i", "--in-interface":
			r.Iface = next()
		case "-d", "--destination":
			destRaw = next()
		case "--dport", "--destination-port":
			dportRaw = next()
		case "--to-destination":
			toRaw = next()
		case "-j", "--jump":
			target = next()
		case "-m", "--match":
			mod := next()
			if mod != "tcp" && mod != "udp" && mod != "comment" {
				unsupported = "含面板不支持的匹配模块（" + mod + "）"
			}
		case "-s", "--source", "--sport", "--source-port", "-o", "--out-interface",
			"--icmp-type", "--tcp-flags", "--set", "--mask", "--mark":
			unsupported = "含面板不支持的匹配（" + t + "）"
			_ = next() // 跳过参数值
		case "!":
			unsupported = "含取反匹配"
		default:
			if strings.HasPrefix(t, "--") && unsupported == "" {
				unsupported = "含未识别选项（" + t + "）"
			}
		}
	}
	r.Source = natExternalSource(r.Chain)

	dportOK := false
	if v, ok := natParseMatchPort(dportRaw); ok {
		r.DportStart, r.DportEnd = v[0], v[1]
		dportOK = true
	}
	destOK := false
	if v, ok := natParseDestMatch(destRaw); ok {
		r.DestIP = v
		destOK = true
	}
	toOK := false
	if ip, ps, ok := natParseToDestination(toRaw); ok {
		r.ToIP = ip
		r.ToPortStart, r.ToPortEnd = ps[0], ps[1]
		toOK = true
	}

	// ---- 可导入判定（顺序即优先级，首个命中原因即输出）----
	_, shellOK := natShellSpec(r.Spec)
	toIP := net.ParseIP(r.ToIP)
	familyOK := toIP != nil && ((r.Family == 4 && toIP.To4() != nil) || (r.Family == 6 && toIP.To4() == nil))
	lSize := natRangeSize(r.DportStart, r.DportEnd)
	tSize := natRangeSize(r.ToPortStart, r.ToPortEnd)
	rangeOK := (lSize == 1) == (tSize == 1) && (lSize <= 1 || lSize == tSize)
	switch {
	case r.Source != "manual":
		r.Reason = natExternalSourceReason(r.Source)
	case r.Chain != "PREROUTING" && r.Chain != "OUTPUT":
		r.Reason = "仅支持接管主链（PREROUTING/OUTPUT）上的规则"
	case target != "DNAT":
		r.Reason = "仅接管 DNAT 规则（回程/伪装类不接管）"
	case unsupported != "":
		r.Reason = unsupported
	case !shellOK:
		r.Reason = "规则含特殊字符，无法安全摘除"
	case r.Proto != "tcp" && r.Proto != "udp":
		r.Reason = "协议仅支持 tcp / udp"
	case dportRaw == "":
		r.Reason = "缺少 --dport 端口匹配"
	case !dportOK:
		r.Reason = "--dport 端口规格不合法"
	case !destOK:
		r.Reason = "「-d」仅支持单个 IP（不接受网段）"
	case !toOK:
		r.Reason = "--to-destination 规格不合法"
	case !familyOK:
		r.Reason = "目标 IP 与地址族不匹配"
	case !rangeOK:
		r.Reason = "端口范围语义不合法（需「范围→同尺寸范围」或「单端口→单端口」）"
	case r.Iface != "" && !natIfacePattern.MatchString(r.Iface):
		r.Reason = "入站网卡名不合法"
	}
	r.Importable = r.Reason == ""
	return r, true
}

// natExternalSource 链名 → 来源分类。
func natExternalSource(chain string) string {
	switch {
	case strings.HasPrefix(chain, "DOCKER"):
		return "docker"
	case strings.HasPrefix(chain, "FWDI_"), strings.HasPrefix(chain, "FWDO_"), strings.HasPrefix(chain, "FWDM_"),
		strings.HasPrefix(chain, "IN_"), strings.HasPrefix(chain, "OUT_"):
		return "firewalld"
	case strings.HasPrefix(chain, "ufw"):
		return "ufw"
	case strings.HasPrefix(chain, "LIBVIRT"):
		return "libvirt"
	case strings.HasPrefix(chain, "KUBE-"), strings.HasPrefix(chain, "CNI-"), strings.HasPrefix(chain, "cali-"),
		strings.HasPrefix(chain, "flannel"), strings.HasPrefix(chain, "cilium_"):
		return "k8s"
	case chain == "PREROUTING" || chain == "OUTPUT" || chain == "POSTROUTING" || chain == "INPUT":
		return "manual"
	default:
		return "custom"
	}
}

// natExternalSourceReason 管理链只读原因。
func natExternalSourceReason(source string) string {
	switch source {
	case "docker":
		return "Docker 容器端口映射，由 dockerd 管理（摘除后会被加回）"
	case "firewalld":
		return "由 firewalld 管理（reload 会还原）"
	case "ufw":
		return "由 ufw 管理"
	case "libvirt":
		return "由 libvirt 管理"
	case "k8s":
		return "由 K8s/CNI 管理"
	default:
		return "位于自定义链，不接管"
	}
}

// natParseMatchPort --dport 规格："p" 或 "a:b"（冒号范围），返回 [起, 止]（止 0 = 单端口）。
func natParseMatchPort(s string) ([2]int, bool) {
	var res [2]int
	if a, b, ok := strings.Cut(s, ":"); ok {
		x, e1 := strconv.Atoi(a)
		y, e2 := strconv.Atoi(b)
		if e1 != nil || e2 != nil || x < 1 || y < x || y > 65535 || y-x+1 > natMaxRangeSize {
			return res, false
		}
		res[0], res[1] = x, y
		return res, true
	}
	x, err := strconv.Atoi(s)
	if err != nil || x < 1 || x > 65535 {
		return res, false
	}
	res[0] = x
	return res, true
}

// natParseDestMatch -d 规格：裸 IP 或 ip/32（归一为裸 IP）；网段不接受（返回 false）。
func natParseDestMatch(s string) (string, bool) {
	if s == "" {
		return "", true
	}
	addr, mask, hasMask := strings.Cut(s, "/")
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", false
	}
	if hasMask {
		if mask != "32" && mask != "128" {
			return "", false
		}
	}
	return ip.String(), true
}

// natParseToDestination --to-destination 规格：v4 "ip:p" / "ip:a-b"；v6 "[ip]:p" / "[ip]:a-b"。
// 返回（IP 已剥方括号, [起, 止]）。
func natParseToDestination(s string) (string, [2]int, bool) {
	var res [2]int
	if s == "" {
		return "", res, false
	}
	addrPart, portPart := "", ""
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", res, false
		}
		addrPart = s[1:end]
		rest := s[end+1:]
		if !strings.HasPrefix(rest, ":") {
			return "", res, false
		}
		portPart = rest[1:]
	} else {
		i := strings.LastIndex(s, ":")
		if i < 0 {
			return "", res, false
		}
		addrPart, portPart = s[:i], s[i+1:]
	}
	ps, ok := natParseToDestPorts(portPart)
	if !ok {
		return "", res, false
	}
	res = ps
	return addrPart, res, true
}

// natParseToDestPorts --to-destination 端口段："p" 或 "a-b"（短横线，iptables 语法差异）。
func natParseToDestPorts(s string) ([2]int, bool) {
	var res [2]int
	if a, b, ok := strings.Cut(s, "-"); ok {
		x, e1 := strconv.Atoi(a)
		y, e2 := strconv.Atoi(b)
		if e1 != nil || e2 != nil || x < 1 || y < x || y > 65535 || y-x+1 > natMaxRangeSize {
			return res, false
		}
		res[0], res[1] = x, y
		return res, true
	}
	x, err := strconv.Atoi(s)
	if err != nil || x < 1 || x > 65535 {
		return res, false
	}
	res[0] = x
	return res, true
}

// NatImportItem 导入项：外部规则 ID + 可选名称覆盖。
type NatImportItem struct {
	RuleID string `json:"ruleId"`
	Name   string `json:"name"`
}

// Import 接管导入：转面板记录 + apply 摘除原规则。服务层持锁串行化；
// 任一条失败整体失败、不建记录（TOCTOU：以导入时刻重新探测的结果为准）。
func (s *NatForwardService) Import(ctx context.Context, nodeId string, items []NatImportItem) ([]model.NatForwardRule, []string, error) {
	if len(items) == 0 {
		return nil, nil, natBadReq("未选择要导入的规则")
	}
	if _, err := s.nodes.ByID(nodeId); err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	available, ext, persist, err := s.probeExternal(ctx, nodeId)
	if err != nil {
		return nil, nil, err
	}
	if !available {
		return nil, nil, natBadReq("目标机 iptables 不可用，无法导入")
	}
	byID := make(map[string]NatExternalRule, len(ext))
	for _, er := range ext {
		byID[er.ID] = er
	}

	created := make([]*model.NatForwardRule, 0, len(items))
	for _, it := range items {
		er, ok := byID[strings.TrimSpace(it.RuleID)]
		if !ok {
			return nil, nil, natBadReq("所选规则已不存在或已变更，请刷新后重试")
		}
		if !er.Importable {
			return nil, nil, natBadReq("规则不可导入: "+er.Reason)
		}
		name := strings.TrimSpace(it.Name)
		if name == "" {
			name = er.Comment
		}
		fallback := "接管-" + natDportSpec([2]int{er.DportStart, er.DportEnd})
		if name == "" || len(name) > 64 {
			name = fallback
		}
		rec := &model.NatForwardRule{
			NodeID: nodeId, Name: name, Protocol: er.Proto, IPFamily: er.Family,
			ListenPort: er.DportStart, ListenPortEnd: er.DportEnd,
			DestIP: er.DestIP, TargetIP: er.ToIP,
			TargetPort: er.ToPortStart, TargetPortEnd: er.ToPortEnd,
			Iface: er.Iface, Enabled: true,
			SrcSpec: er.Chain + "|" + er.Spec,
		}
		if err := natValidateRule(rec); err != nil {
			return nil, nil, err
		}
		created = append(created, rec)
	}

	// 占用检测（DNAT 非监听不占 ss，冲突面与新建一致）
	for _, rec := range created {
		occupied, err := s.CheckPort(ctx, nodeId, rec.Protocol, rec.ListenPort, rec.ListenPortEnd)
		if err != nil {
			return nil, nil, err
		}
		if len(occupied) > 0 {
			return nil, nil, natPortInUseErr(occupied)
		}
	}

	warnings := natPersistWarnings(persist, created)

	for _, rec := range created {
		if err := s.db.Create(rec).Error; err != nil {
			for _, prev := range created {
				if prev.ID != 0 {
					_ = s.db.Delete(prev).Error
				}
			}
			return nil, nil, err
		}
	}

	warn2, err := s.applyNodeLocked(ctx, nodeId)
	if err != nil {
		// 回滚：还原原规则（尽力）→ 删记录 → 重建面板链
		restoreErr := s.restoreImported(context.Background(), nodeId, created)
		for _, rec := range created {
			if e := s.db.Delete(rec).Error; e != nil {
				slog.Error("NAT 导入回滚删记录失败", "id", rec.ID, "err", e.Error())
			}
		}
		if _, e := s.applyNodeLocked(context.Background(), nodeId); e != nil {
			slog.Warn("NAT 导入回滚重建链失败", "node", nodeId, "err", e.Error())
		}
		msg := "导入已回滚（应用失败）: " + err.Error()
		if restoreErr != nil {
			msg += "；原规则还原失败: " + restoreErr.Error()
		}
		return nil, nil, natBadReq(msg)
	}
	warnings = append(warnings, warn2...)

	rows := make([]model.NatForwardRule, 0, len(created))
	for _, rec := range created {
		rows = append(rows, *rec)
	}
	return rows, warnings, nil
}

// restoreImported 导入回滚：逐地址族还原接管的原内核规则（尽力而为，失败仅返回首个错误）。
func (s *NatForwardService) restoreImported(ctx context.Context, nodeId string, created []*model.NatForwardRule) error {
	var firstErr error
	for _, fb := range []struct {
		family int
		bin    string
	}{{4, "iptables"}, {6, "ip6tables"}} {
		need := false
		for _, r := range created {
			if r.IPFamily == fb.family && r.SrcSpec != "" {
				need = true
				break
			}
		}
		if !need {
			continue
		}
		out, code, err := s.exec(ctx, nodeId, buildNatRestoreScript(fb.bin, fb.family, created), 60)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if code != 0 {
			firstErr = errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("%s 原规则还原脚本失败: %s", fb.bin, strings.TrimSpace(out)))
		}
	}
	return firstErr
}

// natPersistWarnings 持久化文件（netfilter-persistent / iptables-services）含疑似同端口
// DNAT 的告警：重启后原规则会随持久化恢复（面板启动重放会再次摘除），不代改文件。
func natPersistWarnings(persistText string, created []*model.NatForwardRule) []string {
	if persistText == "" || len(created) == 0 {
		return nil
	}
	ws := []string{}
	seen := map[string]bool{}
	var curFile string
	for _, line := range strings.Split(persistText, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "---") {
			curFile = strings.TrimPrefix(line, "---")
			continue
		}
		if !strings.HasPrefix(line, "-A ") || !strings.Contains(line, "DNAT") {
			continue
		}
		for _, rec := range created {
			if seen[rec.Name] {
				continue
			}
			if strings.Contains(line, "--dport "+natDportSpec([2]int{rec.ListenPort, rec.ListenPortEnd})) {
				ws = append(ws, fmt.Sprintf("持久化文件 %s 含同端口 DNAT 规则（%s），重启后会随持久化恢复；面板启动重放会再次摘除，彻底移除请编辑该文件", curFile, rec.Name))
				seen[rec.Name] = true
				break
			}
		}
	}
	return ws
}
