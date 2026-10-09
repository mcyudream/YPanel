package service

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"github.com/ypanel/core/internal/model"
)

func dnsRec(id uint, typ, domain, target string, enabled bool, sort int) model.DnsRecord {
	return model.DnsRecord{ID: id, Type: typ, Domain: domain, Target: target, Enabled: enabled, Sort: sort}
}

// 渲染：默认上游/缓存、listen-address+bind-interfaces、三类记录、停用过滤、sort 升序。
func TestRenderDnsmasqConf(t *testing.T) {
	cfg := DnsConfig{ListenIP: "192.168.100.2"}
	records := []model.DnsRecord{
		dnsRec(2, "cname", "mc.example.com", "example.com", true, 0),
		dnsRec(1, "address", "example.com", "192.168.100.2", true, 0),
		dnsRec(3, "txt", "_dmarc.example.com", "v=DMARC1; p=none", true, 5),
		dnsRec(4, "address", "disabled.example.com", "10.0.0.9", false, 0),
	}
	conf := renderDnsmasqConf(cfg, records, nil)
	for _, want := range []string{
		"no-resolv\nserver=223.5.5.5\nserver=119.29.29.29",
		"listen-address=192.168.100.2\nbind-interfaces\n",
		"cache-size=1000",
		"local-ttl=300",
		"address=/example.com/192.168.100.2",
		"cname=mc.example.com,example.com",
		"txt-record=_dmarc.example.com,v=DMARC1; p=none",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("conf 缺少片段 %q\nconf:\n%s", want, conf)
		}
	}
	if strings.Contains(conf, "disabled.example.com") {
		t.Fatal("停用记录不应被渲染")
	}
	// 同 sort 下按 id 升序：address(1) 在 cname(2) 前
	if strings.Index(conf, "address=/example.com/") > strings.Index(conf, "cname=mc.example.com") {
		t.Fatal("记录排序错误（期望 sort,id 升序）")
	}
	// 未设监听 IP 时不渲染绑定段
	conf2 := renderDnsmasqConf(DnsConfig{}, nil, nil)
	if strings.Contains(conf2, "listen-address") || strings.Contains(conf2, "bind-interfaces") {
		t.Fatal("未设监听 IP 时不应渲染绑定段")
	}
}

// 校验：address 归一化、cname/txt、域名形态（下划线允许、通配符拒绝、大小写归一）。
func TestDnsValidateRecord(t *testing.T) {
	r := dnsRec(0, "address", "Example.COM.", "FD00::2", true, 0)
	if err := dnsValidateRecord(&r); err != nil {
		t.Fatalf("合法 address 记录被拒: %v", err)
	}
	if r.Domain != "example.com" || r.Target != "fd00::2" {
		t.Fatalf("归一化失败: %s / %s", r.Domain, r.Target)
	}
	bad := []model.DnsRecord{
		dnsRec(0, "address", "example.com", "not-an-ip", true, 0),
		dnsRec(0, "cname", "a.example.com", "bad_host..com", true, 0),
		dnsRec(0, "txt", "example.com", "line1\nline2", true, 0),
		dnsRec(0, "mx", "example.com", "x", true, 0),
		dnsRec(0, "address", "*.example.com", "10.0.0.1", true, 0),
		dnsRec(0, "address", "", "10.0.0.1", true, 0),
		dnsRec(0, "address", strings.Repeat("a", 260), "10.0.0.1", true, 0),
	}
	for i := range bad {
		if err := dnsValidateRecord(&bad[i]); err == nil {
			t.Fatalf("非法记录 #%d 未被拦截: %+v", i, bad[i])
		}
	}
	// 下划线（_dmarc）与连字符合法
	ok := dnsRec(0, "txt", "_dmarc.example.com", "v=DMARC1", true, 0)
	if err := dnsValidateRecord(&ok); err != nil {
		t.Fatalf("下划线域名被误拒: %v", err)
	}
	ok2 := dnsRec(0, "cname", "mc-srv.example.com", "example.com", true, 0)
	if err := dnsValidateRecord(&ok2); err != nil {
		t.Fatalf("连字符主机名被误拒: %v", err)
	}
}

// 监听 IP 安全硬约束：拒绝 0.0.0.0 与公网地址，接受私网/回环。
func TestDnsValidateIPPrivate(t *testing.T) {
	for _, bad := range []string{"0.0.0.0", "::", "1.2.3.4", "8.8.8.8", "224.0.0.1", "abc"} {
		if err := dnsValidateIPPrivate(bad); err == nil {
			t.Fatalf("监听 IP %q 应被拒绝", bad)
		}
	}
	for _, good := range []string{"192.168.100.2", "10.0.0.5", "172.16.1.5", "fd00::2", "127.0.0.1"} {
		if err := dnsValidateIPPrivate(good); err != nil {
			t.Fatalf("监听 IP %q 应被接受: %v", good, err)
		}
	}
}

// compose 渲染：镜像、容器名、host 网络、只读挂载。
func TestRenderDnsCompose(t *testing.T) {
	yml := renderDnsCompose("dockurr/dnsmasq:latest")
	for _, want := range []string{
		"image: dockurr/dnsmasq:latest",
		"container_name: ypanel-dnsmasq",
		"network_mode: host",
		"restart: unless-stopped",
		"- ./dnsmasq.conf:/etc/dnsmasq.conf:ro",
	} {
		if !strings.Contains(yml, want) {
			t.Fatalf("compose 缺少片段 %q\nyml:\n%s", want, yml)
		}
	}
}

// overview 解析：容器态/镜像、conf 存在性、53 监听明细含绑定地址。
func TestParseDnsOverview(t *testing.T) {
	out := "===C===\nrunning dockurr/dnsmasq:latest\n" +
		"===D===\nyes\n" +
		"===TCP===\nLISTEN 0      32   192.168.100.2:53     0.0.0.0:*   users:((\"dnsmasq\",pid=9,fd=5))\nLISTEN 0      32   [::]:53              [::]:*     users:((\"dnsmasq\",pid=9,fd=6))\n" +
		"===UDP===\nUNCONN 0     0    192.168.100.2:53     0.0.0.0:*   users:((\"dnsmasq\",pid=9,fd=7))\n"
	ov := parseDnsOverview(out)
	if ov.Container != "running" || ov.Image != "dockurr/dnsmasq:latest" {
		t.Fatalf("容器解析错误: %+v", ov)
	}
	if !ov.DirReady {
		t.Fatal("conf 存在性解析错误")
	}
	if len(ov.Port53) != 3 {
		t.Fatalf("53 监听明细条数错误: %+v", ov.Port53)
	}
	if ov.Port53[0].Addr != "192.168.100.2" || ov.Port53[1].Addr != "::" || ov.Port53[0].Proto != "tcp" || ov.Port53[2].Proto != "udp" {
		t.Fatalf("地址/协议解析错误: %+v", ov.Port53)
	}
	if procShortName(ov.Port53[0].Process) != "dnsmasq" {
		t.Fatalf("进程名提取错误: %q", ov.Port53[0].Process)
	}
	// 空输出
	ov2 := parseDnsOverview("===C===\n===D===\nno\n===TCP===\n===UDP===\n")
	if ov2.Container != "missing" || ov2.DirReady || len(ov2.Port53) != 0 {
		t.Fatalf("空探测解析错误: %+v", ov2)
	}
}

// 53 冲突判定：通配/同地址冲突；resolved stub 等其他特定地址共存。
func TestDns53Conflict(t *testing.T) {
	for _, c := range []struct {
		addr    string
		listen  string
		conflic bool
	}{
		{"0.0.0.0", "192.168.100.2", true},
		{"*", "192.168.100.2", true},
		{"::", "fd00::2", true},
		{"192.168.100.2", "192.168.100.2", true},
		{"fd00::2", "FD00::2", true}, // v6 归一化等值
		{"127.0.0.53", "192.168.100.2", false},
		{"127.0.0.1", "127.0.0.53", false},
		{"fe80::1%eth0", "fd00::2", false},
	} {
		if got := dns53IsConflict(c.addr, c.listen); got != c.conflic {
			t.Fatalf("dns53IsConflict(%q, %q) = %v，期望 %v", c.addr, c.listen, got, c.conflic)
		}
	}
	// 分拣：本服务进程放行；stub 归入 others
	entries := []DnsPort53Occupy{
		{Port: 53, Proto: "tcp", Addr: "127.0.0.53", Process: `users:(("systemd-resolve",pid=788,fd=14))`},
		{Port: 53, Proto: "udp", Addr: "0.0.0.0", Process: `users:(("dnscache",pid=9,fd=7))`},
		{Port: 53, Proto: "tcp", Addr: "192.168.100.2", Process: `users:(("dnsmasq",pid=5,fd=6))`},
	}
	conflicts, others := dns53Classify(entries, "192.168.100.2")
	if len(conflicts) != 1 || conflicts[0].Addr != "0.0.0.0" {
		t.Fatalf("冲突分拣错误: %+v", conflicts)
	}
	if len(others) != 1 || others[0].Addr != "127.0.0.53" {
		t.Fatalf("共存分拣错误: %+v", others)
	}
}

// b64 写脚本：内容经 base64 传递可无损还原，含备份/原子替换/权限保留。
func TestBuildB64WriteScript(t *testing.T) {
	content := []byte("address=/example.com/192.168.1.2\ntxt-record=a,x y z\n")
	script := buildB64WriteScript("/opt/ypanel/dns/dnsmasq.conf", content)
	m := regexp.MustCompile(`printf '%s' '([A-Za-z0-9+/=]+)'`)
	mm := m.FindStringSubmatch(script)
	if mm == nil {
		t.Fatalf("脚本缺少 base64 载荷:\n%s", script)
	}
	decoded, err := base64.StdEncoding.DecodeString(mm[1])
	if err != nil || string(decoded) != string(content) {
		t.Fatalf("base64 载荷还原失败: %v", err)
	}
	for _, want := range []string{
		"cp -f '/opt/ypanel/dns/dnsmasq.conf' '/opt/ypanel/dns/dnsmasq.conf.bak'",
		"stat -c '%a'",
		"base64 -d > '/opt/ypanel/dns/dnsmasq.conf.tmp'",
		"mv -f '/opt/ypanel/dns/dnsmasq.conf.tmp' '/opt/ypanel/dns/dnsmasq.conf'",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("脚本缺少片段 %q\nscript:\n%s", want, script)
		}
	}
}

// 站点对齐：域名收集（主+附加 JSON、通配前缀剥离、跨站点去重）与手动优先过滤、渲染小节。
func TestDnsSiteAlign(t *testing.T) {
	sites := []model.Site{
		{Name: "main", Enabled: true, Domain: "Example.COM", Domains: `["*.example.com","api.example.com"]`},
		{Name: "app", Enabled: true, Domain: "app.example.com"},
	}
	entries := collectSiteDomains(sites)
	if len(entries) != 3 {
		t.Fatalf("域名收集错误（期望去重后 3 条）: %+v", entries)
	}
	if entries[0].Domain != "example.com" || entries[0].SiteName != "main" {
		t.Fatalf("主域名归一/归属错误: %+v", entries[0])
	}
	for _, e := range entries {
		if e.Domain == "*.example.com" || e.Domain == "" {
			t.Fatalf("通配前缀未剥离: %+v", entries)
		}
	}

	// 手动优先：启用 address 同域排除；停用/cname 不拦截
	records := []model.DnsRecord{
		dnsRec(1, "address", "app.example.com", "10.9.9.9", true, 0),
		dnsRec(2, "address", "gone.example.com", "10.9.9.9", false, 0),
		dnsRec(3, "cname", "example.com", "other.example.com", true, 0),
	}
	derived := filterSiteAlign(entries, records)
	if len(derived) != 2 {
		t.Fatalf("手动优先过滤错误（期望 2 条）: %+v", derived)
	}
	for _, e := range derived {
		if e.Domain == "app.example.com" {
			t.Fatal("与手动启用 address 同域的派生项未被排除")
		}
	}

	// 渲染：开启对齐 → 独立小节；默认指向 = 监听 IP
	cfg := DnsConfig{ListenIP: "192.168.100.2", SiteAlign: true}
	conf := renderDnsmasqConf(cfg, records, sites)
	if !strings.Contains(conf, "# ---- 站点对齐") {
		t.Fatalf("缺少站点对齐小节:\n%s", conf)
	}
	if !strings.Contains(conf, "address=/example.com/192.168.100.2") {
		t.Fatalf("派生记录缺失或指向错误:\n%s", conf)
	}
	if strings.Contains(conf, "address=/app.example.com/192.168.100.2") {
		t.Fatal("手动同域派生项不应渲染")
	}
	// 指向 IP 覆盖
	conf2 := renderDnsmasqConf(DnsConfig{ListenIP: "192.168.100.2", SiteAlign: true, SiteAlignIP: "192.168.100.9"}, nil, sites)
	if !strings.Contains(conf2, "address=/example.com/192.168.100.9") {
		t.Fatalf("SiteAlignIP 覆盖未生效:\n%s", conf2)
	}
	// 关闭对齐 → 不渲染
	conf3 := renderDnsmasqConf(DnsConfig{ListenIP: "192.168.100.2"}, nil, sites)
	if strings.Contains(conf3, "站点对齐") {
		t.Fatal("未开启对齐时不应渲染小节")
	}
}
