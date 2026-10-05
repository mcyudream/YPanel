// Package errs 定义统一业务错误：错误码 + i18n key + 用户可读消息。
// 业务层返回 *Error，api 层统一转为 dto.Resp；系统错误（非 *Error）一律按 500 处理并记日志。
package errs

import "fmt"

// 错误码段规划：
//   1xxx 通用；2xxx 认证与用户；3xxx 文件；4xxx Docker；5xxx 系统/agent
const (
	CodeInternal      = 1000 // 系统内部错误
	CodeBadRequest    = 1001 // 参数错误
	CodeNotFound      = 1002 // 资源不存在
	CodeConflict      = 1003 // 资源冲突
	CodeAgentUnreach  = 5001 // agent 不可达
	CodeAgentDisabled = 5002 // 功能在当前节点不可用
)

const (
	CodeUnauthorized    = 2001 // 未登录或会话失效
	CodeForbidden       = 2002 // 无权限
	CodeUserOrPassWrong = 2003 // 用户名或密码错误
	CodeUserDisabled    = 2004 // 账号已禁用
	CodeOldPassWrong    = 2005 // 原密码错误
)

const (
	CodePathInvalid    = 3001 // 路径非法（穿越/超出根）
	CodeFileNotFound   = 3002
	CodeFileTooLarge   = 3003
	CodeFileOpFailed   = 3004
)

// Error 携带 i18n key 的业务错误。
type Error struct {
	Code    int    `json:"code"`
	I18nKey string `json:"i18nKey"`
	Message string `json:"message"` // 兜底文案（中文）；接入 i18n 后端渲染后由 key 替代
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%d]%s", e.Code, e.Message)
}

// New 构造业务错误。
func New(code int, i18nKey, message string) *Error {
	return &Error{Code: code, I18nKey: i18nKey, Message: message}
}

// Wrap 在已有业务错误上追加上下文（保持码与 key）。
func Wrap(err *Error, context string) *Error {
	return &Error{Code: err.Code, I18nKey: err.I18nKey, Message: context + ": " + err.Message}
}

// codeKeys 错误码 → i18n key 映射（供 Wrapc 使用）。
var codeKeys = map[int]string{
	CodeInternal:      "error.internal",
	CodeBadRequest:    "error.badRequest",
	CodeNotFound:      "error.notFound",
	CodeConflict:      "error.conflict",
	CodePathInvalid:   "error.pathInvalid",
	CodeFileNotFound:  "error.fileNotFound",
	CodeFileTooLarge:  "error.fileTooLarge",
	CodeFileOpFailed:  "error.fileOpFailed",
	CodeAgentUnreach:  "error.agentUnreachable",
	CodeAgentDisabled: "error.agentDisabled",
}

// Wrapc 以错误码构造错误（消息为具体上下文；key 由错误码映射，未登记回退 error.internal）。
func Wrapc(code int, msg string) *Error {
	key, ok := codeKeys[code]
	if !ok {
		key = "error.internal"
	}
	return &Error{Code: code, I18nKey: key, Message: msg}
}

// 预置常用错误。
var (
	ErrInternal      = New(CodeInternal, "error.internal", "系统内部错误")
	ErrBadRequest    = New(CodeBadRequest, "error.badRequest", "请求参数错误")
	ErrNotFound      = New(CodeNotFound, "error.notFound", "资源不存在")
	ErrUnauthorized  = New(CodeUnauthorized, "error.unauthorized", "登录已失效，请重新登录")
	ErrForbidden     = New(CodeForbidden, "error.forbidden", "无权执行该操作")
	ErrAgentUnreach  = New(CodeAgentUnreach, "error.agentUnreachable", "节点 agent 不可达")
	ErrAgentDisabled = New(CodeAgentDisabled, "error.agentDisabled", "当前节点不支持该功能")
	ErrPathInvalid   = New(CodePathInvalid, "error.pathInvalid", "路径不合法")
)
