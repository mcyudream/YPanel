package service

import (
	"strings"
	"testing"

	"github.com/ypanel/core/internal/model"
)

func hostRec(id uint, ip, hostnames string, enabled bool, sort int) model.HostRecord {
	return model.HostRecord{ID: id, IP: ip, Hostnames: hostnames, Enabled: enabled, Sort: sort}
}

func hostRecP(id uint, ip, hostnames string, enabled bool, sort int) *model.HostRecord {
	r := hostRec(id, ip, hostnames, enabled, sort)
	return &r
}

const hostsSysPart = "127.0.0.1\tlocalhost\n::1\tlocalhost ip6-localhost ip6-loopback\n"

// 托管块渲染：启用记录按 sort,id 升序、含注释、停用过滤、空态占位。
func TestRenderHostsBlock(t *testing.T) {
	records := []model.HostRecord{
		hostRec(2, "192.168.100.3", "mc.example.com", true, 0),
		hostRec(1, "192.168.100.2", "example.com www.example.com", true, 0),
		hostRec(3, "10.0.0.9", "disabled.example.com", false, 0),
	}
	block := renderHostsBlock(records)
	if !strings.HasPrefix(block, hostsBlockBegin+"\n") || !strings.HasSuffix(block, hostsBlockEnd+"\n") {
		t.Fatalf("块标记缺失:\n%s", block)
	}
	for _, want := range []string{
		"192.168.100.2\texample.com www.example.com",
		"192.168.100.3\tmc.example.com",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("块缺少片段 %q\nblock:\n%s", want, block)
		}
	}
	if strings.Contains(block, "disabled.example.com") {
		t.Fatal("停用记录不应渲染")
	}
	if strings.Index(block, "192.168.100.2\t") > strings.Index(block, "192.168.100.3\t") {
		t.Fatal("记录排序错误（期望 sort,id 升序）")
	}
	// 带注释
	r := hostRec(9, "192.168.100.4", "db.example.com", true, 0)
	r.Comment = "数据库"
	line := renderHostsBlock([]model.HostRecord{r})
	if !strings.Contains(line, "192.168.100.4\tdb.example.com\t# 数据库") {
		t.Fatalf("注释渲染错误:\n%s", line)
	}
	// 空态
	empty := renderHostsBlock(nil)
	if !strings.Contains(empty, "# （当前无启用记录）") {
		t.Fatalf("空态占位缺失:\n%s", empty)
	}
}

// compose 幂等与还原：无块追加、已有块替换、漂移块替换、解除托管剥块。
func TestComposeHostsFile(t *testing.T) {
	block := renderHostsBlock([]model.HostRecord{hostRec(1, "192.168.100.2", "example.com", true, 0)})
	// 无块 → 追加（保留系统条目）
	got := composeHostsFile(hostsSysPart, block)
	if !strings.Contains(got, "127.0.0.1\tlocalhost") || !strings.Contains(got, "example.com") {
		t.Fatalf("追加失败:\n%s", got)
	}
	// 幂等：再合成不变
	if again := composeHostsFile(got, block); again != got {
		t.Fatalf("合成不幂等:\n%s\n---\n%s", got, again)
	}
	// 已有块（漂移）→ 替换
	drift := renderHostsBlock([]model.HostRecord{hostRec(1, "10.9.9.9", "old.example.com", true, 0)})
	got2 := composeHostsFile(hostsSysPart+drift, block)
	if strings.Contains(got2, "old.example.com") || !strings.Contains(got2, "example.com") {
		t.Fatalf("漂移块替换失败:\n%s", got2)
	}
	if !strings.Contains(got2, "127.0.0.1\tlocalhost") {
		t.Fatal("替换时不应触碰系统条目")
	}
	// 解除托管（block=""）→ 剥块还原
	got3 := composeHostsFile(hostsSysPart+block, "")
	if strings.Contains(got3, "ypanel-managed") || strings.Contains(got3, "example.com") {
		t.Fatalf("剥块失败:\n%s", got3)
	}
	if !strings.Contains(got3, "127.0.0.1\tlocalhost") {
		t.Fatalf("剥块后系统条目丢失:\n%s", got3)
	}
}

// 块提取与异常形态：无块空串；有 begin 无 end 视为延伸到文件尾。
func TestExtractHostsBlock(t *testing.T) {
	if extractHostsBlock(hostsSysPart) != "" {
		t.Fatal("无块时应返回空串")
	}
	block := renderHostsBlock([]model.HostRecord{hostRec(1, "192.168.100.2", "example.com", true, 0)})
	full := hostsSysPart + block
	got := extractHostsBlock(full)
	if got != block {
		t.Fatalf("块提取不一致:\n%s\n---\n%s", got, block)
	}
	// 未闭合块
	unclosed := hostsSysPart + hostsBlockBegin + "\n1.2.3.4 broken\n"
	if got := extractHostsBlock(unclosed); !strings.Contains(got, "broken") {
		t.Fatalf("未闭合块应延伸到文件尾:\n%s", got)
	}
	if got := stripHostsBlock(unclosed); strings.Contains(got, "broken") || !strings.Contains(got, "localhost") {
		t.Fatalf("未闭合块剥离失败:\n%s", got)
	}
}

// 校验：IP 合法化、主机名分隔/小写/去重、数量与格式。
func TestHostsValidateRecord(t *testing.T) {
	r := hostRec(0, "192.168.100.2", "Example.COM, www.Example.com  example.com", true, 0)
	if err := hostsValidateRecord(&r); err != nil {
		t.Fatalf("合法记录被拒: %v", err)
	}
	if r.IP != "192.168.100.2" || r.Hostnames != "example.com www.example.com" {
		t.Fatalf("归一化失败: %s / %s", r.IP, r.Hostnames)
	}
	// 超量：互异主机名超过上限
	many := make([]string, 0, hostsMaxNames+1)
	for i := 0; i <= hostsMaxNames; i++ {
		many = append(many, "a"+strings.Repeat("b", i)+".example.com")
	}
	bad := []model.HostRecord{
		hostRec(0, "not-an-ip", "a.com", true, 0),
		hostRec(0, "192.168.100.2", "", true, 0),
		hostRec(0, "192.168.100.2", "bad_name..com", true, 0),
		hostRec(0, "192.168.100.2", strings.Join(many, " "), true, 0),
	}
	for i := range bad {
		if err := hostsValidateRecord(&bad[i]); err == nil {
			t.Fatalf("非法记录 #%d 未被拦截: %+v", i, bad[i])
		}
	}
	// IPv6 合法
	r6 := hostRec(0, "FD00::2", "v6.example.com", true, 0)
	if err := hostsValidateRecord(&r6); err != nil || r6.IP != "fd00::2" {
		t.Fatalf("IPv6 校验失败: %v / %s", err, r6.IP)
	}
}

// 启用记录间主机名重复拦截（停用记录与自身不参与）。
func TestHostsHostnameConflict(t *testing.T) {
	existing := []model.HostRecord{hostRec(1, "192.168.100.2", "example.com", true, 0)}
	if !hostsHostnameConflict(existing, hostRecP(0, "10.0.0.1", "EXAMPLE.com", true, 0)) {
		t.Fatal("大小写不敏感的重复主机名未被拦截")
	}
	if hostsHostnameConflict(existing, hostRecP(0, "10.0.0.1", "example.com", false, 0)) {
		t.Fatal("停用记录不应参与冲突判定")
	}
	self := hostRec(1, "192.168.100.2", "example.com", true, 0)
	if hostsHostnameConflict(existing, &self) {
		t.Fatal("记录自身不应参与冲突判定（更新场景）")
	}
	if hostsHostnameConflict(existing, hostRecP(0, "192.168.100.2", "mail.example.com", true, 0)) {
		t.Fatal("不重名的新主机名（哪怕同 IP）不应拦截")
	}
	if !hostsHostnameConflict(existing, hostRecP(0, "192.168.100.2", "example.com mail.example.com", true, 0)) {
		t.Fatal("跨记录共享主机名（即使同 IP）也应拦截（hosts 首条命中语义）")
	}
}
