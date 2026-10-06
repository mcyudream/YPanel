// 文件编码支持：编辑器编码检测/转换（raw 读取与按编码写盘）。
// 白名单与前端 TextDecoder 标签对齐；GBK 由 GB18030（超集）承载，避免编码失败。
package files

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

// encodingSet 编码白名单（id 即前端展示/回传值）。
var encodingSet = map[string]encoding.Encoding{
	"utf-8":        unicode.UTF8,
	"gb18030":      simplifiedchinese.GB18030,
	"big5":         traditionalchinese.Big5,
	"shift_jis":    japanese.ShiftJIS,
	"windows-1252": charmap.Windows1252,
}

// LookupEncoding 按 id 查白名单编码；utf-8/空 返回 nil（无需转码）。
func LookupEncoding(id string) (encoding.Encoding, bool) {
	if id == "" || id == "utf-8" {
		return nil, true
	}
	enc, ok := encodingSet[id]
	return enc, ok
}

// ReadOptions Read 扩展选项。
type ReadOptions struct {
	// Raw 为 true 时返回原始字节（ContentB64），并做二进制检测
	Raw bool
	// Encoding 指定服务端解码编码（非 utf-8 时生效）；Raw 模式下仅回显
	Encoding string
}

// decodeBytes 按白名单编码解码原始字节；encoding 为空/utf-8 时原样返回。
func decodeBytes(data []byte, encodingID string) ([]byte, string, error) {
	enc, ok := LookupEncoding(encodingID)
	if !ok {
		return nil, "", fmt.Errorf("不支持的编码: %s", encodingID)
	}
	if enc == nil {
		return data, "utf-8", nil
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return nil, "", fmt.Errorf("按 %s 解码失败: %w", encodingID, err)
	}
	return decoded, encodingID, nil
}

// encodeBytes 按白名单编码编码文本；encoding 为空/utf-8 时原样返回。
func encodeBytes(text []byte, encodingID string) ([]byte, error) {
	enc, ok := LookupEncoding(encodingID)
	if !ok {
		return nil, fmt.Errorf("不支持的编码: %s", encodingID)
	}
	if enc == nil {
		return text, nil
	}
	encoded, err := enc.NewEncoder().Bytes(text)
	if err != nil {
		return nil, fmt.Errorf("按 %s 编码失败: %w", encodingID, err)
	}
	return encoded, nil
}

// isBinaryData 前 8KiB 含 NUL 即视为二进制。
func isBinaryData(data []byte) bool {
	head := data
	if len(head) > 8192 {
		head = head[:8192]
	}
	return bytes.IndexByte(head, 0) >= 0
}

// b64Encode base64 编码（标准填充）。
func b64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
