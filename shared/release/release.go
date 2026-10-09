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
)

// OnlineRelease 单一源的最新 Release 信息。
type OnlineRelease struct {
	Source    string `json:"source"` // github|gitee
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
func FetchLatestRelease(ctx context.Context, source string) OnlineRelease {
	out := OnlineRelease{Source: source}
	url := GithubLatestURL
	if source == "gitee" {
		url = GiteeLatestURL
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

// DownloadScript 构造在线下载脚本（curl 重试下载 → sha256 校验 → 解包到更新通道目录，
// 产出 ypanel 与 ypagent 两个二进制，调用方按需取用）。
func DownloadScript(channelDir, goarch, assetURL, sumURL string) string {
	return fmt.Sprintf(`set -e
cd %s
curl -fSL --retry 2 --connect-timeout 10 -o ypanel-linux-%s.tar.gz '%s'
if curl -fSL --retry 2 --connect-timeout 10 -o sha256sums.txt '%s' 2>/dev/null; then
  sha256sum -c sha256sums.txt --ignore-missing || { rm -f sha256sums.txt ypanel-linux-%s.tar.gz; echo YPSUMFAIL; exit 1; }
  rm -f sha256sums.txt
fi
tar -xzf ypanel-linux-%s.tar.gz -C %s
rm -f ypanel-linux-%s.tar.gz
echo YPDLOK
`, channelDir, goarch, assetURL, sumURL, goarch, goarch, channelDir, goarch)
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
		strings.HasPrefix(assetURL, "https://objects.githubusercontent.com/") {
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
