//go:build linux

package diskusage

import (
	"os"
	"syscall"
)

// sameDevice 判断两个路径是否在同一文件系统（挂载点边界检测）。
func sameDevice(a, b string) (bool, error) {
	sa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	sb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return sa.Sys().(*syscall.Stat_t).Dev == sb.Sys().(*syscall.Stat_t).Dev, nil
}
