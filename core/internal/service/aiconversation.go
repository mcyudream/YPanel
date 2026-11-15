package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func aiOwner(ctx context.Context) uint { c, _ := rbac.CallerFrom(ctx); return c.UserID }
func aiAdmin(ctx context.Context) bool {
	c, ok := rbac.CallerFrom(ctx)
	return !ok || rbac.Match(c.PermSet, rbac.Wildcard)
}
func aiScope(ctx context.Context, q *gorm.DB) *gorm.DB {
	if aiAdmin(ctx) {
		return q
	}
	return q.Where("owner_id = ? AND owner_id <> 0", aiOwner(ctx))
}
func aiNotFound() error { return errs.New(errs.CodeNotFound, "error.notFound", "AI 资源不存在") }

func (s *AIService) ownedConversation(ctx context.Context, id uint) (*model.AIConversation, error) {
	var row model.AIConversation
	err := aiScope(ctx, s.db.WithContext(ctx)).First(&row, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, aiNotFound()
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func defaultAISettings() AIChatSettings {
	return AIChatSettings{Mode: aiModeConfirm, MaxRounds: 100, ContextLimit: 32768,
		MaxOutputTokens: 4096, AutoCompact: true, CompactThreshold: .8, RetainTurns: 4,
		Skills: []string{}, Memory: true, Knowledge: true, Tools: true}
}

func decodeAISettings(raw string) AIChatSettings {
	out := defaultAISettings()
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func validateAISettings(v AIChatSettings) error {
	if v.ContextLimit < 2048 || v.ContextLimit > 2_000_000 || v.MaxOutputTokens < 128 || v.MaxOutputTokens >= v.ContextLimit {
		return errWrapAI("上下文需为 2048~2000000 tokens，输出上限需小于上下文且至少 128")
	}
	if v.CompactThreshold < .5 || v.CompactThreshold > .95 || v.RetainTurns < 1 || v.RetainTurns > 20 || v.MaxRounds < 10 || v.MaxRounds > 500 {
		return errWrapAI("压缩阈值需为 50%~95%，保留轮次需为 1~20，工具轮次需为 10~500")
	}
	if len(v.SystemPrompt) > 32000 || len(v.Skills) > 30 {
		return errWrapAI("提示词或技能选择过长")
	}
	if v.Temperature != nil && (*v.Temperature < 0 || *v.Temperature > 2) {
		return errWrapAI("temperature 需为 0~2")
	}
	return nil
}

func (s *AIService) AIChatDefaults(ctx context.Context) AIChatSettings {
	raw := ""
	if s.settings != nil {
		raw = s.settings.Get(fmt.Sprintf("ai.user.%d.defaults", aiOwner(ctx)), "")
	}
	cfg := decodeAISettings(raw)
	if raw == "" {
		if s.settings != nil {
			cfg.MaxRounds = s.maxToolRoundsOf()
		}
		if p, err := s.DefaultProvider(); err == nil {
			cfg.ProviderID = p.ID
			cfg.Model = p.Model
			c := modelCapabilities(p)
			if c.ContextLimit > 0 {
				cfg.ContextLimit = c.ContextLimit
			}
			if c.MaxOutputTokens > 0 {
				cfg.MaxOutputTokens = c.MaxOutputTokens
			}
			cfg.MaxOutputTokens = min(cfg.MaxOutputTokens, cfg.ContextLimit-128)
		}
	}
	return cfg
}

func (s *AIService) SaveAIChatDefaults(ctx context.Context, v AIChatSettings) error {
	if err := validateAISettings(v); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.settings.Set(fmt.Sprintf("ai.user.%d.defaults", aiOwner(ctx)), string(b))
}

// migrateConversation 保留旧 JSON，仅在标准消息表无行时幂等导入。
func (s *AIService) conversationMessages(ctx context.Context, row *model.AIConversation) ([]ChatMessage, error) {
	var records []model.AIMessage
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", row.ID).Order("position").Find(&records).Error; err != nil {
		return nil, err
	}
	if len(records) == 0 && strings.TrimSpace(row.Messages) != "" && row.Messages != "null" {
		var legacy []ChatMessage
		if err := json.Unmarshal([]byte(row.Messages), &legacy); err != nil {
			return nil, fmt.Errorf("历史消息格式错误，原始记录已保留: %w", err)
		}
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for i := range legacy {
				legacy[i].ID = fmt.Sprintf("legacy-%d-%d", row.ID, i)
				if legacy[i].CreatedAt.IsZero() {
					legacy[i].CreatedAt = row.CreatedAt
				}
				if err := saveAIMessage(tx, row.ID, i, legacy[i]); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return legacy, nil
	}
	out := make([]ChatMessage, 0, len(records))
	for _, r := range records {
		var m ChatMessage
		if err := json.Unmarshal([]byte(r.Body), &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func saveAIMessage(tx *gorm.DB, convo uint, pos int, m ChatMessage) error {
	if m.Steps == nil {
		m.Steps = []ChatStep{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"body"})}).Create(&model.AIMessage{ID: m.ID, ConversationID: convo, Position: pos, Body: string(b), CreatedAt: m.CreatedAt}).Error
}

func (s *AIService) saveConversation(ctx context.Context, id uint, title string, messages []ChatMessage) (uint, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return s.saveConversationLocked(ctx, id, title, messages)
}
func (s *AIService) saveConversationLocked(ctx context.Context, id uint, title string, messages []ChatMessage) (uint, error) {
	if id != 0 {
		if _, err := s.ownedConversation(ctx, id); err != nil {
			return 0, err
		}
		// 兼容旧客户端自动保存；历史只由服务端追加，绝不整体覆盖。
		if strings.TrimSpace(title) != "" {
			return id, s.db.WithContext(ctx).Model(&model.AIConversation{}).Where("id = ?", id).Update("title", truncText(strings.TrimSpace(title), 128)).Error
		}
		return id, nil
	}
	row := model.AIConversation{OwnerID: aiOwner(ctx), Title: title, Version: 1}
	if row.Title == "" {
		row.Title = firstUserSnippet(messages)
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for i := range messages {
			messages[i].ID = "msg-" + randAskID()
			messages[i].CreatedAt = time.Now()
			if err := saveAIMessage(tx, row.ID, i, messages[i]); err != nil {
				return err
			}
		}
		return nil
	})
	return row.ID, err
}

type AIConversationQuery struct {
	Search         string
	Page, PageSize int
	Archived       bool
}

func (s *AIService) ConversationPage(ctx context.Context, in AIConversationQuery) (map[string]any, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize < 1 || in.PageSize > 100 {
		in.PageSize = 40
	}
	q := aiScope(ctx, s.db.WithContext(ctx).Model(&model.AIConversation{})).Where("archived = ?", in.Archived)
	if kw := strings.TrimSpace(in.Search); kw != "" {
		q = q.Where("title LIKE ?", "%"+kw+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	rows := []model.AIConversation{}
	if err := q.Select("id", "title", "version", "pinned", "archived", "updated_at").Order("pinned DESC, updated_at DESC, id DESC").Offset((in.Page - 1) * in.PageSize).Limit(in.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{"id": r.ID, "title": r.Title, "version": r.Version, "pinned": r.Pinned, "archived": r.Archived, "updatedAt": r.UpdatedAt, "runStatus": s.ConversationRunStatus(r.ID)})
	}
	return map[string]any{"items": items, "total": total, "page": in.Page, "pageSize": in.PageSize}, nil
}

func (s *AIService) ConversationSnapshot(ctx context.Context, id uint, before, limit int) (*AIChatSnapshot, error) {
	row, err := s.ownedConversation(ctx, id)
	if err != nil {
		return nil, err
	}
	if before == 0 {
		if v, ok := s.runs.Load(id); ok {
			return v.(*chatRun).snapshot(), nil
		}
	}

	var count int64
	if err := s.db.Model(&model.AIMessage{}).Where("conversation_id = ?", id).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 && row.Messages != "" {
		if _, err := s.conversationMessages(ctx, row); err != nil {
			return nil, err
		}
		s.db.Model(&model.AIMessage{}).Where("conversation_id = ?", id).Count(&count)
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	end := int(count)
	if before > 0 && before < end {
		end = before
	}
	var records []model.AIMessage
	if err := s.db.Where("conversation_id = ? AND position < ?", id, end).Order("position desc").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}
	messages := make([]ChatMessage, len(records))
	for i, r := range records {
		if err := json.Unmarshal([]byte(r.Body), &messages[len(records)-1-i]); err != nil {
			return nil, err
		}
	}
	status := "idle"
	var run model.AIRun
	usage := AIContextUsage{Summary: row.Summary, SummaryThrough: row.SummaryThrough, Estimated: true}
	if err := s.db.Where("conversation_id = ?", id).Order("created_at DESC").First(&run).Error; err == nil {
		status = run.Status
		_ = json.Unmarshal([]byte(run.Context), &usage)
	}
	usage.Summary = row.Summary
	usage.SummaryThrough = row.SummaryThrough
	settings := decodeAISettings(row.Settings)
	if usage.Limit == 0 {
		usage.Limit = settings.ContextLimit
		usage.OutputReserve = settings.MaxOutputTokens
	}
	return &AIChatSnapshot{ConversationID: id, Title: row.Title, Version: row.Version, RunID: run.ID, Status: status, Messages: messages, Total: int(count), HasMore: end > len(messages), Settings: settings, Context: usage}, nil
}

func (s *AIService) UpdateConversation(ctx context.Context, id uint, title *string, pinned, archived *bool, settings *AIChatSettings) error {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if _, err := s.ownedConversation(ctx, id); err != nil {
		return err
	}
	updates := map[string]any{}
	if title != nil {
		if strings.TrimSpace(*title) == "" || !utf8.ValidString(*title) {
			return errWrapAI("会话标题不能为空")
		}
		updates["title"] = truncText(strings.TrimSpace(*title), 128)
	}
	if pinned != nil {
		updates["pinned"] = *pinned
	}
	if archived != nil {
		updates["archived"] = *archived
	}
	if settings != nil {
		if s.ChatRunStatus(id) == AIRunStatusRunning {
			return errWrapAI("请先停止当前生成再修改会话设置")
		}
		if err := validateAISettings(*settings); err != nil {
			return err
		}
		b, _ := json.Marshal(settings)
		updates["settings"] = string(b)
	}
	if len(updates) == 0 {
		return nil
	}
	updates["version"] = gorm.Expr("version + 1")
	if err := s.db.WithContext(ctx).Model(&model.AIConversation{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	if v, ok := s.runs.Load(id); ok {
		r := v.(*chatRun)
		r.mu.Lock()
		if title != nil {
			r.title = *title
		}
		if settings != nil {
			r.settings = *settings
		}
		r.version++
		r.mu.Unlock()
	}
	return nil
}

func (s *AIService) RemoveConversation(ctx context.Context, id uint) error {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if _, err := s.ownedConversation(ctx, id); err != nil {
		return err
	}
	if s.ChatRunStatus(id) == AIRunStatusRunning {
		return errWrapAI("请先停止任务再删除会话")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", id).Delete(&model.AIAttachmentRef{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", id).Delete(&model.AIMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", id).Delete(&model.AIRun{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AIConversation{}, id).Error
	})
	if err == nil {
		s.runs.Delete(id)
		_ = s.cleanupAttachments()
	}
	return err
}

// ForkConversation 复制到指定消息；编辑/重试使用新分支，不破坏原会话。
func (s *AIService) ForkConversation(ctx context.Context, id uint, through string, include bool) (uint, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	row, err := s.ownedConversation(ctx, id)
	if err != nil {
		return 0, err
	}
	msgs, err := s.conversationMessages(ctx, row)
	if err != nil {
		return 0, err
	}
	if through != "" {
		found := false
		for i := range msgs {
			if msgs[i].ID == through {
				end := i
				if include {
					end++
				}
				msgs = msgs[:end]
				found = true
				break
			}
		}
		if !found {
			return 0, aiNotFound()
		}
	}
	newID, err := s.saveConversationLocked(ctx, 0, truncText(row.Title, 100)+" · 分支", msgs)
	if err == nil {
		err = s.db.Model(&model.AIConversation{}).Where("id = ?", newID).Update("settings", row.Settings).Error
		if err == nil {
			for _, m := range msgs {
				ids := []string{}
				for _, a := range m.Attachments {
					ids = append(ids, a.ID)
				}
				for _, seg := range m.Segments {
					if seg.OutputID != "" {
						ids = append(ids, seg.OutputID)
					}
				}
				for _, id := range ids {
					if err = s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.AIAttachmentRef{AttachmentID: id, ConversationID: newID}).Error; err != nil {
						return newID, err
					}
				}
			}
		}
	}

	return newID, err
}
