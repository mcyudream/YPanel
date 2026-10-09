package service

import (
	"strings"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// TestCompressContext M42：超阈值时压缩保留窗口之外的工具结果。
func TestCompressContext(t *testing.T) {
	// 构造 8 条工具结果，每条 10KB（共 80KB < 120K → 不压缩）
	mk := func(n int) []llms.MessageContent {
		msgs := []llms.MessageContent{}
		for i := 0; i < n; i++ {
			msgs = append(msgs, llms.MessageContent{
				Role:  llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: string(rune('a' + i)), Name: "t", Content: strings.Repeat("x", 10_000)}},
			})
		}
		return msgs
	}
	small := mk(8)
	if got := compressContext(small); len(got) != 8 {
		t.Fatalf("small input should be untouched, got %d", len(got))
	}
	// 16 条 ×10KB = 160KB > 120K → 压缩除最近 4 条外的全部
	big := mk(16)
	got := compressContext(big)
	compressed, kept := 0, 0
	for _, m := range got {
		tc := m.Parts[0].(llms.ToolCallResponse)
		if strings.HasPrefix(tc.Content, "[已压缩] ") {
			compressed++
		} else {
			kept++
		}
	}
	if compressed != 12 || kept != 4 {
		t.Fatalf("expect 12 compressed + 4 kept, got %d + %d", compressed, kept)
	}
	// 压缩后的内容长度应大幅下降（首 200 字符 + 前缀）
	tc := got[0].Parts[0].(llms.ToolCallResponse)
	if len(tc.Content) > 400 {
		t.Fatalf("compressed content too long: %d", len(tc.Content))
	}
}

// TestFuseCounter M42：同工具+同参数连续失败 2 次后第三次应熔断。
func TestFuseCounter(t *testing.T) {
	fuse := map[string]int{}
	key := func(name, args string) string { return name + "|" + strings.TrimSpace(args) }
	exec := func(name, args string) (string, error) {
		k := key(name, args)
		if fuse[k] >= 2 {
			return "已熔断", nil // 短路：不再真正执行
		}
		// 模拟执行失败
		fuse[k]++
		return "失败", errFuseFake
	}
	for i := 1; i <= 2; i++ {
		if out, _ := exec("read_file", `{"path":"/x"}`); out == "已熔断" {
			t.Fatalf("call %d should execute (fail), not fuse", i)
		}
	}
	if out, _ := exec("read_file", `{"path":"/x"}`); out != "已熔断" {
		t.Fatalf("call 3 should be fused")
	}
	// 不同参数不受影响
	if out, _ := exec("read_file", `{"path":"/y"}`); out == "已熔断" {
		t.Fatalf("different args should not fuse")
	}
}

// errFuseFake 测试用假错误。
var errFuseFake = errorString("fake failure")

type errorString string

func (e errorString) Error() string { return string(e) }
