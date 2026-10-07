package service

import (
	"encoding/json"
	"sync"

	"github.com/ypanel/core/internal/wsbus"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
)

// NotificationService 站内通知：关键事件写入（登录异常/告警/实例与站点变更等）+ SSE 实时推送。
type NotificationService struct {
	db   *gorm.DB
	mu   sync.Mutex
	subs map[chan string]struct{} // SSE 订阅者（JSON 事件串）
}

// NewNotificationService 创建。
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db, subs: map[chan string]struct{}{}}
}

// Push 写入通知（供各服务调用；err 静默——通知失败不影响主流程）并广播 SSE。
func (s *NotificationService) Push(level, title, content string) {
	row := &model.Notification{Level: level, Title: title, Content: content}
	if err := s.db.Create(row).Error; err != nil {
		return
	}
	// B7：通知事件 WS 推送（保留兼容）
	wsbus.Default.Publish("notification", level, title, map[string]any{"content": content})
	s.broadcast(map[string]any{
		"type": "notification", "id": row.ID, "level": level, "title": title,
		"content": content, "createdAt": row.CreatedAt,
	})
}

// Subscribe 注册 SSE 订阅者（buffered channel，慢消费者丢帧不阻塞广播）。
func (s *NotificationService) Subscribe() chan string {
	ch := make(chan string, 16)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe 注销订阅者。
func (s *NotificationService) Unsubscribe(ch chan string) {
	s.mu.Lock()
	delete(s.subs, ch)
	s.mu.Unlock()
}

// NotifyUnreadChanged 未读数变化广播（标记已读后调用）。
func (s *NotificationService) NotifyUnreadChanged() {
	s.broadcast(map[string]any{"type": "unread", "count": s.UnreadCount()})
}

func (s *NotificationService) broadcast(payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- string(b):
		default: // 订阅者堆积（断线未感知）时丢帧
		}
	}
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
	if err := q.Update("read", true).Error; err != nil {
		return err
	}
	s.NotifyUnreadChanged()
	return nil
}
