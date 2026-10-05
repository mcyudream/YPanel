//go:build windows

package files

func syscallStat(string) (struct {
	uid, gid, mode uint32
}, bool) {
	return struct {
		uid, gid, mode uint32
	}{}, false
}
