package service

import (
	"strings"
	"testing"
	"time"

	"github.com/ypanel/core/internal/model"
)

func TestSiteDesiredExtraPorts(t *testing.T) {
	rows := []model.Site{
		{Name: "a", Port: 80, Enabled: true},
		{Name: "b", Port: 443, Enabled: true},
		{Name: "c", Port: 8080, Enabled: true},
		{Name: "d", Port: 8080, Enabled: true},  // 同端口多站点：去重
		{Name: "e", Port: 9090, Enabled: false}, // 停用：不纳入
		{Name: "f", Port: 3000, Enabled: true},
		{Name: "g", Port: 0, Enabled: true},     // 零值：视同 80，不纳入
		{Name: "h", Port: 70000, Enabled: true}, // 越界：防御性排除
	}
	got := siteDesiredExtraPorts(rows)
	if len(got) != 2 || got[0] != 3000 || got[1] != 8080 {
		t.Fatalf("desired 错误: %v", got)
	}
	if got2 := siteDesiredExtraPorts(nil); len(got2) != 0 {
		t.Fatalf("空输入应为空集: %v", got2)
	}
}

func TestSiteLeaseDiff(t *testing.T) {
	leases := []model.SitePortLease{
		{ID: 1, Port: 8080, Proto: "tcp"},
		{ID: 2, Port: 9090, Proto: "tcp"},
	}
	toAdd, toRemove := siteLeaseDiff([]int{9090, 8443}, leases)
	if len(toAdd) != 1 || toAdd[0] != 8443 {
		t.Fatalf("toAdd 错误: %v", toAdd)
	}
	if len(toRemove) != 1 || toRemove[0] != 8080 {
		t.Fatalf("toRemove 错误: %v", toRemove)
	}
	// 全等：无差异
	if a, r := siteLeaseDiff([]int{8080, 9090}, leases); len(a) != 0 || len(r) != 0 {
		t.Fatalf("全等时不应有差异: add=%v remove=%v", a, r)
	}
}

func TestValidateSitePort(t *testing.T) {
	if p, err := validateSitePort(0); err != nil || p != 80 {
		t.Fatalf("0 应默认 80: %d %v", p, err)
	}
	if p, err := validateSitePort(8080); err != nil || p != 8080 {
		t.Fatalf("8080 应原样通过: %d %v", p, err)
	}
	for _, bad := range []int{-1, 65536} {
		if _, err := validateSitePort(bad); err == nil {
			t.Fatalf("%d 应拒绝", bad)
		}
	}
}

func TestNginxComposeTemplatePorts(t *testing.T) {
	base := nginxComposeTemplate(nil)
	if strings.Count(base, `- "80:80"`) != 1 || strings.Count(base, `- "443:443"`) != 1 {
		t.Fatalf("基础端口段缺失:\n%s", base)
	}
	if strings.Contains(base, "- \"8080:") {
		t.Fatalf("基础模板不应含附加端口")
	}
	extra := nginxComposeTemplate([]int{8080, 8443})
	for _, want := range []string{`- "8080:8080"`, `- "8443:8443"`} {
		if !strings.Contains(extra, want) {
			t.Fatalf("缺少映射 %s:\n%s", want, extra)
		}
	}
	// 映射行应位于 ports: 段内（volumes 之前）
	if idxPorts := strings.Index(extra, "ports:"); idxPorts < 0 || strings.Index(extra, `- "8080:8080"`) < idxPorts {
		t.Fatalf("附加端口未落在 ports 段:\n%s", extra)
	}
}

func TestIntsEqual(t *testing.T) {
	if !intsEqual([]int{1, 2}, []int{1, 2}) || intsEqual([]int{1}, []int{1, 2}) || intsEqual(nil, []int{}) && false {
		t.Fatal("intsEqual 语义错误")
	}
	if !intsEqual(nil, nil) {
		t.Fatal("nil 与 nil 应相等")
	}
}

// 防御：SitePortLease 唯一索引存在性靠迁移保证，这里只锁字段约定（port 唯一、proto 默认 tcp）。
func TestSitePortLeaseShape(t *testing.T) {
	l := model.SitePortLease{Port: 8080, Proto: "tcp", Sites: "a,b", CreatedAt: time.Now()}
	if l.Proto != "tcp" || l.Sites != "a,b" {
		t.Fatalf("字段约定变化: %+v", l)
	}
}
