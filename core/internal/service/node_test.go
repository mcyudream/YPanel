package service

import (
	"testing"
	"time"
)

// 心跳边沿判定：上次心跳超出在线窗口（90s）才算「离线→在线」恢复。
func TestIsNodeComeback(t *testing.T) {
	cases := []struct {
		name     string
		lastSeen time.Time
		want     bool
	}{
		{"刚心跳过（在线中）不触发", time.Now().Add(-30 * time.Second), false},
		{"超过在线窗口触发", time.Now().Add(-91 * time.Second), true},
		{"远超窗口（节点重启数小时）触发", time.Now().Add(-3 * time.Hour), true},
		{"零值时间（异常兜底）触发", time.Time{}, true},
	}
	for _, c := range cases {
		if got := isNodeComeback(c.lastSeen); got != c.want {
			t.Fatalf("%s: isNodeComeback=%v want %v", c.name, got, c.want)
		}
	}
}
