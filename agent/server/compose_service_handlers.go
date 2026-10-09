// Compose 单服务操作 handler。
package server

import (
	"net/http"

)

// handleComposeServiceAction POST /agent/v1/compose/services/{service}/{action}?project=
func (s *Server) handleComposeServiceAction(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	service := r.URL.Query().Get("service")
	action := r.URL.Query().Get("action")
	out, err := s.compose.ServiceAction(r.Context(), project, service, action)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}

// handleComposeProjectDelete DELETE /agent/v1/compose/projects/{name}（down + 移除编排目录，含数据）
func (s *Server) handleComposeProjectDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.compose.ProjectDelete(r.Context(), r.PathValue("name")); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, struct{}{})
}
