// Package fcgi 实现最小 FastCGI 客户端（Responder 角色、单请求单连接），
// 供 agent 侧直拨 php-fpm 的 pm.status_path（FPM 状态页）等场景，纯标准库实现。
package fcgi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	versionFCGI = 1

	typeBeginRequest = 1
	typeEndRequest   = 3
	typeParams       = 4
	typeStdIn        = 5
	typeStdOut       = 6
	typeStdErr       = 7

	roleResponder = 1
	requestID     = 1

	maxRecordBody = 65535
)

var errRequestAborted = errors.New("fcgi: 对端中止请求")

type connWriter struct {
	w   io.Writer
	buf []byte
}

func (cw *connWriter) writeRecord(typ byte, body []byte) error {
	hdr := make([]byte, 8)
	hdr[0] = versionFCGI
	hdr[1] = typ
	binary.BigEndian.PutUint16(hdr[2:4], requestID)
	binary.BigEndian.PutUint16(hdr[4:6], uint16(len(body)))
	// hdr[6] padding=0, hdr[7] reserved=0
	if _, err := cw.w.Write(hdr); err != nil {
		return err
	}
	if len(body) > 0 {
		if _, err := cw.w.Write(body); err != nil {
			return err
		}
	}
	return nil
}

func encodePairLen(n int) []byte {
	if n < 128 {
		return []byte{byte(n)}
	}
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n)|1<<31)
	return b
}

func writePairs(cw *connWriter, params map[string]string) error {
	body := make([]byte, 0, 256)
	for k, v := range params {
		body = append(body, encodePairLen(len(k))...)
		body = append(body, encodePairLen(len(v))...)
		body = append(body, k...)
		body = append(body, v...)
	}
	for len(body) > maxRecordBody {
		if err := cw.writeRecord(typeParams, body[:maxRecordBody]); err != nil {
			return err
		}
		body = body[maxRecordBody:]
	}
	if err := cw.writeRecord(typeParams, body); err != nil {
		return err
	}
	return cw.writeRecord(typeParams, nil)
}

// Do 发起一次 FastCGI 请求并返回 STDOUT 原始输出（含 CGI 响应头）。
func Do(ctx context.Context, network, addr string, params map[string]string, timeout time.Duration) ([]byte, error) {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("fcgi: 连接 %s 失败: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	cw := &connWriter{w: conn}
	begin := make([]byte, 8)
	binary.BigEndian.PutUint16(begin[0:2], roleResponder)
	if err := cw.writeRecord(typeBeginRequest, begin); err != nil {
		return nil, fmt.Errorf("fcgi: begin request: %w", err)
	}
	if err := writePairs(cw, params); err != nil {
		return nil, fmt.Errorf("fcgi: params: %w", err)
	}
	if err := cw.writeRecord(typeStdIn, nil); err != nil {
		return nil, fmt.Errorf("fcgi: stdin: %w", err)
	}

	var out []byte
	hdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(conn, hdr); err != nil {
			return nil, fmt.Errorf("fcgi: 读取响应: %w", err)
		}
		contentLen := int(hdr[4])<<8 | int(hdr[5])
		padLen := int(hdr[6])
		body := make([]byte, contentLen+padLen)
		if _, err := io.ReadFull(conn, body); err != nil {
			return nil, fmt.Errorf("fcgi: 读取记录体: %w", err)
		}
		switch hdr[1] {
		case typeStdOut:
			out = append(out, body[:contentLen]...)
		case typeStdErr:
			// 状态页场景忽略 stderr
		case typeEndRequest:
			// body: appStatus(4) + protocolStatus(1) + reserved(3)
			if len(body) >= 5 && body[4] != 0 { // protocolStatus != FCGI_REQUEST_COMPLETE
				return nil, errRequestAborted
			}
			return out, nil
		}
	}
}

// StripCGIHeaders 剥离 CGI 响应头（到第一个空行），返回 body。
func StripCGIHeaders(raw []byte) string {
	s := string(raw)
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if i := strings.Index(s, sep); i >= 0 {
			return s[i+len(sep):]
		}
	}
	return s
}
