package server

import (
	"net/http"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// handleDockerGuardStopAll POST /agent/v1/docker/guard/stopall
// 磁盘空间保护：停止全部活动容器（豁免名单除外）并改写重启策略为 no，返回快照。
func (s *Server) handleDockerGuardStopAll(w http.ResponseWriter, r *http.Request) {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[dto.GuardStopAllReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.dock.StopAllForGuard(r.Context(), req.Exclude)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

// handleDockerGuardRestore POST /agent/v1/docker/guard/restore
// 磁盘空间保护一键恢复：按快照还原重启策略并拉起容器。
func (s *Server) handleDockerGuardRestore(w http.ResponseWriter, r *http.Request) {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[dto.GuardRestoreReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.dock.RestoreFromGuard(r.Context(), req.Containers)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}
