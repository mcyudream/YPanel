// Docker daemon.json 配置读写（agent 主机 /etc/docker/daemon.json）。
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

const daemonJSONPath = "/etc/docker/daemon.json"

// restartDockerCap systemctl restart 的等待上限。容器多/数据库容器优雅停机慢时重停可达分钟级；
// 超时只停止等待，systemd 的重启 job 不随客户端断开而取消，仍在后台继续。
const restartDockerCap = 8 * time.Minute

// handleDockerDaemonConfig GET/PUT /agent/v1/docker/daemon-config
// PUT = 写入 + 重启 docker 生效；重启失败自动回滚（写前留底）并附 journal 尾部真实原因。
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
	origin, hadOrigin, err := readDaemonJSON()
	if err != nil {
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, err.Error()))
		return
	}
	if err := os.WriteFile(daemonJSONPath, []byte(req.Content), 0o644); err != nil {
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, err.Error()))
		return
	}
	out, timedOut, err := restartDocker()
	if err == nil {
		writeOK(w, map[string]string{"message": "docker 已重启，配置生效"})
		return
	}
	msg := "重启 docker 失败: " + tailText(out, err)
	if timedOut {
		// systemd job 仍在后台继续，此时回滚会与进行中的重启互相踩踏，只报告状态
		msg += "\n等待重启超时，systemd 重启任务仍在后台继续，请稍后确认 docker 状态后再操作"
		writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, msg))
		return
	}
	if journal := journalTail(); journal != "" {
		msg += "\njournal 尾部：\n" + journal
	}
	msg += "\n" + rollbackDaemonJSON(origin, hadOrigin)
	writeErr(w, errs.Wrapc(errs.CodeFileOpFailed, msg))
}

// readDaemonJSON 读当前 daemon.json；文件不存在返回 hadOrigin=false。
func readDaemonJSON() (string, bool, error) {
	b, err := os.ReadFile(daemonJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(b), true, nil
}

// restartDocker 重启 docker（有等待上限，超时不打断 systemd job）。
func restartDocker() (out string, timedOut bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), restartDockerCap)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "docker")
	b, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(b), true, errors.New("等待重启超时")
	}
	return string(b), false, err
}

// journalTail 取 docker 单元日志尾部（dockerd 起不来的真实原因在这里）；不可得时返回空。
func journalTail() string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "journalctl", "-u", "docker", "-n", "30", "--no-pager").CombinedOutput()
	if err != nil {
		return ""
	}
	return tailText(string(out), nil)
}

// rollbackDaemonJSON 恢复原配置并重启，返回给用户看的结果说明。
func rollbackDaemonJSON(origin string, hadOrigin bool) string {
	var werr error
	if hadOrigin {
		werr = os.WriteFile(daemonJSONPath, []byte(origin), 0o644)
	} else {
		werr = os.Remove(daemonJSONPath)
	}
	if werr != nil {
		return "daemon.json 回滚失败（" + werr.Error() + "），请手工恢复原配置后重启 docker"
	}
	if _, _, err := restartDocker(); err != nil {
		return "已回滚 daemon.json，但恢复重启未完成，docker 可能处于停止态：请在节点终端执行 systemctl restart docker"
	}
	return "daemon.json 已回滚，docker 已按原配置恢复运行（新配置未生效）"
}

// tailText 归并命令输出与错误为一段有界文本（保留尾部，真实错误在后半段）。
func tailText(out string, err error) string {
	s := strings.TrimSpace(out)
	if s == "" {
		if err == nil {
			return ""
		}
		return err.Error()
	}
	const cap = 2000
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}
