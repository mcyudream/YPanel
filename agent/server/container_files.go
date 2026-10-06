// 容器内文件管理 handlers：tar 归档 + 容器内 exec，见 internal/dockerx/cfiles.go。
package server

import (
	"encoding/base64"
	"log/slog"
	"net/http"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// containerFileGuard Docker 可用性前置检查（容器文件能力依赖 dock）。
func (s *Server) containerFileGuard(w http.ResponseWriter) bool {
	if !s.dock.Available() {
		writeErr(w, errs.ErrAgentDisabled)
		return false
	}
	return true
}

func (s *Server) handleContainerFileList(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	out, err := s.dock.ContainerFileList(r.Context(), r.PathValue("id"), qParam(r, "path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, dto.FileListResp{Path: qParam(r, "path"), Entries: out})
}

func (s *Server) handleContainerFileRead(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	content, truncated, err := s.dock.ContainerFileRead(r.Context(), r.PathValue("id"), qParam(r, "path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, dto.FileReadResp{
		Path:       qParam(r, "path"),
		Size:       int64(len(content)),
		Truncated:  truncated,
		ContentB64: base64.StdEncoding.EncodeToString(content),
	})
}

func (s *Server) handleContainerFileDownload(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	path := qParam(r, "path")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+pathValueBase(path)+`"`)
	if err := s.dock.ContainerFileDownload(r.Context(), r.PathValue("id"), path, w); err != nil && r.Context().Err() == nil {
		// 响应头已发出，只能记日志（与宿主 download 行为一致）
		slog.Warn("container file download failed", "err", err)
	}
}

func (s *Server) handleContainerFileWrite(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	req, err := decodeBody[dto.FileWriteReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	// 容器写直存 UTF-8；非 UTF-8 编码请求显式拒绝（与宿主编码转换能力差异）
	if req.Encoding != "" && req.Encoding != "utf-8" {
		writeErr(w, errs.Wrapc(errs.CodeBadRequest, "容器文件暂仅支持 UTF-8 编码保存"))
		return
	}
	if err := s.dock.ContainerFileWrite(r.Context(), r.PathValue("id"), req.Path, []byte(req.Content)); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleContainerFileMkdir(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	req, err := decodeBody[dto.FileMkdirReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.dock.ContainerFileMkdir(r.Context(), r.PathValue("id"), req.Path); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleContainerFileRename(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	req, err := decodeBody[dto.FileRenameReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.dock.ContainerFileRename(r.Context(), r.PathValue("id"), req.From, req.To); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleContainerFileDelete(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	req, err := decodeBody[dto.FileDeleteReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.dock.ContainerFileDelete(r.Context(), r.PathValue("id"), req.Paths); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

func (s *Server) handleContainerFileChmod(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	req, err := decodeBody[dto.FileChmodReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.dock.ContainerFileChmod(r.Context(), r.PathValue("id"), req.Path, req.Mode); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// containerUploadReq 容器文件上传（JSON base64，免 multipart 透传）。
type containerUploadReq struct {
	Path       string `json:"path" binding:"required"` // 目标目录
	Name       string `json:"name" binding:"required"`
	ContentB64 string `json:"contentB64" binding:"required"`
}

func (s *Server) handleContainerFileUpload(w http.ResponseWriter, r *http.Request) {
	if !s.containerFileGuard(w) {
		return
	}
	req, err := decodeBody[containerUploadReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	content, derr := base64.StdEncoding.DecodeString(req.ContentB64)
	if derr != nil {
		writeErr(w, errs.Wrapc(errs.CodeBadRequest, "contentB64 解码失败"))
		return
	}
	if err := s.dock.ContainerFileUpload(r.Context(), r.PathValue("id"), req.Path, req.Name, content); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"path": req.Path + "/" + req.Name})
}

func pathValueBase(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
