// AI 工具扩展（对标 SQLBot/LobeHub）：数据库问答（nl2sql 只读查询）、长期记忆、会话持久化。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

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
func (s *AIService) RememberMemory(content string, scopes ...context.Context) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errs.Wrap(errs.ErrBadRequest, "记忆内容为空")
	}
	var n int64
	if err := s.db.Model(&model.AIMemory{}).Where("owner_id = ? AND content = ?", aiOwner(aiContext(scopes)), maskSensitive(content)).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	row := &model.AIMemory{OwnerID: aiOwner(aiContext(scopes)), Content: maskSensitive(content)}
	return s.db.Create(row).Error
}

// RecallMemories scores the authorized memory corpus in the database and returns five bounded candidates.
func (s *AIService) RecallMemories(query string, scopes ...context.Context) []model.AIMemory {
	ctx := aiContext(scopes)
	rows := []model.AIMemory{}
	q := aiResourceScope(ctx, s.db.WithContext(ctx).Model(&model.AIMemory{}))
	words := tokenizeKnowledgeQuery(query)
	if len(words) > 32 {
		words = words[:32]
	}
	if len(words) == 0 {
		_ = q.Order("id DESC").Limit(5).Find(&rows).Error
		return rows
	}
	score, args := aiKnowledgeScoreSQL(words, "content")
	_ = q.Select("ai_memories.*,("+score+") AS ai_score", args...).Where("("+score+") > 0", args...).Order("ai_score DESC,id DESC").Limit(5).Scan(&rows).Error
	return rows
}

// ---- 会话持久化（LobeHub 式多会话） ----

// SaveConversation 保存会话（upsert：有 ID 更新标题与消息；ID 不存在时落为新建）。
func (s *AIService) SaveConversation(ctx context.Context, id uint, title string, messages []ChatMessage) (uint, error) {
	return s.saveConversation(ctx, id, title, messages)
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
		case unicode.IsLetter(r) || unicode.IsDigit(r):
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
				t = truncText(t, 30)
			}
			return t
		}
	}
	return "新对话"
}

// DeleteAllMemories 清空全部记忆。
func (s *AIService) DeleteAllMemories(scopes ...context.Context) error {
	// GORM v2 拦截无 WHERE 的全表 Delete（ErrMissingWhereClause），清空须显式条件
	return aiScope(aiContext(scopes), s.db).Where("1 = 1").Delete(&model.AIMemory{}).Error
}

// ---- 知识文档（用户注入 md/txt → 分块检索 + read_knowledge 按需深读） ----

// SaveKnowledgeDoc 保存知识文档并切分入库（utf-8，≤2MB）。
func (s *AIService) SaveKnowledgeDoc(title, filename, content string, scopes ...context.Context) (*model.AIKnowledgeDoc, error) {
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
	doc := &model.AIKnowledgeDoc{OwnerID: aiOwner(aiContext(scopes)), Title: title, Filename: filename, Content: content}
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
func (s *AIService) ListKnowledgeDocs(scopes ...context.Context) []map[string]any {
	rows := []model.AIKnowledgeDoc{}
	_ = aiResourceScope(aiContext(scopes), s.db).Omit("content").Order("id desc").Find(&rows).Error
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		var cnt int64
		_ = s.db.Model(&model.AIKnowledgeChunk{}).Where("doc_id = ?", r.ID).Count(&cnt).Error
		out = append(out, map[string]any{
			"id": r.ID, "title": r.Title, "filename": r.Filename,
			"chunks": cnt, "createdAt": r.CreatedAt, "shared": r.Shared, "ownerId": r.OwnerID,
		})
	}
	return out
}

// GetKnowledgeDoc 文档全文。
func (s *AIService) GetKnowledgeDoc(id uint, scopes ...context.Context) (*model.AIKnowledgeDoc, error) {
	var row model.AIKnowledgeDoc
	if err := aiResourceScope(aiContext(scopes), s.db).First(&row, id).Error; err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "文档不存在")
	}
	return &row, nil
}

// DeleteKnowledgeDoc 删除文档（级联分块）。
func (s *AIService) DeleteKnowledgeDoc(id uint, scopes ...context.Context) error {
	var row model.AIKnowledgeDoc
	if err := aiScope(aiContext(scopes), s.db).First(&row, id).Error; err != nil {
		return aiNotFound()
	}
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
