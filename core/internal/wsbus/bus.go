// Package wsbus 事件推送总线（B7）：core 内部事件统一 publish，
// 面板前端经 /api/v1/ws 订阅主题（notification / task / alert）。
package wsbus

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Event 总线事件信封。
type Event struct {
	Topic     string          `json:"topic"`
	Type      string          `json:"type"`
	Title     string          `json:"title,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"ts"`
}

// Hub 订阅中心（进程内广播）。
type Hub struct {
	mu      sync.RWMutex
	clients map[uint]chan Event
	nextID  uint
}

// Default 全局总线实例。
var Default = &Hub{clients: map[uint]chan Event{}}

// Publish 向所有订阅者广播事件（无订阅者时零开销）。
func (h *Hub) Publish(topic, typ, title string, payload any) {
	var raw json.RawMessage
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			raw = b
		}
	}
	ev := Event{Topic: topic, Type: typ, Title: title, Payload: raw, Timestamp: time.Now()}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.clients {
		select {
		case ch <- ev:
		default: // 慢消费者丢帧，避免阻塞发布方
		}
	}
	if topic == "task" || topic == "alert" {
		slog.Debug("wsbus publish", "topic", topic, "type", typ)
	}
}

// HandleGET /api/v1/ws?token=（浏览器 WS 无法带 header，query token 校验）。
func HandleGET(parseToken func(string) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := parseToken(c.Query("token")); err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		handle(c)
	}
}

func handle(c *gin.Context) {
	up := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(*http.Request) bool { return true },
	}
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ch := make(chan Event, 64)
	Default.mu.Lock()
	Default.nextID++
	id := Default.nextID
	Default.clients[id] = ch
	Default.mu.Unlock()
	defer func() {
		Default.mu.Lock()
		delete(Default.clients, id)
		Default.mu.Unlock()
	}()

	// 读泵：客户端 ping/订阅消息一律忽略（当前全主题广播）
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case ev := <-ch:
			_ = conn.WriteJSON(ev)
		case <-ticker.C:
			_ = conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}
