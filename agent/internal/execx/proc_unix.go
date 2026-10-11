//go:build !windows

package execx

import "syscall"

// procAttr 新建进程组，超时可整组终止。
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// killGroup 对进程组整组 SIGKILL（负 pid）。CommandContext 默认只 Kill 主进程，
// sh -c 管道链里的子进程（如 docker pull）存活会握住 stdout 管道使 cmd.Wait 永久阻塞。
func killGroup(pid int) error {
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(-pid, syscall.SIGKILL)
}
