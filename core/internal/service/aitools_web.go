// aitools_web.go AI 联网查询工具（M33）：web_search（Bing 搜索结果抓取）+ fetch_url（网页正文提取）。
// 出站复用 fileext 的 checkPublicHost 逐跳 SSRF 防护（拒绝内网/保留地址）。
package service

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"net/url"
	"strings"
	"time"
)

var (
	wsTitleRe     = regexp.MustCompile(`(?s)<h2><a[^>]*href="(http[^"]+)"[^>]*>(.*?)</a>`)
	wsSnipRe      = regexp.MustCompile(`(?s)<p class="b_lineclamp[^"]*">(.*?)</p>`)
	scriptBlockRe = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>`)
	anyTagRe      = regexp.MustCompile(`<[^>]+>`)
)

// webLookupClient 联网查询专用 HTTP 客户端（超时 + UA；SSRF 由 checkPublicHost 逐跳校验）。
func webLookupClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func stripTags(s string) string {
	s = scriptBlockRe.ReplaceAllString(s, " ")
	s = anyTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// aiToolsWeb 联网查询模块。
func (s *AIService) aiToolsWeb(ctx context.Context) []aiToolDef {
	return []aiToolDef{
		{
			Name: "web_search", Module: aiModWeb, Risk: aiRiskRead,
			Desc: "外部联网搜索（Bing，查软件文档/报错资料/最佳实践）。input JSON：{\"query\":\"搜索关键词\",\"limit\":8}。返回标题/链接/摘要列表；仅作资料参考，面板内操作仍用面板工具。",
			Parameters: schObj(map[string]any{
				"query": schStr("搜索关键词（可用引号精确匹配）"), "limit": schInt("结果条数，默认 8 上限 15"),
			}, "query"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Query) == "" {
					return "", fmt.Errorf("缺少 query")
				}
				if p.Limit < 1 || p.Limit > 15 {
					p.Limit = 8
				}
				req, err := http.NewRequest(http.MethodGet, "https://cn.bing.com/search?q="+urlQueryEscape(p.Query)+"&count="+fmt.Sprint(p.Limit), nil)
				if err != nil {
					return "", err
				}
				req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
				req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
				resp, err := webLookupClient().Do(req)
				if err != nil {
					return "", fmt.Errorf("搜索请求失败（检查面板机外网连通）: %w", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return "", fmt.Errorf("搜索返回 HTTP %d", resp.StatusCode)
				}
				buf := make([]byte, 0, 1<<20)
				tmp := make([]byte, 32*1024)
				for {
					n, rerr := resp.Body.Read(tmp)
					buf = append(buf, tmp[:n]...)
					if rerr != nil {
						break
					}
					if len(buf) > 3<<20 {
						break
					}
				}
				page := string(buf)
				titles := wsTitleRe.FindAllStringSubmatch(page, -1)
				snips := wsSnipRe.FindAllStringSubmatch(page, -1)
				if len(titles) == 0 {
					return "未获取到搜索结果（页面结构变化或被限流），可稍后重试或用 fetch_url 直接抓取已知文档地址", nil
				}
				type item struct {
					Title, URL, Snip string
				}
				items := make([]item, 0, len(titles))
				for i, t := range titles {
					if strings.Contains(t[1], "bing.com") || strings.Contains(t[1], "microsoft.com/bing") {
						continue
					}
					snip := ""
					if i < len(snips) {
						snip = stripTags(snips[i][1])
					}
					items = append(items, item{Title: stripTags(t[2]), URL: t[1], Snip: truncText(snip, 200)})
					if len(items) >= p.Limit {
						break
					}
				}
				if len(items) == 0 {
					return "未解析到有效结果", nil
				}
				lines := make([]string, 0, len(items))
				for i, it := range items {
					lines = append(lines, fmt.Sprintf("%d. %s\n   %s\n   %s", i+1, it.Title, it.URL, it.Snip))
				}
				return strings.Join(lines, "\n"), nil
			},
		},
		{
			Name: "fetch_url", Module: aiModWeb, Risk: aiRiskRead,
			Desc: "抓取网页并提取正文纯文本（查文档页面/报错页/issue 内容；仅 http/https 且拒绝内网地址）。input JSON：{\"url\":\"https://...\",\"maxChars\":8000}",
			Parameters: schObj(map[string]any{
				"url": schStr("完整网页地址"), "maxChars": schInt("返回正文上限字符，默认 8000"),
			}, "url"),
			Fn: func(_ context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					URL      string `json:"url"`
					MaxChars int    `json:"maxChars"`
				}](input)
				if err != nil {
					return "", err
				}
				if !strings.HasPrefix(p.URL, "http://") && !strings.HasPrefix(p.URL, "https://") {
					return "", fmt.Errorf("仅支持 http/https")
				}
				// SSRF 防护：目标主机必须解析到公网（拒绝内网/回环/保留地址）
				if u, perr := url.Parse(p.URL); perr != nil || u.Hostname() == "" {
					return "", fmt.Errorf("URL 不合法")
				} else if err := checkPublicHost(u.Hostname()); err != nil {
					return "", err
				}
				if p.MaxChars < 500 || p.MaxChars > 20000 {
					p.MaxChars = 8000
				}
				req, err := http.NewRequest(http.MethodGet, p.URL, nil)
				if err != nil {
					return "", err
				}
				req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
				resp, err := webLookupClient().Do(req)
				if err != nil {
					return "", fmt.Errorf("抓取失败: %w", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return "", fmt.Errorf("HTTP %d", resp.StatusCode)
				}
				buf := make([]byte, 0, p.MaxChars*4)
				tmp := make([]byte, 32*1024)
				for {
					n, rerr := resp.Body.Read(tmp)
					buf = append(buf, tmp[:n]...)
					if rerr != nil {
						break
					}
					if len(buf) > 2<<20 {
						break
					}
				}
				text := stripTags(string(buf))
				return truncText(text, p.MaxChars), nil
			},
		},
	}
}

func urlQueryEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "+"), "%", "%25")
}
