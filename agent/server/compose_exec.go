package server

import (
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
		writeErr(w, err)
		return
	}
	out, err := execx.Run(r.Context(), req.Command, req.TimeoutSecs)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}
