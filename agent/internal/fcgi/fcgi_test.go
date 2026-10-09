package fcgi

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestEncodePairLen(t *testing.T) {
	if got := encodePairLen(10); len(got) != 1 || got[0] != 10 {
		t.Fatalf("短长度编码错误: %v", got)
	}
	got := encodePairLen(300)
	if len(got) != 4 {
		t.Fatalf("长长度应为 4 字节: %v", got)
	}
	v := binary.BigEndian.Uint32(got)
	if v&(1<<31) == 0 || v&^(1<<31) != 300 {
		t.Fatalf("长长度编码错误: %v", got)
	}
}

// TestWritePairsParams 编码键值对 → 解回，验证 params 帧格式正确。
func TestWritePairsParams(t *testing.T) {
	var buf bytes.Buffer
	cw := &connWriter{w: &buf}
	params := map[string]string{"SCRIPT_NAME": "/status", "QUERY_STRING": "full"}
	if err := writePairs(cw, params); err != nil {
		t.Fatalf("writePairs: %v", err)
	}
	raw := buf.Bytes()
	decoded := map[string]string{}
	for len(raw) >= 8 {
		typ := raw[1]
		cl := int(raw[4])<<8 | int(raw[5])
		pad := int(raw[6])
		body := raw[8 : 8+cl]
		raw = raw[8+cl+pad:]
		if typ != typeParams {
			t.Fatalf("期望 Params 记录，得到 type=%d", typ)
		}
		for len(body) > 0 {
			nl, n := decodeLen(body)
			body = body[n:]
			vl, n := decodeLen(body)
			body = body[n:]
			if len(body) < nl+vl {
				t.Fatalf("params 体越界")
			}
			decoded[string(body[:nl])] = string(body[nl : nl+vl])
			body = body[nl+vl:]
		}
	}
	if decoded["SCRIPT_NAME"] != "/status" || decoded["QUERY_STRING"] != "full" {
		t.Fatalf("params 解码不符: %v", decoded)
	}
}

func decodeLen(b []byte) (int, int) {
	if b[0]&0x80 == 0 {
		return int(b[0]), 1
	}
	v := binary.BigEndian.Uint32(b)
	return int(v &^ (1 << 31)), 4
}

func TestStripCGIHeaders(t *testing.T) {
	raw := []byte("Content-type: text/plain\r\n\r\npool: www\r\nprocesses: 3\r\n")
	body := StripCGIHeaders(raw)
	if !strings.HasPrefix(body, "pool: www") {
		t.Fatalf("CRLF 头剥离失败: %q", body)
	}
	raw2 := []byte("Content-type: text/plain\n\npool: www")
	if body2 := StripCGIHeaders(raw2); !strings.HasPrefix(body2, "pool: www") {
		t.Fatalf("LF 头剥离失败: %q", body2)
	}
	if body3 := StripCGIHeaders([]byte("no headers")); body3 != "no headers" {
		t.Fatalf("无头内容应原样返回: %q", body3)
	}
}
