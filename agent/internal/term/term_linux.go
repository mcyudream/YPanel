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
	"sync"

	"github.com/creack/pty"
)

// Session 一个活跃终端会话。
type Session struct {
	cmd *exec.Cmd
	tty *os.File // pty master

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

// Start 启动会话；onData 在独立 goroutine 中推送 pty 输出，ctx 取消时结束进程。
func Start(ctx context.Context, cols, rows uint16, onData func([]byte)) (*Session, error) {
	shell := findShell()
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("启动 pty 失败: %w", err)
	}
	s := &Session{cmd: cmd, tty: tty}

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
		slog.Debug("term: session closed")
	})
	return nil
}
