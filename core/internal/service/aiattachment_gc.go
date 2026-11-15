package service

import (
	"github.com/ypanel/core/internal/model"
	"os"
	"path/filepath"
	"time"
)

// Only unreferenced files older than 24h are collected, including abandoned uploads.
func (s *AIService) cleanupAttachments() error {
	var rows []model.AIAttachment
	query := s.db.Where("created_at < ? AND id NOT IN (?)", time.Now().Add(-24*time.Hour), s.db.Model(&model.AIAttachmentRef{}).Select("attachment_id"))
	// Legacy rows without explicit references are kept while their conversation exists.
	query = query.Where("conversation_id = 0 OR conversation_id NOT IN (?)", s.db.Model(&model.AIConversation{}).Select("id"))
	if err := query.Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	root, err := s.attachmentRoot()
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.Path != r.ID || filepath.Base(r.Path) != r.Path {
			continue
		}
		if err := os.Remove(filepath.Join(root, r.Path)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := s.db.Delete(&r).Error; err != nil {
			return err
		}
	}
	return nil
}
