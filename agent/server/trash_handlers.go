// 文件回收站 handler（M38）。
package server

import (
	"encoding/json"
	"net/http"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// handleFileTrash POST /agent/v1/files/trash {paths}
func (s *Server) handleFileTrash(w http.ResponseWriter, r *http.Request) {
	var req dto.FileTrashReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Paths) == 0 {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	items, err := s.files.Trash(req.Paths)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, dto.FileTrashListResp{Items: items})
}

// handleFileTrashList GET /agent/v1/files/trash/list
func (s *Server) handleFileTrashList(w http.ResponseWriter, r *http.Request) {
	items, err := s.files.TrashList()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, dto.FileTrashListResp{Items: items})
}

// handleFileTrashRestore POST /agent/v1/files/trash/restore {names}
func (s *Server) handleFileTrashRestore(w http.ResponseWriter, r *http.Request) {
	var req dto.FileTrashNamesReq
	derr := json.NewDecoder(r.Body).Decode(&req)
	if derr != nil || len(req.Names) == 0 {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	if err := s.files.TrashRestore(req.Names); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, struct{}{})
}

// handleFileTrashPurge POST /agent/v1/files/trash/purge {names}
func (s *Server) handleFileTrashPurge(w http.ResponseWriter, r *http.Request) {
	var req dto.FileTrashNamesReq
	derr := json.NewDecoder(r.Body).Decode(&req)
	if derr != nil || len(req.Names) == 0 {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	if err := s.files.TrashPurge(req.Names); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, struct{}{})
}

// handleFileTrashClear POST /agent/v1/files/trash/clear
func (s *Server) handleFileTrashClear(w http.ResponseWriter, r *http.Request) {
	n, err := s.files.TrashClear()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]int{"count": n})
}
