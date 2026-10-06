// 安全基线：TOTP 2FA（RFC 6238，SHA1/6位/30s 步长，与主流验证器兼容）。
// 不引第三方依赖：crypto/hmac 手写，QR 由前端用 otpauth URI 自行渲染。
package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// TOTP 常量。
const (
	totpStep    = 30 * time.Second
	totpDigits  = 6
	totpSkew    = 1 // 允许前后各 1 个步长
)

// GenerateTOTPSecret 生成 TOTP 密钥（20 字节随机，base32 无填充）。
func GenerateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// TOTPCode 计算指定时间步的 6 位验证码。
func totpCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(
		strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if err != nil {
		return "", err
	}
	counter := uint64(t.Unix() / int64(totpStep/time.Second))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	// RFC 4226 动态截断后需模 10^digits
	const hotpMod = 1000000
	return fmt.Sprintf("%0*d", totpDigits, code%hotpMod), nil
}

// VerifyTOTP 校验验证码（允许前后各 1 步长漂移）。
func VerifyTOTP(secret, code string) error {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return errs.Wrap(errs.ErrBadRequest, "验证码格式错误")
	}
	now := time.Now()
	for _, d := range []time.Duration{0, -totpStep, totpStep} {
		want, err := totpCode(secret, now.Add(d))
		if err != nil {
			return err
		}
		if want == code {
			return nil
		}
	}
	return errs.Wrap(errs.ErrBadRequest, "验证码错误")
}

// OtpauthURI 生成 otpauth:// URI（验证器扫码/手动输入用）。
func OtpauthURI(issuer, account, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		issuer, account, secret, issuer, totpDigits, int(totpStep/time.Second))
}
