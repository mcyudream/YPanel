package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/service"
)

// TerminalAPI Web 终端（core 双向桥接浏览器 WS ↔ agent WS）。
type TerminalAPI struct {
	Nodes *service.NodeService
}

func (t *TerminalAPI) client(c *gin.Context) *agentclient.Client {
	node, _ := t.Nodes.ByID(c.DefaultQuery("node", "local"))
	return agentclient.New(node.BaseURL, node.Token)
}

// terminalIn 浏览器入站控制消息（与 agent 协议一致，原样透传）。
type terminalIn struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// Handle GET /api/v1/terminal?cols=&rows=（升级为 WS；token 经 query 传递）
func (t *TerminalAPI) Handle(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     func(*http.Request) bool { return true }, // 鉴权已由中间件完成
	}
	browserWS, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = browserWS.Close() }()

	ac := t.client(c)
	agentURL := ac.WSURL("/agent/v1/terminal?cols=" + c.DefaultQuery("cols", "80") + "&rows=" + c.DefaultQuery("rows", "24"))

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL, ac.Header())
	if err != nil {
		slog.Error("terminal: agent ws dial failed", "url", agentURL, "err", err)
		_ = browserWS.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "agent 连接失败: "+err.Error()),
			time.Now().Add(2*time.Second))
		return
	}
	defer func() { _ = agentConn.Close() }()

	done := make(chan struct{})

	// agent → 浏览器（binary 原样）
	go func() {
		defer close(done)
		for {
			mt, data, err := agentConn.ReadMessage()
			if err != nil {
				return
			}
			if err := browserWS.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	// 浏览器 → agent（text 控制帧原样透传）
	for {
		mt, payload, err := browserWS.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		var msg terminalIn
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		if err := agentConn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}
