package server

// M33 P2.6：VictoriaLogs 自动发现与只读查询代理（core→agent→本机 VL）。

import (
	"net/http"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// handleVLDiscover GET /agent/v1/logs/vl/discover
func (s *Server) handleVLDiscover(w http.ResponseWriter, r *http.Request) {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	out, err := s.dock.DiscoverVL(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

// handleVLQuery POST /agent/v1/logs/vl/query {path, params}
func (s *Server) handleVLQuery(w http.ResponseWriter, r *http.Request) {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[dto.VLProxyReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.dock.QueryVL(r.Context(), req.Path, req.Params)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}
