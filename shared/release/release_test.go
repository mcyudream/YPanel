package release

import (
	"encoding/json"
	"testing"
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
