package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// RoleAPI 角色管理接口（user:manage）。
type RoleAPI struct {
	RBAC *rbac.Service
}

// Catalog GET /api/v1/rbac/catalog（授权树目录）
func (a *RoleAPI) Catalog(c *gin.Context) {
	respOK(c, rbac.Catalog())
}

type roleItem struct {
	dto.RoleInfo
	Perms []string `json:"perms"`
}

// List GET /api/v1/rbac/roles（角色列表，附权限集）
func (a *RoleAPI) List(c *gin.Context) {
	roles, err := a.RBAC.Roles()
	if err != nil {
		respErr(c, err)
		return
	}
	items := make([]roleItem, 0, len(roles))
	for _, r := range roles {
		perms, err := a.RBAC.RolePerms(r.ID)
		if err != nil {
			respErr(c, err)
			return
		}
		items = append(items, roleItem{RoleInfo: toRoleInfo(r), Perms: perms})
	}
	respOK(c, items)
}

// Create POST /api/v1/rbac/roles
func (a *RoleAPI) Create(c *gin.Context) {
	req, ok := bind[dto.RoleCreateReq](c)
	if !ok {
		return
	}
	role, err := a.RBAC.CreateRole(req.Key, req.Name, req.Remark, req.DataScope, req.Perms)
	if err != nil {
		respErr(c, errs.New(errs.CodeBadRequest, "error.roleCreateFailed", err.Error()))
		return
	}
	respOK(c, toRoleInfo(*role))
}

// Update PUT /api/v1/rbac/roles/:id
func (a *RoleAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	req, ok := bind[dto.RoleUpdateReq](c)
	if !ok {
		return
	}
	var perms []string
	if req.Perms != nil {
		perms = *req.Perms
	}
	role, err := a.RBAC.UpdateRole(uint(id), req.Name, req.Remark, req.DataScope, perms, req.Perms != nil)
	if err != nil {
		respErr(c, errs.New(errs.CodeBadRequest, "error.roleUpdateFailed", err.Error()))
		return
	}
	respOK(c, toRoleInfo(*role))
}

// Delete DELETE /api/v1/rbac/roles/:id
func (a *RoleAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	if err := a.RBAC.DeleteRole(uint(id)); err != nil {
		respErr(c, errs.New(errs.CodeBadRequest, "error.roleDeleteFailed", err.Error()))
		return
	}
	respOK(c, struct{}{})
}

func toRoleInfo(r model.Role) dto.RoleInfo {
	return dto.RoleInfo{
		ID: r.ID, Key: r.Key, Name: r.Name, Builtin: r.Builtin,
		ScopeAllNodes: r.ScopeAllNodes, DataScope: r.DataScope, Remark: r.Remark,
	}
}
