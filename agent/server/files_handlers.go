// 文件管理扩展 handlers（chmod/chown/压缩/解压/搜索/属主枚举）。
package server

import (
	"net/http"

	"github.com/ypanel/shared/dto"
)

// handleFileChmod POST /agent/v1/files/chmod {path, mode, recursive}
func (s *Server) handleFileChmod(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileChmodReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Chmod(r.Context(), req.Path, req.Mode, req.Recursive); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleFileChown POST /agent/v1/files/chown {path, owner, group, recursive}
func (s *Server) handleFileChown(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileChownReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Chown(r.Context(), req.Path, req.Owner, req.Group, req.Recursive); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleFileOwners GET /agent/v1/files/owners（系统用户/组枚举）
func (s *Server) handleFileOwners(w http.ResponseWriter, r *http.Request) {
	out, err := s.files.ListOwners()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}

// handleFileCompress POST /agent/v1/files/compress {src?, srcs?, dest}
func (s *Server) handleFileCompress(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.FileCompressReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	srcs := req.Srcs
	if len(srcs) == 0 && req.Src != "" {
		srcs = []string{req.Src} // 兼容旧单源调用方
	}
	if err := s.files.Compress(r.Context(), req.Dest, srcs); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleFileDecompress POST /agent/v1/files/decompress {archive, destDir}
func (s *Server) handleFileDecompress(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[struct {
		Archive string `json:"archive" binding:"required"`
		DestDir string `json:"destDir" binding:"required"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Decompress(req.Archive, req.DestDir); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleFileSearch GET /agent/v1/files/search?dir=&keyword=
func (s *Server) handleFileSearch(w http.ResponseWriter, r *http.Request) {
	list, err := s.files.Search(r.URL.Query().Get("dir"), r.URL.Query().Get("keyword"), 100)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, list)
}
