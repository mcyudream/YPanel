// DangerLock 危险操作锁定（M48）：开启后全局拒绝恢复/回滚/重装/清理/删除类高危端点。
// 锁定对所有登录用户生效（含 admin）；解锁走「安全设置」开关。AI 工具链路同步拦截。
package service

import (
	"strings"

	"github.com/ypanel/shared/errs"
)

// KeyDangerLock 危险操作锁定设置键（"1" = 锁定开启）。
const KeyDangerLock = "security.danger_lock"

// DangerLockEnabled 锁定是否开启。
func (s *SecuritySettingsService) DangerLockEnabled() bool {
	return s.settings.Get(KeyDangerLock, "0") == "1"
}

// SetDangerLock 设置锁定开关。
func (s *SecuritySettingsService) SetDangerLock(on bool) error {
	v := "0"
	if on {
		v = "1"
	}
	return s.settings.Set(KeyDangerLock, v)
}

// dangerPaths 锁定端点清单（method + FullPath 后缀匹配；保守集合，可按需扩充）。
// 覆盖：系统快照恢复、数据库/面板恢复、容器重装、应用升级/卸载、各类 prune/清理、compose down、危险删除。
var dangerPaths = []struct{ method, suffix string }{
	{"POST", "/snapshots/execute"},          // 系统/面板快照恢复
	{"POST", "/restore"},                    // 数据库/面板恢复类（含 system/snapshots/restore）
	{"POST", "/prune"},                      // 各类清理（镜像/容器/卷/构建缓存/快照）
	{"POST", "/uninstall"},                  // 应用卸载
	{"POST", "/upgrade"},                    // 应用升级（重装语义）
	{"POST", "/recreate"},                   // 容器重装
	{"POST", "/compose/down"},               // 编排下线
	{"POST", "/system/clean"},               // 系统清理
	{"DELETE", "/docker/containers/:id"},    // 容器删除
	{"DELETE", "/docker/images/:id"},        // 镜像删除
	{"DELETE", "/docker/volumes/:name"},     // 卷删除
	{"DELETE", "/sites/:id"},                // 站点删除
	{"DELETE", "/nodes/:id"},                // 节点删除
	{"DELETE", "/store/installed/:project"}, // 商店已装删除
	{"DELETE", "/cron/tasks/:id"},           // 计划任务删除
}

// IsDangerPath 判定请求是否命中锁定清单。
func IsDangerPath(method, fullPath string) bool {
	for _, p := range dangerPaths {
		if p.method == method && strings.HasSuffix(fullPath, p.suffix) {
			return true
		}
	}
	return false
}

// CheckDangerLock 供中间件/AI 链路调用：锁定开启且命中清单 → 拒绝。
func (s *SecuritySettingsService) CheckDangerLock(method, fullPath string) error {
	if s.DangerLockEnabled() && IsDangerPath(method, fullPath) {
		return errs.New(errs.CodeForbidden, "error.dangerLock",
			"危险操作锁定已开启：恢复/回滚/重装/清理/删除类操作被禁止，请先在「安全设置」中关闭锁定")
	}
	return nil
}
