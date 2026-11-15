package service

import (
	"context"
	"github.com/ypanel/core/internal/model"
	"gorm.io/gorm"
	"strings"
)

func aiContext(scopes []context.Context) context.Context {
	if len(scopes) > 0 {
		return scopes[0]
	}
	return context.Background()
}

// Legacy owner=0 resources stay admin-only; only explicitly published resources are shared.
func aiResourceScope(ctx context.Context, q *gorm.DB) *gorm.DB {
	if aiAdmin(ctx) {
		return q
	}
	return q.Where("((owner_id = ? AND owner_id <> 0) OR shared = ?)", aiOwner(ctx), true)
}
func (s *AIService) MemoryPage(ctx context.Context, search string, page, size int) (map[string]any, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 40
	}
	q := aiResourceScope(ctx, s.db.WithContext(ctx).Model(&model.AIMemory{}))
	if search != "" {
		q = q.Where("content LIKE ?", "%"+search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	rows := []model.AIMemory{}
	if err := q.Order("id desc").Limit(size).Offset((page - 1) * size).Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"items": rows, "total": total, "page": page, "pageSize": size}, nil
}
func (s *AIService) SaveMemory(ctx context.Context, in model.AIMemory) error {
	in.Content = maskSensitive(strings.TrimSpace(in.Content))
	if in.Content == "" || len(in.Content) > 4000 {
		return errWrapAI("记忆内容不能为空且需小于 4000 字节")
	}
	if in.Shared && !aiAdmin(ctx) {
		return errWrapAI("仅管理员可发布共享记忆")
	}
	if in.ID == 0 {
		in.OwnerID = aiOwner(ctx)
		return s.db.WithContext(ctx).Create(&in).Error
	}
	var old model.AIMemory
	if aiScope(ctx, s.db).First(&old, in.ID).Error != nil {
		return aiNotFound()
	}
	return s.db.Model(&old).Updates(map[string]any{"content": in.Content, "shared": in.Shared}).Error
}
func (s *AIService) DeleteMemory(ctx context.Context, id uint) error {
	result := aiScope(ctx, s.db.WithContext(ctx)).Delete(&model.AIMemory{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return aiNotFound()
	}
	return nil
}

func (s *AIService) ShareKnowledgeDoc(ctx context.Context, id uint, shared bool) error {
	if !aiAdmin(ctx) {
		return errWrapAI("仅管理员可发布共享文档")
	}
	var row model.AIKnowledgeDoc
	if aiScope(ctx, s.db).First(&row, id).Error != nil {
		return aiNotFound()
	}
	return s.db.Model(&row).Update("shared", shared).Error
}
