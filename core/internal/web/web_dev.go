//go:build noembed

// Package web 开发模式构建（-tags noembed）：不嵌入前端，仅返回占位提示。
package web

import (
	"github.com/gin-gonic/gin"
)

// Register 开发模式占位。
func Register(*gin.Engine) error { return nil }
