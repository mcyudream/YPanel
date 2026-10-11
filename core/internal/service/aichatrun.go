// AI 聊天任务运行体（M62）：任务脱离 HTTP 连接在后端 goroutine 运行，
// 事件帧进内存总线（回放 buffer + 多订阅者广播）——换窗口/强刷新可重连介入。
// 生命周期：POST /ai/chat/start 创建 → goroutine 跑 RunAgentChat → 结束统一落库
// → 保留 10min 供迟订阅回放 → 惰性清理；面板重启则运行中任务随进程终止（内存态）。
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/ypanel/core/internal/model"
)

const (
	AIRunStatusRunning = "running"
	aiRunStatusDone    = "done"
	aiRunStatusStopped = "stopped"
	aiRunStatusError   = "error"

	aiRunMaxEvents    = 8000            // 单任务事件帧回放上限保险丝（超限截半）
	aiRunTTL          = 10 * time.Minute // 结束后保留时长（迟订阅回放窗口）
	aiRunTotalTimeout = 30 * time.Minute // 单任务总上限（防泄漏）
)

// chatRun 一轮对话任务的运行体。emit 仅由任务 goroutine 单线程调用。
type chatRun struct {
	id      string
	convoID uint
	cancel  context.CancelFunc

	mu     sync.Mutex
	base   int        // events 已截掉的帧数（客户端 offset = 全局序号）
	events [][]byte   // 事件帧（JSON 原文，不含 SSE 信封）
	subs   map[chan []byte]struct{}
	status string
	doneCh chan struct{} // finish 时关闭（订阅端感知收尾，避免悬挂）

	builder *aiMsgBuilder
	startAt time.Time
}

// emit 记录并广播一帧事件（帧同时喂给消息重建器）。
func (r *chatRun) emit(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	r.builder.feed(b)
	r.mu.Lock()
	r.events = append(r.events, b)
	if len(r.events) > aiRunMaxEvents {
		drop := len(r.events) / 2
		r.events = append([][]byte(nil), r.events[drop:]...)
		r.base += drop
	}
	for ch := range r.subs {
		select {
		case ch <- b:
		default: // 慢订阅丢帧：渲染容错，任务不受影响
		}
	}
	r.mu.Unlock()
}

// ping 心跳帧只推订阅者（注释帧不入回放、不参与 offset 计数）。
func (r *chatRun) ping() {
	b := []byte(": ping\n\n")
	r.mu.Lock()
	for ch := range r.subs {
		select {
		case ch <- b:
		default:
		}
	}
	r.mu.Unlock()
}

// subscribe 注册订阅者并回放 events[offset:]（同锁内快照，不丢注册间隙的帧）。
func (r *chatRun) subscribe(offset int) (replay [][]byte, ch chan []byte) {
	ch = make(chan []byte, 1024)
	r.mu.Lock()
	defer r.mu.Unlock()
	from := offset - r.base
	if from < 0 {
		from = 0
	}
	if from <= len(r.events) {
		replay = append([][]byte(nil), r.events[from:]...)
	}
	r.subs[ch] = struct{}{}
	return replay, ch
}

func (r *chatRun) unsubscribe(ch chan []byte) {
	r.mu.Lock()
	delete(r.subs, ch)
	r.mu.Unlock()
}

func (r *chatRun) finish(status string) {
	r.mu.Lock()
	if r.status == AIRunStatusRunning {
		r.status = status
		close(r.doneCh)
	}
	r.mu.Unlock()
}

// ---- 消息重建器：从事件帧流重建可持久化的消息链（与前端 useAiChat 帧处理同构） ----

type aiMsgBuilder struct {
	msgs    []ChatMessage
	cur     *ChatMessage
	askIdx  map[string]int // askID → cur.Steps 下标
	quesIdx map[string]int // questionID → cur.Steps 下标
}

func newAiMsgBuilder(history []ChatMessage) *aiMsgBuilder {
	msgs := append([]ChatMessage(nil), history...)
	return &aiMsgBuilder{
		msgs:    msgs,
		cur:     &ChatMessage{Role: "assistant"},
		askIdx:  map[string]int{},
		quesIdx: map[string]int{},
	}
}

// feed 处理一帧事件（JSON 原文）。
func (b *aiMsgBuilder) feed(raw []byte) {
	var ev map[string]any
	if json.Unmarshal(raw, &ev) != nil {
		return
	}
	cur := b.cur
	if v, ok := ev["scene"].(map[string]any); ok {
		page, _ := v["page"].(string)
		cur.Steps = append(cur.Steps, ChatStep{Type: "scene", Name: "读取页面数据", Detail: page, Done: true})
	}
	if v, ok := ev["reasoning"].(string); ok {
		cur.Reasoning += v
	}
	if v, ok := ev["step"].(map[string]any); ok {
		if t, _ := v["type"].(string); t == "tool" {
			name, _ := v["name"].(string)
			detail, _ := v["detail"].(string)
			cur.Steps = append(cur.Steps, ChatStep{Type: "tool", Name: name, Detail: detail})
		}
	}
	if v, ok := ev["step_result"].(map[string]any); ok {
		name, _ := v["name"].(string)
		status, _ := v["status"].(string)
		for i := len(cur.Steps) - 1; i >= 0; i-- {
			st := &cur.Steps[i]
			if st.Type == "tool" && st.Name == name && !st.Done {
				st.Done = true
				st.Status = status
				break
			}
		}
	}
	if v, ok := ev["ask"].(map[string]any); ok {
		id, _ := v["id"].(string)
		tool, _ := v["tool"].(string)
		args, _ := v["args"].(string)
		b.askIdx[id] = len(cur.Steps)
		cur.Steps = append(cur.Steps, ChatStep{Type: "ask", Name: tool, Detail: args})
	}
	if v, ok := ev["ask_result"].(map[string]any); ok {
		id, _ := v["id"].(string)
		approved, _ := v["approved"].(bool)
		if i, has := b.askIdx[id]; has && i < len(cur.Steps) {
			cur.Steps[i].Done = true
			if approved {
				cur.Steps[i].Status = "已确认"
			} else if reason, _ := v["reason"].(string); reason == "timeout" {
				cur.Steps[i].Status = "已超时"
			} else {
				cur.Steps[i].Status = "已拒绝"
			}
		}
	}
	if v, ok := ev["ask_user"].(map[string]any); ok {
		id, _ := v["id"].(string)
		detail := ""
		if qs, ok := v["questions"].([]any); ok {
			for i, q := range qs {
				if qm, ok := q.(map[string]any); ok {
					if i > 0 {
						detail += "；"
					}
					q, _ := qm["question"].(string)
					detail += q
				}
			}
		}
		b.quesIdx[id] = len(cur.Steps)
		cur.Steps = append(cur.Steps, ChatStep{Type: "question", Name: "询问用户", Detail: detail})
	}
	if v, ok := ev["ask_user_result"].(map[string]any); ok {
		id, _ := v["id"].(string)
		if i, has := b.quesIdx[id]; has && i < len(cur.Steps) {
			cur.Steps[i].Done = true
			cur.Steps[i].Status = "已取消"
		}
	}
	if v, ok := ev["knowledge"].([]any); ok {
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				t, _ := m["title"].(string)
				body, _ := m["body"].(string)
				cur.Knowledge = append(cur.Knowledge, ChatKnowledge{Title: t, Body: body})
			}
		}
	}
	if v, ok := ev["content"].(string); ok {
		cur.Content += v
	}
	if v, ok := ev["error"].(string); ok {
		cur.Content += "\n\n" + v
	}
}

// finalize 收尾：assistant 有产出才追加，返回完整消息链。
func (b *aiMsgBuilder) finalize() []ChatMessage {
	if b.cur.Content != "" || len(b.cur.Steps) > 0 || b.cur.Reasoning != "" {
		b.msgs = append(b.msgs, *b.cur)
		b.cur = &ChatMessage{Role: "assistant"}
	}
	return b.msgs
}

// ---- 服务侧：注册表 / 启动 / 停止 / 订阅 ----

// StartChatRun 启动一轮后台对话任务，立即返回会话 ID。
// caller 携带 RBAC 身份（WithoutCancel 派生：HTTP 连接断开不影响任务）。
func (s *AIService) StartChatRun(
	caller context.Context,
	convoID uint,
	provider *model.AIProvider,
	history []ChatMessage,
	scenePath, focus, mode string,
) (uint, error) {
	// 同会话并发拒绝（运行体结束后 10min TTL 内仍可回放，但状态非 running 可再次发起）
	if v, ok := s.runs.Load(convoID); ok && convoID != 0 {
		if r, ok := v.(*chatRun); ok && r != nil {
			r.mu.Lock()
			st := r.status
			r.mu.Unlock()
			if st == AIRunStatusRunning {
				return convoID, errWrapAI("该会话有 AI 任务进行中，请等待完成或停止后再发送")
			}
		}
	}

	if convoID == 0 && s.db != nil {
		row := &model.AIConversation{Title: firstUserSnippet(history)}
		if err := s.db.Create(row).Error; err != nil {
			return 0, errWrapAI("创建会话失败: " + err.Error())
		}
		convoID = row.ID
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(caller), aiRunTotalTimeout)
	run := &chatRun{
		id:      randAskID(),
		convoID: convoID,
		cancel:  cancel,
		subs:    map[chan []byte]struct{}{},
		status:  AIRunStatusRunning,
		doneCh:  make(chan struct{}),
		builder: newAiMsgBuilder(history),
		startAt: time.Now(),
	}
	s.runs.Store(convoID, run)
	slog.Info("ai chat run started", "convo", convoID, "run", run.id, "mode", mode)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("ai chat run panic", "convo", convoID, "panic", p)
				run.emit(map[string]string{"error": "任务异常终止"})
				run.emit(map[string]string{"done": "1"})
				run.finish(aiRunStatusError)
			}
		}()
		err := s.RunAgentChat(ctx, run.emit, run.ping, provider, history, scenePath, focus, mode)
		status := aiRunStatusDone
		if err != nil {
			if ctx.Err() != nil {
				status = aiRunStatusStopped
			} else {
				status = aiRunStatusError
				run.emit(map[string]string{"error": err.Error()})
			}
			run.emit(map[string]string{"done": "1"})
		}
		// 统一落库（停止/出错半截也算，与前端旧 abort 语义一致）
		if msgs := run.builder.finalize(); len(msgs) > 0 && s.db != nil {
			if _, err := s.SaveConversation(context.Background(), convoID, "", msgs); err != nil {
				slog.Warn("ai chat run persist failed", "convo", convoID, "err", err)
			}
		}
		run.finish(status)
		slog.Info("ai chat run finished", "convo", convoID, "run", run.id, "status", status, "events", len(run.events))
		time.AfterFunc(aiRunTTL, func() {
			if cur, ok := s.runs.Load(convoID); ok && cur.(*chatRun) == run {
				s.runs.Delete(convoID)
			}
		})
	}()
	return convoID, nil
}

// StopChatRun 停止会话的运行中任务。返回是否找到。
func (s *AIService) StopChatRun(convoID uint) bool {
	v, ok := s.runs.Load(convoID)
	if !ok {
		return false
	}
	v.(*chatRun).cancel()
	return true
}

// SubscribeChatRun 订阅任务事件流。ok=false 表示该会话当前没有任务运行体。
func (s *AIService) SubscribeChatRun(convoID uint, offset int) (replay [][]byte, ch chan []byte, doneCh chan struct{}, status string, ok bool) {
	v, loaded := s.runs.Load(convoID)
	if !loaded {
		return nil, nil, nil, "", false
	}
	r, ok := v.(*chatRun)
	if !ok {
		return nil, nil, nil, "", false
	}
	replay, ch = r.subscribe(offset)
	r.mu.Lock()
	status = r.status
	doneCh = r.doneCh
	r.mu.Unlock()
	return replay, ch, doneCh, status, true
}

// ChatRunStatus 查询会话任务状态（无运行体返回空串）。
func (s *AIService) ChatRunStatus(convoID uint) string {
	v, ok := s.runs.Load(convoID)
	if !ok {
		return ""
	}
	r, ok := v.(*chatRun)
	if !ok {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}
