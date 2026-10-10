package gwserver

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadAnchor(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"裸 head", "<html><head>", 12},
		{"带属性", `<head lang="zh-CN">`, 19},
		{"大小写", "<HEAD>", 6},
		{"自闭合", "<head/>", 7},
		{"header 不匹配", "<header class=x>", -1},
		{"headx 不匹配", "<headx>", -1},
		{"属性值含大于号", `<head data-x="a>b" data-y='c>d'>`, 32},
		{"标签截断", "<head lang", -1},
		{"head 字样但非标签", "xhead>", -1},
		{"无 head", "<html><body>", -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := headAnchor([]byte(c.in)); got != c.want {
				t.Fatalf("headAnchor(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestHeadAnchorPartialAccumulate(t *testing.T) {
	// 分块到达：<head 落在前块边界，补齐后应命中
	b := []byte("<html><hea")
	if got := headAnchor(b); got != -1 {
		t.Fatalf("截断标签应返回 -1，got %d", got)
	}
	b = append(b, []byte("d><title>x")...)
	if got := headAnchor(b); got != 12 {
		t.Fatalf("补齐后应命中 12，got %d", got)
	}
}

func TestDoctypeEnd(t *testing.T) {
	if got := doctypeEnd([]byte("<!DOCTYPE html><html>")); got != 15 {
		t.Fatalf("doctypeEnd = %d, want 15", got)
	}
	if got := doctypeEnd([]byte("<!doctype html>")); got != 15 {
		t.Fatalf("小写 = %d, want 15", got)
	}
	if got := doctypeEnd([]byte("<html>")); got != -1 {
		t.Fatalf("无 doctype 应 -1，got %d", got)
	}
}

func TestSplitCrossHost(t *testing.T) {
	b64 := base64.RawURLEncoding.EncodeToString([]byte("http://192.168.1.9:9000"))
	b64b := base64.RawURLEncoding.EncodeToString([]byte("http://10.0.0.2:80"))
	cases := []struct {
		rest          string
		wantB64       string
		wantAfter     string
		wantOK        bool
	}{
		{"/~h/" + b64 + "/api/x", b64, "/api/x", true},
		{"/~h/" + b64, b64, "/", true},
		{"/~h/" + b64b + "/", b64b, "/", true},
		{"/api/x", "", "", false},
		{"/", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		gotB64, gotAfter, ok := splitCrossHost(c.rest)
		if ok != c.wantOK || gotB64 != c.wantB64 || gotAfter != c.wantAfter {
			t.Fatalf("splitCrossHost(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.rest, gotB64, gotAfter, ok, c.wantB64, c.wantAfter, c.wantOK)
		}
	}
}

func TestDecodeCrossOrigin(t *testing.T) {
	ok := base64.RawURLEncoding.EncodeToString([]byte("https://192.168.5.5:8443"))
	got, err := decodeCrossOrigin(ok)
	if err != nil || got != "https://192.168.5.5:8443" {
		t.Fatalf("decodeCrossOrigin = (%q,%v)", got, err)
	}
	bad := []string{
		base64.RawURLEncoding.EncodeToString([]byte("ftp://x/")),
		base64.RawURLEncoding.EncodeToString([]byte("http://")),
		"!!!invalid-b64!!!",
	}
	for _, b := range bad {
		if _, err := decodeCrossOrigin(b); err == nil {
			t.Fatalf("应拒绝 %q", b)
		}
	}
}

// flushRecorder 记录底层 Flush 调用次数。
type flushRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (f *flushRecorder) Flush() { f.flushes++ }

func newHTMLInjector(rec *flushRecorder) *injectWriter {
	iw := newInjectWriter(rec, "/s/abc/__yp_proxy.js", http.MethodGet)
	rec.Header().Set("Content-Type", "text/html; charset=utf-8")
	return iw
}

func TestInjectWriterHTML(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	iw := newHTMLInjector(rec)
	iw.WriteHeader(http.StatusOK)
	_, _ = iw.Write([]byte(`<!doctype html><html><head><title>t</title></head><body>hi</body></html>`))
	iw.drain()
	body := rec.Body.String()
	want := `<!doctype html><html><head><script src="/s/abc/__yp_proxy.js"></script><title>`
	if !strings.HasPrefix(body, want) {
		t.Fatalf("注入位置错误:\n%s", body)
	}
	if !strings.Contains(body, "hi</body></html>") {
		t.Fatalf("正文丢失:\n%s", body)
	}
}

func TestInjectWriterCLStripped(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	rec.Header().Set("Content-Length", "123")
	iw := newHTMLInjector(rec)
	iw.WriteHeader(http.StatusOK)
	if v := rec.Header().Get("Content-Length"); v != "" {
		t.Fatalf("html 注入模式应剥 Content-Length，got %q", v)
	}
}

func TestInjectWriterNonHTMLPass(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "application/json")
	iw := newInjectWriter(rec, "/s/abc/__yp_proxy.js", http.MethodGet)
	iw.WriteHeader(http.StatusOK)
	payload := `{"head":"<head>"}`
	_, _ = iw.Write([]byte(payload))
	iw.drain()
	if got := rec.Body.String(); got != payload {
		t.Fatalf("非 html 应零改动透传，got %q", got)
	}
}

func TestInjectWriterNoHeadSmallDoc(t *testing.T) {
	// 无 <head> 的小文档：Write/drain 后必须完整收尾（降级链），数据不得滞留
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	iw := newHTMLInjector(rec)
	_, _ = iw.Write([]byte("<!doctype html><html><body>x</body></html>"))
	iw.drain()
	body := rec.Body.String()
	want := `<!doctype html><script src="/s/abc/__yp_proxy.js"></script><html>`
	if !strings.HasPrefix(body, want) {
		t.Fatalf("doctype 降级注入错误:\n%s", body)
	}
}

func TestInjectWriterNoDoctypeFront(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	iw := newHTMLInjector(rec)
	_, _ = iw.Write([]byte("<p>fragment</p>"))
	iw.drain()
	if !strings.HasPrefix(rec.Body.String(), `<script src="/s/abc/__yp_proxy.js"></script><p>`) {
		t.Fatalf("无 doctype 应注入文档最前:\n%s", rec.Body.String())
	}
}

func TestInjectWriterFlushSuppressed(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	iw := newHTMLInjector(rec)
	_, _ = iw.Write([]byte("<!do")) // 第一个极小 chunk
	iw.Flush()                  // 上游 FlushInterval=-1 的冲刷应被抑制（锚点未决）
	if rec.flushes != 0 {
		t.Fatalf("html 未注入完成前 Flush 应被抑制，got %d 次", rec.flushes)
	}
	_, _ = iw.Write([]byte("ctype html><html><head></head><body>ok</body></html>"))
	iw.Flush()
	if rec.flushes != 1 {
		t.Fatalf("注入完成后 Flush 应透传，got %d 次", rec.flushes)
	}
}

func TestInjectWriterHeadSplitAcrossChunks(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	iw := newHTMLInjector(rec)
	_, _ = iw.Write([]byte("<html><head"))
	_, _ = iw.Write([]byte(`er><title>`)) // <header>：不作为锚点
	_, _ = iw.Write([]byte("</header><head><meta charset=utf-8>"))
	iw.drain()
	if !strings.Contains(rec.Body.String(), `</script><meta charset=utf-8>`) {
		t.Fatalf("应注入在真正的 <head> 后:\n%s", rec.Body.String())
	}
}

func TestInjectWriterHeadOversize(t *testing.T) {
	// 超过缓冲上限仍无 head：强制降级注入，不悬挂
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	iw := newHTMLInjector(rec)
	big := make([]byte, injectBufMax) // 恰好占满上限且无锚点
	for i := range big {
		big[i] = 'x'
	}
	_, _ = iw.Write(big)
	if !iw.injected {
		t.Fatal("缓冲满应触发降级注入")
	}
	tag := `<script src="/s/abc/__yp_proxy.js"></script>`
	if !strings.HasPrefix(rec.Body.String(), tag) {
		t.Fatalf("无锚点应注入文档最前:\n%.80s", rec.Body.String())
	}
	if got := rec.Body.Len(); got != injectBufMax+len(tag) {
		t.Fatalf("降级后应全量写出，got %d", got)
	}
}

func TestInjectWriterHeadMethodSkipped(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec.Header().Set("Content-Type", "text/html")
	iw := newInjectWriter(rec, "/s/abc/__yp_proxy.js", http.MethodHead)
	iw.WriteHeader(http.StatusOK)
	_, _ = iw.Write([]byte("<html><head></head></html>"))
	iw.drain()
	if strings.Contains(rec.Body.String(), "<script") {
		t.Fatal("HEAD 请求不应注入")
	}
}
