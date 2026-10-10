// 外部 NAT 规则探测/解析/接管单测（M58）。
package service

import (
	"strings"
	"testing"

	"github.com/ypanel/core/internal/model"
)

// natExtByID 按 ID 索引解析结果。
func natExtByID(rules []NatExternalRule) map[string]NatExternalRule {
	m := map[string]NatExternalRule{}
	for _, r := range rules {
		m[r.ID] = r
	}
	return m
}

// 探测输出全量解析：分类、可导入判定、YPANEL 排除、v6、垃圾行容忍。
func TestParseNatExternal(t *testing.T) {
	out := strings.Join([]string{
		"===V4===",
		"-P PREROUTING ACCEPT",
		"-P OUTPUT ACCEPT",
		"-N DOCKER",
		"-N ufw-user-input",
		"-N MYCHAIN",
		// 手工裸 DNAT（无 -d）
		`-A PREROUTING -p tcp -m tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22`,
		// 手工 DNAT：-d/32 + 网卡 + 注释
		`-A PREROUTING -d 192.168.100.142/32 -i eth0 -p tcp -m tcp --dport 8080 -m comment --comment "web to nginx" -j DNAT --to-destination 10.0.0.5:80`,
		// 主链 MASQUERADE：manual 但非 DNAT，只读
		`-A POSTROUTING -d 10.0.0.5/32 -p tcp -m tcp --dport 80 -j MASQUERADE`,
		// docker 链
		`-A DOCKER ! -i docker0 -p tcp -m tcp --dport 80 -j DNAT --to-destination 172.17.0.2:80`,
		// ufw 链
		`-A ufw-user-input -p tcp --dport 22 -j ACCEPT`,
		// firewalld 链
		`-A FWDI_eth0_allow -p tcp -m tcp --dport 8081 -j DNAT --to-destination 10.0.0.6:8081`,
		// 自定义链
		`-A MYCHAIN -p tcp --dport 9000 -j DNAT --to-destination 10.0.0.7:9000`,
		// multiport：不支持
		`-A PREROUTING -p tcp -m multiport --dports 8001,8002 -j DNAT --to-destination 10.0.0.8:8001`,
		// 源匹配：不支持
		`-A PREROUTING -s 192.168.1.0/24 -p tcp -m tcp --dport 8003 -j DNAT --to-destination 10.0.0.8:8003`,
		// 注释含特殊字符：白名单拦截
		`-A PREROUTING -p tcp -m tcp --dport 8004 -m comment --comment "a;b$(rm)" -j DNAT --to-destination 10.0.0.8:8004`,
		// 面板自建链：排除
		`-A YPANEL_FWD -p tcp -m tcp --dport 8090 -m comment --comment ypanel-fwd-1 -j DNAT --to-destination 10.0.0.9:80`,
		"-A YPANEL_FWD_POST -p tcp -d 10.0.0.9 --dport 80 -j MASQUERADE",
		// 面板挂在主链上的跳转：排除
		"-A PREROUTING -j YPANEL_FWD",
		"-A OUTPUT -j YPANEL_FWD",
		"-A POSTROUTING -j YPANEL_FWD_POST",
		// 网段 -d：不支持
		`-A PREROUTING -d 192.168.1.0/24 -p tcp -m tcp --dport 8005 -j DNAT --to-destination 10.0.0.8:8005`,
		"===V6===",
		"-P PREROUTING ACCEPT",
		`-A PREROUTING -p tcp -m tcp --dport 18443 -j DNAT --to-destination [fd00::5]:443`,
		"===PERSIST===",
		"---/etc/iptables/rules.v4",
		`-A PREROUTING -p tcp -m tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22`,
		"YPOK",
	}, "\n")

	available, rules := parseNatExternal(out)
	if !available {
		t.Fatal("v4 有输出应 available=true")
	}
	byID := natExtByID(rules)
	// YPANEL 两条被排除
	if len(rules) != 12 {
		t.Fatalf("应解析出 12 条（排除 YPANEL 自建链），实得 %d: %+v", len(rules), rules)
	}

	// 手工裸 DNAT：可导入
	r1, ok := byID[natExtRuleID("PREROUTING", "-p tcp -m tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22")]
	if !ok || !r1.Importable || r1.Source != "manual" || r1.DportStart != 18099 || r1.ToIP != "127.0.0.1" || r1.ToPortStart != 22 {
		t.Fatalf("裸 DNAT 解析错误: %+v", r1)
	}
	// -d/32 + 网卡 + 带引号注释：可导入（引号注释经 natShellSpec 安全化），归一为裸 IP
	r2 := byID[natExtRuleID("PREROUTING", `-d 192.168.100.142/32 -i eth0 -p tcp -m tcp --dport 8080 -m comment --comment "web to nginx" -j DNAT --to-destination 10.0.0.5:80`)]
	if !r2.Importable || r2.DestIP != "192.168.100.142" || r2.Iface != "eth0" || r2.Comment != "web to nginx" || r2.DportStart != 8080 || r2.ToIP != "10.0.0.5" {
		t.Fatalf("带 -d DNAT 解析错误: %+v", r2)
	}
	if ss, ok := natShellSpec(r2.Spec); !ok || !strings.Contains(ss, `--comment 'web to nginx'`) {
		t.Fatalf("引号注释应安全化为单引号形式: %q %v", ss, ok)
	}
	// 主链 MASQUERADE：manual 但只读
	r3 := byID[natExtRuleID("POSTROUTING", `-d 10.0.0.5/32 -p tcp -m tcp --dport 80 -j MASQUERADE`)]
	if r3.Importable || !strings.Contains(r3.Reason, "主链") {
		t.Fatalf("POSTROUTING MASQUERADE 应不可导入: %+v", r3)
	}
	// docker 链只读
	r4 := byID[natExtRuleID("DOCKER", `! -i docker0 -p tcp -m tcp --dport 80 -j DNAT --to-destination 172.17.0.2:80`)]
	if r4.Importable || r4.Source != "docker" {
		t.Fatalf("docker 链应只读: %+v", r4)
	}
	// ufw / firewalld / 自定义链
	if r := byID[natExtRuleID("ufw-user-input", `-p tcp --dport 22 -j ACCEPT`)]; r.Importable || r.Source != "ufw" {
		t.Fatalf("ufw 链应只读: %+v", r)
	}
	if r := byID[natExtRuleID("FWDI_eth0_allow", `-p tcp -m tcp --dport 8081 -j DNAT --to-destination 10.0.0.6:8081`)]; r.Importable || r.Source != "firewalld" {
		t.Fatalf("firewalld 链应只读: %+v", r)
	}
	if r := byID[natExtRuleID("MYCHAIN", `-p tcp --dport 9000 -j DNAT --to-destination 10.0.0.7:9000`)]; r.Importable || r.Source != "custom" {
		t.Fatalf("自定义链应只读: %+v", r)
	}
	// multiport / -s / 特殊字符 / 网段 -d：解析级拦截
	if r := byID[natExtRuleID("PREROUTING", `-p tcp -m multiport --dports 8001,8002 -j DNAT --to-destination 10.0.0.8:8001`)]; r.Importable || !strings.Contains(r.Reason, "multiport") {
		t.Fatalf("multiport 应不可导入: %+v", r)
	}
	if r := byID[natExtRuleID("PREROUTING", `-s 192.168.1.0/24 -p tcp -m tcp --dport 8003 -j DNAT --to-destination 10.0.0.8:8003`)]; r.Importable || !strings.Contains(r.Reason, "-s") {
		t.Fatalf("源匹配应不可导入: %+v", r)
	}
	if r := byID[natExtRuleID("PREROUTING", `-p tcp -m tcp --dport 8004 -m comment --comment "a;b$(rm)" -j DNAT --to-destination 10.0.0.8:8004`)]; r.Importable || !strings.Contains(r.Reason, "特殊字符") {
		t.Fatalf("特殊字符应不可导入: %+v", r)
	}
	if r := byID[natExtRuleID("PREROUTING", `-d 192.168.1.0/24 -p tcp -m tcp --dport 8005 -j DNAT --to-destination 10.0.0.8:8005`)]; r.Importable || !strings.Contains(r.Reason, "网段") {
		t.Fatalf("网段 -d 应不可导入: %+v", r)
	}
	// v6 DNAT 可导入，方括号已剥
	r6 := byID[natExtRuleID("PREROUTING", `-p tcp -m tcp --dport 18443 -j DNAT --to-destination [fd00::5]:443`)]
	if !r6.Importable || r6.Family != 6 || r6.ToIP != "fd00::5" || r6.ToPortStart != 443 {
		t.Fatalf("v6 DNAT 解析错误: %+v", r6)
	}
	// ID 稳定性：同 spec 重复解析 ID 一致
	if _, rules2 := parseNatExternal(out); natExtByID(rules2)[r1.ID].ID != r1.ID {
		t.Fatal("ID 应稳定")
	}
}

// iptables 缺失：available=false、零规则。
func TestParseNatExternalUnavailable(t *testing.T) {
	available, rules := parseNatExternal(strings.Join([]string{
		"===V4===",
		"===V6===",
		"===PERSIST===",
		"YPOK",
	}, "\n"))
	if available || len(rules) != 0 {
		t.Fatalf("空输出应 available=false 零规则: %v %d", available, len(rules))
	}
}

// 垃圾行（告警/错误文本）不影响解析。
func TestParseNatExternalJunkLines(t *testing.T) {
	out := strings.Join([]string{
		"===V4===",
		"iptables v1.8.7 (nf_tables): Could not fetch rule set generation id: Permission denied",
		"-P PREROUTING ACCEPT",
		"random text line",
		"===V6===",
		"YPOK",
	}, "\n")
	available, rules := parseNatExternal(out)
	if !available || len(rules) != 0 {
		t.Fatalf("告警行应被跳过: %v %d", available, len(rules))
	}
}

func natExtRuleID(chain, spec string) string {
	r, ok := parseNatExternalRule(4, chain+" "+spec)
	if !ok {
		panic("parse failed")
	}
	return r.ID
}

func TestNatParseMatchPort(t *testing.T) {
	if v, ok := natParseMatchPort("8080"); !ok || v != [2]int{8080, 0} {
		t.Fatalf("单端口: %v %v", v, ok)
	}
	if v, ok := natParseMatchPort("8000:8007"); !ok || v != [2]int{8000, 8007} {
		t.Fatalf("范围: %v %v", v, ok)
	}
	for _, bad := range []string{"", "0", "65536", "abc", "8007:8000", "1:99999"} {
		if _, ok := natParseMatchPort(bad); ok {
			t.Fatalf("%q 应不合法", bad)
		}
	}
}

func TestNatParseDestMatch(t *testing.T) {
	if v, ok := natParseDestMatch("10.0.0.5"); !ok || v != "10.0.0.5" {
		t.Fatalf("裸 IP: %v %v", v, ok)
	}
	if v, ok := natParseDestMatch("10.0.0.5/32"); !ok || v != "10.0.0.5" {
		t.Fatalf("/32 应归一: %v %v", v, ok)
	}
	if v, ok := natParseDestMatch("fd00::5/128"); !ok || v != "fd00::5" {
		t.Fatalf("v6 /128 应归一: %v %v", v, ok)
	}
	if v, ok := natParseDestMatch(""); !ok || v != "" {
		t.Fatalf("空应合法: %v %v", v, ok)
	}
	for _, bad := range []string{"10.0.0.0/24", "not-ip", "10.0.0.5/33"} {
		if _, ok := natParseDestMatch(bad); ok {
			t.Fatalf("%q 应不合法", bad)
		}
	}
}

func TestNatParseToDestination(t *testing.T) {
	if ip, ps, ok := natParseToDestination("10.0.0.5:80"); !ok || ip != "10.0.0.5" || ps != [2]int{80, 0} {
		t.Fatalf("v4 单端口: %v %v %v", ip, ps, ok)
	}
	if ip, ps, ok := natParseToDestination("10.0.0.5:9000-9007"); !ok || ip != "10.0.0.5" || ps != [2]int{9000, 9007} {
		t.Fatalf("v4 范围: %v %v %v", ip, ps, ok)
	}
	if ip, ps, ok := natParseToDestination("[fd00::5]:443"); !ok || ip != "fd00::5" || ps != [2]int{443, 0} {
		t.Fatalf("v6: %v %v %v", ip, ps, ok)
	}
	for _, bad := range []string{"", "10.0.0.5", "10.0.0.5:", "10.0.0.5:abc", "[fd00::5]:0", "10.0.0.5:80-"} {
		if _, _, ok := natParseToDestination(bad); ok {
			t.Fatalf("%q 应不合法", bad)
		}
	}
}

func TestNatSplitSrcSpec(t *testing.T) {
	if ch, sp, ok := natSplitSrcSpec("PREROUTING|-p tcp --dport 80 -j DNAT"); !ok || ch != "PREROUTING" || sp != "-p tcp --dport 80 -j DNAT" {
		t.Fatalf("合法拆分: %v %v %v", ch, sp, ok)
	}
	for _, bad := range []string{"", "|spec", "CHAIN|", "BAD CHAIN|-p tcp"} {
		if _, _, ok := natSplitSrcSpec(bad); ok {
			t.Fatalf("%q 应不合法", bad)
		}
	}
	// spec 文本安全化由 natShellSpec 负责（结构拆分不做字符白名单）
	if _, _, ok := natSplitSrcSpec(`PREROUTING|-m comment --comment "x" -j DNAT`); !ok {
		t.Fatal("带引号 spec 结构拆分应通过（安全化在 natShellSpec）")
	}
}

func TestNatShellSpec(t *testing.T) {
	// 引号注释 → 单引号重写
	if s, ok := natShellSpec(`-p tcp -m comment --comment "web fwd" --dport 80 -j DNAT`); !ok || s != `-p tcp -m comment --comment 'web fwd' --dport 80 -j DNAT` {
		t.Fatalf("注释重写错误: %q %v", s, ok)
	}
	// 单引号注释同样处理
	if s, ok := natShellSpec(`-m comment --comment 'a b' -j DNAT`); !ok || s != `-m comment --comment 'a b' -j DNAT` {
		t.Fatalf("单引号注释: %q %v", s, ok)
	}
	// 裸单词注释统一重写为单引号形式（语义等价，规避转义分支）
	if s, ok := natShellSpec(`-m comment --comment plain -j DNAT`); !ok || s != `-m comment --comment 'plain' -j DNAT` {
		t.Fatalf("裸注释: %q %v", s, ok)
	}
	// 注释内容含 shell 元字符 → 拒绝
	for _, bad := range []string{
		`-m comment --comment "a;b" -j DNAT`,
		"-m comment --comment \"a$(rm)\" -j DNAT",
		`-m comment --comment "a'b" -j DNAT`,
		"-m comment --comment \"a`b\" -j DNAT",
	} {
		if _, ok := natShellSpec(bad); ok {
			t.Fatalf("%q 应不安全", bad)
		}
	}
	// 非注释部分含特殊字符 → 拒绝
	if _, ok := natShellSpec(`-p tcp --dport 80 -j DNAT; rm`); ok {
		t.Fatal("分号应拒绝")
	}
	if _, ok := natShellSpec(`-j DNAT "stray`); ok {
		t.Fatal("游离引号应拒绝")
	}
}

// 摘除段：-C 守卫 + 幂等 -D + 上限 5 次；规则 ID 定位错误。
func TestBuildNatRemovalSection(t *testing.T) {
	s := buildNatRemovalSection([]natRemovalSpec{
		{Chain: "PREROUTING", Spec: "-p tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22", RuleID: 11},
		{Chain: "OUTPUT", Spec: "-p udp --dport 5353 -j DNAT --to-destination 10.0.0.1:53", RuleID: 12},
	})
	for _, want := range []string{
		`_n=0; while [ $_n -lt 5 ] && $IPT -t nat -C PREROUTING -p tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22 2>/dev/null; do $IPT -t nat -D PREROUTING -p tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22 || { echo 'YPERR:规则 #11 原规则摘除失败'; exit 67; }; _n=$((_n+1)); done`,
		"$IPT -t nat -C OUTPUT -p udp --dport 5353",
		"YPERR:规则 #12 原规则摘除失败",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("摘除段缺少 %q:\n%s", want, s)
		}
	}
	if buildNatRemovalSection(nil) != "" {
		t.Fatal("空摘除清单应生成空段")
	}
}

// 还原脚本：! -C 守卫 + -A 补回 + 族过滤 + 非法 SrcSpec 跳过。
func TestBuildNatRestoreScript(t *testing.T) {
	rules := []*model.NatForwardRule{
		{ID: 1, IPFamily: 4, SrcSpec: "PREROUTING|-p tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22"},
		{ID: 2, IPFamily: 6, SrcSpec: "PREROUTING|-p tcp --dport 18443 -j DNAT --to-destination [fd00::5]:443"},
		{ID: 3, IPFamily: 4, SrcSpec: ""},
		{ID: 4, IPFamily: 4, SrcSpec: "PREROUTING|bad;injection"},
	}
	s4 := buildNatRestoreScript("iptables", 4, rules)
	if !strings.Contains(s4, "if ! $IPT -t nat -C PREROUTING -p tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22 2>/dev/null; then if ! _o=$($IPT -t nat -A PREROUTING -p tcp --dport 18099") {
		t.Fatalf("v4 还原段错误:\n%s", s4)
	}
	if strings.Contains(s4, "18443") {
		t.Fatal("v4 还原脚本不应含 v6 规则")
	}
	if strings.Contains(s4, "injection") {
		t.Fatal("非法 SrcSpec 应被跳过")
	}
	s6 := buildNatRestoreScript("ip6tables", 6, rules)
	if !strings.Contains(s6, "18443") || strings.Contains(s6, "18099") {
		t.Fatalf("v6 还原段过滤错误:\n%s", s6)
	}
}

// 渲染脚本集成：-d 渲染 + 摘除段 + MASQUERADE 不带 -d 匹配。
func TestBuildNatApplyScriptWithDestAndRemoval(t *testing.T) {
	v := view(7, "tcp", "", 8080, 0, "10.0.0.5", 80, 0)
	v.DestIP = "192.168.1.10"
	script := buildNatApplyScript("iptables", 4, []natRenderView{v}, nil, []natRemovalSpec{
		{Chain: "PREROUTING", Spec: "-p tcp --dport 8080 -j DNAT --to-destination 10.0.0.5:80", RuleID: 7},
	})
	if !strings.Contains(script, "-A YPANEL_FWD -p tcp -d '192.168.1.10' --dport 8080") {
		t.Fatalf("应渲染 -d 匹配:\n%s", script)
	}
	if !strings.Contains(script, "-A YPANEL_FWD_POST -p tcp -d '10.0.0.5' --dport 80") {
		t.Fatalf("MASQUERADE 行不应带接管 -d 匹配:\n%s", script)
	}
	if !strings.Contains(script, "原规则摘除失败") {
		t.Fatal("应用脚本应含摘除段")
	}
}

// 持久化告警：同端口 DNAT 命中、MASQUERADE 行与非 DNAT 行忽略、去重。
func TestNatPersistWarnings(t *testing.T) {
	persist := strings.Join([]string{
		"---/etc/iptables/rules.v4",
		`-A PREROUTING -p tcp -m tcp --dport 18099 -j DNAT --to-destination 127.0.0.1:22`,
		`-A POSTROUTING -s 172.17.0.0/16 -j MASQUERADE`,
		"-A PREROUTING -p tcp -m tcp --dport 1 -j DNAT --to-destination 10.0.0.1:1",
	}, "\n")
	created := []*model.NatForwardRule{
		{Name: "r1", ListenPort: 18099, ListenPortEnd: 0},
	}
	ws := natPersistWarnings(persist, created)
	if len(ws) != 1 || !strings.Contains(ws[0], "rules.v4") || !strings.Contains(ws[0], "r1") {
		t.Fatalf("持久化告警错误: %+v", ws)
	}
	// 无持久化段
	if ws := natPersistWarnings("", created); ws != nil {
		t.Fatalf("空持久化段应无告警: %+v", ws)
	}
	// 无命中
	if ws := natPersistWarnings(persist, []*model.NatForwardRule{{Name: "r2", ListenPort: 9999}}); len(ws) != 0 {
		t.Fatalf("不应命中: %+v", ws)
	}
}
