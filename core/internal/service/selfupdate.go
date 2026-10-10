// SelfUpdateService 面板自更新（在线双源 + 本地文件通道）。
// 在线：检查 GitHub / Gitee 最新 Release（并行探测，互为兜底）→ agent 主机侧下载并
// sha256 校验 → 解包 → 复用本地 apply 通道（备份当前二进制 → 替换 → systemctl restart，
// detached 脚本脱离 core 进程生命周期）。本机 agent 与 core 同进程，重启即同步更新。
// Release 探测/下载/apply 脚本/版本比较等公共能力在 shared/release（节点 CLI 共用）。
package service

import (
	"context"
	"log/slog"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"github.com/ypanel/shared/release"
)

// 面板二进制与服务名（apply 脚本参数）。
const (
	panelBin  = "/opt/ypanel/ypanel"
	panelUnit = "ypanel"
	panelLog  = "/tmp/ypanel-apply.log"
)

const updateTimeoutSecs = 1800 // 下载+校验+解包上限

// OnlineRelease 单一源的最新 Release 信息（shared.release 别名，API 序列化形态不变）。
type OnlineRelease = release.OnlineRelease

// OnlineCheck 在线检查结果。
type OnlineCheck struct {
	CurrentVersion string          `json:"currentVersion"`
	CurrentIsDev   bool            `json:"currentIsDev"`
	Latest         string          `json:"latest,omitempty"` // 可达源中的最新 tag
	Updatable      bool            `json:"updatable"`
	Sources        []OnlineRelease `json:"sources"`
}

// UpdateChannelDir 更新通道目录（agent 主机侧 / CLI 通用）。
const UpdateChannelDir = "/opt/ypanel/updates"

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
	out, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL2(UpdateChannelDir))
	if err != nil {
		return map[string]any{
			"currentVersion": s.currentVer,
			"channelDir":     UpdateChannelDir,
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
		"channelDir":     UpdateChannelDir,
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
	src := path.Join(UpdateChannelDir, file)
	applyScript := path.Join(UpdateChannelDir, "apply-update.sh")
	script := release.ApplyScript(src, panelBin, panelUnit)
	// 写脚本
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: applyScript, Content: script}); err != nil {
		return nil, err
	}
	// detached 执行（core 将在 2s 后被重启，脚本独立存活）
	cmd := release.ApplyDetachedCommand(applyScript, panelLog)
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

// CheckOnline 并行探测双源，判定可更新性。
// 结果 30 秒缓存（互斥复用）：Gitee 匿名 API 限额低，频繁刷新设置页会把两源检查打出限流/超时，
// 造成「不可达」假象；日志记录不可达的具体原因（HTTP 码/超时），前端 chip 的 hover 提示同源展示。
var (
	checkCacheMu      sync.Mutex
	checkCacheAt      time.Time
	checkCacheResult  *OnlineCheck
)

func (s *SelfUpdateService) CheckOnline(ctx context.Context) (*OnlineCheck, error) {
	checkCacheMu.Lock()
	if checkCacheResult != nil && time.Since(checkCacheAt) < 30*time.Second {
		defer checkCacheMu.Unlock()
		return checkCacheResult, nil
	}
	checkCacheMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	res := map[string]*release.OnlineRelease{}
	for _, src := range []string{"github", "gitee"} {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			r := release.FetchLatestRelease(ctx, src)
			mu.Lock()
			res[src] = &r
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	for _, src := range res {
		if !src.Reachable {
			slog.Warn("更新源检查失败", "source", src.Source, "error", src.Error)
		}
	}
	isDev := release.IsDevVersion(s.currentVer)
	check := &OnlineCheck{CurrentVersion: s.currentVer, CurrentIsDev: isDev, Sources: []OnlineRelease{*res["gitee"], *res["github"]}}
	for _, src := range res {
		if !src.Reachable {
			continue
		}
		if check.Latest == "" || release.CompareVersions(src.Version, check.Latest) > 0 {
			check.Latest = src.Version
		}
	}
	if check.Latest != "" {
		check.Updatable = isDev || release.CompareVersions(s.currentVer, check.Latest) < 0
	}
	checkCacheMu.Lock()
	checkCacheAt, checkCacheResult = time.Now(), check
	checkCacheMu.Unlock()
	return check, nil
}

// Upgrade 在线升级：agent 主机侧下载（curl 重试）→ sha256 校验 → 解包 → 复用本地 Apply 通道。
func (s *SelfUpdateService) Upgrade(ctx context.Context, source string) (map[string]any, error) {
	if source != "github" && source != "gitee" {
		return nil, errs.Wrap(errs.ErrBadRequest, "未知更新源（github|gitee）")
	}
	rel := release.FetchLatestRelease(ctx, source)
	if !rel.Reachable {
		return nil, errs.New(errs.CodeAgentUnreach, "error.badRequest", source+" 不可达: "+rel.Error)
	}
	if rel.AssetURL == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, rel.Error)
	}
	if !release.IsDevVersion(s.currentVer) {
		if release.CompareVersions(s.currentVer, rel.Version) >= 0 {
			return map[string]any{"message": "已是最新版本 " + s.currentVer, "version": rel.Version}, nil
		}
	}
	// 资产域名白名单（URL 来自 Release API 响应，防注入重定向到任意主机）
	if err := release.ValidateAssetURL(rel.AssetURL); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, err.Error())
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	script := release.DownloadScript(UpdateChannelDir, runtime.GOARCH, rel.AssetURL, rel.SumURL)
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
