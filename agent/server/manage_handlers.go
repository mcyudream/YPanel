// Docker 管理扩展 handlers（镜像/网络/卷/容器详情/清理）。
package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ypanel/shared/errs"
)

func (s *Server) dockerOK() bool { return s.dock != nil }

func writeDockerErr(w http.ResponseWriter, err error) {
	writeErr(w, err)
}

// handleDockerImageList GET /agent/v1/docker/images
func (s *Server) handleDockerImageList(w http.ResponseWriter, r *http.Request) {
	if !s.dockerOK() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	list, err := s.dock.ImageList(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}

// handleDockerImagePull POST /agent/v1/docker/images/pull {ref}
func (s *Server) handleDockerImagePull(w http.ResponseWriter, r *http.Request) {
	if !s.dockerOK() {
		writeErr(w, errs.ErrAgentDisabled)
		return
	}
	req, err := decodeBody[struct {
		Ref string `json:"ref" binding:"required"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.dock.ImagePull(r.Context(), req.Ref)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}

// handleDockerImageRemove DELETE /agent/v1/docker/images/{id}
func (s *Server) handleDockerImageRemove(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "1"
	if err := s.dock.ImageRemove(r.Context(), r.PathValue("id"), force); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleDockerImagesPrune POST /agent/v1/docker/images/prune
func (s *Server) handleDockerImagesPrune(w http.ResponseWriter, r *http.Request) {
	out, err := s.dock.ImagesPrune(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}

// handleDockerNetworkList GET /agent/v1/docker/networks
func (s *Server) handleDockerNetworkList(w http.ResponseWriter, r *http.Request) {
	list, err := s.dock.NetworkList(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}

// handleDockerNetworkCreate POST /agent/v1/docker/networks/{name} {driver}
func (s *Server) handleDockerNetworkCreate(w http.ResponseWriter, r *http.Request) {
	driver := "bridge"
	var body struct {
		Driver string `json:"driver"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	if body.Driver != "" {
		driver = body.Driver
	}
	if err := s.dock.NetworkCreate(r.Context(), r.PathValue("name"), driver); err != nil {
		writeDockerErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleDockerNetworkRemove DELETE /agent/v1/docker/networks/{name}
func (s *Server) handleDockerNetworkRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.dock.NetworkRemove(r.Context(), r.PathValue("name")); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleDockerVolumeList GET /agent/v1/docker/volumes
func (s *Server) handleDockerVolumeList(w http.ResponseWriter, r *http.Request) {
	list, err := s.dock.VolumeList(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}

// handleDockerVolumeCreate POST /agent/v1/docker/volumes/{name}
func (s *Server) handleDockerVolumeCreate(w http.ResponseWriter, r *http.Request) {
	if err := s.dock.VolumeCreate(r.Context(), r.PathValue("name")); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleDockerVolumeRemove DELETE /agent/v1/docker/volumes/{name}
func (s *Server) handleDockerVolumeRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.dock.VolumeRemove(r.Context(), r.PathValue("name")); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleDockerVolumesPrune POST /agent/v1/docker/volumes/prune
func (s *Server) handleDockerVolumesPrune(w http.ResponseWriter, r *http.Request) {
	out, err := s.dock.VolumesPrune(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}

// handleDockerContainerInspect GET /agent/v1/docker/containers/{id}/inspect
func (s *Server) handleDockerContainerInspect(w http.ResponseWriter, r *http.Request) {
	raw, err := s.dock.ContainerInspectRaw(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, json.RawMessage(raw))
}

// handleDockerContainerStats GET /agent/v1/docker/containers/{id}/stats
func (s *Server) handleDockerContainerStats(w http.ResponseWriter, r *http.Request) {
	raw, err := s.dock.ContainerStatsRaw(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, raw)
}

// handleDockerContainerRemove DELETE /agent/v1/docker/containers/{id}
func (s *Server) handleDockerContainerRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.dock.ContainerRemove(r.Context(), r.PathValue("id"), r.URL.Query().Get("force") == "1"); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleDockerContainersPrune POST /agent/v1/docker/containers/prune
func (s *Server) handleDockerContainersPrune(w http.ResponseWriter, r *http.Request) {
	out, err := s.dock.ContainersPrune(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"output": out})
}
