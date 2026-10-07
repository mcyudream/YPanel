package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ypanel/agent/internal/procs"
	"github.com/ypanel/shared/errs"
)

// handleProcessList GET /agent/v1/processes?sort=&order=&limit=
func (s *Server) handleProcessList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(qParam(r, "limit"))
	list, err := procs.List(context.Background(), procs.ListOpts{
		Sort:  qParam(r, "sort"),
		Order: qParam(r, "order"),
		Limit: limit,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}

// handleProcessKill POST /agent/v1/processes/kill {pid}
func (s *Server) handleProcessKill(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[struct {
		Pid int32 `json:"pid"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := procs.Kill(r.Context(), req.Pid); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleServiceList GET /agent/v1/services
func (s *Server) handleServiceList(w http.ResponseWriter, _ *http.Request) {
	list, err := procs.ListServices(context.Background())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}

// handleServiceAction POST /agent/v1/services/{name}/{action}
func (s *Server) handleServiceAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	action := r.PathValue("action")
	switch action {
	case "start", "stop", "restart":
	default:
		writeErr(w, errs.ErrBadRequest)
		return
	}
	out, err := procs.ServiceAction(r.Context(), name, action)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}


