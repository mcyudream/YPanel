package rbac

import "testing"

func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

func TestMatch(t *testing.T) {
	cases := []struct {
		name string
		set  map[string]struct{}
		key  string
		want bool
	}{
		{"通配", set("*"), "site:write", true},
		{"精确命中", set("site:read"), "site:read", true},
		{"精确未命中", set("site:read"), "site:write", false},
		{"模块通配", set("site:*"), "site:write", true},
		{"模块通配不跨模块", set("site:*"), "docker:write", false},
		{"插件点模块通配", set("plugin.db-admin:*"), "plugin.db-admin:use", true},
		{"空集", set(), "dashboard:read", false},
	}
	for _, c := range cases {
		if got := Match(c.set, c.key); got != c.want {
			t.Errorf("%s: Match=%v want %v", c.name, got, c.want)
		}
	}
}

func TestValidPermKey(t *testing.T) {
	for _, k := range []string{"*", "site:read", "plugin.db-admin:use", "site:*"} {
		if !ValidPermKey(k) {
			t.Errorf("%s 应合法", k)
		}
	}
	for _, k := range []string{"site", "site:", ":read", "unknown:read", "site:**", "*:read"} {
		if ValidPermKey(k) {
			t.Errorf("%s 应非法", k)
		}
	}
}

func TestCheckTool(t *testing.T) {
	ctx := WithCaller(t.Context(), Caller{PermSet: set("docker:read", "terminal:access")})
	if err := CheckTool(ctx, "docker_containers", "read"); err != nil {
		t.Errorf("docker 读应放行: %v", err)
	}
	if err := CheckTool(ctx, "docker_containers", "write"); err == nil {
		t.Error("docker 写应拒绝")
	}
	if err := CheckTool(ctx, "exec", "danger"); err != nil {
		t.Errorf("terminal:access 应放行 exec: %v", err)
	}
	if err := CheckTool(t.Context(), "docker_containers", "write"); err != nil {
		t.Error("未注入调用者的内部链路应放行")
	}
	super := WithCaller(t.Context(), Caller{PermSet: set("*")})
	if err := CheckTool(super, "databases", "danger"); err != nil {
		t.Errorf("通配应放行: %v", err)
	}
}
