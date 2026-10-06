// 容器 exec WS 终端（TTY 双向桥接）。
package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/ypanel/shared/errs"
)

// handleDockerExecWS GET /agent/v1/docker/containers/{id}/exec?cmd=/bin/sh
func (s *Server) handleDockerExecWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	cmd := r.URL.Query().Get("cmd")
	if cmd == "" {
		cmd = "/bin/sh"
	}
	execID, err := s.dock.ExecCreate(r.Context(), id, []string{cmd})
	if err != nil {
		writeErr(w, err)
		return
	}
	reader, writer, err := s.dock.ExecAttach(r.Context(), execID)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer func() {
		if closer, ok := writer.(io.Closer); ok {
			_ = closer.Close()
		}
	}()

	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = ws.Close() }()

	// 上行：容器输出 → WS 文本帧
	go func() {
		buf := make([]byte, 4096)
		for {
			n, rerr := reader.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.TextMessage, trimCRBytes(buf[:n])); werr != nil {
					return
				}
			}
			if rerr != nil {
				return
			}
		}
	}()

	// 下行：WS 文本帧 → 容器 stdin；JSON resize 控制帧 → 调整 exec TTY 尺寸
	for {
		mt, payload, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		var ctl struct {
			Type string `json:"type"`
			Cols uint16 `json:"cols"`
			Rows uint16 `json:"rows"`
		}
		if json.Unmarshal(payload, &ctl) == nil && ctl.Type == "resize" {
			_ = s.dock.ExecResize(r.Context(), execID, ctl.Cols, ctl.Rows)
			continue
		}
		if _, werr := writer.Write(payload); werr != nil {
			return
		}
	}
}

func trimCRBytes(b []byte) []byte { return b }
