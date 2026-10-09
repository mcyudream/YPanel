// aitools_diag.go AI 工具：诊断排查场景（端口/HTTP/DNS/journal/下载）+ fail2ban（M31 二批）。
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var (
	diagPortPattern   = regexp.MustCompile(`^[0-9]{1,5}$`)
	diagDomainPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
	diagUnitPattern   = regexp.MustCompile(`^[a-zA-Z0-9@._\\-]{1,64}$`)
	diagIPPattern     = regexp.MustCompile(`^[0-9a-fA-F.:]{3,45}$`)
	diagURLPattern    = regexp.MustCompile(`^https?://[^\s'";<>]+$`)
	diagPathPattern   = regexp.MustCompile(`^/[^\s'";<>]*$`)
)

// shellQuote 单引号包裹防注入（内部单引号按 POSIX 规则转义）。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\'"'`) + "'"
}

// aiToolsDiag 诊断排查工具集（场景：服务挂了/访问不了的定位链路）。
func (s *AIService) aiToolsDiag(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "check_port", Module: aiModDiag, Risk: aiRiskRead,
			Desc: "检查端口监听状态（哪些进程在监听该端口，tcp/udp）。input JSON：{\"port\":8080,\"proto\":\"tcp\"}",
			Parameters: schObj(map[string]any{
				"port": schInt("端口号"), "proto": schEnum("协议", "tcp", "udp"),
			}, "port"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Port  int    `json:"port"`
					Proto string `json:"proto"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Port < 1 || p.Port > 65535 {
					return "", fmt.Errorf("端口不合法: %d", p.Port)
				}
				if p.Proto != "udp" {
					p.Proto = "tcp"
				}
				flag := "-tlnp"
				if p.Proto == "udp" {
					flag = "-ulnp"
				}
				return s.hostExec(ctx, fmt.Sprintf("ss %s sport = :%d", flag, p.Port))
			},
		},
		{
			Name: "http_probe", Module: aiModDiag, Risk: aiRiskRead,
			Desc: "HTTP 探测：请求 URL 返回状态码与耗时（用于验证服务连通性，支持本机/内网地址）。input JSON：{\"url\":\"http://127.0.0.1:8080/health\",\"timeout\":5}",
			Parameters: schObj(map[string]any{
				"url": schStr("完整 URL（http/https）"), "timeout": schInt("超时秒数，默认 5"),
			}, "url"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					URL     string `json:"url"`
					Timeout int    `json:"timeout"`
				}](input)
				if err != nil {
					return "", err
				}
				if !diagURLPattern.MatchString(p.URL) {
					return "", fmt.Errorf("URL 不合法（需 http/https 且无特殊字符）")
				}
				if p.Timeout < 1 || p.Timeout > 30 {
					p.Timeout = 5
				}
				return s.hostExec(ctx, fmt.Sprintf("curl -sS -m %d -o /dev/null -w 'HTTP %%{http_code}, %%{time_total}s' %s", p.Timeout, shellQuote(p.URL)))
			},
		},
		{
			Name: "dns_resolve", Module: aiModDiag, Risk: aiRiskRead,
			Desc: "DNS 解析验证：解析域名返回 IP（验证 hosts/内网 DNS 是否生效）。input JSON：{\"domain\":\"nas.internal\"}",
			Parameters: schObj(map[string]any{"domain": schStr("域名")}, "domain"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Domain string `json:"domain"`
				}](input)
				if err != nil {
					return "", err
				}
				if !diagDomainPattern.MatchString(p.Domain) {
					return "", fmt.Errorf("域名不合法")
				}
				return s.hostExec(ctx, "getent hosts "+shellQuote(p.Domain)+" || nslookup "+shellQuote(p.Domain)+" 2>&1 | tail -4")
			},
		},
		{
			Name: "journal_logs", Module: aiModDiag, Risk: aiRiskRead,
			Desc: "查看 systemd 服务日志尾部（服务排障第一入口；配合 list_services 找服务名）。input JSON：{\"unit\":\"nginx\",\"lines\":100}",
			Parameters: schObj(map[string]any{
				"unit": schStr("服务单元名（如 nginx / ypanel，@结尾实例如 ssh@）"), "lines": schInt("尾部行数，默认 100"),
			}, "unit"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Unit  string `json:"unit"`
					Lines int    `json:"lines"`
				}](input)
				if err != nil {
					return "", err
				}
				if !diagUnitPattern.MatchString(p.Unit) {
					return "", fmt.Errorf("服务名不合法")
				}
				if p.Lines < 1 {
					p.Lines = 100
				}
				if p.Lines > 500 {
					p.Lines = 500
				}
				return s.hostExec(ctx, fmt.Sprintf("journalctl -u %s -n %d --no-pager", shellQuote(p.Unit), p.Lines))
			},
		},
		{
			Name: "download_file", Module: aiModDiag, Risk: aiRiskWrite,
			Desc: "从 URL 下载文件到服务器（部署脚本/安装包等，会先向用户确认）。input JSON：{\"url\":\"https://...\",\"path\":\"/opt/down/pkg.tar.gz\"}",
			Parameters: schObj(map[string]any{
				"url": schStr("下载地址（http/https）"), "path": schStr("保存绝对路径"),
			}, "url", "path"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					URL  string `json:"url"`
					Path string `json:"path"`
				}](input)
				if err != nil {
					return "", err
				}
				if !diagURLPattern.MatchString(p.URL) || !diagPathPattern.MatchString(p.Path) {
					return "", fmt.Errorf("url 或 path 不合法")
				}
				out, err := s.hostExec(ctx, fmt.Sprintf("curl -fsSL -m 300 --create-dirs -o %s %s && ls -la %s", shellQuote(p.Path), shellQuote(p.URL), shellQuote(p.Path)))
				if err != nil {
					return "", err
				}
				return "下载完成：\n" + out, nil
			},
		},
		{
			Name: "list_fail2ban", Module: aiModNetSec, Risk: aiRiskRead,
			Desc: "查看 fail2ban 状态与封禁列表（各 jail 的当前封禁 IP）。input 传 {}。",
			Parameters: schObj(map[string]any{}),
			Fn: func(_ context.Context, _ string) (string, error) {
				out, err := s.f2b.Status(ctx)
				if err != nil {
					return "", err
				}
				return toolOut(out)
			},
		},
		{
			Name: "fail2ban_ban", Module: aiModNetSec, Risk: aiRiskDanger,
			Desc: "手动封禁 IP（立即拉黑，误封会切断访问，会先向用户确认）。input JSON：{\"jail\":\"sshd\",\"ip\":\"1.2.3.4\"}",
			Parameters: schObj(map[string]any{
				"jail": schStr("jail 名（list_fail2ban 可查）"), "ip": schStr("要封禁的 IP"),
			}, "jail", "ip"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Jail string `json:"jail"`
					IP   string `json:"ip"`
				}](input)
				if err != nil {
					return "", err
				}
				if !diagIPPattern.MatchString(p.IP) {
					return "", fmt.Errorf("IP 不合法")
				}
				if err := s.f2b.Ban(ctx, p.Jail, p.IP); err != nil {
					return "", err
				}
				return fmt.Sprintf("已封禁 %s（jail: %s）", p.IP, p.Jail), nil
			},
		},
		{
			Name: "fail2ban_unban", Module: aiModNetSec, Risk: aiRiskWrite,
			Desc: "解封 IP。input JSON：{\"jail\":\"sshd\",\"ip\":\"1.2.3.4\"}",
			Parameters: schObj(map[string]any{
				"jail": schStr("jail 名"), "ip": schStr("要解封的 IP"),
			}, "jail", "ip"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Jail string `json:"jail"`
					IP   string `json:"ip"`
				}](input)
				if err != nil {
					return "", err
				}
				if !diagIPPattern.MatchString(p.IP) {
					return "", fmt.Errorf("IP 不合法")
				}
				if err := s.f2b.Unban(ctx, p.Jail, p.IP); err != nil {
					return "", err
				}
				return fmt.Sprintf("已解封 %s（jail: %s）", p.IP, p.Jail), nil
			},
		},
	}
}
