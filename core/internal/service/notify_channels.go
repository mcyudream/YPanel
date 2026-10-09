// 通知渠道扩展（M37）：bark / email(SMTP) 发送 + SMTP 全局设置。
package service

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// SMTPConfig 告警邮件通道全局配置（Setting 键 smtp.*）。
type SMTPConfig struct {
	Host   string
	Port   int
	User   string
	Pass   string
	From   string
	SSL    bool // true=465 隐式 TLS；false=STARTTLS(587) 或明文
	ToAddr string
}

// LoadSMTPConfig 从设置读取 SMTP 配置（settings 服务注入）。
func LoadSMTPConfig(settings *SettingService) SMTPConfig {
	get := func(k string) string {
		v := settings.Get("smtp."+k, "")
		return v
	}
	port, _ := strconv.Atoi(get("port"))
	return SMTPConfig{
		Host: get("host"), Port: port, User: get("user"), Pass: get("pass"),
		From: get("from"), SSL: get("ssl") == "true", ToAddr: get("to"),
	}
}

// SendMail 经 SMTP 发送（465 隐式 TLS / 587 STARTTLS / 明文）。
func SendMail(ctx context.Context, cfg SMTPConfig, subject, body string) error {
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" || cfg.ToAddr == "" {
		return errs.Wrap(errs.ErrBadRequest, "SMTP 配置不完整（host/port/from/收件人）")
	}
	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	to := strings.Split(cfg.ToAddr, ",")
	header := strings.Join([]string{
		"From: " + cfg.From,
		"To: " + strings.Join(to, ", "),
		"Subject: " + mimeEncodeHeader(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n") + "\r\n"
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	d := net.Dialer{}
	var err error
	if cfg.SSL {
		conn, derr := d.DialContext(ctx, "tcp", addr)
		if derr != nil {
			return derr
		}
		tconn := tls.Client(conn, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		cli, cerr := smtp.NewClient(tconn, cfg.Host)
		if cerr != nil {
			return cerr
		}
		defer func() { _ = cli.Quit() }()
		err = sendMailFlow(cli, auth, cfg.From, to, header)
	} else {
		sendCtx := func() error {
			cli, cerr := smtp.Dial(addr)
			if cerr != nil {
				return cerr
			}
			defer func() { _ = cli.Quit() }()
			if ok, _ := cli.Extension("STARTTLS"); ok {
				if terr := cli.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); terr != nil {
					return terr
				}
			}
			return sendMailFlow(cli, auth, cfg.From, to, header)
		}
		done := make(chan error, 1)
		go func() { done <- sendCtx() }()
		select {
		case err = <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	if err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "SMTP 发送失败: "+err.Error())
	}
	return nil
}

func sendMailFlow(cli *smtp.Client, auth smtp.Auth, from string, to []string, msg string) error {
	if err := cli.Hello("localhost"); err != nil {
		_ = err // HELO 失败不少服务器仍可继续
	}
	if err := cli.Auth(auth); err != nil && auth != nil {
		return fmt.Errorf("认证失败: %w", err)
	}
	if err := cli.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := cli.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return err
		}
	}
	w, err := cli.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	_ = w.Close()
	return nil
}

// mimeEncodeHeader 非 ASCII 主题 RFC 2047 编码。
func mimeEncodeHeader(s string) string {
	if isASCII(s) {
		return s
	}
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// SendBark bark 推送（url = https://api.day.app/<deviceKey>，GET 语义但走 POST JSON 亦可；这里用官方 GET 路径形式）。
func SendBark(ctx context.Context, key, title, body string) error {
	key = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(key), "https://api.day.app/"))
	if key == "" || strings.ContainsAny(key, "/?#") {
		return errs.Wrap(errs.ErrBadRequest, "bark deviceKey 不合法")
	}
	u := fmt.Sprintf("https://api.day.app/%s/%s/%s", key, url.PathEscape(title), url.PathEscape(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "bark 推送失败: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", fmt.Sprintf("bark 返回 HTTP %d", resp.StatusCode))
	}
	return nil
}
