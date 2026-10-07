package service

import (
	"strings"
	"testing"

	"github.com/ypanel/core/internal/model"
)

func view(id uint, proto, iface string, lp, lpe int, ip string, tp, tpe int) natRenderView {
	return natRenderView{ID: id, Protocol: proto, Iface: iface, Listen: [2]int{lp, lpe}, TargetIP: ip, Target: [2]int{tp, tpe}}
}

// 单端口 v4：dport 冒号、to-destination 冒号、注释、链挂载。
func TestBuildNatApplyScriptSingleV4(t *testing.T) {
	script := buildNatApplyScript("iptables", 4, []natRenderView{view(7, "tcp", "", 8080, 0, "10.0.0.5", 80, 0)}, []string{"192.168.1.10"})
	for _, want := range []string{
		"-N YPANEL_FWD",
		"-C PREROUTING -j YPANEL_FWD 2>/dev/null || $IPT -t nat -I PREROUTING 1 -j YPANEL_FWD",
		"-C OUTPUT -j YPANEL_FWD",
		"-C POSTROUTING -j YPANEL_FWD_POST",
		"-F YPANEL_FWD",
		"-A YPANEL_FWD -p tcp --dport 8080 -m comment --comment 'ypanel-fwd-7' -j DNAT --to-destination '10.0.0.5:80'",
		"command -v iptables",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("脚本缺少片段 %q\n脚本:\n%s", want, script)
		}
	}
	// 10.0.0.5 不在本机地址集 → 需要 MASQUERADE
	if !strings.Contains(script, "-A YPANEL_FWD_POST -p tcp -d '10.0.0.5' --dport 80 -m comment --comment 'ypanel-fwd-7' -j MASQUERADE") {
		t.Fatalf("非本机目标缺少 MASQUERADE:\n%s", script)
	}
	// v4 sysctl
	if !strings.Contains(script, "net.ipv4.ip_forward") {
		t.Fatal("缺少 v4 转发开关检测")
	}
}

// 本机目标（回环 / 本机地址）不追加 MASQUERADE。
func TestBuildNatApplyScriptNoMasqForLocal(t *testing.T) {
	loopback := buildNatApplyScript("iptables", 4, []natRenderView{view(1, "tcp", "", 18022, 0, "127.0.0.1", 22, 0)}, nil)
	if strings.Contains(loopback, "MASQUERADE") {
		t.Fatalf("回环目标不应有 MASQUERADE:\n%s", loopback)
	}
	local := buildNatApplyScript("iptables", 4, []natRenderView{view(2, "tcp", "", 18022, 0, "192.168.1.10", 22, 0)}, []string{"192.168.1.10", "10.8.0.1"})
	if strings.Contains(local, "MASQUERADE") {
		t.Fatalf("本机地址目标不应有 MASQUERADE:\n%s", local)
	}
}

// 范围→同尺寸范围：dport 冒号、to-destination 短横线（iptables 语法差异，真机踩坑点）。
func TestBuildNatApplyScriptRange(t *testing.T) {
	script := buildNatApplyScript("iptables", 4, []natRenderView{view(3, "udp", "eth0", 8000, 8007, "10.0.0.9", 9000, 9007)}, nil)
	if !strings.Contains(script, "--dport 8000:8007") {
		t.Fatalf("dport 范围应为冒号语法:\n%s", script)
	}
	if !strings.Contains(script, "--to-destination '10.0.0.9:9000-9007'") {
		t.Fatalf("to-destination 范围应为短横线语法:\n%s", script)
	}
	if !strings.Contains(script, "-i 'eth0'") {
		t.Fatal("入站网卡未附加")
	}
	if !strings.Contains(script, "--dport 9000:9007 -m comment --comment 'ypanel-fwd-3' -j MASQUERADE") {
		t.Fatalf("MASQUERADE 应用目标端口范围:\n%s", script)
	}
}

// v6：ip6tables + [addr]:port + v6 转发开关；v6 目标必须纯 v6 地址（无冒号判断由校验层保证）。
func TestBuildNatApplyScriptV6(t *testing.T) {
	script := buildNatApplyScript("ip6tables", 6, []natRenderView{view(9, "tcp", "", 8443, 0, "fd00::5", 443, 0)}, []string{"fd11::1"})
	if !strings.Contains(script, "command -v ip6tables") {
		t.Fatal("v6 应使用 ip6tables")
	}
	if !strings.Contains(script, "--to-destination '[fd00::5]:443'") {
		t.Fatalf("v6 目标应为 [addr]:port:\n%s", script)
	}
	if !strings.Contains(script, "net.ipv6.conf.all.forwarding") {
		t.Fatal("缺少 v6 转发开关检测")
	}
	if !strings.Contains(script, "-d 'fd00::5' --dport 443") {
		t.Fatalf("v6 MASQUERADE 缺失:\n%s", script)
	}
	if strings.Contains(script, "net.ipv4.ip_forward") {
		t.Fatal("v6 脚本不应检测 v4 开关")
	}
}

// 失败语义：脚本内单条规则失败要带 id 定位。
func TestBuildNatApplyScriptErrorMarker(t *testing.T) {
	script := buildNatApplyScript("iptables", 4, []natRenderView{view(5, "tcp", "", 80, 0, "10.0.0.1", 80, 0)}, nil)
	if !strings.Contains(script, "YPERR:规则 #5 应用失败") {
		t.Fatal("缺少规则级失败标记")
	}
}

func TestParseNatInterfaces(t *testing.T) {
	out := strings.Join([]string{
		`1: lo    inet 127.0.0.1/8 scope host lo\       valid_lft forever preferred_lft forever`,
		`1: lo    inet6 ::1/128 scope host \       valid_lft forever preferred_lft forever`,
		`2: eth0    inet 192.168.100.142/24 brd 192.168.100.255 scope global eth0\       valid_lft forever`,
		`2: eth0    inet6 fe80::20c:29ff:fe11:2233/64 scope link \       valid_lft forever`,
		`3: docker0    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0\       valid_lft forever`,
		`4: br-abc123    inet 172.18.0.1/16 brd 172.18.255.255 scope global br-abc123`,
		`5: veth9f3a@if6:    inet 169.254.9.34/16 scope link veth9f3a`,
		"junk-line-without-colon",
	}, "\n")
	got := parseNatInterfaces(out)
	if len(got) != 7 {
		t.Fatalf("应解析出 7 条，实得 %d: %+v", len(got), got)
	}
	byAddr := map[string]NatInterface{}
	for _, it := range got {
		byAddr[it.Addr] = it
	}
	if it := byAddr["127.0.0.1"]; !it.Internal || it.Name != "lo" || it.Family != 4 {
		t.Fatalf("lo 判定错误: %+v", it)
	}
	if it := byAddr["::1"]; !it.Internal || it.Family != 6 {
		t.Fatalf("::1 判定错误: %+v", it)
	}
	if it := byAddr["192.168.100.142"]; it.Internal || it.Name != "eth0" {
		t.Fatalf("eth0 应为外部网卡: %+v", it)
	}
	if it := byAddr["172.17.0.1"]; !it.Internal {
		t.Fatalf("docker0 应为 internal: %+v", it)
	}
	if it := byAddr["172.18.0.1"]; !it.Internal {
		t.Fatalf("br-* 应为 internal: %+v", it)
	}
	if it := byAddr["fe80::20c:29ff:fe11:2233"]; it.Internal {
		t.Fatalf("eth0 v6 应为外部: %+v", it)
	}
}

func TestParseNatListeningPorts(t *testing.T) {
	out := strings.Join([]string{
		"===TCP===",
		"LISTEN 0      128          0.0.0.0:22         0.0.0.0:*    users:((\"sshd\",pid=800,fd=3))",
		"LISTEN 0      511        127.0.0.1:8880         0.0.0.0:*    users:((\"ypanel\",pid=1200,fd=8))",
		"LISTEN 0      4096               [::]:9100            [::]:*",
		"===UDP===",
		"UNCONN 0      0            0.0.0.0:53          0.0.0.0:*    users:((\"dnsmasq\",pid=900,fd=5))",
		"",
	}, "\n")
	got := parseNatListeningPorts(out)
	if len(got[22]) != 1 || got[22][0].Proto != "tcp" || !strings.Contains(got[22][0].Process, "sshd") {
		t.Fatalf("22/tcp 解析错误: %+v", got[22])
	}
	if len(got[8880]) != 1 || !strings.Contains(got[8880][0].Process, "ypanel") {
		t.Fatalf("8880 解析错误: %+v", got[8880])
	}
	if len(got[9100]) != 1 || got[9100][0].Process != "" {
		t.Fatalf("无权限进程信息应容忍为空: %+v", got[9100])
	}
	if len(got[53]) != 1 || got[53][0].Proto != "udp" || !strings.Contains(got[53][0].Process, "dnsmasq") {
		t.Fatalf("53/udp 解析错误: %+v", got[53])
	}
}

func natRule(id uint, proto string, family int, lp, lpe, tp, tpe int, ip, iface string) *model.NatForwardRule {
	return &model.NatForwardRule{
		ID: id, NodeID: "local", Name: "r", Protocol: proto, IPFamily: family,
		ListenPort: lp, ListenPortEnd: lpe, TargetIP: ip, TargetPort: tp, TargetPortEnd: tpe, Iface: iface,
	}
}

func TestNatValidateRule(t *testing.T) {
	cases := []struct {
		name    string
		rule    *model.NatForwardRule
		wantErr string
	}{
		{"合法单端口", natRule(0, "tcp", 4, 8080, 0, 80, 0, "10.0.0.5", ""), ""},
		{"合法同尺寸范围", natRule(0, "udp", 4, 8000, 8007, 9000, 9007, "10.0.0.5", "eth0"), ""},
		{"名称为空", func() *model.NatForwardRule {
			r := natRule(0, "tcp", 4, 8080, 0, 80, 0, "10.0.0.5", "")
			r.Name = ""
			return r
		}(), "名称"},
		{"协议非法", natRule(0, "sctp", 4, 8080, 0, 80, 0, "10.0.0.5", ""), "协议"},
		{"IP 族非法", natRule(0, "tcp", 5, 8080, 0, 80, 0, "10.0.0.5", ""), "IP 族"},
		{"监听端口越界", natRule(0, "tcp", 4, 0, 0, 80, 0, "10.0.0.5", ""), "端口"},
		{"目标端口越界", natRule(0, "tcp", 4, 8080, 0, 65536, 0, "10.0.0.5", ""), "端口"},
		{"范围倒置", natRule(0, "tcp", 4, 8100, 8000, 80, 0, "10.0.0.5", ""), "范围"},
		{"尺寸超限", natRule(0, "tcp", 4, 1000, 1000+1000, 1, 1, "10.0.0.5", ""), "尺寸"},
		{"单端口对范围", natRule(0, "tcp", 4, 8080, 0, 80, 90, "10.0.0.5", ""), "范围语义"},
		{"范围对单端口", natRule(0, "tcp", 4, 8080, 8090, 80, 0, "10.0.0.5", ""), "范围语义"},
		{"范围尺寸不等", natRule(0, "tcp", 4, 8080, 8089, 80, 85, "10.0.0.5", ""), "范围语义"},
		{"目标 IP 非法", natRule(0, "tcp", 4, 8080, 0, 80, 0, "not-an-ip", ""), "目标 IP"},
		{"v4 族给 v6 地址", natRule(0, "tcp", 4, 8080, 0, 80, 0, "fd00::5", ""), "不匹配"},
		{"v6 族给 v4 地址", natRule(0, "tcp", 6, 8080, 0, 80, 0, "10.0.0.5", ""), "不匹配"},
		{"v6 族给 v4 映射地址", natRule(0, "tcp", 6, 8080, 0, 80, 0, "::ffff:10.0.0.5", ""), "不匹配"},
		{"网卡名非法", natRule(0, "tcp", 4, 8080, 0, 80, 0, "10.0.0.5", "eth0; rm -rf /"), "网卡"},
	}
	for _, tc := range cases {
		err := natValidateRule(tc.rule)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("%s: 期望通过，实得 %v", tc.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: 期望错误含 %q，实得 %v", tc.name, tc.wantErr, err)
		}
	}
	// 规范化：目标 IP 归一（v4 落地为点分十进制）
	r := natRule(0, "tcp", 4, 8080, 0, 80, 0, " 10.0.0.5 ", "")
	if err := natValidateRule(r); err != nil || r.TargetIP != "10.0.0.5" {
		t.Errorf("目标 IP 应去空白并归一: %v %q", err, r.TargetIP)
	}
}

func TestNatPortRangeHelpers(t *testing.T) {
	if natRangeSize(8080, 0) != 1 || natRangeSize(8000, 8007) != 8 || natPortEnd(8000, 0) != 8000 {
		t.Fatal("端口范围辅助函数语义错误")
	}
}

// 渲染视图只取启用态且族匹配的规则（停用规则不得进内核链）。
func TestNatEnabledViews(t *testing.T) {
	rules := []model.NatForwardRule{
		{ID: 1, IPFamily: 4, Protocol: "tcp", Enabled: true, ListenPort: 8080, TargetIP: "10.0.0.1", TargetPort: 80},
		{ID: 2, IPFamily: 4, Protocol: "tcp", Enabled: false, ListenPort: 8081, TargetIP: "10.0.0.1", TargetPort: 80},
		{ID: 3, IPFamily: 6, Protocol: "tcp", Enabled: true, ListenPort: 8082, TargetIP: "fd00::1", TargetPort: 80},
	}
	v4 := natEnabledViews(rules, 4)
	if len(v4) != 1 || v4[0].ID != 1 {
		t.Fatalf("v4 启用视图应仅含 id=1: %+v", v4)
	}
	v6 := natEnabledViews(rules, 6)
	if len(v6) != 1 || v6[0].ID != 3 {
		t.Fatalf("v6 启用视图应仅含 id=3: %+v", v6)
	}
	// 全停用 → 空视图（上层应改走容忍式冲刷）
	all := natEnabledViews(rules[:2], 6)
	if len(all) != 0 {
		t.Fatalf("应无启用规则: %+v", all)
	}
}

// 容忍式冲刷脚本：命令缺失静默成功、链存在判定后才冲刷、无 -A 无 sysctl。
func TestBuildNatFlushScript(t *testing.T) {
	s := buildNatFlushScript("ip6tables")
	for _, want := range []string{
		"command -v ip6tables >/dev/null 2>&1 || exit 0",
		"ip6tables -t nat -L YPANEL_FWD -n >/dev/null 2>&1 && ip6tables -t nat -F YPANEL_FWD",
		"ip6tables -t nat -L YPANEL_FWD_POST -n >/dev/null 2>&1 && ip6tables -t nat -F YPANEL_FWD_POST",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("冲刷脚本缺少 %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "-A ") || strings.Contains(s, "sysctl") {
		t.Fatalf("冲刷脚本不应含 -A 或 sysctl:\n%s", s)
	}
}
