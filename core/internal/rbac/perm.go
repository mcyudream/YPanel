// Package rbac 角色权限点（M54）：目录静态定义于代码，角色持有集入库，运行期带缓存解析。
// 权限点形如 "site:write"（模块:动作）；超级角色持 "*" 通配，模块级 "site:*" 亦合法。
package rbac

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Wildcard 超级通配权限点（仅内置超级角色持有）。
const Wildcard = "*"

// Group 权限分组（授权树用；标题前端经 i18n 键 rbac.group.<Key> 渲染）。
type Group struct {
	Key string `json:"key"`
}

// Perm 权限点定义。
type Perm struct {
	Key   string `json:"key"`
	Group string `json:"group"`
}

// Groups 分组目录（顺序即授权树展示顺序）。
var Groups = []Group{
	{Key: "dashboard"}, {Key: "site"}, {Key: "cert"}, {Key: "runtime"}, {Key: "db"},
	{Key: "compose"}, {Key: "docker"}, {Key: "store"}, {Key: "file"}, {Key: "terminal"},
	{Key: "webgw"}, {Key: "cron"}, {Key: "script"}, {Key: "ai"}, {Key: "monitor"},
	{Key: "alert"}, {Key: "log"}, {Key: "node"}, {Key: "host"}, {Key: "tools"},
	{Key: "backup"}, {Key: "update"}, {Key: "panel"},
}

// Perms 权限点目录（新增路由必须在此登记，并在 router.go 标注）。
var Perms = []Perm{
	{Key: "dashboard:read", Group: "dashboard"},

	{Key: "site:read", Group: "site"},
	{Key: "site:write", Group: "site"},
	{Key: "cert:read", Group: "cert"},
	{Key: "cert:write", Group: "cert"},
	{Key: "runtime:read", Group: "runtime"},
	{Key: "runtime:write", Group: "runtime"},
	{Key: "db:read", Group: "db"},
	{Key: "db:write", Group: "db"},
	{Key: "plugin.db-admin:use", Group: "db"},

	{Key: "compose:read", Group: "compose"},
	{Key: "compose:write", Group: "compose"},
	{Key: "docker:read", Group: "docker"},
	{Key: "docker:write", Group: "docker"},
	{Key: "docker:terminal", Group: "docker"},
	{Key: "store:read", Group: "store"},
	{Key: "store:write", Group: "store"},

	{Key: "file:read", Group: "file"},
	{Key: "file:write", Group: "file"},
	{Key: "file:share", Group: "file"},
	{Key: "terminal:access", Group: "terminal"},
	{Key: "webgw:use", Group: "webgw"},

	{Key: "cron:read", Group: "cron"},
	{Key: "cron:write", Group: "cron"},
	{Key: "script:read", Group: "script"},
	{Key: "script:write", Group: "script"},
	{Key: "ai:use", Group: "ai"},
	{Key: "ai:admin", Group: "ai"},

	{Key: "monitor:read", Group: "monitor"},
	{Key: "monitor:write", Group: "monitor"},
	{Key: "alert:read", Group: "alert"},
	{Key: "alert:write", Group: "alert"},
	{Key: "log:read", Group: "log"},
	{Key: "log:write", Group: "log"},

	{Key: "node:read", Group: "node"},
	{Key: "node:manage", Group: "node"},
	{Key: "node:exec", Group: "node"},
	{Key: "host:manage", Group: "host"},

	{Key: "tool:firewall", Group: "tools"},
	{Key: "tool:nat", Group: "tools"},
	{Key: "tool:dns", Group: "tools"},
	{Key: "tool:hosts", Group: "tools"},
	{Key: "tool:ftp", Group: "tools"},
	{Key: "tool:ssh", Group: "tools"},
	{Key: "tool:bruteforce", Group: "tools"},
	{Key: "tool:vpn", Group: "tools"},

	{Key: "backup:manage", Group: "backup"},
	{Key: "update:manage", Group: "update"},

	{Key: "user:manage", Group: "panel"},
	{Key: "setting:write", Group: "panel"},
	{Key: "audit:read", Group: "panel"},
	{Key: "mcp:manage", Group: "panel"},
}

// permKeySet 目录快速索引。
var permKeySet = func() map[string]bool {
	m := make(map[string]bool, len(Perms))
	for _, p := range Perms {
		m[p.Key] = true
	}
	return m
}()

// ValidPermKey 是否为合法权限点（含通配形态）。
func ValidPermKey(key string) bool {
	if key == Wildcard {
		return true
	}
	if permKeySet[key] {
		return true
	}
	if i := strings.LastIndex(key, ":"); i > 0 && key[:i+1]+"*" == key {
		return true // 模块级通配 site:*
	}
	return false
}

// Match 判定权限集是否覆盖 key（支持 * 与 模块:* 通配）。
func Match(set map[string]struct{}, key string) bool {
	if _, ok := set[Wildcard]; ok {
		return true
	}
	if _, ok := set[key]; ok {
		return true
	}
	if i := strings.LastIndex(key, ":"); i > 0 {
		if _, ok := set[key[:i+1]+"*"]; ok {
			return true
		}
	}
	return false
}

// Expand 将权限集展开为具体权限点列表（* → 全目录；模块:* → 模块全部点）。
// fa 前端 hasPermission 是精确 includes 匹配、不支持通配——下发给前端前必须展开。
func Expand(set map[string]struct{}) []string {
	_, super := set[Wildcard]
	out := make([]string, 0, len(set))
	for _, p := range Perms {
		if super {
			out = append(out, p.Key)
			continue
		}
		if _, ok := set[p.Key]; ok {
			out = append(out, p.Key)
			continue
		}
		if i := strings.LastIndex(p.Key, ":"); i > 0 {
			if _, ok := set[p.Key[:i+1]+"*"]; ok {
				out = append(out, p.Key)
			}
		}
	}
	sort.Strings(out)
	return out
}

// ---- 内置角色 ----

// BuiltinRole 内置角色定义（seed 幂等同步，不可经 API 修改）。
type BuiltinRole struct {
	Key    string
	Name   string
	Remark string
	Perms  []string
}

var viewerPerms = []string{
	"dashboard:read",
	"site:read", "cert:read", "runtime:read", "db:read",
	"compose:read", "docker:read", "store:read",
	"file:read", "cron:read", "script:read",
	"monitor:read", "alert:read", "log:read", "node:read",
}

var operatorPerms = append(append([]string{}, viewerPerms...),
	"site:write", "cert:write", "runtime:write", "db:write", "plugin.db-admin:use",
	"compose:write", "docker:write", "docker:terminal", "store:write",
	"file:write", "file:share", "cron:write", "script:write",
	"terminal:access", "webgw:use", "ai:use",
	"monitor:write", "alert:write", "log:write",
)

// Builtins 内置角色：超级管理员 / 只读 / 运维。存量 admin→super-admin、user→operator（能力不缩水）。
var Builtins = []BuiltinRole{
	{Key: "super-admin", Name: "超级管理员", Remark: "全部权限（内置）", Perms: []string{Wildcard}},
	{Key: "viewer", Name: "只读用户", Remark: "各模块只读 + 修改自己密码（内置）", Perms: viewerPerms},
	{Key: "operator", Name: "运维用户", Remark: "业务读写 + 终端，不含节点/面板管理（内置）", Perms: operatorPerms},
}

// ---- 调用者上下文（鉴权中间件注入，AI 工具链消费） ----

type ctxKey struct{}

// Caller 请求调用者的授权上下文。
type Caller struct {
	UserID  uint
	PermSet map[string]struct{}
}

// WithCaller 将调用者授权上下文挂入 ctx（AI 对话 / MCP 请求链路）。
func WithCaller(ctx context.Context, c Caller) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// CallerFrom 取调用者上下文；ok=false 表示链路未注入（面板内部调用，放行）。
func CallerFrom(ctx context.Context) (Caller, bool) {
	c, ok := ctx.Value(ctxKey{}).(Caller)
	return c, ok
}

// aiModulePerm AI 工具模块 → [读权限点, 写权限点]。
// write/danger 风险一律取写点；模块未登记时回落 ai:use / ai:admin。
var aiModulePerm = map[string][2]string{
	"system":           {"dashboard:read", "host:manage"},
	"docker_containers": {"docker:read", "docker:write"},
	"docker_images":     {"docker:read", "docker:write"},
	"docker_networks":   {"docker:read", "docker:write"},
	"docker_volumes":    {"docker:read", "docker:write"},
	"compose":           {"compose:read", "compose:write"},
	"files":             {"file:read", "file:write"},
	"processes":         {"monitor:read", "host:manage"},
	"exec":              {"terminal:access", "terminal:access"},
	"databases":         {"db:read", "db:write"},
	"sites_certs":       {"site:read", "site:write"},
	"runtimes":          {"runtime:read", "runtime:write"},
	"firewall_net":      {"tool:firewall", "tool:firewall"},
	"tasks":             {"cron:read", "cron:write"},
	"store":             {"store:read", "store:write"},
	"srcbuild":          {"compose:read", "compose:write"},
	"diagnostics":       {"monitor:read", "monitor:read"},
	"panel_ai":          {"ai:use", "ai:admin"},
	"meta":              {"ai:use", "ai:use"},
	"mcp":               {"mcp:manage", "mcp:manage"},
}

// CheckTool 校验调用者是否可执行该模块该风险级的工具。
// ctx 未携带调用者信息时放行（面板内部链路）；拒绝时返回可向模型透出的错误。
func CheckTool(ctx context.Context, module, risk string) error {
	caller, ok := CallerFrom(ctx)
	if !ok {
		return nil
	}
	key, known := aiModulePerm[module]
	perm := key[1]
	if risk == "read" {
		perm = key[0]
	}
	if !known {
		perm = "ai:use"
		if risk != "read" {
			perm = "ai:admin"
		}
	}
	if Match(caller.PermSet, perm) {
		return nil
	}
	return fmt.Errorf("当前账号权限不足（缺少 %s），已拒绝执行该工具", perm)
}
