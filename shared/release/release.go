// Package release：YPanel Release 探测与更新脚本（面板 core 与节点 agent CLI 共用）。
// Release 资产为 ypanel-linux-<arch>.tar.gz（内含 ypanel 与 ypagent 两个二进制），
// 下载 → sha256 校验 → 解包 → detached apply（备份→替换→systemctl restart）全链路在此封装。
package release

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Release API 地址（服务端固定，不拼接用户输入）。
const (
	GithubLatestURL = "https://api.github.com/repos/mcyudream/YPanel/releases/latest"
	GiteeLatestURL  = "https://gitee.com/api/v5/repos/mcyudream/ypanel/releases/latest"
	// GiteeLatestPageURL latest 页（非 API，无匿名限额）：302 的 Location 直接携带最新 tag——
	// Gitee 匿名 API 限额低（按 IP），生产服务器跑 CLI/面板检查几次就可能 403 造成「Gitee 不可达」假象，
	// API 失败时用它兜底探测。
	GiteeLatestPageURL = "https://gitee.com/mcyudream/ypanel/releases/latest"
	GiteeRepoBase      = "https://gitee.com/mcyudream/ypanel"
	GithubRepoBase     = "https://github.com/mcyudream/YPanel"
	// 自建 GitLab 兜底源（yudream 团队私有实例；CI 从 GitHub Release 自动同步资产）。
	GitlabBase      = "https://gitlab.yudream.online"
	GitlabLatestURL = GitlabBase + "/api/v4/projects/yudream%2Fypanel/releases?per_page=1"
)

// OnlineRelease 单一源的最新 Release 信息。
type OnlineRelease struct {
	Source    string `json:"source"` // github|gitee|gitlab
	Reachable bool   `json:"reachable"`
	Version   string `json:"version,omitempty"` // tag
	Name      string `json:"name,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Published string `json:"published,omitempty"`
	AssetURL  string `json:"assetUrl,omitempty"` // ypanel-linux-<arch>.tar.gz
	AssetSize int64  `json:"assetSize,omitempty"`
	SumURL    string `json:"sumUrl,omitempty"` // sha256sums.txt
	Error     string `json:"error,omitempty"`
}

// releasePayload Release API 响应的公共字段（GitHub / Gitee 结构兼容子集）。
type releasePayload struct {
	TagName   string `json:"tag_name"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	Published string `json:"published_at"`
	Assets    []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// FetchLatestRelease 拉取单源最新 Release 并挑出当前架构资产。
// Gitee 分支在 API 失败（匿名限流/超时）时回退 latest 页 302 探测（tag 从 Location 提取，
// 资产 URL 按等价形式拼接），保证国内源检查几乎不误报。
func FetchLatestRelease(ctx context.Context, source string) OnlineRelease {
	out := fetchViaAPI(ctx, source)
	if !out.Reachable && source == "gitee" {
		if probed := giteeProbeViaRedirect(ctx); probed != nil {
			probed.Error = strings.TrimSpace("API: " + out.Error + "；" + probed.Error)
			return *probed
		}
	}
	return out
}

func fetchViaAPI(ctx context.Context, source string) OnlineRelease {
	out := OnlineRelease{Source: source}
	url := GithubLatestURL
	if source == "gitee" {
		url = GiteeLatestURL
	}
	if source == "gitlab" {
		return fetchViaGitlab(ctx)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		out.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return out
	}
	var pl releasePayload
	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		out.Error = err.Error()
		return out
	}
	out.Reachable = true
	out.Version = normalizeTag(pl.TagName)
	out.Name = pl.Name
	out.Notes = pl.Body
	out.Published = pl.Published
	asset := "ypanel-linux-" + runtime.GOARCH + ".tar.gz"
	for _, a := range pl.Assets {
		if a.Name == asset {
			out.AssetURL = a.BrowserDownloadURL
			out.AssetSize = a.Size
		}
		if a.Name == "sha256sums.txt" {
			out.SumURL = a.BrowserDownloadURL
		}
	}
	if out.AssetURL == "" {
		out.Error = "Release 中缺少资产 " + asset
	}
	return out
}

// gitlabReleasePayload 自建 GitLab Release 响应子集（assets.links 结构与 GitHub 不同）。
type gitlabReleasePayload struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ReleasedAt  string `json:"released_at"`
	Assets      struct {
		Links []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"links"`
	} `json:"assets"`
}

// fetchViaGitlab 自建 GitLab 兜底源（匿名可读；资产由 GitHub CI 自动同步）。
func fetchViaGitlab(ctx context.Context) OnlineRelease {
	out := OnlineRelease{Source: "gitlab"}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitlabLatestURL, nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		out.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return out
	}
	var list []gitlabReleasePayload
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		out.Error = err.Error()
		return out
	}
	if len(list) == 0 {
		out.Error = "GitLab 暂无 Release（等 CI 同步）"
		return out
	}
	gl := list[0]
	out.Reachable = true
	out.Version = normalizeTag(gl.TagName)
	out.Name = gl.Name
	out.Notes = gl.Description
	out.Published = gl.ReleasedAt
	asset := "ypanel-linux-" + runtime.GOARCH + ".tar.gz"
	for _, a := range gl.Assets.Links {
		u := a.URL
		if u != "" && !strings.HasPrefix(u, "http") {
			u = GitlabBase + u
		}
		if a.Name == asset && out.AssetURL == "" {
			out.AssetURL = u
		}
		if a.Name == "sha256sums.txt" && out.SumURL == "" {
			out.SumURL = u
		}
	}
	if out.AssetURL == "" {
		out.Error = "Release 中缺少资产 " + asset
	}
	return out
}

// giteeProbeViaRedirect latest 页 302 探测（HTTP 200/302 均取 Location；无 API 限额）。
func giteeProbeViaRedirect(ctx context.Context) *OnlineRelease {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GiteeLatestPageURL, nil)
	if err != nil {
		return nil
	}
	client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse // 不跟随，从 Location 拿 tag
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	loc := resp.Header.Get("Location")
	prefix := GiteeRepoBase + "/releases/tag/"
	if loc == "" || !strings.HasPrefix(loc, prefix) {
		return nil
	}
	tag := normalizeTag(strings.TrimPrefix(loc, prefix))
	if tag == "" {
		return nil
	}
	asset := "ypanel-linux-" + runtime.GOARCH + ".tar.gz"
	base := GiteeRepoBase + "/releases/download/" + tag + "/"
	return &OnlineRelease{
		Source: "gitee", Reachable: true, Version: tag, Name: "YPanel " + tag,
		AssetURL: base + asset, SumURL: base + "sha256sums.txt",
		Error: "已从 latest 页 302 兜底解析（Gitee API 匿名限额，资产大小未知）",
	}
}

// FallbackAssetURL 多源互备：解析资产 URL 的 tag 与文件名，拼出对端源等价 URL
// （Gitee 下载超时切 GitHub/GitLab，反之亦然；解析失败返回空）。
func FallbackAssetURL(assetURL string) string {
	file := ""
	for _, base := range []string{
		GiteeRepoBase + "/releases/download/",
		GithubRepoBase + "/releases/download/",
		GitlabBase + "/yudream/ypanel/-/releases/download/",
	} {
		if strings.HasPrefix(assetURL, base) {
			file = strings.TrimPrefix(assetURL, base)
			break
		}
	}
	if file == "" {
		return ""
	}
	for _, alt := range []string{
		GiteeRepoBase + "/releases/download/" + file,
		GithubRepoBase + "/releases/download/" + file,
		GitlabBase + "/yudream/ypanel/-/releases/download/" + file,
	} {
		if alt != assetURL {
			return alt
		}
	}
	return ""
}

// DownloadScript 构造在线下载脚本（curl 重试下载 → sha256 校验 → 解包到更新通道目录，
// 产出 ypanel 与 ypagent 两个二进制，调用方按需取用）。
// fallbackURL 非空时主源下载失败自动切换对端源等价 URL 重试（Gitee 超时切 GitHub、反之亦然）。
func DownloadScript(channelDir, goarch, assetURL, sumURL, fallbackURL string) string {
	primary := fmt.Sprintf("curl -fSL --retry 2 --connect-timeout 10 -o ypanel-linux-%s.tar.gz '%s'", goarch, assetURL)
	if fallbackURL != "" && fallbackURL != assetURL {
		primary += fmt.Sprintf(" || { echo '[ypanel] 主源超时/失败，切换备用源…'; curl -fSL --retry 2 --connect-timeout 10 -o ypanel-linux-%s.tar.gz '%s'; }", goarch, fallbackURL)
	}
	sumFallback := ""
	if fallbackURL != "" && fallbackURL != assetURL {
		if alt := FallbackAssetURL(sumURL); alt != "" {
			sumFallback = fmt.Sprintf(" || curl -fSL --retry 2 --connect-timeout 10 -o sha256sums.txt '%s'", alt)
		}
	}
	return fmt.Sprintf(`set -e
cd %s
%s
if curl -fSL --retry 2 --connect-timeout 10 -o sha256sums.txt '%s' 2>/dev/null %s; then
  sha256sum -c sha256sums.txt --ignore-missing || { rm -f sha256sums.txt ypanel-linux-%s.tar.gz; echo YPSUMFAIL; exit 1; }
  rm -f sha256sums.txt
fi
tar -xzf ypanel-linux-%s.tar.gz -C %s
rm -f ypanel-linux-%s.tar.gz
echo YPDLOK
`, channelDir, primary, sumURL, sumFallback, goarch, goarch, channelDir, goarch)
}

// ApplyScript 构造应用更新脚本（备份→替换→systemctl restart，detached 执行）。
func ApplyScript(src, targetBin, unit string) string {
	return fmt.Sprintf(`#!/bin/sh
sleep 2
cp %s %s.bak || exit 1
mv %s %s || exit 1
chmod +x %s
systemctl restart %s
`, targetBin, targetBin, src, targetBin, targetBin, unit)
}

// ApplyDetachedCommand detached 执行 apply 脚本的 shell 命令（服务将在 2s 后被重启，脚本独立存活）。
func ApplyDetachedCommand(applyScript, logFile string) string {
	return fmt.Sprintf("setsid nohup sh %s >%s 2>&1 </dev/null &", applyScript, logFile)
}

// ValidateAssetURL 资产域名白名单校验（URL 来自 Release API 响应，防注入重定向到任意主机）。
func ValidateAssetURL(assetURL string) error {
	if strings.HasPrefix(assetURL, "https://gitee.com/") || strings.HasPrefix(assetURL, "https://github.com/") ||
		strings.HasPrefix(assetURL, "https://objects.githubusercontent.com/") ||
		strings.HasPrefix(assetURL, GitlabBase+"/") {
		return nil
	}
	return fmt.Errorf("资产地址域名不在白名单: %s", assetURL)
}

// IsDevVersion 版本串是否为 dev 形态（无法比较大小，始终允许更新）。
func IsDevVersion(v string) bool {
	dev, _ := parseVersion(v)
	return dev
}

// CompareVersions 比较 vX.Y.Z（a>b 返回 1）。
func CompareVersions(a, b string) int {
	_, av := parseVersion(a)
	_, bv := parseVersion(b)
	for i := 0; i < 3; i++ {
		if av[i] != bv[i] {
			if av[i] > bv[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// normalizeTag 归一 tag（统一 v 前缀形态）。
func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	if tag[0] == 'v' || tag[0] == 'V' {
		return "v" + tag[1:]
	}
	return tag
}

// parseVersion 解析 vX.Y.Z（解析失败 = dev 版本，isDev=true）。
func parseVersion(v string) (bool, [3]int) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	segs := strings.Split(v, ".")
	var out [3]int
	if len(segs) < 2 {
		return true, out
	}
	for i := 0; i < 3 && i < len(segs); i++ {
		seg := segs[i]
		if j := strings.IndexFunc(seg, func(r rune) bool { return r < '0' || r > '9' }); j >= 0 {
			seg = seg[:j]
		}
		n, err := strconv.Atoi(seg)
		if err != nil {
			return true, out
		}
		out[i] = n
	}
	return false, out
}
