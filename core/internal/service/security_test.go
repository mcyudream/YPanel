package service

import (
	"fmt"
	"testing"
	"time"
)

// TestTOTPRFCVector RFC 6238 附录 B 标准向量（SHA1/8位 → 取后 6 位）。
func TestTOTPRFCVector(t *testing.T) {
	// RFC 6238 测试密钥（ASCII: 12345678901234567890）的 base32
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	// RFC 6238 附录 B：T=59 → 94287082（8位，取后6位 287082）
	got, err := totpCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("T=59 → %s（期望 287082）\n", got)
	if got != "287082" {
		t.Fatalf("RFC 向量不匹配: got %s want 287082", got)
	}
	// 与标准实现交叉验证：当前时刻的码
	now := time.Now()
	got2, _ := totpCode(secret, now)
	fmt.Printf("now → %s\n", got2)
}
