// Docker 资源用量统计 handlers。
package server

import (
	"net/http"

	"github.com/ypanel/shared/errs"
)

// handleDockerUsage GET /agent/v1/docker/usage（system df；verbose 实算 size，懒加载）
func (s *Server) handleDockerUsage(w http.ResponseWriter, r *http.Request) {
	if !s.dockerOK() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	out, err := s.dock.Usage(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

// handleDockerBuildCachePrune POST /agent/v1/docker/buildcache/prune（清空构建缓存，返回释放字节数）
func (s *Server) handleDockerBuildCachePrune(w http.ResponseWriter, r *http.Request) {
	if !s.dockerOK() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	freed, err := s.dock.BuildCachePrune(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]int64{"freed": freed})
}
