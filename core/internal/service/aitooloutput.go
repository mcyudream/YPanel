package service

import (
	"context"
	"github.com/ypanel/core/internal/model"
	"gorm.io/gorm/clause"
	"strings"
)

func (s *AIService) persistAIToolOutput(ctx context.Context, name, result string) (string, string) {
	masked := maskSensitive(result)
	if len(masked) <= toolOutputTrunc {
		return masked, ""
	}
	ident, _ := ctx.Value(aiRunContextKey{}).(aiRunIdentity)
	if ident.ConversationID == 0 || s.dataDir == "" {
		return masked, ""
	}
	if len(masked) > aiAttachmentLimit {
		masked = truncText(masked, aiAttachmentLimit-100) + "\n[输出超过 20MB，已明确截断]"
	}
	a, err := s.UploadChatAttachment(ctx, "tool-"+name+".txt", strings.NewReader(masked))
	if err != nil {
		return masked + "\n[独立结果文件保存失败，已在消息中保留内容]", ""
	}
	if err = s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.AIAttachmentRef{AttachmentID: a.ID, ConversationID: ident.ConversationID}).Error; err != nil {
		return masked, ""
	}
	return truncText(masked, toolOutputTrunc) + "\n[完整脱敏结果可下载]", a.ID
}
