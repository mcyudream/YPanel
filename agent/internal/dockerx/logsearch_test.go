package dockerx

import (
	"regexp"
	"testing"
	"time"
)

func TestDetectLevel(t *testing.T) {
	cases := map[string]string{
		"ERROR: connection refused":   "ERROR",
		"err: disk full":              "ERROR",
		"warning: deprecated":         "WARN",
		"WARN low memory":             "WARN",
		"INFO service started":        "INFO",
		"debug: cache miss":           "DEBUG",
		"notification sent to user":   "", // 词边界：notification 不含独立 notice
		"CRITICAL fault in core":      "FATAL",
		"panic: runtime error":        "FATAL",
		"FATAL: cannot start":         "FATAL",
		"plain line":                  "",
	}
	for line, want := range cases {
		if got := DetectLevel(line); got != want {
			t.Errorf("DetectLevel(%q) = %q, want %q", line, got, want)
		}
	}
}

func TestSplitTimestamp(t *testing.T) {
	last := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// docker 时间戳含纳秒与时区偏移
	ts, content := splitTimestamp("2026-10-08T01:46:54.760893296+08:00 INFO CORE: up", last)
	if content != "INFO CORE: up" {
		t.Fatalf("content = %q", content)
	}
	if ts.IsZero() || ts.Unix() == 0 && ts.Equal(last) {
		t.Fatalf("ts 未解析: %v", ts)
	}
	// 斜杠日期等非 RFC3339 行：整行保留、继承 lastTs
	ts2, content2 := splitTimestamp("2026/10/07 17:01:15 [notice] reload", last)
	if content2 != "2026/10/07 17:01:15 [notice] reload" || !ts2.Equal(last) {
		t.Fatalf("继承失败: ts=%v content=%q", ts2, content2)
	}
}

func TestLineParserRingAndFilter(t *testing.T) {
	re := compilePatternMust(t, "timeout")
	p := &lineParser{limit: 3, re: re}
	lines := []string{
		"2026-10-08T00:00:01Z ok-1",
		"2026-10-08T00:00:02Z timeout-2",
		"2026-10-08T00:00:03Z timeout-3",
		"2026-10-08T00:00:04Z ok-4",
		"2026-10-08T00:00:05Z timeout-5",
		"2026-10-08T00:00:06Z timeout-6",
	}
	for _, l := range lines {
		p.feed(l)
	}
	hits := p.hits()
	// 命中行：timeout-2/3/5/6 共 4 条，环形缓冲保最后 3 条
	if len(hits) != 3 {
		t.Fatalf("ring len = %d, want 3", len(hits))
	}
	wantContent := []string{"timeout-3", "timeout-5", "timeout-6"}
	for i, h := range hits {
		if h.item.Line != wantContent[i] {
			t.Errorf("hits[%d] = %q, want %q", i, h.item.Line, wantContent[i])
		}
	}
	if p.scanned != int64(len(lines)) {
		t.Errorf("scanned = %d, want %d", p.scanned, len(lines))
	}
	// 命中行带解析时间戳与级别字段（此处无级别 token）
	if !hits[0].ts.Equal(time.Date(2026, 10, 8, 0, 0, 3, 0, time.UTC)) {
		t.Errorf("ts = %v", hits[0].ts)
	}
}

func TestLineParserNegateAndPartialWrite(t *testing.T) {
	re := compilePatternMust(t, "error")
	p := &lineParser{limit: 10, re: re, negate: true}
	// 模拟 stdcopy 分片写入：一行被拆到多次 Write
	p.Write([]byte("2026-10-08T00:00:01Z has er"))
	p.Write([]byte("ror inside\n2026-10-08T00:00:02Z clean line\ntr"))
	p.flush()
	hits := p.hits()
	// negate 语义：保留未命中行（clean line、tr），排除命中行（has error inside）
	if len(hits) != 2 || hits[0].item.Line != "clean line" || hits[1].item.Line != "tr" {
		t.Fatalf("negate 过滤/分片写入失败: %+v", hits)
	}
	if p.scanned != 3 { // error 行 + clean line + tr 尾行都计入扫描
		t.Errorf("scanned = %d, want 3", p.scanned)
	}
}

func TestLineParserLevelAndNoTimestamp(t *testing.T) {
	p := &lineParser{limit: 10}
	p.feed("2026-10-08T00:00:01Z ERROR boot failed")
	p.feed("  continuation line") // 无时间戳行：继承上一条 ts
	hits := p.hits()
	if hits[0].item.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", hits[0].item.Level)
	}
	if !hits[1].ts.Equal(hits[0].ts) {
		t.Errorf("continuation 未继承时间戳: %v vs %v", hits[1].ts, hits[0].ts)
	}
}

func TestValidTimeArg(t *testing.T) {
	ok := []string{"", "15m", "1h30m", "1770000000", "2026-01-02T15:04:05Z", "2026-01-02 15:04:05", "2026-01-02", "2026-01-02T15:04:05.123456789+08:00"}
	for _, s := range ok {
		if !validTimeArg(s) {
			t.Errorf("validTimeArg(%q) = false, want true", s)
		}
	}
	bad := []string{"garbage", "2026/01/02", "15x"}
	for _, s := range bad {
		if validTimeArg(s) {
			t.Errorf("validTimeArg(%q) = true, want false", s)
		}
	}
}

func TestClampLimit(t *testing.T) {
	if clampLimit(0, 100, 500) != 100 || clampLimit(-1, 100, 500) != 100 ||
		clampLimit(999, 100, 500) != 500 || clampLimit(42, 100, 500) != 42 {
		t.Fatal("clampLimit 语义不符")
	}
}

func TestCompilePattern(t *testing.T) {
	if re, err := compilePattern(""); err != nil || re != nil {
		t.Fatalf("空正则应放行: %v %v", re, err)
	}
	if _, err := compilePattern("("); err == nil {
		t.Fatal("非法正则应报错")
	}
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := compilePattern(string(long)); err == nil {
		t.Fatal("超长正则应报错")
	}
}

func compilePatternMust(t *testing.T, p string) *regexp.Regexp {
	t.Helper()
	re, err := compilePattern(p)
	if err != nil {
		t.Fatalf("compilePattern(%q): %v", p, err)
	}
	return re
}
