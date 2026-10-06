// ScriptService 脚本库（B13）：计划任务引用的脚本片段管理。
package service

import (
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
)

// ScriptService 脚本库。
type ScriptService struct {
	db *gorm.DB
}

// NewScriptService 创建。
func NewScriptService(db *gorm.DB) *ScriptService {
	return &ScriptService{db: db}
}

// List 全量列表（脚本数量有限，不分页）。
func (s *ScriptService) List() []model.Script {
	out := []model.Script{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
}

// Create 创建脚本。
func (s *ScriptService) Create(name, content string) (*model.Script, error) {
	if name == "" || content == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "脚本名与内容必填")
	}
	row := &model.Script{Name: name, Content: content}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Update 更新脚本。
func (s *ScriptService) Update(id uint, name, content string) (*model.Script, error) {
	var row model.Script
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "脚本不存在")
	}
	if name != "" {
		row.Name = name
	}
	if content != "" {
		row.Content = content
	}
	if err := s.db.Model(&row).Updates(map[string]any{"name": row.Name, "content": row.Content}).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Delete 删除脚本。
func (s *ScriptService) Delete(id uint) error {
	return s.db.Delete(&model.Script{}, id).Error
}
