//go:build !windows

package files

import "syscall"

func syscallStat(path string) (struct {
	uid, gid, mode uint32
}, bool) {
	st := &syscall.Stat_t{}
	if err := syscall.Stat(path, st); err != nil {
		return struct {
			uid, gid, mode uint32
		}{}, false
	}
	return struct {
		uid, gid, mode uint32
	}{uid: st.Uid, gid: st.Gid, mode: uint32(st.Mode)}, true
}
