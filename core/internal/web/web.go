//go:build !noembed

// Package web 前端产物嵌入与静态服务。
// 构建前由 scripts/build.sh 将 web 构建产物拷贝至本目录 dist/。
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// Register 将前端产物挂载到 gin（SPA fallback：未命中文件回退 index.html）。
func Register(r *gin.Engine) error {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(sub))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/agent/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}
		// 静态文件命中则直接服务
		if path != "/" {
			if f, err := sub.Open(strings.TrimPrefix(path, "/")); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// SPA fallback
		index, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.String(http.StatusServiceUnavailable, "前端产物未构建：请在 web/ 下执行 pnpm build 后重新编译")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
	return nil
}
