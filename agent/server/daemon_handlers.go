// Docker daemon.json 配置读写（agent 主机 /etc/docker/daemon.json）。
package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/ypanel/shared/errs"
)

const daemonJSONPath = "/etc/docker/daemon.json"

// handleDockerDaemonConfig GET/PUT /agent/v1/docker/daemon-config
func (s *Server) handleDockerDaemonConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		b, err := os.ReadFile(daemonJSONPath)
		content := "{}"
		if err == nil && len(b) > 0 {
			content = string(b)
		}
		writeOK(w, map[string]string{"content": content})
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	if !json.Valid([]byte(req.Content)) {
		writeErr(w, errs.Wrap(errs.ErrBadRequest, "daemon.json 不是合法 JSON"))
		return
	}
	if err := os.WriteFile(daemonJSONPath, []byte(req.Content), 0o644); err != nil {
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, err.Error()))
		return
	}
	// 重启 docker 使配置生效（宿主 systemd）
	cmd := exec.Command("systemctl", "restart", "docker")
	if out, err := cmd.CombinedOutput(); err != nil {
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, "重启 docker 失败: "+string(out)))
		return
	}
	writeOK(w, map[string]string{"message": "docker 已重启，配置生效"})
}
