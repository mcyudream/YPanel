package service

import (
	"strings"
	"testing"
)

func TestParseCentralLines(t *testing.T) {
	body := strings.Join([]string{
		`{"_msg":"b line","_stream":"{container=\"nginx\"}","_time":"2026-10-08T09:00:02Z","extra":"x"}`,
		``,
		`{"_msg":"a line","_stream":"{container=\"app\"}","_time":"2026-10-08T09:00:01Z"}`,
		`not-json-garbage`,
		`{"_msg":"c line","_stream":"{container=\"nginx\"}","_time":"2026-10-08T09:00:03Z"}`,
	}, "\n")
	out, err := parseCentralLines(strings.NewReader(body), 10)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out.Truncated || len(out.Items) != 3 {
		t.Fatalf("items=%d truncated=%v", len(out.Items), out.Truncated)
	}
	if out.Items[0].Msg != "a line" || out.Items[1].Msg != "b line" || out.Items[2].Msg != "c line" {
		t.Fatalf("排序错误: %+v", out.Items)
	}
	if out.Items[0].Stream != `{container="app"}` {
		t.Fatalf("stream 字段丢失: %+v", out.Items[0])
	}

	out2, err := parseCentralLines(strings.NewReader(body), 2)
	if err != nil {
		t.Fatalf("parse2: %v", err)
	}
	if !out2.Truncated || len(out2.Items) != 2 {
		t.Fatalf("limit 截断失败: items=%d truncated=%v", len(out2.Items), out2.Truncated)
	}
}
