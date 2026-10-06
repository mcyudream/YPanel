// 系统管理端点（B21）：主机名 / 时区 / DNS。agent 侧受控执行。
package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ypanel/agent/internal/execx"
	"github.com/ypanel/shared/errs"
)

// handleSysManage POST /agent/v1/sysmanage {action, value}
// action: hostname / timezone / dns
func (s *Server) handleSysManage(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[struct {
		Action string `json:"action" binding:"required"`
		Value  string `json:"value" binding:"required"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	value := strings.TrimSpace(req.Value)
	if value == "" {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	var cmd string
	switch req.Action {
	case "hostname":
		if len(value) > 63 || strings.ContainsAny(value, " /\\$;|&<>") {
			writeErr(w, errs.Wrap(errs.ErrBadRequest, "主机名不合法"))
			return
		}
		cmd = fmt.Sprintf("hostnamectl set-hostname %s", value)
	case "timezone":
		// IANA 时区名（如 Asia/Shanghai）：允许字母数字 _ / + -
		if strings.ContainsAny(value, " ;|&<>*?[]{}()#~%\"'`") || strings.Contains(value, "\\") || len(value) > 64 {
			writeErr(w, errs.Wrap(errs.ErrBadRequest, "时区不合法"))
			return
		}
		cmd = fmt.Sprintf("timedatectl set-timezone %s", value)
	case "dns":
		for _, ns := range strings.Fields(value) {
			if strings.ContainsAny(ns, ";|&$<>") {
				writeErr(w, errs.Wrap(errs.ErrBadRequest, "DNS 地址不合法"))
				return
			}
		}
		cmd = fmt.Sprintf("printf '%s\\n' > /etc/resolv.conf", strings.ReplaceAll(value, "\n", ""))
	default:
		writeErr(w, errs.Wrap(errs.ErrBadRequest, "不支持的操作: "+req.Action))
		return
	}

	out, err := execx.Run(r.Context(), cmd, 30)
	if err != nil {
		writeErr(w, err)
		return
	}
	if out.ExitCode != 0 {
		writeErr(w, errs.Wrap(errs.ErrBadRequest, out.Output))
		return
	}
	writeOK(w, map[string]string{"action": req.Action, "value": value})
}
