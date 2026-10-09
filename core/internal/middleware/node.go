package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/shared/errs"
)

// CtxNodeAll / CtxNodeSet 节点范围（Auth 中间件注入）：true=不限节点；否则以集合为准。
const (
	CtxNodeAll = "nodeAll"
	CtxNodeSet = "nodeSet"
)

// localImplicitPrefixes 无显式节点参数时视作「操作本机节点」的路径前缀（/api/v1 后）。
// 覆盖所有落在 local 节点的业务域；面板级数据（通知/任务/AI 会话/审计/用户/设置等）不在列——
// 与节点无关，任何角色可访问（仍受权限点约束）。
var localImplicitPrefixes = []string{
	"/sites", "/certs", "/nginx", "/database", "/docker", "/compose", "/runtimes",
	"/store/install", "/store/installed", "/scripts", "/cron",
	"/system/manage", "/system/swap", "/system/bbr", "/system/clean", "/system/snapshots",
	"/snapshots", "/panel/backups", "/storage-accounts", "/backups",
	"/vpn", "/firewall", "/fail2ban", "/nat", "/dns", "/hosts", "/ftp", "/ssh",
	"/files", "/terminal", "/webgw", "/processes", "/services", "/logs/search",
	"/git/credentials", "/config-revisions", "/nodes/exec",
}

func localImplicitPath(path string) bool {
	for _, pre := range localImplicitPrefixes {
		if strings.HasPrefix(path, "/api/v1"+pre) {
			return true
		}
	}
	return false
}

// NodeScope 节点范围闸门（M54-P2，置于 Auth 之后）：
//  1. ScopeAllNodes=true 直接放行；
//  2. 显式节点参数（?node= / ?nodeId=）→ 校验该节点；
//  3. 无节点参数但路径属「隐式 local」前缀 → 校验 local；
//  4. 其余路径（面板级数据）不做节点校验。
func NodeScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(CtxNodeAll) {
			c.Next()
			return
		}
		set, _ := c.Get(CtxNodeSet)
		nodeSet, _ := set.(map[string]struct{})
		node := c.Query("node")
		if node == "" {
			node = c.Query("nodeId")
		}
		if node == "" {
			if !localImplicitPath(c.Request.URL.Path) {
				c.Next()
				return
			}
			node = "local"
		}
		if _, ok := nodeSet[node]; ok {
			c.Next()
			return
		}
		abort(c, errs.New(errs.CodeForbidden, "error.nodeForbidden", "无该节点的操作权限"))
	}
}

// NodeScopeFromCtx 供 handler 收窄数据面使用：返回 (不限节点, 允许集)。
func NodeScopeFromCtx(c *gin.Context) (bool, map[string]struct{}) {
	set, _ := c.Get(CtxNodeSet)
	nodeSet, _ := set.(map[string]struct{})
	if nodeSet == nil {
		nodeSet = map[string]struct{}{}
	}
	return c.GetBool(CtxNodeAll), nodeSet
}
