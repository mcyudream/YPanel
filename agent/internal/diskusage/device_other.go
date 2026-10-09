//go:build !linux

package diskusage

// sameDevice 非 linux 平台无跨挂载点检测（agent 面向 linux 部署，此桩仅为本地构建通过）。
func sameDevice(_, _ string) (bool, error) {
	return true, nil
}
