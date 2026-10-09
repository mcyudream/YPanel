// SecuritySettingsService 面板安全基线设置（2FA/安全入口/IP 白名单/会话超时）。
// 存储复用 SettingService 键值表；由 auth 中间件与登录流程消费。
package service

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"github.com/ypanel/shared/errs"
)

// SecuritySettings 安全基线键值。
const (
	KeyTwoFAEnabled     = "security.2fa_enabled"      // "1"/"0"（admin 全局开关）
	KeyTwoFASecret      = "security.2fa_secret"       // TOTP 密钥（base32）
	KeySafeEntry        = "security.safe_entry"       // 安全入口路径段，如 "my-panel"；空=关闭
	KeyAllowedIPs       = "security.allowed_ips"      // 逗号分隔白名单；空=不限制
	KeySessionTimeoutHr = "security.session_timeout"  // 会话超时（小时），默认 24
	KeyMinPasswordLen   = "security.min_password_len" // 密码最小长度（M40，0=不限制）
)

// SecuritySettingsData 设置数据。
type SecuritySettingsData struct {
	TwoFAEnabled   bool   `json:"twoFaEnabled"`
	TwoFASecret    string `json:"twoFaSecret,omitempty"` // 仅绑定流程返回
	SafeEntry      string `json:"safeEntry"`
	AllowedIPs     string `json:"allowedIps"`
	SessionHours   int    `json:"sessionHours"`
	MinPasswordLen int    `json:"minPasswordLen"`
	DangerLock     bool   `json:"dangerLock"`
}

// SecuritySettingsService 安全设置服务。
type SecuritySettingsService struct {
	settings *SettingService
}

// NewSecuritySettingsService 创建。
func NewSecuritySettingsService(settings *SettingService) *SecuritySettingsService {
	return &SecuritySettingsService{settings: settings}
}

// Get 读取当前安全设置（secret 不回显）。
func (s *SecuritySettingsService) Get(ctx context.Context) SecuritySettingsData {
	enabled := s.settings.Get(KeyTwoFAEnabled, "0") == "1"
	hours, _ := strconv.Atoi(s.settings.Get(KeySessionTimeoutHr, "24"))
	if hours <= 0 {
		hours = 24
	}
	minLen, _ := strconv.Atoi(s.settings.Get(KeyMinPasswordLen, "0"))
	dangerLock := s.DangerLockEnabled()
	return SecuritySettingsData{
		TwoFAEnabled:   enabled,
		TwoFASecret:    "", // 密钥不回显，仅 2FA setup 时返回一次
		SafeEntry:      s.settings.Get(KeySafeEntry, ""),
		AllowedIPs:     s.settings.Get(KeyAllowedIPs, ""),
		SessionHours:   hours,
		MinPasswordLen: minLen,
		DangerLock:     dangerLock,
	}
}

// GenerateSafeEntry 生成 8 位随机安全入口（无易混淆字符）。
func GenerateSafeEntry() string {
	const chars = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

// EnsureSafeEntry 启动时确保安全入口已开启（M49 强制策略）：为空则生成 8 位随机入口。
func (s *SecuritySettingsService) EnsureSafeEntry() (string, bool, error) {
	entry := strings.Trim(s.settings.Get(KeySafeEntry, ""), "/ ")
	if entry != "" {
		return entry, false, nil
	}
	entry = GenerateSafeEntry()
	if err := s.settings.Set(KeySafeEntry, entry); err != nil {
		return "", false, err
	}
	return entry, true, nil
}

// CheckPasswordPolicy 密码策略校验（M40）。len<=0 表示不限制。
func (s *SecuritySettingsService) CheckPasswordPolicy(password string) error {
	minLen, _ := strconv.Atoi(s.settings.Get(KeyMinPasswordLen, "0"))
	if minLen > 0 && len([]rune(password)) < minLen {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("密码长度不足（至少 %d 位）", minLen))
	}
	return nil
}

// SetMinPasswordLen 设置密码最小长度。
func (s *SecuritySettingsService) SetMinPasswordLen(n int) error {
	if n < 0 || n > 64 {
		return errs.Wrap(errs.ErrBadRequest, "密码最小长度需在 0-64（0=不限制）")
	}
	err := s.settings.Set(KeyMinPasswordLen, strconv.Itoa(n))
	return err
}

// Enable2FA 生成并启用 2FA 密钥（返回 secret 与 otpauth URI 供绑定）。
func (s *SecuritySettingsService) Enable2FA(ctx context.Context, issuer, account string) (secret, uri string, err error) {
	secret, err = GenerateTOTPSecret()
	if err != nil {
		return "", "", err
	}
	if err := s.settings.Set(KeyTwoFASecret, secret); err != nil {
		return "", "", err
	}
	if err := s.settings.Set(KeyTwoFAEnabled, "1"); err != nil {
		return "", "", err
	}
	return secret, OtpauthURI(issuer, account, secret), nil
}

// Disable2FA 关闭 2FA。
func (s *SecuritySettingsService) Disable2FA(ctx context.Context) error {
	return s.settings.Set(KeyTwoFAEnabled, "0")
}

// Update 更新入口/白名单/会话超时（clientIP 用于防自锁：白名单非空时当前 IP 必须在列）。
func (s *SecuritySettingsService) Update(ctx context.Context, safeEntry, allowedIPs string, sessionHours int, clientIP string) error {
	// M49：安全入口强制开启，长度 ≥6 位
	safeEntry = strings.Trim(safeEntry, "/ ")
	if len(safeEntry) < 6 {
		return errs.Wrap(errs.ErrBadRequest, "安全入口强制开启且长度不得小于 6 位")
	}
	for _, r := range safeEntry {
		if !isSafeEntryRune(r) {
			return errs.Wrap(errs.ErrBadRequest, "安全入口仅允许字母/数字/中划线/下划线")
		}
	}
	for _, ip := range strings.Split(allowedIPs, ",") {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if !isAllowedIPFormat(ip) {
			return errs.Wrap(errs.ErrBadRequest, "IP 白名单格式不合法: "+ip)
		}
	}
	if sessionHours <= 0 || sessionHours > 24*30 {
		return errs.Wrap(errs.ErrBadRequest, "会话超时需在 1-720 小时")
	}
	cleaned := make([]string, 0, 8)
	for _, ip := range strings.Split(allowedIPs, ",") {
		if ip = strings.TrimSpace(ip); ip != "" {
			cleaned = append(cleaned, ip)
		}
	}
	if len(cleaned) > 0 && clientIP != "" && !ipAllowedIn(clientIP, strings.Join(cleaned, ",")) {
		return errs.Wrap(errs.ErrBadRequest, "当前 IP "+clientIP+" 不在新白名单内，保存后将无法访问面板；请先把当前 IP 加入白名单")
	}
	if err := s.settings.Set(KeySafeEntry, safeEntry); err != nil {
		return err
	}
	if err := s.settings.Set(KeyAllowedIPs, strings.Join(cleaned, ",")); err != nil {
		return err
	}
	return s.settings.Set(KeySessionTimeoutHr, strconv.Itoa(sessionHours))
}

// IsIPAllowed 白名单校验（空=放行所有）。
func (s *SecuritySettingsService) IsIPAllowed(ip string) bool {
	return ipAllowedIn(ip, s.settings.Get(KeyAllowedIPs, ""))
}

// ipAllowedIn 按给定白名单串（逗号分隔，支持 a.b.c.* 通配）校验；空串放行。
func ipAllowedIn(ip, raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	for _, allowed := range strings.Split(raw, ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(ip, prefix) {
				return true
			}
		} else if allowed == ip {
			return true
		}
	}
	return false
}

// SessionHours 会话超时小时数。
func (s *SecuritySettingsService) SessionHours() int {
	h, _ := strconv.Atoi(s.settings.Get(KeySessionTimeoutHr, "24"))
	if h <= 0 {
		h = 24
	}
	return h
}

// SafeEntry 安全入口段。
func (s *SecuritySettingsService) SafeEntry() string {
	return strings.TrimSpace(s.settings.Get(KeySafeEntry, ""))
}

// TwoFAEnabled 是否启用 2FA。
func (s *SecuritySettingsService) TwoFAEnabled() bool {
	return s.settings.Get(KeyTwoFAEnabled, "0") == "1"
}

// TwoFASecret 取密钥（登录校验用）。
func (s *SecuritySettingsService) TwoFASecret() string {
	return s.settings.Get(KeyTwoFASecret, "")
}

func isSafeEntryRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'
}

func isAllowedIPFormat(ip string) bool {
	// 三种形式：精确 IP、CIDR（a.b.c.d/nn）、通配（a.b.c.* 或 a.b.* 等）
	if strings.HasSuffix(ip, ".*") {
		parts := strings.Split(strings.TrimSuffix(ip, ".*"), ".")
		if len(parts) == 0 || len(parts) > 3 {
			return false
		}
		for _, o := range parts {
			if o == "" || len(o) > 3 {
				return false
			}
			for _, c := range o {
				if c < '0' || c > '9' {
					return false
				}
			}
		}
		return true
	}
	// 精确 IP 或 CIDR 简校验
	parts := strings.Split(ip, "/")
	if len(parts) > 2 {
		return false
	}
	octets := strings.Split(parts[0], ".")
	if len(octets) != 4 {
		return false
	}
	for _, o := range octets {
		n := 0
		if len(o) == 0 || len(o) > 3 {
			return false
		}
		for _, c := range o {
			if c < '0' || c > '9' {
				return false
			}
			n = n*10 + int(c-'0')
		}
		if n > 255 {
			return false
		}
	}
	if len(parts) == 2 {
		bits, err := strconv.Atoi(parts[1])
		if err != nil || bits < 0 || bits > 32 {
			return false
		}
	}
	return true
}
