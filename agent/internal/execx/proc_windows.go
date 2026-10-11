//go:build windows

package execx

import (
	"os"
	"syscall"
)

// procAttr Windows 平台无进程组需求。
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

// killGroup Windows 无进程组语义，由调用方对主进程 Kill（Taskkill /T 属替代方案，暂不引入）。
func killGroup(pid int) error {
	if pid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
