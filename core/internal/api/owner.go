package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/errs"
)

// ---- M54-P3 数据范围（属主）助手 ----
// 语义：角色 DataScope=assigned 时，资源仅 owner∈{uid, 0=公共} 可见可管；
// 单资源越权一律回 404（不暴露存在性）；all 范围不受限。

// dataScopeOf 当前调用者数据范围。
func dataScopeOf(c *gin.Context) string {
	if v, ok := c.Get(middleware.CtxDataScope); ok {
		if s, is := v.(string); is {
			return s
		}
	}
	return "all"
}

// callerUID 当前调用者用户 ID。
func callerUID(c *gin.Context) uint {
	return c.GetUint(middleware.CtxUID)
}

// ownerAllowed assigned 范围下 owner∈{uid,0} 才放行。
func ownerAllowed(c *gin.Context, ownerID uint) bool {
	if dataScopeOf(c) != "assigned" {
		return true
	}
	return ownerID == 0 || ownerID == callerUID(c)
}

// isSuperCaller 调用者是否持通配权限（等价旧 admin 全量判断）。
func isSuperCaller(c *gin.Context) bool {
	set, _ := c.Get(middleware.CtxPerms)
	permSet, _ := set.(map[string]struct{})
	return rbac.Match(permSet, rbac.Wildcard)
}

// errNotFound 统一越权/不存在响应体（不暴露存在性差异）。
func errNotFound(msg string) *errs.Error {
	return errs.New(errs.CodeNotFound, "error.notFound", msg)
}

// ownedSiteID 解析站点 :id 并在 assigned 范围下校验属主。
// 失败时返回错误（由调用方既有 respErr 通道渲染 404），不重复写响应。
func ownedSiteID(c *gin.Context, sites *service.SiteService) (uint, error) {
	id, err := idParam(c)
	if err != nil {
		return 0, err
	}
	if dataScopeOf(c) != "assigned" {
		return id, nil
	}
	site, err := sites.GetByIDF(id)
	if err != nil {
		return 0, errNotFound("站点不存在")
	}
	if !ownerAllowed(c, site.OwnerID) {
		return 0, errNotFound("站点不存在")
	}
	return id, nil
}

// ownedInstanceID 解析数据库实例 :id 并校验属主。
func ownedInstanceID(c *gin.Context, dbs *service.DatabaseService) (uint, error) {
	id, err := idParam(c)
	if err != nil {
		return 0, err
	}
	if dataScopeOf(c) != "assigned" {
		return id, nil
	}
	inst, err := dbs.ByID(id)
	if err != nil {
		return 0, errNotFound("实例不存在")
	}
	if !ownerAllowed(c, inst.OwnerID) {
		return 0, errNotFound("实例不存在")
	}
	return id, nil
}

// ownedProject 解析商店应用 :project 并校验属主（install 记录缺失也按 404 由调用方处理）。
func ownedProject(c *gin.Context, store *service.StoreService) (string, error) {
	project := c.Param("project")
	if project == "" {
		return "", errBadRequest("项目名不合法")
	}
	if dataScopeOf(c) != "assigned" {
		return project, nil
	}
	inst, err := store.InstalledByProject(project)
	if err != nil {
		return "", errNotFound("应用不存在")
	}
	if !ownerAllowed(c, inst.OwnerID) {
		return "", errNotFound("应用不存在")
	}
	return project, nil
}

// stampOwnerForCreate 创建归属规则（M54-P3）：assigned 创建 → 自己；all 创建 → 公共(0)。
func stampOwnerForCreate(c *gin.Context) uint {
	if dataScopeOf(c) == "assigned" {
		return callerUID(c)
	}
	return 0
}

// requireAllScope 属主再分配仅限数据范围不受限（all）的账号。
func requireAllScope(c *gin.Context) bool {
	if dataScopeOf(c) == "all" {
		return true
	}
	respErr(c, errs.New(errs.CodeForbidden, "error.ownerManageForbidden", "仅数据范围不受限的账号可调整资源属主"))
	return false
}

// parseOwnerBody 解析属主分配请求 {ownerId}。
func parseOwnerBody(c *gin.Context) (uint, bool) {
	req, ok := bind[struct {
		OwnerID *uint `json:"ownerId"`
	}](c)
	if !ok || req.OwnerID == nil {
		return 0, false
	}
	return *req.OwnerID, true
}
