package middleware

import "testing"

func TestLocalImplicitPath(t *testing.T) {
	// 隐式 local：无节点参数也应校验 local 节点
	for _, p := range []string{
		"/api/v1/sites", "/api/v1/docker/containers", "/api/v1/files/list", "/api/v1/store/install",
		"/api/v1/terminal", "/api/v1/database/instances", "/api/v1/nodes/exec", "/api/v1/system/manage",
	} {
		if !localImplicitPath(p) {
			t.Errorf("%s 应属隐式 local", p)
		}
	}
	// 面板级数据：不做节点校验
	for _, p := range []string{
		"/api/v1/auth/me", "/api/v1/notifications", "/api/v1/rbac/roles", "/api/v1/system/overview",
		"/api/v1/tasks", "/api/v1/settings", "/api/v1/ai/conversations", "/api/v1/logs/central/status",
	} {
		if localImplicitPath(p) {
			t.Errorf("%s 不应属隐式 local", p)
		}
	}
}
