// Package api 控制器层（薄）：参数绑定 → service → 统一响应。
package api

import (
	"strconv"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/shared/errs"
)

// respOK 成功响应。
func respOK[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, errs.RespOK(data))
}

// respErr 业务错误响应（HTTP 恒 200，业务码区分）；非业务错误记日志。
// 内部错误（code 1000）向前端透出真实原因摘要——YPanel 为自托管管理员面板，
// 排查价值高于隐藏收益（完整细节仍在服务端日志），否则用户只见「系统内部错误」无法反馈。
func respErr(c *gin.Context, err error) {
	be := errs.From(err)
	if be.Code == errs.CodeInternal && err.Error() != be.Message {
		slog.Error("api internal error", "err", err.Error(), "path", c.Request.URL.Path)
		be = &errs.Error{Code: be.Code, I18nKey: be.I18nKey, Message: truncErr(be.Message, err.Error())}
	}
	c.JSON(http.StatusOK, errs.RespErr(be))
}

// truncErr 内部错误透传摘要：「系统内部错误：<原因前 160 字符>」。
func truncErr(base, detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return base
	}
	r := []rune(detail)
	if len(r) > 160 {
		r = append(r[:160], '…')
	}
	return base + ": " + string(r)
}

// bind JSON 绑定，失败自动响应。
func bind[T any](c *gin.Context) (*T, bool) {
	var v T
	if err := c.ShouldBindJSON(&v); err != nil {
		respErr(c, errs.Wrap(errs.ErrBadRequest, err.Error()))
		return nil, false
	}
	return &v, true
}

// clientIP 取真实客户端 IP。
func clientIP(c *gin.Context) string { return c.ClientIP() }

// idParam 解析路径 :id（旧签名，供 runtime/sites 等历史引用点使用；certs.go 用 certIDParam）。
func idParam(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errBadRequest("ID 不合法")
	}
	return uint(id), nil
}
