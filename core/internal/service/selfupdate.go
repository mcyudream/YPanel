// SelfUpdateService 面板自更新（在线双源 + 本地文件通道）。
// 在线：检查 GitHub / Gitee 最新 Release（并行探测，互为兜底）→ agent 主机侧下载并
// sha256 校验 → 解包 → 复用本地 apply 通道（备份当前二进制 → 替换 → systemctl restart，
// detached 脚本脱离 core 进程生命周期）。本机 agent 与 core 同进程，重启即同步更新。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// Release 源常量（服务端固定，不拼接用户输入）。
const (
	githubLatestURL   = "https://api.github.com/repos/mcyudream/YPanel/releases/latest"
	giteeLatestURL    = "https://gitee.com/api/v5/repos/mcyudream/ypanel/releases/latest"
	updateTimeoutSecs = 1800 // 下载+校验+解包上限
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

// OnlineCheck 在线检查结果。
type OnlineCheck struct {
	CurrentVersion string          `json:"currentVersion"`
	CurrentIsDev   bool            `json:"currentIsDev"`
	Latest         string          `json:"latest,omitempty"` // 可达源中的最新 tag
	Updatable      bool            `json:"updatable"`
	Sources        []OnlineRelease `json:"sources"`
}

// updateChannelDir 更新通道目录（agent 主机侧）。
const updateChannelDir = "/opt/ypanel/updates"

// SelfUpdateService 自更新服务（面板本机 + 被管节点 agent 一键更新，任务化走 tasks）。
type SelfUpdateService struct {
	nodes      *NodeService
	tasks      *TaskService
	currentVer string
}

// NewSelfUpdateService 创建（tasks 可 nil——仅在线检查可用）。
func NewSelfUpdateService(nodes *NodeService, tasks *TaskService, currentVersion string) *SelfUpdateService {
	return &SelfUpdateService{nodes: nodes, tasks: tasks, currentVer: currentVersion}
}

func (s *SelfUpdateService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// Status 更新通道状态：当前版本 + 可用更新文件列表。
func (s *SelfUpdateService) Status(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(updateChannelDir))
	if err != nil {
		return map[string]any{
			"currentVersion": s.currentVer,
			"channelDir":     updateChannelDir,
			"available":      []map[string]any{},
		}, nil
	}
	files := []map[string]any{}
	for _, e := range out.Entries {
		if e.IsDir || e.Size < 1<<20 { // 二进制至少 1MB，过滤杂项
			continue
		}
		files = append(files, map[string]any{
			"name": e.Name, "sizeMb": float64(e.Size) / 1024 / 1024,
			"modTime": e.ModTime, "path": e.Path,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		a, _ := files[i]["modTime"].(time.Time)
		b, _ := files[j]["modTime"].(time.Time)
		return a.After(b)
	})
	return map[string]any{
		"currentVersion": s.currentVer,
		"channelDir":     updateChannelDir,
		"available":      files,
	}, nil
}

// validateUpdateFile 更新文件名白名单。
func validateUpdateFile(file string) error {
	if file == "" || strings.Contains(file, "/") || strings.Contains(file, "..") ||
		!strings.HasPrefix(file, "ypanel") {
		return errs.Wrap(errs.ErrBadRequest, "更新文件名不合法（需 ypanel 前缀且无路径分隔）")
	}
	return nil
}

// Apply 应用更新：写 detached 脚本（备份→替换→重启），立即返回。
func (s *SelfUpdateService) Apply(ctx context.Context, file string) (map[string]any, error) {
	if err := validateUpdateFile(file); err != nil {
		return nil, err
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	src := path.Join(updateChannelDir, file)
	applyScript := path.Join("/opt/ypanel/updates", "apply-update.sh")
	script := fmt.Sprintf(`#!/bin/sh
sleep 2
cp /opt/ypanel/ypanel /opt/ypanel/ypanel.bak || exit 1
mv %s /opt/ypanel/ypanel || exit 1
chmod +x /opt/ypanel/ypanel
systemctl restart ypanel
`, src)
	// 写脚本
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: applyScript, Content: script}); err != nil {
		return nil, err
	}
	// detached 执行（core 将在 2s 后被重启，脚本独立存活）
	cmd := fmt.Sprintf("setsid nohup sh %s >/tmp/ypanel-apply.log 2>&1 </dev/null &", applyScript)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 15})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return map[string]any{"output": out.Output}, errs.Wrapc(errs.CodeFileOpFailed, "apply 脚本启动失败")
	}
	return map[string]any{
		"message": "更新已启动：约 2 秒后替换二进制并重启服务，请稍后通过 /health 确认新版本",
		"file":    file,
	}, nil
}

// ---- 在线更新（GitHub / Gitee 双源） ----

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

// fetchLatest 拉取单源最新 Release 并挑出当前架构资产。
func (s *SelfUpdateService) fetchLatest(ctx context.Context, source string) OnlineRelease {
	out := OnlineRelease{Source: source}
	url := githubLatestURL
	if source == "gitee" {
		url = giteeLatestURL
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

// CheckOnline 并行探测双源，判定可更新性。
func (s *SelfUpdateService) CheckOnline(ctx context.Context) (*OnlineCheck, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	res := map[string]*OnlineRelease{}
	for _, src := range []string{"github", "gitee"} {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			r := s.fetchLatest(ctx, src)
			mu.Lock()
			res[src] = &r
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	isDev, _ := parseVersion(s.currentVer)
	check := &OnlineCheck{CurrentVersion: s.currentVer, CurrentIsDev: isDev, Sources: []OnlineRelease{*res["gitee"], *res["github"]}}
	for _, src := range res {
		if !src.Reachable {
			continue
		}
		if check.Latest == "" || compareVersions(src.Version, check.Latest) > 0 {
			check.Latest = src.Version
		}
	}
	if check.Latest != "" {
		check.Updatable = isDev || compareVersions(s.currentVer, check.Latest) < 0
	}
	return check, nil
}

// Upgrade 在线升级：agent 主机侧下载（curl 重试）→ sha256 校验 → 解包 → 复用本地 Apply 通道。
func (s *SelfUpdateService) Upgrade(ctx context.Context, source string) (map[string]any, error) {
	if source != "github" && source != "gitee" {
		return nil, errs.Wrap(errs.ErrBadRequest, "未知更新源（github|gitee）")
	}
	rel := s.fetchLatest(ctx, source)
	if !rel.Reachable {
		return nil, errs.New(errs.CodeAgentUnreach, "error.badRequest", source+" 不可达: "+rel.Error)
	}
	if rel.AssetURL == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, rel.Error)
	}
	isDev := false
	if isDev, _ = parseVersion(s.currentVer); !isDev {
		if compareVersions(s.currentVer, rel.Version) >= 0 {
			return map[string]any{"message": "已是最新版本 " + s.currentVer, "version": rel.Version}, nil
		}
	}
	// 资产域名白名单（URL 来自 Release API 响应，防注入重定向到任意主机）
	if !strings.HasPrefix(rel.AssetURL, "https://gitee.com/") && !strings.HasPrefix(rel.AssetURL, "https://github.com/") &&
		!strings.HasPrefix(rel.AssetURL, "https://objects.githubusercontent.com/") {
		return nil, errs.Wrap(errs.ErrBadRequest, "资产地址域名不在白名单: "+rel.AssetURL)
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf(`set -e
cd %s
curl -fSL --retry 2 --connect-timeout 10 -o ypanel-linux-%s.tar.gz '%s'
if curl -fSL --retry 2 --connect-timeout 10 -o sha256sums.txt '%s' 2>/dev/null; then
  sha256sum -c sha256sums.txt --ignore-missing || { rm -f sha256sums.txt ypanel-linux-%s.tar.gz; echo YPSUMFAIL; exit 1; }
  rm -f sha256sums.txt
fi
tar -xzf ypanel-linux-%s.tar.gz -C %s
rm -f ypanel-linux-%s.tar.gz
echo YPDLOK
`, updateChannelDir, runtime.GOARCH, rel.AssetURL, rel.SumURL, runtime.GOARCH, runtime.GOARCH, updateChannelDir, runtime.GOARCH)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: script, TimeoutSecs: updateTimeoutSecs})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 || strings.Contains(out.Output, "YPSUMFAIL") {
		return map[string]any{"output": tailStr(out.Output, 1000)}, errs.New(errs.CodeFileOpFailed, "error.badRequest", "下载/校验失败，请稍后重试或更换更新源")
	}
	// 解包产物名固定 ypanel（与 CI 打包一致），走既有本地 apply 通道
	return s.Apply(ctx, "ypanel")
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

// compareVersions 比较 vX.Y.Z（a>b 返回 1）。
func compareVersions(a, b string) int {
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
