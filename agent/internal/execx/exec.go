// Package execx 受控命令执行：计划任务/脚本通道。
// 安全：仅经 agent PSK 鉴权暴露；输出截断；超时强杀进程组。
package execx

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// outputLimit 单次执行输出上限（1MB）。
const outputLimit = 1 << 20

// DefaultTimeoutSecs 默认超时（秒）。
const DefaultTimeoutSecs = 300

// MaxTimeoutSecs 超时上限（秒）。
const MaxTimeoutSecs = 86400

// Run 以 sh -c 执行命令，返回输出与超时/退出码。timeoutSecs<=0 时用默认值。
func Run(ctx context.Context, command string, timeoutSecs int) (dto.ExecResp, error) {
	return RunEnv(ctx, command, nil, timeoutSecs)
}

// RunStream 同 RunEnv，但输出（stdout/stderr 合并）按行经 onLine 流式回调，命令结束后返回退出码。
// 供 core 流式任务日志使用；回调应快速返回。超时/进程组语义与 RunEnv 一致。
// 返回 (exitCode, timedOut, err)：err 仅表示启动失败，命令自身的非零退出经 exitCode 表达。
func RunStream(ctx context.Context, command string, timeoutSecs int, onLine func(line string)) (int, bool, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return 0, false, errs.ErrBadRequest
	}
	if timeoutSecs <= 0 {
		timeoutSecs = DefaultTimeoutSecs
	}
	if timeoutSecs > MaxTimeoutSecs {
		timeoutSecs = MaxTimeoutSecs
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.SysProcAttr = procAttr() // 超时 Kill 时连同子进程组一起结束
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, false, err
	}
	eout, err := cmd.StderrPipe()
	if err != nil {
		return 0, false, err
	}
	if err := cmd.Start(); err != nil {
		return 0, false, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	lines := make(chan string, 256)
	var wg sync.WaitGroup
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}
	wg.Add(2)
	go scan(out)
	go scan(eout)
	go func() { wg.Wait(); close(lines) }()
	for line := range lines {
		if onLine != nil {
			onLine(line)
		}
	}
	runErr := cmd.Wait()
	timedOut := ctx.Err() == context.DeadlineExceeded
	if timedOut {
		return -1, true, nil
	}
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		return exitErr.ExitCode(), false, nil
	}
	if runErr != nil {
		return 0, false, errs.Wrapc(errs.CodeFileOpFailed, runErr.Error())
	}
	return 0, false, nil
}

// RunEnv 同 Run，额外注入环境变量（敏感值走 env 不进 argv）。
func RunEnv(ctx context.Context, command string, env map[string]string, timeoutSecs int) (dto.ExecResp, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return dto.ExecResp{}, errs.ErrBadRequest
	}
	if timeoutSecs <= 0 {
		timeoutSecs = DefaultTimeoutSecs
	}
	if timeoutSecs > MaxTimeoutSecs {
		timeoutSecs = MaxTimeoutSecs
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	// 超时Kill 时连同子进程组一起结束（setpgid）
	cmd.SysProcAttr = procAttr()

	runErr := cmd.Run()
	resp := dto.ExecResp{Output: buf.String(), TimedOut: ctx.Err() == context.DeadlineExceeded}
	if resp.TimedOut {
		resp.Output += "\n[执行超时，已终止]"
		return resp, nil
	}
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		resp.ExitCode = exitErr.ExitCode()
		return resp, nil
	}
	if runErr != nil {
		return dto.ExecResp{}, errs.Wrapc(errs.CodeFileOpFailed, runErr.Error())
	}
	return resp, nil
}
