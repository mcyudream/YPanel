//go:build windows

package execx

import "syscall"

// procAttr Windows 平台无进程组需求。
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
