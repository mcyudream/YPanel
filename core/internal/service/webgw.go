// M51 内网浏览器：会话式反代网关 service。
// 桌面工作台浏览器窗的所有流量以面板网络环境出站（loopback/私网目标），
// 公网目标默认拒绝（设置键 webgw_public_allow 开启）。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/shared/errs"
)

// webGwCookie 网关端口鉴权 Cookie 名（值=面板 JWT）。
const WebGwCookie = "yp_gw_t"

// webGwPublicAllowKey 公网目标放行设置键。
const webGwPublicAllowKey = "webgw_public_allow"

const webGwSessionTTL = 8 * time.Hour

// WebGwSession 代理会话（内存态，core 重启即失效，前端自动重建）。
type WebGwSession struct {
	Sid       string    `json:"sid"`
	Target    string    `json:"target"` // 归一化后的目标根地址（scheme://host:port）
	UID       uint      `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// WebGwService 会话管理 + 目标校验 + 反代构造。
type WebGwService struct {
	settings  *SettingService
	mu        sync.RWMutex
	sessions  map[string]*WebGwSession
	proxy     map[string]*httputil.ReverseProxy // sid → 专用反代（目标固定，Director 闭包缓存）
	tokens    map[string]webGwToken             // 网关 Cookie 令牌 → uid（随机不透明值，不存 JWT）
	lastSid   map[string]string                 // 网关令牌 → 最近使用会话（根逃逸无 Referer 兜底）
	transport *http.Transport
	// 面板自身识别：目标是本机面板时注入 X-Safe-Entry（安全入口使 iframe 内登录 404）
	selfPort  int
	selfEntry func() string
	nodes     *NodeService
}

// SetSelfEntry 声明面板自身端口与安全入口取值函数（main 装配时注入）。
func (s *WebGwService) SetSelfEntry(port int, entry func() string) {
	s.mu.Lock()
	s.selfPort = port
	s.selfEntry = entry
	s.mu.Unlock()
}

// isLocalHost 判断目标主机是否为本机（loopback 或任一本地网卡地址）。
func isLocalHost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]" {
		return true
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	local := map[string]bool{}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				local[ipn.IP.String()] = true
			}
		}
	}
	for _, ip := range ips {
		if ip.IsLoopback() || local[ip.String()] {
			return true
		}
	}
	return false
}

// applySelfEntry 目标是本机面板时注入安全入口头（登录等请求经 iframe 代理时无法携带 ?entry=）。
func (s *WebGwService) applySelfEntry(r *http.Request) {
	s.mu.RLock()
	port, entryFn := s.selfPort, s.selfEntry
	s.mu.RUnlock()
	if port == 0 || entryFn == nil {
		return
	}
	if _, p, err := net.SplitHostPort(r.Host); err != nil || p != strconv.Itoa(port) {
		return
	}
	if !isLocalHost(r.Host[:strings.LastIndex(r.Host, ":")]) {
		return
	}
	if e := entryFn(); e != "" {
		r.Header.Set("X-Safe-Entry", e)
	}
}

// webGwToken 网关访问令牌：Path=/ Cookie（根路径逃逸请求带不上 Path=/s 的会话 Cookie）。
type webGwToken struct {
	UID       uint
	ExpiresAt time.Time
}

// EnsureToken 取 uid 的网关令牌，每次建会话轮换并刷新有效期。
func (s *WebGwService) EnsureToken(uid uint) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err) // 系统熵源不可用属进程级致命错误（与 randomHex 同口径）
	}
	tok := hex.EncodeToString(raw[:])
	s.mu.Lock()
	s.tokens[tok] = webGwToken{UID: uid, ExpiresAt: time.Now().Add(webGwSessionTTL)}
	s.mu.Unlock()
	return tok
}

// TokenUID 校验令牌并返回归属 uid。
func (s *WebGwService) TokenUID(token string) (uint, bool) {
	s.mu.RLock()
	t, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok || time.Now().After(t.ExpiresAt) {
		return 0, false
	}
	return t.UID, true
}

// TouchSession 记录令牌最近使用的会话（根逃逸请求无 Referer 时的回捞兜底）。
func (s *WebGwService) TouchSession(token, sid string) {
	s.mu.Lock()
	s.lastSid[token] = sid
	s.mu.Unlock()
}

// FallbackSid 取令牌最近使用的存活会话；无则空串。
func (s *WebGwService) FallbackSid(token string) string {
	s.mu.RLock()
	sid := s.lastSid[token]
	s.mu.RUnlock()
	if sid == "" || s.Get(sid) == nil {
		return ""
	}
	return sid
}

func NewWebGwService(ctx context.Context, settings *SettingService) *WebGwService {
	s := &WebGwService{
		settings:  settings,
		sessions:  map[string]*WebGwSession{},
		proxy:     map[string]*httputil.ReverseProxy{},
		tokens:    map[string]webGwToken{},
		lastSid:   map[string]string{},
		transport: webGwTransport(),
	}
	// 过期会话惰性清理：每分钟一轮
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.cleanup()
			}
		}
	}()
	return s
}

func (s *WebGwService) cleanup() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for sid, ses := range s.sessions {
		if now.After(ses.ExpiresAt) {
			delete(s.sessions, sid)
			delete(s.proxy, sid)
		}
	}
	for tok, t := range s.tokens {
		if now.After(t.ExpiresAt) {
			delete(s.tokens, tok)
			delete(s.lastSid, tok)
		}
	}
}

// ---- 目标校验 ----

// validateTarget 校验目标主机：默认仅 loopback + RFC1918 私网；
// link-local/unspecified/multicast 恒拒；公网仅在设置开关打开时允许。
func (s *WebGwService) validateTarget(host string) error {
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return errs.Wrap(errs.ErrBadRequest, "域名解析失败: "+err.Error())
	}
	if len(ips) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "域名解析为空")
	}
	allowPublic := s.settings.Get(webGwPublicAllowKey, "false") == "true"
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() {
			continue
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return errs.Wrap(errs.ErrBadRequest, "拒绝访问链路本地/保留地址")
		}
		if !allowPublic {
			return errs.Wrap(errs.ErrBadRequest, "默认仅允许内网地址（面板设置 webgw_public_allow 可放开公网）")
		}
	}
	return nil
}

// ---- 会话管理 ----

// Create 校验并创建会话，返回会话与归一化目标地址。
func (s *WebGwService) Create(rawURL string, uid uint) (*WebGwSession, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "仅支持 http(s) 地址")
	}
	if u.User != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "目标地址不允许携带用户信息")
	}
	host := u.Hostname()
	if host == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "目标主机为空")
	}
	if err := s.validateTarget(host); err != nil {
		return nil, err
	}
	// 归一化目标根：scheme://host:port（port 缺省按 scheme 补，显式写出便于回显）
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	var sidRaw [16]byte
	if _, err := rand.Read(sidRaw[:]); err != nil {
		return nil, errs.Wrap(errs.ErrInternal, "生成会话失败")
	}
	now := time.Now()
	ses := &WebGwSession{
		Sid:       hex.EncodeToString(sidRaw[:]),
		Target:    u.Scheme + "://" + net.JoinHostPort(host, port),
		UID:       uid,
		CreatedAt: now,
		ExpiresAt: now.Add(webGwSessionTTL),
	}
	s.mu.Lock()
	s.sessions[ses.Sid] = ses
	s.proxy[ses.Sid] = s.newProxy(ses)
	s.mu.Unlock()
	slog.Info("webgw: 会话创建", "sid", ses.Sid[:8], "target", ses.Target, "uid", uid)
	return ses, nil
}

// Get 取会话（自动判过期）。
func (s *WebGwService) Get(sid string) *WebGwSession {
	s.mu.RLock()
	ses := s.sessions[sid]
	s.mu.RUnlock()
	if ses == nil || time.Now().After(ses.ExpiresAt) {
		return nil
	}
	return ses
}

// Delete 销毁会话（关窗 best effort）。
func (s *WebGwService) Delete(sid string) {
	s.mu.Lock()
	_, ok := s.sessions[sid]
	delete(s.sessions, sid)
	delete(s.proxy, sid)
	s.mu.Unlock()
	if ok {
		slog.Info("webgw: 会话销毁", "sid", sid[:8])
	}
}

// ---- 反代 ----

// Proxy 取会话专用反代。
func (s *WebGwService) Proxy(sid string) *httputil.ReverseProxy {
	s.mu.RLock()
	p := s.proxy[sid]
	s.mu.RUnlock()
	return p
}

// webGwTransport 出站 Transport：DialContext 每连接复核目标 IP（防 DNS rebinding）。
func webGwTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	baseDial := t.DialContext
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("resolve %s: empty", host)
		}
		// 与 validateTarget 同口径：只允许 loopback/私网 IP 落连（公网在会话创建时已把关）
		var dialIP net.IP
		for _, ip := range ips {
			if ip.IsLoopback() || ip.IsPrivate() {
				dialIP = ip
				break
			}
		}
		if dialIP == nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "目标解析到非内网 IP，已拦截")
		}
		return baseDial(ctx, network, net.JoinHostPort(dialIP.String(), port))
	}
	return t
}

// newProxy 构造会话专用反代：目标固定，改写响应头保障窗口内可渲染。
func (s *WebGwService) newProxy(ses *WebGwSession) *httputil.ReverseProxy {
	target, err := url.Parse(ses.Target)
	if err != nil {
		slog.Error("webgw: 会话目标非法", "target", ses.Target, "err", err)
	}
	return &httputil.ReverseProxy{
		Transport:     s.transport,
		FlushInterval: -1, // SSE/日志类流即时冲刷
		Director: func(r *http.Request) {
			origHost := r.Host
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
			r.Host = target.Host // 保留目标 Host（应用虚拟主机/校验场景）
			s.applySelfEntry(r)  // 目标是本机面板时补安全入口头
			r.Header.Set("X-Forwarded-Proto", "http")
			if r.TLS != nil {
				r.Header.Set("X-Forwarded-Proto", "https")
			}
			r.Header.Set("X-Forwarded-Host", origHost)
			r.Header.Set("X-Forwarded-For", r.RemoteAddr)
		},
		ModifyResponse: func(w *http.Response) error {
			h := w.Header
			prefix := "/s/" + ses.Sid
			// 允许被窗口 iframe 渲染
			h.Del("X-Frame-Options")
			if v := h.Get("Content-Security-Policy"); v != "" {
				h.Set("Content-Security-Policy", relaxFrameAncestors(v))
			}
			// Set-Cookie：去 Domain（host-only 到面板域）+ Path 前插会话前缀（多会话隔离）
			cookies := h.Values("Set-Cookie")
			if len(cookies) > 0 {
				h.Del("Set-Cookie")
				for _, c := range cookies {
					h.Add("Set-Cookie", rewriteSetCookie(c, prefix))
				}
			}
			// Location：绝对目标地址与站内绝对路径都改回会话前缀
			if loc := h.Get("Location"); loc != "" {
				h.Set("Location", rewriteLocation(loc, target, prefix))
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("webgw: 目标不可达", "sid", ses.Sid[:8], "target", ses.Target, "err", err)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>目标不可达</title>
<body style="font-family:system-ui;background:#f8fafc;color:#334155;display:flex;min-height:100vh;margin:0;align-items:center;justify-content:center">
<div style="max-width:420px;padding:24px;text-align:center">
<h2 style="margin:0 0 8px;font-size:18px">面板无法连接到目标</h2>
<p style="margin:0 0 4px;font-size:14px;color:#64748b">%s</p>
<p style="margin:0;font-size:12px;color:#94a3b8">请确认目标服务已启动且端口正确，可尝试刷新。</p>
</div></body>`, ses.Target)
		},
	}
}

// relaxFrameAncestors 把 CSP 中 frame-ancestors 指令宽化为 *（其余指令原样保留）。
func relaxFrameAncestors(csp string) string {
	parts := strings.Split(csp, ";")
	for i, p := range parts {
		k := strings.TrimSpace(strings.SplitN(p, " ", 2)[0])
		if strings.EqualFold(k, "frame-ancestors") {
			parts[i] = " frame-ancestors *"
		}
	}
	return strings.Join(parts, ";")
}

// rewriteSetCookie 删 Domain、Path 前插会话前缀；Secure 按目标 scheme 保留原样。
func rewriteSetCookie(c, prefix string) string {
	segs := strings.Split(c, ";")
	out := make([]string, 0, len(segs)+1)
	out = append(out, strings.TrimSpace(segs[0]))
	pathSeen := false
	for _, seg := range segs[1:] {
		kv := strings.TrimSpace(seg)
		name := strings.ToLower(strings.TrimSpace(strings.SplitN(kv, "=", 2)[0]))
		switch name {
		case "domain":
			continue // host-only：限定在面板域
		case "path":
			v := "/"
			if kv := strings.SplitN(kv, "=", 2); len(kv) == 2 {
				if x := strings.TrimSpace(kv[1]); x != "" {
					v = x
				}
			}
			if !strings.HasPrefix(v, prefix+"/") && v != prefix {
				if v == "/" {
					v = prefix + "/"
				} else {
					v = prefix + v
				}
			}
			out = append(out, "Path="+v)
			pathSeen = true
		case "samesite":
			// 跨端口 iframe 内请求属同站（same-site）：强制 Lax 保证可携带
			out = append(out, "SameSite=Lax")
		default:
			out = append(out, kv)
		}
	}
	if !pathSeen {
		out = append(out, "Path="+prefix+"/")
	}
	return strings.Join(out, "; ")
}

// rewriteLocation 目标绝对地址改会话前缀；站内根路径同样补前缀（iframe 文档在 /s/{sid}/ 下）。
func rewriteLocation(loc string, target *url.URL, prefix string) string {
	u, err := url.Parse(loc)
	if err != nil {
		return loc
	}
	if u.IsAbs() {
		if strings.EqualFold(u.Scheme, target.Scheme) && strings.EqualFold(u.Host, target.Host) {
			return prefix + addSlashQuery(u)
		}
		return loc // 跳往其它站点：交给兜底页
	}
	if u.Path == "" || !strings.HasPrefix(u.Path, "/") {
		return loc // 相对路径：浏览器基于文档 URL 自动落在前缀内
	}
	return prefix + addSlashQuery(u)
}

func addSlashQuery(u *url.URL) string {
	s := u.EscapedPath()
	if u.RawQuery != "" {
		s += "?" + u.RawQuery
	}
	return s
}
