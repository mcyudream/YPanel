// 终端 WS 协议探针：连接 /api/v1/terminal，发送 echo，统计收帧（排障用，不提交）。
package main

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	base, token := os.Args[1], os.Args[2]
	u, _ := url.Parse(base + "/api/v1/terminal?token=" + url.QueryEscape(token) + "&cols=100&rows=30")
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("DIAL_ERR:", err)
		return
	}
	defer ws.Close()
	go func() {
		time.Sleep(800 * time.Millisecond)
		_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"input","data":"echo ws-probe-ok\n"}`))
	}()
	deadline := time.Now().Add(5 * time.Second)
	frames := 0
	tail := ""
	for time.Now().Before(deadline) {
		ws.SetReadDeadline(deadline)
		_, data, err := ws.ReadMessage()
		if err != nil {
			fmt.Println("READ_ERR:", err)
			break
		}
		frames++
		tail += string(data)
		if len(tail) > 8000 {
			tail = tail[4000:]
		}
		if frames > 200 {
			break
		}
	}
	fmt.Printf("frames=%d\nprompt_seen=%v\necho_seen=%v\ntail=%q\n",
		frames,
		len(tail) > 0 && (contains(tail, "root@") || contains(tail, "$") || contains(tail, "#")),
		contains(tail, "ws-probe-ok"),
		lastLine(tail))
}

func contains(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
func lastLine(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return s[i+1:]
		}
	}
	return s
}
