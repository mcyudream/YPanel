// M51 内网浏览器网关服务器：独立第二端口的会话式反代。
// 门禁独立于主端口：创建会话时签发的随机令牌 Cookie（yp_gw_t）在此校验。
// 路径两种形态：
//   - /s/{sid}/...  会话前缀内，直接按 sid 代理；
//   - 其余路径（根逃逸）：目标应用用 /assets 等绝对路径引用资源时，浏览器按 8881 根发起，
//     凭 Referer 中的 /s/{sid} 找回会话继续代理（找不到则兜底页）。
package gwserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ypanel/core/internal/service"
)

// Server 网关服务器（第二端口）。
type Server struct {
	svc     *service.WebGwService
	http    *http.Server
	tlsCert string
	tlsKey  string
}

// New 构造网关服务器（未启动）。addr 形如 ":8881"；tlsCert/tlsKey 与面板共用（空=HTTP）。
func New(svc *service.WebGwService, addr, tlsCert, tlsKey string) *Server {
	s := &Server{svc: svc}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.withAuth(s.handle))
	s.http = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
	}
	s.tlsCert, s.tlsKey = tlsCert, tlsKey
	return s
}

// ListenAndServe 启动并阻塞；ctx 取消即优雅退出。返回 nil 表示正常关闭。
func (s *Server) ListenAndServe(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shCtx)
	}()
	slog.Info("webgw: 网关端口监听中", "addr", s.http.Addr, "tls", s.tlsCert != "")
	var err error
	if s.tlsCert != "" && s.tlsKey != "" {
		err = s.http.ListenAndServeTLS(s.tlsCert, s.tlsKey)
	} else {
		err = s.http.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// withAuth 网关全端口鉴权：Cookie 中的网关令牌必须有效。
// 同名 Cookie 可能多条（历史 Cookie 残留，路径不同），任一有效即通过。
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, c := range r.Cookies() {
			if c.Name != service.WebGwCookie || c.Value == "" {
				continue
			}
			if _, ok := s.svc.TokenUID(c.Value); ok {
				next(w, r.WithContext(context.WithValue(r.Context(), ctxToken{}, c.Value)))
				return
			}
		}
		s.renderGate(w, "未登录或登录已过期")
	}
}

// ctxToken 请求上下文键：已通过门禁的网关令牌。
type ctxToken struct{}

func tokenOf(r *http.Request) string {
	v, _ := r.Context().Value(ctxToken{}).(string)
	return v
}

// handle 路由分发：会话前缀 / 根逃逸回捞 / 兜底页。
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	ep := r.URL.EscapedPath()
	if ep == "/s" || strings.HasPrefix(ep, "/s/") {
		s.proxyBySid(w, r, ep)
		return
	}
	// 根逃逸：Referer 里的 /s/{sid} 找回会话（iframe 内文档发起的绝对路径子请求）；
	// Referer 缺失（部分动态 import 竞态）时用令牌最近使用的会话兜底
	sid := sidFromReferer(r.Referer())
	if sid == "" {
		sid = s.svc.FallbackSid(tokenOf(r))
	}
	if sid != "" {
		s.proxyRoot(w, r, sid, ep)
		return
	}
	s.renderMiss(w, r)
}

// sidFromReferer 从 Referer URL 路径中解析 /s/{sid} 前缀。
func sidFromReferer(referer string) string {
	if referer == "" {
		return ""
	}
	u, err := url.Parse(referer)
	if err != nil {
		return ""
	}
	p := u.EscapedPath()
	if !strings.HasPrefix(p, "/s/") {
		return ""
	}
	rest := strings.TrimPrefix(p, "/s/")
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	if rest == "" {
		return ""
	}
	return rest
}

// proxyBySid 会话前缀内请求：剥离 /s/{sid} 前缀后按会话代理（路径结构与直连一致）。
func (s *Server) proxyBySid(w http.ResponseWriter, r *http.Request, ep string) {
	rest := strings.TrimPrefix(ep, "/s")
	rest = strings.TrimPrefix(rest, "/")
	slash := strings.IndexByte(rest, '/')
	var sid, remainder string
	if slash < 0 {
		sid, remainder = rest, ""
	} else {
		sid, remainder = rest[:slash], rest[slash:]
	}
	ses := s.svc.Get(sid)
	p := s.svc.Proxy(sid)
	if ses == nil || p == nil {
		s.renderMiss(w, r)
		return
	}
	s.svc.TouchSession(tokenOf(r), sid)
	applyPath(r, remainder)
	p.ServeHTTP(w, r)
}

// proxyRoot 根逃逸请求：按 Referer 找回的会话代理，目标路径 = 原始根路径。
func (s *Server) proxyRoot(w http.ResponseWriter, r *http.Request, sid, ep string) {
	ses := s.svc.Get(sid)
	p := s.svc.Proxy(sid)
	if ses == nil || p == nil {
		s.renderMiss(w, r)
		return
	}
	applyPath(r, ep)
	p.ServeHTTP(w, r)
}

// applyPath 写回目标路径（保留原始转义；空路径归一为根）。
func applyPath(r *http.Request, escapedPath string) {
	if escapedPath == "" {
		r.URL.Path = "/"
		r.URL.RawPath = ""
		return
	}
	r.URL.Path = escapedPath
	r.URL.RawPath = escapedPath
}

// renderGate 未授权访问的引导页。
func (s *Server) renderGate(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>YPanel 内网浏览器</title>
<body style="font-family:system-ui;background:#f1f5f9;color:#334155;display:flex;min-height:100vh;margin:0;align-items:center;justify-content:center">
<div style="max-width:440px;padding:24px;text-align:center">
<h2 style="margin:0 0 8px;font-size:18px">YPanel 内网浏览器网关</h2>
<p style="margin:0 0 4px;font-size:14px;color:#64748b">%s</p>
<p style="margin:0;font-size:12px;color:#94a3b8">请从 YPanel 桌面工作台的「内网浏览器」应用窗口访问内网资源。</p>
</div></body>`, msg)
}

// renderMiss 无会话可回捞的路径（无 Referer 的直接访问 / 会话失效）。
func (s *Server) renderMiss(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>不在代理会话内</title>
<body style="font-family:system-ui;background:#f8fafc;color:#334155;display:flex;min-height:100vh;margin:0;align-items:center;justify-content:center">
<div style="max-width:440px;padding:24px;text-align:center">
<h2 style="margin:0 0 8px;font-size:18px">该地址不在浏览器会话内</h2>
<p style="margin:0 0 4px;font-size:14px;color:#64748b">目标页面跳转到了会话范围之外（%s）</p>
<p style="margin:0;font-size:12px;color:#94a3b8">请回到内网浏览器窗口的地址栏直接输入目标地址访问。</p>
</div></body>`, r.URL.Path)
}
