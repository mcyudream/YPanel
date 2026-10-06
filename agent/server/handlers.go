package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ypanel/agent/internal/compose"
	"github.com/ypanel/agent/internal/files"
	"github.com/ypanel/agent/internal/term"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

const maxBodyBytes = 32 << 20 // 请求体上限 32MB（含上传）

func decodeBody[T any](r *http.Request) (*T, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, err.Error())
	}
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "JSON 解析失败: "+err.Error())
	}
	return &v, nil
}

func qParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// ---- 健康 ----

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, dto.HealthResp{Status: "ok", Version: version})
}

// version 由构建注入（-ldflags），见 cmd/agent。
var version = "dev"

// ---- 系统 ----

func (s *Server) handleOverview(w http.ResponseWriter, _ *http.Request) {
	ov, err := s.sys.Overview()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, ov)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	seconds, _ := strconv.Atoi(qParam(r, "seconds"))
	if seconds <= 0 || seconds > 3600 {
		seconds = 600
	}
	writeOK(w, s.sys.History(seconds))
}

// ---- 文件 ----

func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	out, err := s.files.List(qParam(r, "path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

func (s *Server) handleFileRead(w http.ResponseWriter, r *http.Request) {
	opts := files.ReadOptions{
		Raw:      qParam(r, "raw") == "1",
		Encoding: qParam(r, "encoding"),
	}
	out, err := s.files.ReadOpts(qParam(r, "path"), opts)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	path := qParam(r, "path")
	// 目录 → tar.gz 流式打包；文件 → 原样下载
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(path)+`.tar.gz"`)
		if err := s.files.Archive(path, w); err != nil && r.Context().Err() == nil {
			slog.Warn("archive failed", "err", err)
		}
		return
	}
	f, st, err := s.files.Download(path)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer func() { _ = f.Close() }()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(path)+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(st.Size(), 10))
	_, _ = io.Copy(w, f)
}

func (s *Server) handleFileWrite(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileWriteReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.WriteEncoded(req.Path, req.Content, req.Encoding); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleFileMkdir(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileMkdirReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Mkdir(req.Path); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleFileRename(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileRenameReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Rename(req.From, req.To); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileDeleteReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Delete(req.Paths); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxBodyBytes); err != nil {
		writeErr(w, errs.Wrap(errs.ErrBadRequest, err.Error()))
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, errs.Wrap(errs.ErrBadRequest, "缺少 file 字段"))
		return
	}
	defer func() { _ = file.Close() }()
	saved, err := s.files.Upload(qParam(r, "path"), hdr.Filename, file)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"path": saved})
}

// ---- Docker ----

func (s *Server) handleDockerList(w http.ResponseWriter, r *http.Request) {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	list, err := s.dock.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}

func (s *Server) handleDockerAction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	action := r.PathValue("action")
	switch action {
	case "start", "stop", "restart":
	default:
		writeErr(w, errs.ErrBadRequest)
		return
	}
	if err := s.dock.Action(r.Context(), id, action); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleDockerLogs(w http.ResponseWriter, r *http.Request) {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	id := r.PathValue("id")
	follow := qParam(r, "follow") == "1"
	tail := qParam(r, "tail")
	if tail == "" {
		tail = "500"
	}
	if follow {
		// 持续推送：chunked 文本流
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, _ := w.(http.Flusher)
		err := s.dock.Logs(r.Context(), id, tail, true, writeFlush{w, flusher})
		if err != nil && r.Context().Err() == nil && !errors.Is(err, context.Canceled) {
			slog.Warn("docker follow logs failed", "err", err)
		}
		return
	}
	var sb bufferedWriter
	if err := s.dock.Logs(r.Context(), id, tail, false, &sb); err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(sb.String()))
}

type writeFlush struct {
	w http.ResponseWriter
	f http.Flusher
}

func (x writeFlush) Write(p []byte) (int, error) {
	n, err := x.w.Write(p)
	if x.f != nil {
		x.f.Flush()
	}
	return n, err
}

// ---- 终端 ----

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true }, // 鉴权由 token 中间件完成
}

// termCtrlMsg 终端入站控制消息。
type termCtrlMsg struct {
	Type string `json:"type"` // input / resize
	Data string `json:"data"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = ws.Close() }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sess, err := term.Start(ctx, parseUint16(qParam(r, "cols"), 80), parseUint16(qParam(r, "rows"), 24), func(out []byte) {
		if out == nil { // EOF 哨兵：会话退出
			_ = ws.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session exited"), time.Now().Add(time.Second))
			cancel()
			return
		}
		_ = ws.WriteMessage(websocket.BinaryMessage, out)
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	defer func() { _ = sess.Close() }()

	for {
		mt, payload, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		var msg termCtrlMsg
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "input":
			if _, err := sess.Write([]byte(msg.Data)); err != nil {
				return
			}
		case "resize":
			_ = sess.Resize(msg.Cols, msg.Rows)
		}
	}
}

func parseUint16(s string, def uint16) uint16 {
	v, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return def
	}
	return uint16(v)
}

// ---- 小工具 ----

type bufferedWriter struct{ b []byte }

func (w *bufferedWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

func (w *bufferedWriter) String() string { return string(w.b) }

// handleComposeScan GET /agent/v1/compose/scan?dir=（B12：外部 compose 项目发现）
func (s *Server) handleComposeScan(w http.ResponseWriter, r *http.Request) {
	dir := qParam(r, "dir")
	if dir == "" {
		dir = "/opt"
	}
	out, err := compose.ScanProjects(dir)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}
