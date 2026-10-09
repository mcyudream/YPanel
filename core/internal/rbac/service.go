package rbac

import (
	"fmt"
	"regexp"
	"sort"
	"sync"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
)

// RoleScope 角色解析结果（权限点 + 节点范围），按角色缓存、角色变更主动失效。
// 权限/节点实时解析、token 不携带——改角色后在线用户下一个请求即生效。
type RoleScope struct {
	Perms     map[string]struct{}
	AllNodes  bool
	Nodes     map[string]struct{}
	DataScope string // all / assigned（M54-P3）
}

// Service 角色权限服务。
type Service struct {
	db *gorm.DB

	mu    sync.RWMutex
	cache map[uint]*RoleScope
}

// New 创建服务。
func New(db *gorm.DB) *Service {
	return &Service{db: db, cache: map[uint]*RoleScope{}}
}

// Seed 内置角色幂等同步 + 存量用户 role_id 回填（每次启动执行：内置角色权限集以代码为准）。
func (s *Service) Seed() error {
	for _, b := range Builtins {
		var role model.Role
		err := s.db.Where("`key` = ?", b.Key).First(&role).Error
		if err != nil {
			role = model.Role{Key: b.Key, Name: b.Name, Builtin: true, ScopeAllNodes: true, DataScope: "all", Remark: b.Remark}
			if err := s.db.Create(&role).Error; err != nil {
				return fmt.Errorf("创建内置角色 %s 失败: %w", b.Key, err)
			}
		}
		// 内置角色权限集每次启动与代码对齐（自愈，内置不可经 API 改）
		if err := s.db.Where("role_id = ?", role.ID).Delete(&model.RolePermission{}).Error; err != nil {
			return fmt.Errorf("重置内置角色 %s 权限失败: %w", b.Key, err)
		}
		rows := make([]model.RolePermission, 0, len(b.Perms))
		for _, k := range b.Perms {
			rows = append(rows, model.RolePermission{RoleID: role.ID, PermKey: k})
		}
		if len(rows) > 0 {
			if err := s.db.Create(&rows).Error; err != nil {
				return fmt.Errorf("写入内置角色 %s 权限失败: %w", b.Key, err)
			}
		}
		s.invalidate(role.ID)
	}
	// 存量二值角色回填（幂等）：admin → super-admin，user → operator
	if err := s.db.Exec("UPDATE users SET role_id = (SELECT id FROM roles WHERE `key` = 'super-admin') WHERE role = 'admin' AND role_id = 0").Error; err != nil {
		return fmt.Errorf("回填管理员角色失败: %w", err)
	}
	if err := s.db.Exec("UPDATE users SET role_id = (SELECT id FROM roles WHERE `key` = 'operator') WHERE role = 'user' AND role_id = 0").Error; err != nil {
		return fmt.Errorf("回填普通用户角色失败: %w", err)
	}
	return nil
}

// RoleByID 按 ID 取角色。
func (s *Service) RoleByID(id uint) (*model.Role, error) {
	var role model.Role
	if err := s.db.First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Roles 全量角色（按内置优先、key 排序）。
func (s *Service) Roles() ([]model.Role, error) {
	rows := []model.Role{}
	err := s.db.Order("builtin desc, key").Find(&rows).Error
	return rows, err
}

// RolePerms 角色持有的权限点列表。
func (s *Service) RolePerms(roleID uint) ([]string, error) {
	rows := []model.RolePermission{}
	if err := s.db.Where("role_id = ?", roleID).Find(&rows).Error; err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.PermKey)
	}
	return keys, nil
}

// RoleNodes 角色允许的节点 ID 列表（ScopeAllNodes=true 时为空表）。
func (s *Service) RoleNodes(roleID uint) ([]string, error) {
	rows := []model.RoleNode{}
	if err := s.db.Where("role_id = ?", roleID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.NodeID)
	}
	return ids, nil
}

// PermSetForUser 解析用户权限集（兼容旧调用方；RoleID=0 时按旧 Role 字符串回退映射）。
// 返回共享缓存 map，调用方只读。
func (s *Service) PermSetForUser(u *model.User) map[string]struct{} {
	return s.ScopeForUser(u).Perms
}

// ScopeForUser 解析用户完整授权范围（权限 + 节点）。
func (s *Service) ScopeForUser(u *model.User) *RoleScope {
	roleID := u.RoleID
	if roleID == 0 {
		roleID = s.legacyRoleID(u.Role)
	}
	return s.scopeOf(roleID)
}

// RoleKeyForUser 解析用户角色 key（RoleID=0 走旧值映射）。
func (s *Service) RoleKeyForUser(u *model.User) string {
	roleID := u.RoleID
	if roleID == 0 {
		return legacyKey(u.Role)
	}
	role, err := s.RoleByID(roleID)
	if err != nil {
		return legacyKey(u.Role)
	}
	return role.Key
}

func legacyKey(role string) string {
	if role == "admin" {
		return "super-admin"
	}
	return "operator"
}

// RoleKeyToID 按 key 查角色 ID（不存在返回 0）。
func (s *Service) RoleKeyToID(key string) uint {
	var role model.Role
	if err := s.db.Where("`key` = ?", key).First(&role).Error; err != nil {
		return 0
	}
	return role.ID
}

func (s *Service) legacyRoleID(role string) uint {
	var m model.Role
	if err := s.db.Where("`key` = ?", legacyKey(role)).First(&m).Error; err != nil {
		return 0
	}
	return m.ID
}

func (s *Service) scopeOf(roleID uint) *RoleScope {
	s.mu.RLock()
	sc, ok := s.cache[roleID]
	s.mu.RUnlock()
	if ok {
		return sc
	}
	perms := []model.RolePermission{}
	_ = s.db.Where("role_id = ?", roleID).Find(&perms).Error
	permSet := make(map[string]struct{}, len(perms))
	for _, r := range perms {
		permSet[r.PermKey] = struct{}{}
	}
	var role model.Role
	allNodes := true
	_ = s.db.First(&role, roleID).Error
	if role.ID > 0 {
		allNodes = role.ScopeAllNodes
	}
	nodeRows := []model.RoleNode{}
	_ = s.db.Where("role_id = ?", roleID).Find(&nodeRows).Error
	nodeSet := make(map[string]struct{}, len(nodeRows))
	for _, r := range nodeRows {
		nodeSet[r.NodeID] = struct{}{}
	}
	sc = &RoleScope{Perms: permSet, AllNodes: allNodes, Nodes: nodeSet, DataScope: normScope(role.DataScope)}
	s.mu.Lock()
	s.cache[roleID] = sc
	s.mu.Unlock()
	return sc
}

func (s *Service) invalidate(roleID uint) {
	s.mu.Lock()
	delete(s.cache, roleID)
	s.mu.Unlock()
}

var roleKeyRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

// validatePermKeys 权限点校验（自定义角色禁通配）。
func validatePermKeys(keys []string) error {
	for _, k := range keys {
		if !ValidPermKey(k) {
			return fmt.Errorf("未知权限点: %s", k)
		}
		if k == Wildcard {
			return fmt.Errorf("自定义角色不能持有通配权限，请显式勾选")
		}
	}
	return nil
}

// CreateRole 创建自定义角色（scopeAllNodes=false 时以 nodeIDs 为节点范围）。
func (s *Service) CreateRole(key, name, remark, dataScope string, permKeys []string, scopeAllNodes bool, nodeIDs []string) (*model.Role, error) {
	if !roleKeyRe.MatchString(key) {
		return nil, fmt.Errorf("角色标识需为 2~32 位小写字母/数字/连字符，且以字母开头")
	}
	if err := validatePermKeys(permKeys); err != nil {
		return nil, err
	}
	var count int64
	_ = s.db.Model(&model.Role{}).Where("`key` = ?", key).Count(&count).Error
	if count > 0 {
		return nil, fmt.Errorf("角色标识已存在: %s", key)
	}
	role := model.Role{Key: key, Name: name, DataScope: normScope(dataScope), Remark: remark}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		// bool 零值 + default:true 标签会被 GORM Create 跳过插入（落成 DB 默认 true），显式回写
		if err := tx.Model(&role).Update("scope_all_nodes", scopeAllNodes).Error; err != nil {
			return err
		}
		role.ScopeAllNodes = scopeAllNodes
		if err := replacePerms(tx, role.ID, permKeys); err != nil {
			return err
		}
		return replaceNodes(tx, role.ID, nodeIDs)
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(role.ID)
	return &role, nil
}

// UpdateRole 更新自定义角色（内置角色不可改）。permsProvided/nodesProvided=false 时对应集合不动。
func (s *Service) UpdateRole(id uint, name, remark, dataScope *string, permKeys []string, permsProvided bool, scopeAllNodes *bool, nodeIDs []string, nodesProvided bool) (*model.Role, error) {
	role, err := s.RoleByID(id)
	if err != nil {
		return nil, fmt.Errorf("角色不存在")
	}
	if role.Builtin {
		return nil, fmt.Errorf("内置角色不可修改，可复制为自定义角色后调整")
	}
	if permsProvided {
		if err := validatePermKeys(permKeys); err != nil {
			return nil, err
		}
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{}
		if name != nil {
			updates["name"] = *name
		}
		if remark != nil {
			updates["remark"] = *remark
		}
		if dataScope != nil {
			updates["data_scope"] = normScope(*dataScope)
		}
		if scopeAllNodes != nil {
			updates["scope_all_nodes"] = *scopeAllNodes
		}
		if len(updates) > 0 {
			if err := tx.Model(role).Updates(updates).Error; err != nil {
				return err
			}
		}
		if permsProvided {
			if err := replacePerms(tx, role.ID, permKeys); err != nil {
				return err
			}
		}
		if nodesProvided {
			return replaceNodes(tx, role.ID, nodeIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(role.ID)
	return role, nil
}

// DeleteRole 删除自定义角色（内置或仍有用户关联时拒绝）。
func (s *Service) DeleteRole(id uint) error {
	role, err := s.RoleByID(id)
	if err != nil {
		return fmt.Errorf("角色不存在")
	}
	if role.Builtin {
		return fmt.Errorf("内置角色不可删除")
	}
	var count int64
	_ = s.db.Model(&model.User{}).Where("role_id = ?", id).Count(&count).Error
	if count > 0 {
		return fmt.Errorf("仍有 %d 个用户使用该角色，请先调整其角色", count)
	}
	if err := s.db.Select("RolePermissions", "RoleNodes").Delete(role).Error; err != nil {
		return err
	}
	s.invalidate(role.ID)
	return nil
}

func replacePerms(tx *gorm.DB, roleID uint, keys []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		return err
	}
	rows := make([]model.RolePermission, 0, len(keys))
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k] {
			continue
		}
		seen[k] = true
		rows = append(rows, model.RolePermission{RoleID: roleID, PermKey: k})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func replaceNodes(tx *gorm.DB, roleID uint, nodeIDs []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleNode{}).Error; err != nil {
		return err
	}
	rows := make([]model.RoleNode, 0, len(nodeIDs))
	seen := map[string]bool{}
	for _, n := range nodeIDs {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		rows = append(rows, model.RoleNode{RoleID: roleID, NodeID: n})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func normScope(s string) string {
	if s == "assigned" {
		return s
	}
	return "all"
}

// IsSuperUser 用户是否持通配权限（内置 super-admin 或权限集含 "*"）。
func (s *Service) IsSuperUser(u *model.User) bool {
	return Match(s.PermSetForUser(u), Wildcard)
}

// CountActiveSuperUsers 其他在用超级账号数（排除 excludeUID）——防止把最后一个超管降级/禁用/删除。
func (s *Service) CountActiveSuperUsers(excludeUID uint) (int64, error) {
	var n int64
	err := s.db.Model(&model.User{}).
		Joins("JOIN role_permissions rp ON rp.role_id = users.role_id").
		Where("rp.perm_key = ? AND users.status = 1 AND users.id != ?", Wildcard, excludeUID).
		Count(&n).Error
	return n, err
}

// Catalog 授权树目录（分组 + 权限点，前端渲染用）。
type CatalogGroup struct {
	Key   string `json:"key"`
	Perms []Perm `json:"perms"`
}

func Catalog() []CatalogGroup {
	byGroup := map[string][]Perm{}
	for _, p := range Perms {
		byGroup[p.Group] = append(byGroup[p.Group], p)
	}
	out := make([]CatalogGroup, 0, len(Groups))
	for _, g := range Groups {
		out = append(out, CatalogGroup{Key: g.Key, Perms: byGroup[g.Key]})
	}
	return out
}

// SortedIDs 节点 ID 排序导出（下发前端稳定展示）。
func SortedIDs(m map[string]struct{}) []string { return sortedIDs(m) }

// sortedIDs 节点 ID 排序（下发前端稳定展示）。
func sortedIDs(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
