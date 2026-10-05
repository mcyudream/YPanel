//go:build !linux

// Package term 非 linux 平台无 pty 支持，Start 返回能力不可用错误。
package term

import (
	"context"
	"errors"

	"github.com/ypanel/shared/errs"
)

// Session 占位类型。
type Session struct{}

// Start 平台不支持。
func Start(_ context.Context, _, _ uint16, _ func([]byte)) (*Session, error) {
	return nil, errs.ErrAgentDisabled
}

// Write 占位。
func (*Session) Write(p []byte) (int, error) { return 0, errors.New("unsupported") }

// Resize 占位。
func (*Session) Resize(_, _ uint16) error { return errors.New("unsupported") }

// Close 占位。
func (*Session) Close() error { return nil }
