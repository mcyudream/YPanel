//go:build linux

// Package term Web 终端：每条 WS 连接一个 pty 会话。
// 协议：入站 text 帧 = JSON 控制消息（input/resize），出站 binary 帧 = pty 原始输出。
package term

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"

	"github.com/creack/pty"
)

// Session 一个活跃终端会话。
type Session struct {
	cmd   *exec.Cmd
	tty   *os.File // pty master
	tmpRC string   // bash OSC7 注入用的临时 rcfile（会话结束删除）

	closeOnce sync.Once
}

// shellCandidates 依次探测可用 shell。
var shellCandidates = []string{"/bin/bash", "/bin/sh", "/bin/ash"}

func findShell() string {
	for _, s := range shellCandidates {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	return "sh"
}

// buildShellCmd 构造 shell 启动命令。bash 通过临时 rcfile 在加载用户 ~/.bashrc
// 之前置位发行版模板自带的彩色提示符开关（force_color_prompt），source 之后补
// ls/grep 颜色别名与 dircolors 兜底（发行版默认 root bashrc 的颜色块可能整段
// 注释，如 Debian），最后追加 OSC 7 目录上报；sh/ash 无钩子机制，保持原样
// （该类会话无目录跟随）。HOME 由 Start 显式传入：systemd 启动的 agent 无 HOME，
// bash 不会自行从 passwd 补齐（实测），缺 HOME 时 $HOME/.bashrc 判空失败、
// 用户 shell 配置（别名/历史/~ 展开/配色）整体不加载。
func buildShellCmd(shell string) (*exec.Cmd, string) {
	if shell != "/bin/bash" {
		return exec.Command(shell), ""
	}
	tmp, err := os.CreateTemp("", "ypanel-bashrc-*.sh")
	if err != nil {
		return exec.Command(shell), "" // 注入失败不阻塞终端可用性
	}
	content := "# YPanel 终端：加载用户 bashrc（前置彩色提示符开关），追加颜色兜底与 OSC 7 目录上报\n" +
		"force_color_prompt=yes\n" +
		"[ -f \"$HOME/.bashrc\" ] && . \"$HOME/.bashrc\"\n" +
		"# 颜色兜底：用户 rc 未配置时补最小配色，已配置（LS_COLORS/alias 已有）则不覆盖\n" +
		"[ -z \"$LS_COLORS\" ] && [ -x /usr/bin/dircolors ] && eval \"$(dircolors -b)\"\n" +
		"alias ls >/dev/null 2>&1 || alias ls='ls --color=auto'\n" +
		"alias grep >/dev/null 2>&1 || alias grep='grep --color=auto'\n" +
		"__yp_osc7() { printf '\\033]7;file://%s\\007' \"$PWD\"; }\n" +
		"case \";${PROMPT_COMMAND}\" in *__yp_osc7*) ;; *) PROMPT_COMMAND=\"__yp_osc7${PROMPT_COMMAND:+; }$PROMPT_COMMAND\" ;; esac\n"
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return exec.Command(shell), ""
	}
	_ = tmp.Close()
	return exec.Command(shell, "--rcfile", tmp.Name()), tmp.Name()
}

// ensureHome 显式补 HOME：systemd 启动的 agent 进程无 HOME，bash 不会自行从
// passwd 补齐（实测仅登录场景处理），缺 HOME 会导致 shell 内 $HOME/.bashrc、
// ~ 展开、命令历史全部失效。
func ensureHome(env []string) []string {
	for _, e := range env {
		if strings.HasPrefix(e, "HOME=") && e != "HOME=" {
			return env
		}
	}
	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		return env
	}
	return append(env, "HOME="+u.HomeDir)
}

// Start 启动会话；onData 在独立 goroutine 中推送 pty 输出，ctx 取消时结束进程。
func Start(ctx context.Context, cols, rows uint16, onData func([]byte)) (*Session, error) {
	shell := findShell()
	cmd, tmpRC := buildShellCmd(shell)
	cmd.Env = ensureHome(append(os.Environ(), "TERM=xterm-256color"))
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		if tmpRC != "" {
			_ = os.Remove(tmpRC)
		}
		return nil, fmt.Errorf("启动 pty 失败: %w", err)
	}
	s := &Session{cmd: cmd, tty: tty, tmpRC: tmpRC}

	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := tty.Read(buf)
			if n > 0 {
				out := make([]byte, n)
				copy(out, buf[:n])
				onData(out)
			}
			if err != nil {
				// 会话结束：发一个 EOF 哨兵（空帧）由上层关闭
				onData(nil)
				return
			}
		}
	}()
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	return s, nil
}

// Write 向 pty 写入用户输入。
func (s *Session) Write(p []byte) (int, error) {
	return s.tty.Write(p)
}

// Resize 调整窗口大小。
func (s *Session) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return nil
	}
	return pty.Setsize(s.tty, &pty.Winsize{Cols: cols, Rows: rows})
}

// Close 结束会话（幂等）。
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		_ = s.tty.Close()
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		_ = s.cmd.Wait()
		if s.tmpRC != "" {
			_ = os.Remove(s.tmpRC)
		}
		slog.Debug("term: session closed")
	})
	return nil
}
