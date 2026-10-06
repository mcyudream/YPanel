package service

import (
	"github.com/ypanel/core/internal/wsbus"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
)

// NotificationService 站内通知：关键事件写入（登录异常/告警/实例与站点变更等）。
type NotificationService struct {
	db *gorm.DB
}

// NewNotificationService 创建。
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// Push 写入通知（供各服务调用；err 静默——通知失败不影响主流程）。
func (s *NotificationService) Push(level, title, content string) {
	row := &model.Notification{Level: level, Title: title, Content: content}
	_ = s.db.Create(row).Error
	// B7：通知事件推送
	wsbus.Default.Publish("notification", level, title, map[string]any{"content": content})
}

// List 通知列表（分页由调用方限制条数）。
func (s *NotificationService) List(limit int) []model.Notification {
	out := []model.Notification{}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	_ = s.db.Order("id desc").Limit(limit).Find(&out).Error
	return out
}

// UnreadCount 未读数。
func (s *NotificationService) UnreadCount() int64 {
	var n int64
	_ = s.db.Model(&model.Notification{}).Where("read = ?", false).Count(&n).Error
	return n
}

// MarkRead 标记已读（id=0 表示全部）。
func (s *NotificationService) MarkRead(id uint) error {
	q := s.db.Model(&model.Notification{}).Where("read = ?", false)
	if id > 0 {
		q = s.db.Model(&model.Notification{}).Where("id = ?", id)
	}
	return q.Update("read", true).Error
}
