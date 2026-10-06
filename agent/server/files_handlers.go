// 文件管理扩展 handlers（chmod/压缩/解压/搜索）。
package server

import (
	"net/http"

)

// handleFileChmod POST /agent/v1/files/chmod {path, mode}
func (s *Server) handleFileChmod(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[struct {
		Path string `json:"path" binding:"required"`
		Mode string `json:"mode" binding:"required"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Chmod(r.Context(), req.Path, req.Mode); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleFileCompress POST /agent/v1/files/compress {src, dest}
func (s *Server) handleFileCompress(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[struct {
		Src  string `json:"src" binding:"required"`
		Dest string `json:"dest" binding:"required"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.files.Compress(r.Context(), req.Src, req.Dest); err != nil {
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
