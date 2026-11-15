package model

import "time"

// AIMessage 是完整历史的真源；AIConversation.Messages 仅保留旧版原始数据。
type AIMessage struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	ConversationID uint      `gorm:"uniqueIndex:idx_ai_message_position;index" json:"conversationId"`
	Position       int       `gorm:"uniqueIndex:idx_ai_message_position" json:"position"`
	Body           string    `gorm:"type:text;not null" json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AIRun 保存运行检查点。进程重启只展示中断状态，绝不自动重做工具。
type AIRun struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	ConversationID uint      `gorm:"index" json:"conversationId"`
	UserID         uint      `gorm:"index" json:"userId"`
	RequestID      string    `gorm:"uniqueIndex:idx_ai_run_request;size:96" json:"requestId"`
	Status         string    `gorm:"index;size:24" json:"status"`
	Context        string    `gorm:"type:text" json:"-"`
	Snapshot       string    `gorm:"type:text" json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AIAttachment struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	OwnerID        uint      `gorm:"index" json:"-"`
	ConversationID uint      `gorm:"index" json:"conversationId"`
	Name           string    `gorm:"size:255" json:"name"`
	MIME           string    `gorm:"size:128" json:"mime"`
	Size           int64     `json:"size"`
	Kind           string    `gorm:"size:16" json:"kind"`
	Path           string    `gorm:"type:text" json:"-"`
	Text           string    `gorm:"type:text" json:"-"`
	CreatedAt      time.Time `gorm:"index" json:"createdAt"`
}

// References allow a fork to keep its attachments when the source conversation is deleted.
type AIAttachmentRef struct {
	AttachmentID   string `gorm:"primaryKey;size:64"`
	ConversationID uint   `gorm:"primaryKey;index"`
}
