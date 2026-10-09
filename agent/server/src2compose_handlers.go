// M26 P2：源码构建（clone / preview / detect）与 compose 构建 handler。
package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ypanel/agent/internal/compose"
	"github.com/ypanel/agent/internal/src2compose"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// src2composeTmpPrefix 流式预检临时目录前缀（与 src2compose.CloneTmp 的 MkdirTemp pattern 一致）。
const src2composeTmpPrefix = "/tmp/yp-src2-"

// handleSrc2ComposePreview POST /agent/v1/src2compose/clone-tmp（临时目录克隆，供 core 流式预检分步编排）
func (s *Server) handleSrc2ComposePreview(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.Src2ComposeCloneTmpReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	dir, commit, err := src2compose.CloneTmp(r.Context(), req.GitURL, req.Branch,
		src2compose.GitAuth{Token: req.Token, Username: req.Username, PrivateKey: req.PrivateKey})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, dto.Src2ComposeCloneTmpResp{Dir: dir, Commit: commit})
}

// handleSrc2ComposeClone POST /agent/v1/src2compose/clone（克隆到托管项目 src/，重复创建走更新）
func (s *Server) handleSrc2ComposeClone(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[dto.Src2ComposeCloneReq](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := compose.ValidateName(req.Name); err != nil {
		writeErr(w, err)
		return
	}
	dest := filepath.Join(s.compose.BaseDir(), req.Name, "src")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, err.Error()))
		return
	}
	commit, err := src2compose.CloneTo(r.Context(), req.GitURL, req.Branch, src2compose.GitAuth{Token: req.Token, Username: req.Username, PrivateKey: req.PrivateKey}, dest)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]string{"commit": commit})
}

// handleSrc2ComposeDetect GET /agent/v1/src2compose/detect?dir=（目录锚定托管目录或 /tmp/yp-src2-* 预检临时目录）
func (s *Server) handleSrc2ComposeDetect(w http.ResponseWriter, r *http.Request) {
	dir := filepath.Clean(qParam(r, "dir"))
	base := s.compose.BaseDir()
	if !strings.HasPrefix(dir, base+string(filepath.Separator)) && !strings.HasPrefix(dir, src2composeTmpPrefix) {
		writeErr(w, errs.ErrPathInvalid)
		return
	}
	if _, err := os.Stat(dir); err != nil {
		writeErr(w, errs.Wrap(errs.ErrNotFound, "目录不存在: "+dir))
		return
	}
	writeOK(w, dto.Src2ComposeDetectResp{Items: src2compose.Detect(dir)})
}
