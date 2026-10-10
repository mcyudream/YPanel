// M56 webgw proxy-lib 注入器：对会话内 text/html 响应流式注入 proxy-lib 脚本标签。
// 约束：FlushInterval=-1 下上游每 chunk 都可能触发 Flush，锚点查找完成前必须抑制冲刷，
// 防止 <head> 被冲走；缓冲上限内找不到锚点按 doctype→文档头降级，保证响应必然收尾。
package gwserver

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/ypanel/core/internal/service"
)

// proxyLibJS 内网浏览器前端请求拦截库（随二进制分发，源码即本目录 proxy-lib.js）。
//
//go:embed proxy-lib.js
var proxyLibJS string

// injectBufMax 注入锚点查找的缓冲上限；超过即按降级链落笔（正常页面 <head> 都在前几百字节）。
const injectBufMax = 16 * 1024

// proxyScriptPath 会话前缀内的 proxy-lib 脚本端点路径。
const proxyScriptPath = "/__yp_proxy.js"

// renderProxyScript 输出会话配置行 + go:embed 的库体。
// 配置值（sid=hex、target=已校验 origin）均不含敏感与 </script> 序列，内联安全。
func renderProxyScript(w http.ResponseWriter, sid, target string) {
	cfg, err := json.Marshal(map[string]string{
		"base":   "/s/" + sid,
		"sid":    sid,
		"target": target,
	})
	if err != nil {
		http.Error(w, "render config failed", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "application/javascript; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(append(append([]byte("window.__YP_WEBGW__="), cfg...), []byte(";\n"+proxyLibJS+"\n")...))
}

// newInjectWriter 包装会话响应：html 流式注入脚本标签，其余类型零改动透传。
// scriptURL 为注入标签的 src（会话根绝对路径）。method 为 HEAD 时不注入。
func newInjectWriter(w http.ResponseWriter, scriptURL, method string) *injectWriter {
	return &injectWriter{ResponseWriter: w, scriptURL: scriptURL, skip: method == http.MethodHead}
}

// injectWriter html 注入状态机：decide（判定类型）→ 缓冲找锚点 → 注入 → 透传。
type injectWriter struct {
	http.ResponseWriter
	scriptURL string
	decided   bool // 已判定 Content-Type（且已完成 Content-Length 处理）
	isHTML    bool
	injected  bool // 已完成注入（或决定放弃）
	buf       []byte
	skip      bool // HEAD 等无体请求：完全不介入
}

// decide 判定 html 并在注入模式下剥 Content-Length（注入使 body 变长，交给 chunked）。
func (w *injectWriter) decide() {
	if w.decided {
		return
	}
	w.decided = true
	w.isHTML = !w.skip && service.IsHTMLContentType(w.Header().Get("Content-Type"))
	if w.isHTML {
		w.Header().Del("Content-Length")
	}
}

func (w *injectWriter) WriteHeader(code int) {
	w.decide()
	w.ResponseWriter.WriteHeader(code)
}

func (w *injectWriter) Write(p []byte) (int, error) {
	w.decide()
	if !w.isHTML || w.injected {
		return w.ResponseWriter.Write(p)
	}
	w.buf = append(w.buf, p...)
	// 优先锚点注入；缓冲达上限仍无锚点才降级（正常页面 <head> 都在前几百字节）
	if idx := headAnchor(w.buf); idx >= 0 {
		w.emit(idx)
		return len(p), nil
	}
	if len(w.buf) >= injectBufMax {
		w.forceInject()
	}
	return len(p), nil
}

// Flush 抑制 html 未注入完成前的上游冲刷（防 <head> 锚点被冲走），其余场景透传。
func (w *injectWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok && (!w.isHTML || w.injected) {
		f.Flush()
	}
}

// Hijack 透传底层连接接管（双保险：升级类请求已在上层绕过本包装，
// 此处兜底保证任何到达的 hijack 语义不因包装而丢失）。
func (w *injectWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("injectWriter: 底层 ResponseWriter 不支持 Hijack")
}

// drain 兜底收尾：响应体已写完仍带缓冲（无 <head> 的小文档）按降级链注入写出，防数据滞留。
func (w *injectWriter) drain() {
	if w.isHTML && !w.injected && len(w.buf) > 0 {
		w.forceInject()
	}
}

// emit 在锚点偏移处插入脚本标签并写出全部缓冲。
func (w *injectWriter) emit(idx int) {
	w.injected = true
	tag := []byte(`<script src="` + w.scriptURL + `"></script>`)
	out := make([]byte, 0, len(w.buf)+len(tag))
	out = append(out, w.buf[:idx]...)
	out = append(out, tag...)
	out = append(out, w.buf[idx:]...)
	w.buf = nil
	_, _ = w.ResponseWriter.Write(out)
}

// forceInject 降级链：doctype 结束之后；再无 → 文档最前（正常 HTML 不会走到）。
func (w *injectWriter) forceInject() {
	idx := doctypeEnd(w.buf)
	if idx < 0 {
		idx = 0
	}
	w.emit(idx)
}

// headAnchor 返回 b 中 <head…> 开标签结束（'>' 之后）的偏移；找不到或截断返回 -1。
// 大小写不敏感；<header 等不以空白/斜杠/闭环接的标签不匹配；属性值内的 '>' 按引号状态跳过。
func headAnchor(b []byte) int {
	for i := 0; i+5 <= len(b); i++ {
		if b[i] != '<' || !eqFold(b[i+1:i+5], "head") {
			continue
		}
		j := i + 5
		if j >= len(b) {
			return -1 // 截断：等更多数据
		}
		if c := b[j]; c != '>' && c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '/' {
			continue // <header> 等
		}
		inS, inD := false, false
		for ; j < len(b); j++ {
			switch b[j] {
			case '\'':
				if !inD {
					inS = !inS
				}
			case '"':
				if !inS {
					inD = !inD
				}
			case '>':
				if !inS && !inD {
					return j + 1
				}
			}
		}
		return -1 // 标签跨块截断
	}
	return -1
}

// doctypeEnd 返回 b 中 doctype 声明结束（'>' 之后）的偏移；找不到 -1（仅在前 2KB 找）。
func doctypeEnd(b []byte) int {
	limit := len(b)
	if limit > 2048 {
		limit = 2048
	}
	for i := 0; i+9 <= limit; i++ {
		if b[i] != '<' || b[i+1] != '!' || !eqFold(b[i+2:i+9], "doctype") {
			continue
		}
		for j := i + 9; j < limit; j++ {
			if b[j] == '>' {
				return j + 1
			}
		}
		return -1
	}
	return -1
}

// eqFold ASCII 大小写不敏感比较（lower 为小写字面量）。
func eqFold(a []byte, lower string) bool {
	if len(a) != len(lower) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if lowcase(a[i]) != lower[i] {
			return false
		}
	}
	return true
}

func lowcase(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}
