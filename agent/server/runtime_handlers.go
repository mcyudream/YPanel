package server

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/agent/internal/execx"
	"github.com/ypanel/agent/internal/fcgi"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// containerNamePattern 容器名白名单（进 shell 前防注入）。
var containerNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

// fpmInspectTemplate 打印容器在各网络中的 IP。
const fpmInspectTemplate = `{{range $_, $v := .NetworkSettings.Networks}}{{if $v.IPAddress}}{{$v.IPAddress}} {{end}}{{end}}`

// handleRuntimeFpmStatus GET /agent/v1/runtime/php/fpm-status?container=php-xxx&query=
// 解析容器 IP 后以 FastCGI 请求 pm.status_path（模板内置 /status），解析 key/value 文本。
func (s *Server) handleRuntimeFpmStatus(w http.ResponseWriter, r *http.Request) {
	container := qParam(r, "container")
	if !containerNamePattern.MatchString(container) {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	out, err := execx.Run(r.Context(), fmt.Sprintf("docker inspect -f '%s' %s", fpmInspectTemplate, container), 15)
	if err != nil {
		writeErr(w, err)
		return
	}
	ip := ""
	for _, f := range strings.Fields(out.Output) {
		if net.ParseIP(f) != nil {
			ip = f
			break
		}
	}
	if ip == "" {
		writeErr(w, errs.New(errs.CodeNotFound, "error.notFound", "未找到运行中的容器 "+container))
		return
	}
	params := map[string]string{
		"REQUEST_METHOD":  "GET",
		"SCRIPT_NAME":     "/status",
		"SCRIPT_FILENAME": "/status",
		"QUERY_STRING":    qParam(r, "query"),
		"SERVER_SOFTWARE": "ypanel-agent",
		"SERVER_NAME":     "localhost",
	}
	raw, err := fcgi.Do(r.Context(), "tcp", net.JoinHostPort(ip, "9000"), params, 10*time.Second)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, dto.FpmStatusResp{Items: parseFpmStatus(fcgi.StripCGIHeaders(raw))})
}

func parseFpmStatus(body string) []dto.FpmStatusItem {
	items := make([]dto.FpmStatusItem, 0, 16)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		items = append(items, dto.FpmStatusItem{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v)})
	}
	return items
}
