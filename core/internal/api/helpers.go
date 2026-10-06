// Package api 控制器层（薄）：参数绑定 → service → 统一响应。
package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ypanel/shared/errs"
)

// respOK 成功响应。
func respOK[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, errs.RespOK(data))
}

// respErr 业务错误响应（HTTP 恒 200，业务码区分）；非业务错误记日志。
func respErr(c *gin.Context, err error) {
	be := errs.From(err)
	if be.Code == errs.CodeInternal && err.Error() != be.Message {
		slog.Error("api internal error", "err", err.Error(), "path", c.Request.URL.Path)
	}
	c.JSON(http.StatusOK, errs.RespErr(be))
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
