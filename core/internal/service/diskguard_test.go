package service

import (
	"testing"

	"github.com/ypanel/shared/dto"
)

func TestGuardMounts(t *testing.T) {
	cases := []struct {
		name  string
		disks []dto.DiskUsage
		want  []string
	}{
		{"根分区与 docker 分区独立挂载", []dto.DiskUsage{
			{Mountpoint: "/", Free: 100},
			{Mountpoint: "/data", Free: 50},
			{Mountpoint: "/var/lib/docker", Free: 30},
		}, []string{"/", "/var/lib/docker"}},
		{"docker 在子挂载上（最长前缀）", []dto.DiskUsage{
			{Mountpoint: "/", Free: 100},
			{Mountpoint: "/var/lib", Free: 40},
		}, []string{"/", "/var/lib"}},
		{"仅根分区", []dto.DiskUsage{{Mountpoint: "/", Free: 100}}, []string{"/"}},
		{"无根分区不误判", []dto.DiskUsage{{Mountpoint: "/data", Free: 1}}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := guardMounts(c.disks)
			if len(got) != len(c.want) {
				t.Fatalf("guardMounts(%v) = %v 挂载点数 %d, 期望 %d", c.disks, got, len(got), len(c.want))
			}
			for i := range got {
				if got[i].Mountpoint != c.want[i] {
					t.Fatalf("guardMounts(%v)[%d] = %s, 期望 %s", c.disks, i, got[i].Mountpoint, c.want[i])
				}
			}
		})
	}
}

func TestGuardMountsLow(t *testing.T) {
	const gb uint64 = 1 << 30
	disks := []dto.DiskUsage{
		{Mountpoint: "/", Free: 6 * gb},
		{Mountpoint: "/var/lib/docker", Free: 4 * gb},
	}
	free, low := guardMountsLow(disks, 5*gb)
	if !low || free != 4*gb {
		t.Fatalf("任一挂载点低于阈值应触发: free=%d low=%v", free, low)
	}
	free, low = guardMountsLow(disks, 3*gb)
	if low {
		t.Fatalf("全部高于阈值不应触发")
	}
	// 无 guard 挂载点（缺根分区）不触发
	if _, low := guardMountsLow([]dto.DiskUsage{{Mountpoint: "/data", Free: 0}}, 5*gb); low {
		t.Fatalf("无 guard 挂载点不应触发")
	}
}

func TestSanitizeExclude(t *testing.T) {
	got := sanitizeExclude([]string{" ypanel-nginx ", "", "a", "a", "ypanel-dnsmasq"})
	want := "ypanel-nginx,a,ypanel-dnsmasq"
	if joinComma(got) != want {
		t.Fatalf("sanitizeExclude 去空白/去重失败: %v", got)
	}
	if len(sanitizeExclude([]string{stringsRepeat("x", 200)})) != 0 {
		t.Fatalf("超长容器名应被丢弃")
	}
}

func TestParseExcludeList(t *testing.T) {
	if got := parseExcludeList("a,,b, c"); joinComma(got) != "a,b,c" {
		t.Fatalf("parseExcludeList = %v", got)
	}
	if got := parseExcludeList(""); len(got) != 0 {
		t.Fatalf("空串应解析为空清单: %v", got)
	}
}

func joinComma(in []string) string {
	out := ""
	for i, s := range in {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
