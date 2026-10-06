// 站点分组管理（对齐 1Panel：创建/编辑/删除/设为默认，默认分组不可删，删除时站点归入默认组）。
package service

import (
	"regexp"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

var groupNamePattern = regexp.MustCompile(`^[\p{Han}a-zA-Z0-9_-]{1,32}$`)

// SiteGroupService 站点分组服务。
type SiteGroupService struct {
	db *gorm.DB
}

// NewSiteGroupService 创建分组服务。
func NewSiteGroupService(db *gorm.DB) *SiteGroupService {
	return &SiteGroupService{db: db}
}

// EnsureDefaultGroup 保证存在默认分组（首次访问时自动创建）。
func (s *SiteGroupService) EnsureDefaultGroup() (*model.SiteGroup, error) {
	var g model.SiteGroup
	if err := s.db.Where("is_default = ?", true).First(&g).Error; err == nil {
		return &g, nil
	}
	g = model.SiteGroup{Name: "默认", IsDefault: true}
	if err := s.db.Create(&g).Error; err != nil {
		// 并发创建冲突时回读
		var out model.SiteGroup
		if e := s.db.Where("is_default = ?", true).First(&out).Error; e == nil {
			return &out, nil
		}
		return nil, err
	}
	return &g, nil
}

// List 分组列表（默认分组置顶，附站点计数）。
func (s *SiteGroupService) List() ([]map[string]any, error) {
	if _, err := s.EnsureDefaultGroup(); err != nil {
		return nil, err
	}
	var groups []model.SiteGroup
	if err := s.db.Order("is_default desc, id").Find(&groups).Error; err != nil {
		return nil, err
	}
	type cnt struct {
		GroupID uint
		N       int64
	}
	var counts []cnt
	if err := s.db.Model(&model.Site{}).Select("group_id, count(*) as n").Group("group_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	cm := map[uint]int64{}
	for _, c := range counts {
		cm[c.GroupID] = c.N
	}
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		n := cm[g.ID]
		if g.IsDefault {
			n += cm[0] // 未分组站点归入默认组
		}
		out = append(out, map[string]any{
			"id": g.ID, "name": g.Name, "isDefault": g.IsDefault, "sites": n,
		})
	}
	return out, nil
}

// Create 创建分组。
func (s *SiteGroupService) Create(name string) (*model.SiteGroup, error) {
	if !groupNamePattern.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "分组名不合法（1-32 位中文/字母/数字/中划线/下划线）")
	}
	if _, err := s.EnsureDefaultGroup(); err != nil {
		return nil, err
	}
	var count int64
	_ = s.db.Model(&model.SiteGroup{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "分组已存在")
	}
	g := &model.SiteGroup{Name: name}
	if err := s.db.Create(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

// Update 重命名分组（默认分组可改名）。
func (s *SiteGroupService) Update(id uint, name string) error {
	if !groupNamePattern.MatchString(name) {
		return errs.Wrap(errs.ErrBadRequest, "分组名不合法")
	}
	g, err := s.byID(id)
	if err != nil {
		return err
	}
	var count int64
	_ = s.db.Model(&model.SiteGroup{}).Where("name = ? AND id != ?", name, id).Count(&count).Error
	if count > 0 {
		return errs.New(errs.CodeConflict, "error.conflict", "分组名已存在")
	}
	return s.db.Model(g).Update("name", name).Error
}

// Delete 删除分组（站点归入默认组；默认分组不可删）。
func (s *SiteGroupService) Delete(id uint) error {
	def, err := s.EnsureDefaultGroup()
	if err != nil {
		return err
	}
	if id == def.ID {
		return errs.Wrap(errs.ErrBadRequest, "默认分组不可删除")
	}
	g, err := s.byID(id)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Site{}).Where("group_id = ?", id).Update("group_id", def.ID).Error; err != nil {
			return err
		}
		return tx.Delete(g).Error
	})
}

// SetDefault 设为默认分组（新站点默认落组；原默认组取消标记）。
func (s *SiteGroupService) SetDefault(id uint) error {
	g, err := s.byID(id)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.SiteGroup{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(g).Update("is_default", true).Error
	})
}

func (s *SiteGroupService) byID(id uint) (*model.SiteGroup, error) {
	var g model.SiteGroup
	if err := s.db.First(&g, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "分组不存在")
	}
	return &g, nil
}
