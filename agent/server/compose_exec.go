package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ypanel/agent/internal/execx"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// ---- Compose ----

// composeReady 能力降级检查（托管目录创建失败等场景）。
func (s *Server) composeReady() bool {
	return s.compose != nil
}

func (s *Server) handleComposeList(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	projects, err := s.compose.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, projects)
}

// handleComposeTopology GET /agent/v1/compose/topology?name=&dir=（M26 P1：项目服务拓扑）
func (s *Server) handleComposeTopology(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	out, err := s.compose.Topology(r.Context(), qParam(r, "name"), qParam(r, "dir"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

func (s *Server) handleComposeConfig(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	out, err := s.compose.Config(qParam(r, "name"), qParam(r, "dir"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

func (s *Server) handleComposeWrite(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[dto.ComposeWriteReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.compose.WriteConfig(req.Name, req.Content); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleComposeUp(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[dto.ComposeActionReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.compose.Up(r.Context(), req.Name, req.Dir)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}

func (s *Server) handleComposeDown(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[dto.ComposeActionReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.compose.Down(r.Context(), req.Name, req.Dir)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}

func (s *Server) handleComposeLogs(w http.ResponseWriter, r *http.Request) {
	if !s.composeReady() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	follow := qParam(r, "follow") == "1"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	if !follow {
		var sb bufferedWriter
		if err := s.compose.Logs(r.Context(), qParam(r, "name"), qParam(r, "dir"), qParam(r, "tail"), qParam(r, "service"), &sb); err != nil {
			writeErr(w, err)
			return
		}
		_, _ = w.Write([]byte(sb.String()))
		return
	}
	err := s.compose.Logs(r.Context(), qParam(r, "name"), qParam(r, "dir"), qParam(r, "tail"), qParam(r, "service"), writeFlush{w, flusher})
	if err != nil && r.Context().Err() == nil {
		slogWarn("compose logs failed", err)
	}
}

// ---- Exec（计划任务通道）----

func (s *Server) handleExec(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.ExecReq](r)
	if err != nil {
		slog.Error("agent exec decode failed", "err", err)
		writeErr(w, err)
		return
	}
	slog.Info("agent exec", "cmdPrefix", req.Command[:min(60, len(req.Command))], "timeout", req.TimeoutSecs)
	out, err := execx.RunEnv(r.Context(), req.Command, req.Env, req.TimeoutSecs)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

// handleExecStream POST /agent/v1/exec/stream {command,timeoutSecs}
// NDJSON chunked 流式（全程 HTTP 200，终态/错误走事件行）：{"line":...} 输出行（stdout/stderr 合并）/
// {"exit":N} 终态 / {"timeout":true} 超时 / {"error":...} 启动失败。供 core 长构建类任务日志实时滚动。
func (s *Server) handleExecStream(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.ExecReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	slog.Info("agent exec stream", "cmdPrefix", req.Command[:min(60, len(req.Command))], "timeout", req.TimeoutSecs)
	fl, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, "当前连接不支持流式响应"))
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	fl.Flush()
	enc := json.NewEncoder(w)
	exit, timedOut, err := execx.RunStream(r.Context(), req.Command, req.TimeoutSecs, func(line string) {
		_ = enc.Encode(map[string]any{"line": line})
		fl.Flush()
	})
	if err != nil {
		_ = enc.Encode(map[string]any{"error": err.Error()})
		fl.Flush()
		return
	}
	if timedOut {
		_ = enc.Encode(map[string]any{"timeout": true})
		fl.Flush()
		return
	}
	_ = enc.Encode(map[string]any{"exit": exit})
	fl.Flush()
}
