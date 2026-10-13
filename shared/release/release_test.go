package release

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// 自建 GitLab Release 响应解析：assets.links 结构与 GitHub 不同，逐链接挑当前架构资产。
func TestParseGitlabRelease(t *testing.T) {
	body := `[{
	  "tag_name": "v0.9.32",
	  "name": "YPanel v0.9.32",
	  "description": "synced from GitHub",
	  "released_at": "2026-10-13T01:00:00.000Z",
	  "assets": {
	    "links": [
	      { "name": "ypanel-linux-amd64.tar.gz", "url": "https://gitlab.yudream.online/yudream/ypanel/-/releases/v0.9.32/downloads/ypanel-linux-amd64.tar.gz" },
	      { "name": "sha256sums.txt", "url": "https://gitlab.yudream.online/yudream/ypanel/-/releases/v0.9.32/downloads/sha256sums.txt" }
	    ]
	  }
	}]`
	var list []gitlabReleasePayload
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 || normalizeTag(list[0].TagName) != "v0.9.32" {
		t.Fatalf("bad payload: %+v", list)
	}
	if len(list[0].Assets.Links) != 2 || list[0].Assets.Links[0].URL == "" {
		t.Fatalf("bad links: %+v", list[0].Assets.Links)
	}
}

// 三源资产 URL 互备：任一源形态都能换算出另外两源的等价 URL。
func TestFallbackAssetURLThreeSources(t *testing.T) {
	file := "v0.9.32/ypanel-linux-amd64.tar.gz"
	cases := map[string]string{
		GiteeRepoBase + "/releases/download/" + file:               GithubRepoBase + "/releases/download/" + file,
		GithubRepoBase + "/releases/download/" + file:              GiteeRepoBase + "/releases/download/" + file,
		GitlabBase + "/yudream/ypanel/-/releases/download/" + file: GiteeRepoBase + "/releases/download/" + file,
	}
	for from, want := range cases {
		if got := FallbackAssetURL(from); got != want {
			t.Fatalf("FallbackAssetURL(%s) = %s, want %s", from, got, want)
		}
	}
	if got := FallbackAssetURL("https://evil.example.com/x.tar.gz"); got != "" {
		t.Fatalf("外域 URL 应返回空，得到 %s", got)
	}
}

// M69 自动选源：gitee > gitlab > github 首个可达且有资产（联网用例，-short 跳过）。
func TestFetchLatestReleaseAuto(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过联网用例")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	r := FetchLatestReleaseAuto(ctx)
	if !r.Reachable || r.AssetURL == "" {
		t.Fatalf("自动选源失败: %+v", r)
	}
	if r.Source != "gitee" && r.Source != "gitlab" && r.Source != "github" {
		t.Fatalf("选源异常: %s", r.Source)
	}
	t.Logf("auto 选源: %s %s (%s)", r.Source, r.Version, r.AssetURL)
}
