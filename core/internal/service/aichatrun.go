package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/core/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AIRunStatusRunning = "running"
	aiRunStatusDone    = "done"
	aiRunStatusStopped = "stopped"
	aiRunStatusError   = "error"
	aiRunMaxEvents     = 8000
	aiRunTTL           = 10 * time.Minute
	aiRunTotalTimeout  = 60 * time.Minute
)

type aiRunEvent struct {
	ID   int
	Data []byte
}
type chatRun struct {
	id        string
	convoID   uint
	userID    uint
	requestID string
	cancel    context.CancelFunc
	mu        sync.Mutex
	base      int
	seq       int
	events    []aiRunEvent
	subs      map[chan aiRunEvent]struct{}
	status    string
	pending   string
	doneCh    chan struct{}
	builder   *aiMsgBuilder
	startAt   time.Time
	title     string
	version   uint
	settings  AIChatSettings
	usage     AIContextUsage
}

// emit 与快照使用同一把锁，快照内容和 lastEventID 永远来自同一时刻。
func (r *chatRun) emit(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	var ev map[string]any
	if json.Unmarshal(raw, &ev) != nil {
		return
	}
	if _, ok := ev["done"]; ok {
		return
	} // 终态帧只在持久化完成后发出
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != AIRunStatusRunning {
		return
	}
	r.builder.feed(ev)
	if _, ok := ev["ask"]; ok {
		r.pending = "ask"
	}
	if _, ok := ev["ask_result"]; ok {
		r.pending = ""
	}
	if _, ok := ev["ask_user"]; ok {
		r.pending = "question"
	}
	if _, ok := ev["ask_user_result"]; ok {
		r.pending = ""
	}
	if value, ok := ev["context"]; ok {
		b, _ := json.Marshal(value)
		_ = json.Unmarshal(b, &r.usage)
	}
	r.seq++
	ev["eventId"] = r.seq
	ev["runId"] = r.id
	raw, err = json.Marshal(ev)
	if err != nil {
		return
	}
	frame := aiRunEvent{ID: r.seq, Data: raw}
	r.events = append(r.events, frame)
	if len(r.events) > aiRunMaxEvents {
		drop := len(r.events) / 2
		r.base = r.events[drop-1].ID
		r.events = append([]aiRunEvent(nil), r.events[drop:]...)
	}
	for ch := range r.subs {
		select {
		case ch <- frame:
		default:
			close(ch)
			delete(r.subs, ch)
		}
	}
}

func (r *chatRun) ping() {} // HTTP 订阅独立发送心跳，不占业务事件序号
func (r *chatRun) snapshotLocked() *AIChatSnapshot {
	all := r.builder.snapshot()
	start := max(0, len(all)-50)
	out := &AIChatSnapshot{ConversationID: r.convoID, Title: r.title, Version: r.version,
		RunID: r.id, Status: r.status, LastEventID: r.seq, Messages: all[start:], Total: len(all),
		HasMore: start > 0, Settings: r.settings, Context: r.usage}
	// 防止解锁后切片/段对象被后续 delta 修改。
	b, _ := json.Marshal(out)
	var cp AIChatSnapshot
	_ = json.Unmarshal(b, &cp)
	return &cp
}
func (r *chatRun) snapshot() *AIChatSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshotLocked()
}

// AIChatSubscription 的 Close 捕获准确运行体，避免新运行替换后清理错订阅。
type AIChatSubscription struct {
	Snapshot *AIChatSnapshot
	Replay   []aiRunEvent
	Events   <-chan aiRunEvent
	Done     <-chan struct{}
	Close    func()
}

func (s *AIService) SubscribeChatRun(ctx context.Context, id uint, runID string, offset int) (*AIChatSubscription, error) {
	if _, err := s.ownedConversation(ctx, id); err != nil {
		return nil, err
	}
	v, ok := s.runs.Load(id)
	if !ok {
		snap, err := s.ConversationSnapshot(ctx, id, 0, 50)
		return &AIChatSubscription{Snapshot: snap, Close: func() {}}, err
	}
	r := v.(*chatRun)
	r.mu.Lock()
	defer r.mu.Unlock()
	out := &AIChatSubscription{Done: r.doneCh, Close: func() {}}
	if runID != r.id || offset < r.base || offset > r.seq || offset < 0 {
		out.Snapshot = r.snapshotLocked()
	} else {
		for _, frame := range r.events {
			if frame.ID > offset {
				out.Replay = append(out.Replay, frame)
			}
		}
	}
	if r.status != AIRunStatusRunning {
		out.Snapshot = r.snapshotLocked()
		out.Replay = nil
		return out, nil
	}
	ch := make(chan aiRunEvent, 256)
	r.subs[ch] = struct{}{}
	out.Events = ch
	var once sync.Once
	out.Close = func() {
		once.Do(func() {
			r.mu.Lock()
			if _, exists := r.subs[ch]; exists {
				delete(r.subs, ch)
				close(ch)
			}
			r.mu.Unlock()
		})
	}
	return out, nil
}

type aiMsgBuilder struct {
	msgs      []ChatMessage
	cur       *ChatMessage
	content   strings.Builder
	reasoning strings.Builder
	text      strings.Builder
}

func newAiMsgBuilder(history []ChatMessage) *aiMsgBuilder {
	return &aiMsgBuilder{msgs: append([]ChatMessage(nil), history...), cur: &ChatMessage{ID: "msg-" + randAskID(), Role: "assistant", Pending: true, Steps: []ChatStep{}, CreatedAt: time.Now()}}
}
func str(m map[string]any, k string) string { v, _ := m[k].(string); return v }
func (b *aiMsgBuilder) feed(ev map[string]any) {
	cur := b.cur
	if v, ok := ev["content"].(string); ok {
		b.content.WriteString(v)
		cur.Content = b.content.String()
		n := len(cur.Segments)
		if n == 0 || cur.Segments[n-1].Type != "text" {
			b.text.Reset()
			cur.Segments = append(cur.Segments, ChatSegment{Type: "text"})
		}
		b.text.WriteString(v)
		cur.Segments[len(cur.Segments)-1].Text = b.text.String()
	}
	if v, ok := ev["reasoning"].(string); ok {
		b.reasoning.WriteString(v)
		cur.Reasoning = b.reasoning.String()
	}
	if v, ok := ev["scene"].(map[string]any); ok {
		cur.Steps = append(cur.Steps, ChatStep{Type: "scene", Name: "读取页面数据", Detail: str(v, "page"), Done: true})
	}
	if v, ok := ev["step"].(map[string]any); ok && str(v, "type") == "tool" {
		cur.Segments = append(cur.Segments, ChatSegment{Type: "tool", ID: str(v, "id"), Name: str(v, "name"), Args: str(v, "detail"), Status: "running"})
	}
	if v, ok := ev["step_result"].(map[string]any); ok {
		for i := len(cur.Segments) - 1; i >= 0; i-- {
			seg := &cur.Segments[i]
			if seg.Type == "tool" && seg.Name == str(v, "name") && (str(v, "id") == "" || seg.ID == str(v, "id")) {
				seg.Summary = str(v, "detail")
				seg.Output = str(v, "output")
				seg.OutputID = str(v, "outputId")
				seg.Status = str(v, "status")
				seg.Done = true
				break
			}
		}
	}
	if v, ok := ev["ask"].(map[string]any); ok {
		for i := len(cur.Segments) - 1; i >= 0; i-- {
			seg := &cur.Segments[i]
			if seg.Type == "tool" && !seg.Done {
				seg.AskID = str(v, "id")
				seg.Risk = str(v, "risk")
				seg.Status = "waiting"
				break
			}
		}
	}
	if v, ok := ev["ask_result"].(map[string]any); ok {
		for i := range cur.Segments {
			seg := &cur.Segments[i]
			if seg.AskID == str(v, "id") {
				seg.Done = true
				seg.Status = "已拒绝"
				if approved, _ := v["approved"].(bool); approved {
					seg.Status = "running"
					seg.Done = false
				}
			}
		}
	}
	if v, ok := ev["ask_user"].(map[string]any); ok {
		var qs []map[string]any
		bts, _ := json.Marshal(v["questions"])
		_ = json.Unmarshal(bts, &qs)
		cur.Segments = append(cur.Segments, ChatSegment{Type: "question", Name: "ask_user", AskID: str(v, "id"), Questions: qs, Status: "waiting"})
	}
	if v, ok := ev["ask_user_result"].(map[string]any); ok {
		for i := range cur.Segments {
			seg := &cur.Segments[i]
			if seg.AskID == str(v, "id") {
				seg.Done = true
				seg.Status = "已回答"
				seg.Summary = str(v, "answer")
				if cancelled, _ := v["cancelled"].(bool); cancelled {
					seg.Status = "已取消"
				}
			}
		}
	}
	if v, ok := ev["knowledge"]; ok {
		raw, _ := json.Marshal(v)
		_ = json.Unmarshal(raw, &cur.Knowledge)
	}
	if v, ok := ev["error"].(string); ok {
		cur.Error = true
		cur.Content += "\n\n" + v
		cur.Segments = append(cur.Segments, ChatSegment{Type: "text", Text: "\n\n" + v})
	}
}
func (b *aiMsgBuilder) snapshot() []ChatMessage {
	out := append([]ChatMessage(nil), b.msgs...)
	return append(out, *b.cur)
}
func (b *aiMsgBuilder) finalize() []ChatMessage { b.cur.Pending = false; return b.snapshot() }

func (s *AIService) StartChat(ctx context.Context, in AIChatStart) (*AIChatSnapshot, error) {
	if strings.TrimSpace(in.Content) == "" && len(in.AttachmentIDs) == 0 {
		return nil, errWrapAI("消息不能为空")
	}
	if len(in.Scene) > 1000 || len(in.Focus) > 4000 || len(in.Content) > 1<<20 || len(in.AttachmentIDs) > 10 || len(in.RequestID) < 8 || len(in.RequestID) > 80 {
		return nil, errWrapAI("消息过长、附件过多或请求标识无效")
	}
	if err := validateAISettings(in.Settings); err != nil {
		return nil, err
	}
	in.Settings.Mode = normalizeAIMode(in.Settings.Mode)
	s.runMu.Lock()
	defer s.runMu.Unlock()
	requestKey := fmt.Sprintf("%d:%s", aiOwner(ctx), in.RequestID)
	var old model.AIRun
	if err := s.db.Where("request_id = ?", requestKey).First(&old).Error; err == nil {
		if old.UserID != aiOwner(ctx) {
			return nil, aiNotFound()
		}
		if in.ConversationID != 0 && in.ConversationID != old.ConversationID {
			return nil, errWrapAI("请求标识已用于其他会话")
		}
		return s.ConversationSnapshot(ctx, old.ConversationID, 0, 50)
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	var row *model.AIConversation
	var err error
	if in.ConversationID > 0 {
		row, err = s.ownedConversation(ctx, in.ConversationID)
		if err != nil {
			return nil, err
		}
		if s.ChatRunStatus(row.ID) == AIRunStatusRunning {
			return nil, errWrapAI("该会话有任务正在进行")
		}
		if in.Version != nil && row.Version != *in.Version {
			return nil, errWrapAI("会话已更新，请重新载入后发送")
		}
	} else {
		row = &model.AIConversation{OwnerID: aiOwner(ctx), Title: truncText(strings.TrimSpace(in.Content), 60)}
	}
	var provider *model.AIProvider
	if in.Settings.ProviderID > 0 {
		provider, err = s.ProviderByID(in.Settings.ProviderID)
	} else {
		provider, err = s.DefaultProvider()
	}
	if err != nil {
		return nil, err
	}
	cp := *provider
	provider = &cp
	if in.Settings.Model != "" {
		provider.Model = in.Settings.Model
	}
	in.Settings.ProviderID = provider.ID
	in.Settings.Model = provider.Model
	cap := modelCapabilities(provider)
	if cap.ContextLimit > 0 && in.Settings.ContextLimit > cap.ContextLimit {
		return nil, errWrapAI("会话窗口超过已配置的模型上限，请调整会话设置")
	}
	if cap.MaxOutputTokens > 0 && in.Settings.MaxOutputTokens > cap.MaxOutputTokens {
		return nil, errWrapAI("输出预算超过已配置的模型上限")
	}
	if in.Settings.Temperature != nil && !cap.Temperature {
		return nil, errWrapAI("当前模型未声明支持温度参数")
	}
	history := []ChatMessage{}
	if row.ID > 0 {
		history, err = s.conversationMessages(ctx, row)
		if err != nil {
			return nil, err
		}
	}
	if !modelCapabilities(provider).Vision {
		for _, m := range history {
			for _, a := range m.Attachments {
				if a.Kind == "image" {
					return nil, errWrapAI("历史含图片，当前模型未声明视觉能力。请选择视觉模型或从图片之前分叉。")
				}
			}
		}
	}
	attachments, err := s.chatAttachments(ctx, row.ID, in.AttachmentIDs, provider)
	if err != nil {
		return nil, err
	}
	user := ChatMessage{ID: "msg-" + randAskID(), Role: "user", Content: strings.TrimSpace(in.Content), Attachments: attachments, Steps: []ChatStep{}, CreatedAt: time.Now()}
	history = append(history, user)
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), aiRunTotalTimeout)
	r := &chatRun{id: randAskID(), userID: aiOwner(ctx), requestID: requestKey, cancel: cancel, subs: map[chan aiRunEvent]struct{}{}, status: AIRunStatusRunning, doneCh: make(chan struct{}), builder: newAiMsgBuilder(history), startAt: time.Now(), settings: in.Settings}
	rawSettings, _ := json.Marshal(in.Settings)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.ID == 0 {
			if row.Title == "" {
				row.Title = "新对话"
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		row.Version++
		row.Settings = string(rawSettings)
		if err := tx.Model(row).Updates(map[string]any{"version": row.Version, "settings": row.Settings, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		r.convoID = row.ID
		r.title = row.Title
		r.version = row.Version
		if err := saveAIMessage(tx, row.ID, len(history)-1, user); err != nil {
			return err
		}
		if err := saveAIMessage(tx, row.ID, len(history), *r.builder.cur); err != nil {
			return err
		}
		if len(in.AttachmentIDs) > 0 {
			if err := tx.Model(&model.AIAttachment{}).Where("id IN ? AND conversation_id = 0", in.AttachmentIDs).Update("conversation_id", row.ID).Error; err != nil {
				return err
			}
		}
		for _, id := range in.AttachmentIDs {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.AIAttachmentRef{AttachmentID: id, ConversationID: row.ID}).Error; err != nil {
				return err
			}
		}
		b, _ := json.Marshal(r.builder.cur)
		return tx.Create(&model.AIRun{ID: r.id, ConversationID: row.ID, UserID: r.userID, RequestID: requestKey, Status: AIRunStatusRunning, Snapshot: string(b)}).Error
	})
	if err != nil {
		cancel()
		return nil, err
	}
	s.runs.Store(row.ID, r)
	snap := r.snapshot()
	go s.executeChatRun(runCtx, r, provider, history, in.Scene, in.Focus)
	return snap, nil
}

func (s *AIService) checkpointRun(r *chatRun, status string) error {
	r.mu.Lock()
	b, err := json.Marshal(r.builder.cur)
	usage, _ := json.Marshal(r.usage)
	record := model.AIMessage{ID: r.builder.cur.ID, ConversationID: r.convoID, Position: len(r.builder.msgs), Body: string(b), CreatedAt: r.builder.cur.CreatedAt}
	r.mu.Unlock()
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AIMessage{}).Where("id = ? AND conversation_id = ?", record.ID, record.ConversationID).Update("body", record.Body).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AIConversation{}).Where("id = ?", r.convoID).Update("updated_at", time.Now()).Error; err != nil {
			return err
		}
		return tx.Model(&model.AIRun{}).Where("id = ?", r.id).Updates(map[string]any{"status": status, "snapshot": string(b), "context": string(usage)}).Error
	})
}

func (s *AIService) executeChatRun(ctx context.Context, r *chatRun, provider *model.AIProvider, history []ChatMessage, scene, focus string) {
	ctx = context.WithValue(ctx, aiRunContextKey{}, aiRunIdentity{UserID: r.userID, ConversationID: r.convoID, RunID: r.id})
	status := aiRunStatusDone
	checkpointStop := make(chan struct{})
	checkpointDone := make(chan struct{})
	go func() {
		defer close(checkpointDone)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := s.checkpointRun(r, AIRunStatusRunning); err != nil {
					slog.Error("ai checkpoint failed", "run", r.id, "err", err)
				}
			case <-checkpointStop:
				return
			}
		}
	}()
	defer func() {
		if p := recover(); p != nil {
			status = aiRunStatusError
			slog.Error("ai run panic", "run", r.id, "type", fmt.Sprintf("%T", p))
			r.emit(map[string]string{"error": "任务异常终止，已保留当前进度"})
		}
		close(checkpointStop)
		<-checkpointDone
		r.mu.Lock()
		r.builder.cur.Pending = false
		r.builder.cur.DurationMS = time.Since(r.startAt).Milliseconds()
		for i := range r.builder.cur.Segments {
			seg := &r.builder.cur.Segments[i]
			if !seg.Done && (seg.Type == "tool" || seg.Type == "question") {
				seg.Done = true
				seg.Status = "已中断"
			}
		}
		r.mu.Unlock()
		if err := s.checkpointRun(r, status); err != nil {
			status = aiRunStatusError
			r.emit(map[string]string{"error": "保存失败，请导出当前回答后重试"})
			slog.Error("ai final persistence failed", "run", r.id, "err", err)
		}
		r.mu.Lock()
		r.status = status
		r.pending = ""
		r.seq++
		raw, _ := json.Marshal(map[string]any{"done": true, "status": status, "runId": r.id, "eventId": r.seq, "durationMs": r.builder.cur.DurationMS})
		frame := aiRunEvent{ID: r.seq, Data: raw}
		r.events = append(r.events, frame)
		for ch := range r.subs {
			select {
			case ch <- frame:
			default:
			}
			close(ch)
			delete(r.subs, ch)
		}
		close(r.doneCh)
		r.mu.Unlock()
		r.cancel()
		time.AfterFunc(aiRunTTL, func() { s.runs.CompareAndDelete(r.convoID, r) })
		slog.Info("ai run finished", "conversation", r.convoID, "run", r.id, "status", status)
	}()
	if err := s.RunAgentChat(ctx, r.emit, r.ping, provider, history, scene, focus, r.settings.Mode, r.settings.MaxRounds, r.settings); err != nil {
		status = aiRunStatusError
		if ctx.Err() != nil {
			status = aiRunStatusStopped
		} else {
			r.emit(map[string]string{"error": maskSensitive(err.Error())})
		}
	}
}

func (s *AIService) StopChat(ctx context.Context, id uint, expectedRun ...string) error {
	if _, err := s.ownedConversation(ctx, id); err != nil {
		return err
	}
	if v, ok := s.runs.Load(id); ok {
		r := v.(*chatRun)
		if len(expectedRun) > 0 && expectedRun[0] != "" && expectedRun[0] != r.id {
			return errWrapAI("会话运行已变更，请重新载入当前任务")
		}
		r.cancel()
		return nil
	}
	return errWrapAI("没有进行中的任务")
}
func (s *AIService) ConversationRunStatus(id uint) string {
	v, ok := s.runs.Load(id)
	if !ok {
		return ""
	}
	r := v.(*chatRun)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != AIRunStatusRunning {
		return ""
	}
	switch r.pending {
	case "ask":
		return "waiting_confirm"
	case "question":
		return "waiting_answer"
	default:
		return "running"
	}
}
func (s *AIService) ChatRunStatus(id uint) string {
	v, ok := s.runs.Load(id)
	if !ok {
		return ""
	}
	r := v.(*chatRun)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (s *AIService) RecoverChatRuns() error {
	var rows []model.AIRun
	if err := s.db.Where("status = ?", AIRunStatusRunning).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			var m ChatMessage
			if err := json.Unmarshal([]byte(row.Snapshot), &m); err != nil {
				return err
			}
			m.Pending = false
			m.Interrupted = true
			for i := range m.Segments {
				if !m.Segments[i].Done {
					m.Segments[i].Done = true
					m.Segments[i].Status = "已中断"
				}
			}
			raw, err := json.Marshal(m)
			if err != nil {
				return err
			}
			if err := tx.Model(&model.AIMessage{}).Where("id = ? AND conversation_id = ?", m.ID, row.ConversationID).Update("body", string(raw)).Error; err != nil {
				return err
			}
			return tx.Model(&model.AIRun{}).Where("id = ?", row.ID).Updates(map[string]any{"status": "interrupted", "snapshot": string(raw)}).Error
		}); err != nil {
			return err
		}
	}
	return s.cleanupAttachments()
}
