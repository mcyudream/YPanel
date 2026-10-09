// AI 工具扩展（对标 SQLBot/LobeHub）：数据库问答（nl2sql 只读查询）、长期记忆、会话持久化。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// ---- 数据库问答工具（SQLBot 式 nl2sql，复用 DBAdmin 只读白名单） ----

// aiQueryDatabase AI 专用数据库查询：只读白名单 + 行数上限，失败返回错误文本供 AI 自纠。
func (s *AIService) aiQueryDatabase(ctx context.Context, instanceID uint, database, sqlText string) (string, error) {
	if s.dbSvc == nil {
		return "", fmt.Errorf("数据库服务未就绪")
	}
	res, err := s.adminSvc.RunQuery(ctx, instanceID, database, sqlText, 200)
	if err != nil {
		return "", err
	}
	// 结果紧凑化为 AI 可读文本（列名 + 行，上限 40 行）
	var b strings.Builder
	b.WriteString("列: " + strings.Join(res.Columns, " | ") + "\n")
	limit := len(res.Rows)
	if limit > 40 {
		limit = 40
	}
	for i := 0; i < limit; i++ {
		cells := make([]string, 0, len(res.Rows[i]))
		for _, c := range res.Rows[i] {
			cells = append(cells, fmt.Sprintf("%v", c))
		}
		b.WriteString("行: " + strings.Join(cells, " | ") + "\n")
	}
	if len(res.Rows) > 40 {
		b.WriteString(fmt.Sprintf("（仅显示前 40 行，共 %d 行）\n", len(res.Rows)))
	}
	b.WriteString("耗时: " + res.Elapsed)
	return b.String(), nil
}

// listDatabaseInstances 供 AI 选择实例。
func (s *AIService) listDatabaseInstances(ctx context.Context) (string, error) {
	if s.dbSvc == nil {
		return "", fmt.Errorf("数据库服务未就绪")
	}
	out, err := s.dbSvc.List(ctx)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// ---- 长期记忆（AI 自动沉淀的运维经验/用户偏好，与手动知识库区分） ----

// RememberMemory AI 调用沉淀记忆。
func (s *AIService) RememberMemory(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errs.Wrap(errs.ErrBadRequest, "记忆内容为空")
	}
	row := &model.AIMemory{Content: content}
	return s.db.Create(row).Error
}

// RecallMemories 检索与 query 相关的记忆（最近 20 条中关键词命中前 5）。
func (s *AIService) RecallMemories(query string) []model.AIMemory {
	rows := []model.AIMemory{}
	_ = s.db.Order("id desc").Limit(20).Find(&rows).Error
	if query == "" {
		return rows[:min5(len(rows))]
	}
	words := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127)
	})
	var hits []model.AIMemory
	for _, r := range rows {
		low := strings.ToLower(r.Content)
		for _, w := range words {
			if w != "" && strings.Contains(low, w) {
				hits = append(hits, r)
				break
			}
		}
	}
	if len(hits) > 5 {
		hits = hits[:5]
	}
	return hits
}

func min5(n int) int {
	if n > 5 {
		return 5
	}
	return n
}

// ---- 会话持久化（LobeHub 式多会话） ----

// SaveConversation 保存会话（upsert：有 ID 更新标题与消息；ID 不存在时落为新建）。
func (s *AIService) SaveConversation(ctx context.Context, id uint, title string, messages []ChatMessage) (uint, error) {
	b, _ := json.Marshal(messages)
	if id > 0 {
		var row model.AIConversation
		if err := s.db.First(&row, id).Error; err == nil {
			// title 为空表示沿用原标题（前端自动保存不传标题）
			updates := map[string]any{"messages": string(b)}
			if title != "" {
				updates["title"] = title
			}
			_ = s.db.Model(&row).Updates(updates).Error
			return id, nil
		}
		// 会话已不存在（如被其他端删除）：落为新建
	}
	if title == "" {
		title = firstUserSnippet(messages)
	}
	row := &model.AIConversation{Title: title, Messages: string(b)}
	if err := s.db.Create(row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// ListConversations 会话列表（不含消息体）。
func (s *AIService) ListConversations() []map[string]any {
	rows := []model.AIConversation{}
	_ = s.db.Order("updated_at desc").Limit(50).Find(&rows).Error
	out := []map[string]any{}
	for _, r := range rows {
		out = append(out, map[string]any{"id": r.ID, "title": r.Title, "updatedAt": r.UpdatedAt})
	}
	return out
}

// GetConversation 读取会话消息。
func (s *AIService) GetConversation(id uint) (map[string]any, error) {
	var row model.AIConversation
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "会话不存在")
	}
	var msgs []ChatMessage
	_ = json.Unmarshal([]byte(row.Messages), &msgs)
	return map[string]any{"id": row.ID, "title": row.Title, "messages": msgs}, nil
}

// DeleteConversation 删除会话。
func (s *AIService) DeleteConversation(id uint) error {
	return s.db.Delete(&model.AIConversation{}, id).Error
}

// tokenizeKnowledgeQuery 检索分词：ASCII 字母数字成词，CJK 逐字（去重）。
// 原实现用 FieldsFunc 且保留所有 >127 字符，整句中文（含标点）会连成一个词，永远无法命中。
func tokenizeKnowledgeQuery(query string) []string {
	seen := map[string]bool{}
	var words []string
	var ascii strings.Builder
	flush := func() {
		if ascii.Len() > 0 {
			w := strings.ToLower(ascii.String())
			if !seen[w] {
				seen[w] = true
				words = append(words, w)
			}
			ascii.Reset()
		}
	}
	for _, r := range query {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			ascii.WriteRune(r)
		case r > 127:
			flush()
			w := string(r)
			if !seen[w] {
				seen[w] = true
				words = append(words, w)
			}
		default:
			flush()
		}
	}
	flush()
	return words
}

func firstUserSnippet(messages []ChatMessage) string {
	for _, m := range messages {
		if m.Role == "user" {
			t := strings.TrimSpace(m.Content)
			if len(t) > 30 {
				t = t[:30]
			}
			return t
		}
	}
	return "新对话"
}

// DeleteAllMemories 清空全部记忆。
func (s *AIService) DeleteAllMemories() error {
	return s.db.Delete(&model.AIMemory{}).Error
}

// ---- 知识文档（用户注入 md/txt → 分块检索 + read_knowledge 按需深读） ----

// SaveKnowledgeDoc 保存知识文档并切分入库（utf-8，≤2MB）。
func (s *AIService) SaveKnowledgeDoc(title, filename, content string) (*model.AIKnowledgeDoc, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errWrapAI("文档内容为空")
	}
	if len(content) > 2<<20 {
		return nil, errWrapAI("文档过大（上限 2MB）")
	}
	if title == "" {
		title = filename
		if i := strings.LastIndex(title, "."); i > 0 {
			title = title[:i]
		}
	}
	doc := &model.AIKnowledgeDoc{Title: title, Filename: filename, Content: content}
	chunks := chunkKnowledgeDoc(content)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			return err
		}
		for i := range chunks {
			chunks[i].DocID = doc.ID
			chunks[i].Idx = i
		}
		if len(chunks) > 0 {
			return tx.Create(&chunks).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// ListKnowledgeDocs 文档列表（不含正文）。
func (s *AIService) ListKnowledgeDocs() []map[string]any {
	rows := []model.AIKnowledgeDoc{}
	_ = s.db.Order("id desc").Find(&rows).Error
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		var cnt int64
		_ = s.db.Model(&model.AIKnowledgeChunk{}).Where("doc_id = ?", r.ID).Count(&cnt).Error
		out = append(out, map[string]any{
			"id": r.ID, "title": r.Title, "filename": r.Filename,
			"chunks": cnt, "createdAt": r.CreatedAt,
		})
	}
	return out
}

// GetKnowledgeDoc 文档全文。
func (s *AIService) GetKnowledgeDoc(id uint) (*model.AIKnowledgeDoc, error) {
	var row model.AIKnowledgeDoc
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "文档不存在")
	}
	return &row, nil
}

// DeleteKnowledgeDoc 删除文档（级联分块）。
func (s *AIService) DeleteKnowledgeDoc(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("doc_id = ?", id).Delete(&model.AIKnowledgeChunk{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AIKnowledgeDoc{}, id).Error
	})
}

// chunkKnowledgeDoc 文本分块：markdown 标题行（#+）作为章节路径，段落聚合到 ~600 字成块。
func chunkKnowledgeDoc(content string) []model.AIKnowledgeChunk {
	type block struct {
		heading string
		buf     strings.Builder
	}
	out := []model.AIKnowledgeChunk{}
	cur := &block{heading: ""}
	flush := func() {
		if strings.TrimSpace(cur.buf.String()) == "" {
			return
		}
		heading := cur.heading
		if heading == "" {
			heading = "（开头）"
		}
		out = append(out, model.AIKnowledgeChunk{Heading: heading, Body: strings.TrimSpace(cur.buf.String())})
		cur.buf.Reset()
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			flush()
			cur.heading = trimmed
			continue
		}
		cur.buf.WriteString(line)
		cur.buf.WriteString("\n")
		if cur.buf.Len() >= 600 && trimmed == "" {
			flush()
			continue
		}
		if cur.buf.Len() >= 1200 {
			flush()
		}
	}
	flush()
	return out
}
