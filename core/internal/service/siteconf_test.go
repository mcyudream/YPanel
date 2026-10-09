package service

import (
	"testing"

	"github.com/ypanel/core/internal/model"
)

func TestCertCovers(t *testing.T) {
	cert := &model.Certificate{
		Domain:     "example.com",
		AltDomains: `["www.example.com", "*.example.org", "a.b.example.net", "example.net"]`,
	}
	cases := []struct {
		domain string
		want   bool
	}{
		{"example.com", true},       // 主域名
		{"www.example.com", true},   // 其他域名精确命中
		{"example.org", false},      // 泛域名不覆盖裸域
		{"sub.example.org", true},   // 泛域名覆盖一级子域
		{"a.b.example.org", false},  // 泛域名不覆盖多级子域
		{"a.b.example.net", true},   // 其他域名精确命中（非泛匹配）
		{"mail.example.net", false}, // 未列入的子域
		{"other.com", false},
	}
	for _, c := range cases {
		if got := certCovers(cert, c.domain); got != c.want {
			t.Errorf("certCovers(%q) = %v, want %v", c.domain, got, c.want)
		}
	}
}

func TestSwitchedExtras(t *testing.T) {
	// 前端交换预览形态：新主域名已从列表移除，旧主域名已放回
	got := switchedExtras([]string{"old.example.com"}, "old.example.com", "new.example.com")
	if len(got) != 1 || got[0] != "old.example.com" {
		t.Errorf("交换形态: got %v, want [old.example.com]", got)
	}
	// 直调形态：新主域名仍在列表，旧主域名缺失 → 移除新主、补入旧主（集合守恒）
	got = switchedExtras([]string{"new.example.com", "other.example.com"}, "old.example.com", "new.example.com")
	if len(got) != 2 || got[0] != "other.example.com" || got[1] != "old.example.com" {
		t.Errorf("直调形态: got %v, want [other.example.com old.example.com]", got)
	}
	// 混合形态：新主与旧主都在列表
	got = switchedExtras([]string{"new.example.com", "old.example.com", "x.example.com"}, "old.example.com", "new.example.com")
	if len(got) != 2 || got[0] != "old.example.com" || got[1] != "x.example.com" {
		t.Errorf("混合形态: got %v, want [old.example.com x.example.com]", got)
	}
	// 空列表：仅剩旧主域名兜底
	got = switchedExtras(nil, "old.example.com", "new.example.com")
	if len(got) != 1 || got[0] != "old.example.com" {
		t.Errorf("空列表: got %v, want [old.example.com]", got)
	}
}
