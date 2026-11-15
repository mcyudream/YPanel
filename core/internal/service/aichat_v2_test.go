package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/tmc/langchaingo/llms"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAIChatTest(t *testing.T) *AIService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ai.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.Setting{}, &model.AIProvider{}, &model.AIConversation{}, &model.AIMessage{}, &model.AIRun{}, &model.AIAttachment{}, &model.AIAttachmentRef{}, &model.AIMemory{}, &model.AIKnowledge{}, &model.AIKnowledgeDoc{}, &model.AIKnowledgeChunk{}, &model.AIOperationLog{}, &model.AIToolFlag{}); err != nil {
		t.Fatal(err)
	}
	s := NewAIService(db, AIDeps{DataDir: t.TempDir(), Settings: NewSettingService(db)})
	t.Cleanup(func() {
		s.runs.Range(func(_, v any) bool {
			r := v.(*chatRun)
			r.cancel()
			select {
			case <-r.doneCh:
			case <-time.After(3 * time.Second):
			}
			return true
		})
		_ = conn.Close()
	})
	return s
}
func aiTestUser(id uint) context.Context {
	return rbac.WithCaller(context.Background(), rbac.Caller{UserID: id, PermSet: map[string]struct{}{"ai:use": {}}, AllNodes: true})
}
func aiTestProvider(t *testing.T, s *AIService, handler http.HandlerFunc) model.AIProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p := model.AIProvider{Name: "mock", APIType: "openai", BaseURL: server.URL, APIKey: "test-only", Model: "mock", AllowPrivate: true, IsDefault: true}
	if err := s.SaveProvider(&p); err != nil {
		t.Fatal(err)
	}
	return p
}
func aiTestSSE(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/event-stream")
	raw, _ := json.Marshal(map[string]any{"id": "mock", "choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": text}}}})
	fmt.Fprintf(w, "data: %s\n\ndata: {\"id\":\"mock\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", raw)
}
func awaitAI(t *testing.T, s *AIService, id uint) *AIChatSnapshot {
	t.Helper()
	v, _ := s.runs.Load(id)
	r := v.(*chatRun)
	select {
	case <-r.doneCh:
	case <-time.After(5 * time.Second):
		t.Fatal("run timeout")
	}
	return r.snapshot()
}

func TestAIChatHistoryResumeAndIsolation(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	var calls atomic.Int32
	aiTestProvider(t, s, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); aiTestSSE(w, "完整答案") })
	history := []ChatMessage{}
	for i := 0; i < 44; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		history = append(history, ChatMessage{Role: role, Content: fmt.Sprintf("message-%d", i)})
	}
	id, err := s.SaveConversation(ctx, 0, "long history", history)
	if err != nil {
		t.Fatal(err)
	}
	cfg := defaultAISettings()
	cfg.Tools = false
	in := AIChatStart{ConversationID: id, RequestID: "idempotent-1234", Content: "next", Settings: cfg}
	start, err := s.StartChat(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.StartChat(ctx, in)
	if err != nil || again.RunID != start.RunID {
		t.Fatalf("idempotent: %v", err)
	}
	done := awaitAI(t, s, id)
	if done.Total != 46 || done.Messages[45].Content != "完整答案" {
		t.Fatalf("bad completion %#v", done)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one provider call, got %d", calls.Load())
	}
	sub, err := s.SubscribeChatRun(ctx, id, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	if sub.Events != nil || len(sub.Replay) != 0 || sub.Snapshot.Total != 46 {
		t.Fatal("completed run replayed deltas")
	}
	if _, err := s.ConversationSnapshot(aiTestUser(2), id, 0, 50); err == nil {
		t.Fatal("other user read conversation")
	}
	if err := s.StopChat(aiTestUser(2), id); err == nil {
		t.Fatal("other user stopped conversation")
	}
	title := "renamed"
	if err := s.UpdateConversation(ctx, id, &title, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	s.runs.Delete(id)
	snap, err := s.ConversationSnapshot(ctx, id, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Messages) != 10 || snap.Total != 46 || !snap.HasMore || snap.Title != title {
		t.Fatalf("bad paged snapshot %+v", snap)
	}
	older, err := s.ConversationSnapshot(ctx, id, 36, 50)
	if err != nil || older.Messages[0].Content != "message-0" {
		t.Fatalf("lost original history %v", err)
	}
}

func TestAIChatConcurrentStartAndSubscribers(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	gate := make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(gate) }) })
	aiTestProvider(t, s, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-gate:
			aiTestSSE(w, "ok")
		case <-r.Context().Done():
		}
	})
	id, _ := s.SaveConversation(ctx, 0, "test", nil)
	cfg := defaultAISettings()
	cfg.Tools = false
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			_, err := s.StartChat(ctx, AIChatStart{ConversationID: id, RequestID: fmt.Sprintf("concurrent-%d", i), Content: "hi", Settings: cfg})
			if err == nil {
				successes.Add(1)
			}
		})
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("starts=%d", successes.Load())
	}
	v, _ := s.runs.Load(id)
	run := v.(*chatRun)
	if err := s.StopChat(ctx, id, "stale-previous-run"); err == nil {
		t.Fatal("stale stop was accepted for another run")
	}
	sub, err := s.SubscribeChatRun(ctx, id, "wrong-run", 0)
	if err != nil || sub.Snapshot == nil {
		t.Fatalf("missing atomic reset: %v", err)
	}
	sub.Close()
	sub.Close()
	run.mu.Lock()
	n := len(run.subs)
	run.mu.Unlock()
	if n != 0 {
		t.Fatal("subscriber leak")
	}
	slow, _ := s.SubscribeChatRun(ctx, id, run.id, run.snapshot().LastEventID)
	defer slow.Close()
	for i := 0; i < aiRunMaxEvents+10; i++ {
		run.emit(map[string]string{"content": "x"})
	}
	run.mu.Lock()
	n = len(run.subs)
	run.mu.Unlock()
	if n != 0 {
		t.Fatal("overflow subscriber was not detached")
	}
	reset, _ := s.SubscribeChatRun(ctx, id, run.id, 0)
	defer reset.Close()
	if reset.Snapshot == nil || reset.Snapshot.LastEventID <= aiRunMaxEvents {
		t.Fatal("expired cursor didn't resync")
	}
	once.Do(func() { close(gate) })
	awaitAI(t, s, id)
}

func TestAIChatRestartAndAttachmentFork(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	a, err := s.UploadChatAttachment(ctx, "notes.md", strings.NewReader("document content"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.AttachmentFile(aiTestUser(2), a.ID); err == nil {
		t.Fatal("attachment leaked")
	}
	id, _ := s.SaveConversation(ctx, 0, "attached", []ChatMessage{{Role: "user", Content: "read", Attachments: []ChatAttachment{*a}}})
	row, _ := s.ownedConversation(ctx, id)
	messages, _ := s.conversationMessages(ctx, row)
	s.db.Create(&model.AIAttachmentRef{AttachmentID: a.ID, ConversationID: id})
	fork, err := s.ForkConversation(ctx, id, messages[0].ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.chatAttachments(ctx, fork, []string{a.ID}, &model.AIProvider{}); err != nil {
		t.Fatal("fork cannot reuse owned document", err)
	}
	if err := s.RemoveConversation(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.AttachmentFile(ctx, a.ID); err != nil {
		t.Fatal("source deletion broke fork", err)
	}
	if err := s.DeleteChatAttachment(ctx, a.ID); err == nil {
		t.Fatal("deleted referenced attachment")
	}
	pending := ChatMessage{ID: "pending-msg", Role: "assistant", Pending: true, Segments: []ChatSegment{{Type: "tool", Name: "write", Status: "running"}}}
	saveAIMessage(s.db, fork, 1, pending)
	raw, _ := json.Marshal(pending)
	s.db.Create(&model.AIRun{ID: "restart-run", ConversationID: fork, UserID: 1, RequestID: "restart-request", Status: "running", Snapshot: string(raw)})
	if err := s.RecoverChatRuns(); err != nil {
		t.Fatal(err)
	}
	snap, err := s.ConversationSnapshot(ctx, fork, 0, 50)
	if err != nil || snap.Status != "interrupted" || snap.Messages[1].Pending || !snap.Messages[1].Interrupted || !snap.Messages[1].Segments[0].Done {
		t.Fatalf("bad restart %+v, %v", snap, err)
	}
}

func TestAIChatFilesBudgetAndMasking(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	a, err := s.UploadChatAttachment(ctx, "a.png", bytes.NewReader(pngData.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.chatAttachments(ctx, 0, []string{a.ID}, &model.AIProvider{}); err == nil {
		t.Fatal("vision disabled image accepted")
	}
	if _, err := s.chatAttachments(ctx, 0, []string{a.ID}, &model.AIProvider{Model: "mock", Capabilities: `{"mock":{"vision":true}}`}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UploadChatAttachment(ctx, "bad.txt", bytes.NewReader([]byte{0, 255})); err == nil {
		t.Fatal("binary text accepted")
	}
	if _, err := s.UploadChatAttachment(ctx, "bad.png", strings.NewReader("fake image")); err == nil {
		t.Fatal("fake image accepted")
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	f, _ := z.Create("word/document.xml")
	f.Write([]byte(`<w:document xmlns:w="test"><w:p><w:r><w:t>中文 DOCX</w:t></w:r></w:p></w:document>`))
	z.Close()
	doc, err := s.UploadChatAttachment(ctx, "doc.docx", bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	parts, _, err := s.attachmentParts(ctx, []ChatAttachment{*doc}, "中文")
	if err != nil || !strings.Contains(parts[0].(llms.TextContent).Text, "中文 DOCX") {
		t.Fatal("docx content missing", err)
	}
	msgs := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, strings.Repeat("用户约束", 1000))}
	original := msgs[0].Parts[0].(llms.TextContent).Text
	if fitAIToolResults(msgs, 100) == nil || msgs[0].Parts[0].(llms.TextContent).Text != original {
		t.Fatal("over-budget user data was deleted")
	}
	for _, input := range []string{`[{"apiKey":"top-secret","nested":[{"password":"top-secret"}]}]`, `Authorization: Bearer top-secret`, `password=top-secret`, `-----BEGIN PRIVATE KEY-----top-secret-----END PRIVATE KEY-----`, `https://user:top-secret@example.com`} {
		if strings.Contains(maskSensitive(input), "top-secret") {
			t.Fatalf("secret leaked: %s", input)
		}
	}
	cfg := decodeAISettings(`{"memory":false,"knowledge":false,"tools":false,"autoCompact":false}`)
	if cfg.Memory || cfg.Knowledge || cfg.Tools || cfg.AutoCompact {
		t.Fatal("false settings overridden")
	}
}

func TestAIChatScopeApprovalAndOutbound(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	s.RememberMemory("私有记忆", ctx)
	s.RememberMemory("其他用户", aiTestUser(2))
	s.RememberMemory("legacy")
	p, _ := s.MemoryPage(ctx, "", 1, 40)
	if p["total"].(int64) != 1 {
		t.Fatal("memory scope leak")
	}
	req := &aiAskRequest{ch: make(chan bool, 1), identity: aiRunIdentity{UserID: 1, ConversationID: 42, RunID: "one"}}
	s.pendingAsks.Store("approval", req)
	s.auditOperation(ctx, aiToolDef{Name: "test", Module: aiModAI, Risk: aiRiskWrite}, "{}", "ok", "", true, "executed")
	if page := s.ListOperationLogs(1, 20, aiTestUser(2)); page["total"].(int64) != 0 {
		t.Fatal("foreign audit data exposed")
	}
	if s.ResolveAsk("approval", true, aiTestUser(2)) {
		t.Fatal("foreign approval accepted")
	}
	if !s.ResolveAsk("approval", true, ctx) || s.ResolveAsk("approval", true, ctx) {
		t.Fatal("approval is not single-use")
	}
	if _, err := s.RunWorkspace(ctx, "echo hi"); err == nil {
		t.Fatal("ai:use bypassed terminal permission")
	}
	var contacted atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { contacted.Add(1); w.Write([]byte("ok")) }))
	defer server.Close()
	d, _ := newAIHTTP(&model.AIProvider{BaseURL: server.URL})
	request, _ := http.NewRequest("GET", server.URL, nil)
	if _, err := d.client.Do(request); err == nil || contacted.Load() != 0 {
		t.Fatal("private guard bypassed by MCP's http.Client")
	}
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect leaked credentials") }))
	defer second.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, second.URL, http.StatusFound) }))
	defer redirect.Close()
	d, _ = newAIHTTP(&model.AIProvider{BaseURL: redirect.URL, AllowPrivate: true})
	request, _ = http.NewRequest("GET", redirect.URL, nil)
	request.Header.Set("Authorization", "Bearer test")
	if _, err := d.Do(request); err == nil {
		t.Fatal("cross origin redirect accepted")
	}
}

func TestAIChatPDFSourceAndKnowledgeScope(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	if _, err := exec.LookPath("pdftotext"); err == nil {
		var pdf bytes.Buffer
		pdf.WriteString("%PDF-1.4\n")
		offsets := []int{0}
		stream := "BT /F1 12 Tf 72 720 Td (Hello PDF) Tj ET"
		objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>", fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)}
		for i, obj := range objects {
			offsets = append(offsets, pdf.Len())
			fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
		}
		xref := pdf.Len()
		fmt.Fprintf(&pdf, "xref\n0 6\n0000000000 65535 f \n")
		for _, offset := range offsets[1:] {
			fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
		}
		fmt.Fprintf(&pdf, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", xref)
		a, err := s.UploadChatAttachment(ctx, "source.pdf", bytes.NewReader(pdf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		row, _ := s.ownedAttachment(ctx, a.ID)
		if !strings.Contains(row.Text, "Hello PDF") || !strings.Contains(row.Text, "第 1 页") {
			t.Fatal("PDF text/page source lost")
		}
	}
	doc, err := s.SaveKnowledgeDoc("private", "private.md", "目标 中文权限专属", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetKnowledgeDoc(doc.ID, aiTestUser(2)); err == nil {
		t.Fatal("foreign knowledge doc exposed")
	}
	if err := s.DeleteKnowledgeDoc(doc.ID, aiTestUser(2)); err == nil {
		t.Fatal("foreign knowledge doc deleted")
	}
	s.SaveKnowledge(&model.AIKnowledge{Title: "best", Body: "中文权限专属"}, ctx)
	s.SaveKnowledge(&model.AIKnowledge{Title: "foreign", Body: "中文权限专属"}, aiTestUser(2))
	hits := s.searchKnowledge("中文权限专属", ctx)
	if len(hits) == 0 {
		t.Fatal("authorized knowledge retrieval returned no candidates")
	}
	for _, hit := range hits {
		if strings.Contains(hit.Title, "foreign") {
			t.Fatal("knowledge retrieval crossed owners")
		}
	}
	// Missing PDF runtime is an explicit parsing error, never a successful empty upload.
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())
	_, err = extractPDF(ctx, "absent.pdf")
	if err == nil || !strings.Contains(err.Error(), "pdftotext") {
		t.Fatal("missing parser was not reported")
	}
	t.Setenv("PATH", originalPath)
}

func TestAIKnowledgeRanksAcrossAllCandidates(t *testing.T) {
	s := newAIChatTest(t)
	ctx := aiTestUser(1)
	for i := 0; i < 30; i++ {
		if err := s.SaveKnowledge(&model.AIKnowledge{Title: fmt.Sprintf("low-%d", i), Body: "alpha"}, ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SaveKnowledge(&model.AIKnowledge{Title: "best-match", Body: "alpha beta gamma delta"}, ctx); err != nil {
		t.Fatal(err)
	}
	s.RememberMemory("memory_unique_alpha", ctx)
	junk := make([]model.AIMemory, 250)
	for i := range junk {
		junk[i] = model.AIMemory{OwnerID: 1, Content: fmt.Sprintf("irrelevant-%d", i)}
	}
	if err := s.db.Create(&junk).Error; err != nil {
		t.Fatal(err)
	}
	recalled := s.RecallMemories("memory_unique_alpha", ctx)
	if len(recalled) == 0 || recalled[0].Content != "memory_unique_alpha" {
		t.Fatal("older relevant memory was hidden by recent unrelated entries")
	}
	hits := s.searchKnowledge("alpha beta gamma delta", ctx)
	if len(hits) != 3 || !strings.Contains(hits[0].Title, "best-match") {
		t.Fatalf("relevance ordering failed: %+v", hits)
	}
}
