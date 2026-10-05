package errs

import "github.com/ypanel/shared/dto"

// RespOK 成功响应。
func RespOK[T any](data T) dto.Resp[T] {
	return dto.Resp[T]{Code: 0, Message: "ok", Data: data}
}

// RespErr 业务错误响应。
func RespErr(e *Error) dto.Resp[struct{}] {
	return dto.Resp[struct{}]{Code: e.Code, Message: e.Message}
}

// From 恢复业务错误：非 *Error 视为内部错误。
func From(err error) *Error {
	if err == nil {
		return nil
	}
	if be, ok := err.(*Error); ok {
		return be
	}
	return &Error{Code: CodeInternal, I18nKey: ErrInternal.I18nKey, Message: ErrInternal.Message}
}
