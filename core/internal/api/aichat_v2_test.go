package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/service"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestAIChatHTTPOwnershipAndSnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Setting{}, &model.AIConversation{}, &model.AIMessage{}, &model.AIRun{}, &model.AIAttachment{}, &model.AIAttachmentRef{}); err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	ai := service.NewAIService(db, service.AIDeps{DataDir: t.TempDir(), Settings: service.NewSettingService(db)})
	owner := rbac.WithCaller(context.Background(), rbac.Caller{UserID: 1, PermSet: map[string]struct{}{"ai:use": {}}})
	id, err := ai.SaveConversation(owner, 0, "api test", []service.ChatMessage{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "saved response"}})
	if err != nil {
		t.Fatal(err)
	}
	handler := &AIAPI{AI: ai}
	router := gin.New()
	router.GET("/conversations/:id", handler.GetConversation)
	router.GET("/stream", handler.ChatStream)
	router.POST("/stop", handler.ChatStop)
	request := func(method, path, body string, user uint) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(rbac.WithCaller(r.Context(), rbac.Caller{UserID: user, PermSet: map[string]struct{}{"ai:use": {}}}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	w := request("GET", fmt.Sprintf("/stream?conversationId=%d", id), "", 2)
	if strings.Contains(w.Header().Get("Content-Type"), "event-stream") {
		t.Fatal("SSE headers sent before ownership check")
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.Code == 0 {
		t.Fatal("foreign stream allowed")
	}
	w = request("GET", fmt.Sprintf("/stream?conversationId=%d", id), "", 1)
	if !strings.Contains(w.Body.String(), `"sync"`) || strings.Count(w.Body.String(), "saved response") != 1 {
		t.Fatalf("bad finished stream: %s", w.Body.String())
	}
	w = request("GET", fmt.Sprintf("/conversations/%d?limit=1", id), "", 1)
	json.Unmarshal(w.Body.Bytes(), &envelope)
	var snapshot service.AIChatSnapshot
	json.Unmarshal(envelope.Data, &snapshot)
	if len(snapshot.Messages) != 1 || snapshot.Total != 2 || !snapshot.HasMore {
		t.Fatal("pagination contract mismatch")
	}
	w = request(http.MethodPost, "/stop", fmt.Sprintf(`{"conversationId":%d}`, id), 2)
	json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.Code == 0 {
		t.Fatal("foreign stop allowed")
	}
}
