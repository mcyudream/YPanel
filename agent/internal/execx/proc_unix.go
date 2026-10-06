//go:build !windows

package execx

import "syscall"

// procAttr 新建进程组，超时可整组终止。
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
