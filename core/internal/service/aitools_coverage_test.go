package service

// AI 工具覆盖固化测试（M33）：扫描 router.go 的 authed 业务路由，按域前缀分组，
// 断言每个域在 AI 工具权限表（aiModulePerm）中有对应模块——
// 新增功能路由若没有同步提供 AI 工具域，本测试直接点名（防「功能上线工具缺失」回归）。
// 新增路由域时：①在 aiModuleRegistry/aiModulePerm 注册模块与工具；②在本测试的 routeDomainMap 补映射。
import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/ypanel/core/internal/rbac"
)

// routeDomainMap 路由前缀 → AI 工具模块（key 为 aiModulePerm/aiModuleRegistry 的模块名）。
// 未列入映射且不在白名单的路由前缀会被测试点名。
var routeDomainMap = map[string]string{
	"/app":              "store",
	"/config-revisions": "panel_ops",
	"/disk":             "system",
	"/nginx":            "sites_certs",
	"/notifications":    "monitor_alert",
	"/rbac":             "panel_ops",
	"/scripts":          "tasks",
	"/security":         "panel_ops",
	"/storage":          "panel_ops",
	"/selfupdate":       "panel_ops",
	"/version":          "panel_ops",
	"/settings":         "panel_ops",
	"/mcpserver":        "panel_ops",
	"/probes":           "monitor_alert",
	"/git":              "srcbuild",
	"/host":             "sites_certs",
	"/mcp":              "panel_ops",
	"/docker":           "docker_containers",
	"/files":            "files",
	"/sites":            "sites_certs",
	"/certs":            "sites_certs",
	"/database":         "databases",
	"/db-admin":         "databases",
	"/cron":             "tasks",
	"/store":            "store",
	"/compose":          "compose",
	"/system":           "system",
	"/processes":        "processes",
	"/services":         "processes",
	"/site-groups":      "sites_certs",
	"/tasks":            "tasks",
	"/monitor":          "monitor_alert",
	"/logs":             "monitor_alert",
	"/audit":            "monitor_alert",
	"/firewall":         "firewall_net",
	"/nat":              "firewall_net",
	"/hosts":            "firewall_net",
	"/dns":              "firewall_net",
	"/ai":               "panel_ai",
	"/panel":            "panel_ops",
	"/runtimes":         "runtimes",
	"/nodes":            "system",
	"/vpn":              "firewall_net",
}

// routeWhitelist 非业务路由（认证/健康/静态/SSE 流/插件内部），无需 AI 工具。
var routeWhitelist = []string{
	"/auth", "/health", "/assets", "/snap", "/plugin", "/notifications/stream",
	"/terminal", "/browser_upgrade", "/webgw",
}

func TestAIToolsCoverAllRouteDomains(t *testing.T) {
	src, err := os.ReadFile("../router/router.go")
	if err != nil {
		t.Skipf("router.go 不可读（跳过覆盖检查）: %v", err)
	}
	re := regexp.MustCompile(`authed\.(GET|POST|PUT|DELETE)\("([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Skip("router.go 未解析到 authed 路由")
	}

	// 提取各路由的域前缀（/xxx/yyy 的第一段 + 第二段组合，尽量精细）
	prefixes := map[string]bool{}
	for _, m := range matches {
		path := m[2]
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 {
			continue
		}
		first := "/" + parts[0]
		if len(parts) >= 2 {
			prefixes[first+"/"+parts[1]] = true
		}
		prefixes[first] = true
	}

	uncovered := map[string][]string{}
	for prefix := range prefixes {
		whitelisted := false
		for _, w := range routeWhitelist {
			if strings.HasPrefix(prefix, w) {
				whitelisted = true
				break
			}
		}
		if whitelisted {
			continue
		}
		if module, ok := routeDomainMap[prefix]; ok {
			if _, has := rbacToolModuleSet()[module]; !has {
				uncovered[prefix] = append(uncovered[prefix], "映射的模块 "+module+" 未在工具权限表注册")
			}
			continue
		}
		// 二段前缀找不到就回退一段
		parts := strings.Split(strings.Trim(prefix, "/"), "/")
		if len(parts) >= 2 {
			if module, ok := routeDomainMap["/"+parts[0]]; ok {
				if _, has := rbacToolModuleSet()[module]; !has {
					uncovered[prefix] = append(uncovered[prefix], "映射的模块 "+module+" 未注册")
				}
				continue
			}
		}
		uncovered[prefix] = append(uncovered[prefix], "无 AI 工具域映射（请在 routeDomainMap 补映射并同步提供工具）")
	}

	if len(uncovered) > 0 {
		keys := make([]string, 0, len(uncovered))
		for k := range uncovered {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t.Errorf("路由域 %s: %v", k, uncovered[k])
		}
	}
}

// rbacToolModuleSet 权限表模块集合。
func rbacToolModuleSet() map[string]bool {
	set := map[string]bool{}
	for _, m := range rbac.ToolModules() {
		set[m] = true
	}
	return set
}

// TestAIModuleRegistryConsistent 校验模块注册表（目录/管理页）与权限表模块一一对应。
func TestAIModuleRegistryConsistent(t *testing.T) {
	permModules := rbacToolModuleSet()
	for _, m := range aiModuleRegistry {
		if !permModules[m.Key] {
			t.Errorf("模块 %s 在注册表（aiModuleRegistry）中但缺少权限映射（aiModulePerm）", m.Key)
		}
		delete(permModules, m.Key)
	}
	for m := range permModules {
		t.Errorf("模块 %s 在 aiModulePerm 中但未列入注册表 aiModuleRegistry（目录将不显示）", m)
	}
}
