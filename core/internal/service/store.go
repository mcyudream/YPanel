// StoreService 统一应用商店：多源驱动（1Panel zip / yp-url index.json / yp-git 仓库）。
//
// yp 源格式（index.json）：
//
//	{ "name": "...", "apps": [{ "id","name","category","kind","description","author","arch",
//	  "logo","readme","reverseProxy","versions":[{ "id","package","releaseNotes","env":[{key,label,type,default,required,rule}] }]}]}
//
// kind：app 普通应用 / service 面板功能依赖的环境服务 / middleware 可共享复用中间件。
// 版本包形态：downloadUrl（远端 tar.gz，agent 端下载解压）或 package（yp-git 仓库内目录，core 读文件写入）。
// 安装 = compose 目录 + .env 渲染 + docker compose up（接入统一网络，与站点反代互通；支持可选网络/时区/hosts）。
package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

const (
	storeSourceDirName = "store-sources"
	storeSyncTTL       = 24 * time.Hour
	// yp 包文件限制（防超大/超多文件写入 agent）
	ypPkgMaxFileCount = 200
	ypPkgMaxFileSize  = 2 << 20
)

var storeAppNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)

var gitURLPattern = regexp.MustCompile(`^(https?://|ssh://|git@|file:///)[^\s]+$`)

// StoreVersion 版本（versions_json 内；两源格式统一）。
type StoreVersion struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	DownloadURL  string           `json:"downloadUrl"` // 远端包地址（onepanel / yp-url）
	LocalDir     string           `json:"localDir"`    // yp-git：仓库内包目录（相对仓库根）
	ReleaseNotes string           `json:"releaseNotes"`
	FormFields   []StoreFormField `json:"formFields"`
}

// StoreFormField 版本参数定义（兼容 1Panel formFields 全量形态）。
type StoreFormField struct {
	EnvKey      string            `json:"envKey"`
	Label       map[string]string `json:"label"`
	Default     interface{}       `json:"default"`
	Type        string            `json:"type"` // text / number / password / select / service / apps / ...
	Rule        string            `json:"rule"` // paramPort / paramCommon / paramComplexity / ...
	Required    bool              `json:"required"`
	Random      bool              `json:"random"` // 安装时随机生成（密码/名称类）
	Edit        *bool             `json:"edit"`   // false = 只读展示
	Disabled    bool              `json:"disabled"`
	Description string            `json:"description"`
	Values      []StoreFormValue  `json:"values"` // select 选项
}

// StoreFormValue select 选项。
type StoreFormValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// isPortField 端口类字段（预检与随机默认的依据）。
func (f StoreFormField) isPortField() bool {
	return f.Rule == "paramPort" || f.Rule == "paramPortRange" ||
		strings.Contains(strings.ToUpper(f.EnvKey), "PORT")
}

// ypListDTO yp 源清单（index.json）。
type ypListDTO struct {
	Name string   `json:"name"`
	Apps []ypApps `json:"apps"`
}

type ypApps struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Category     string     `json:"category"`
	Kind         string     `json:"kind"`
	Description  string     `json:"description"`
	Author       string     `json:"author"`
	Arch         []string   `json:"arch"`
	Logo         string     `json:"logo"`
	Readme       string     `json:"readme"`
	Website      string     `json:"website"`   // 官网
	SourceURL    string     `json:"sourceUrl"` // 开源社区
	Document     string     `json:"document"`  // 文档
	ReverseProxy string     `json:"reverseProxy"`
	Versions     []ypVerion `json:"versions"`
}

type ypVerion struct {
	ID           string      `json:"id"`
	Package      string      `json:"package"`
	ReleaseNotes string      `json:"releaseNotes"`
	Env          []ypEnvItem `json:"env"`
	Ports        []ypPortDef `json:"ports"`
}

type ypEnvItem struct {
	Key      string      `json:"key"`
	Label    string      `json:"label"`
	Type     string      `json:"type"` // text / number / password / select
	Default  interface{} `json:"default"`
	Required bool        `json:"required"`
	Rule     string      `json:"rule"`
}

type ypPortDef struct {
	EnvKey  string `json:"envKey"`
	Default int    `json:"default"`
}

// StoreListQuery 列表查询。
type StoreListQuery struct {
	Search   string
	Tag      string
	SourceID uint
	Kind     string
	Status   string // all / installed / notInstalled / upgradable
	OrderBy  string // name / lastModified / 空=id
	Order    string // asc / desc
	Page     int
	PageSize int
}

// StoreAppItem 列表项（带安装状态）。
type StoreAppItem struct {
	model.AppStoreApp
	Installed    bool                `json:"installed"`
	Upgradable   bool                `json:"upgradable"`
	LatestVer    string              `json:"latestVer"`
	InstallInfo  *model.AppStoreInstall `json:"installInfo,omitempty"`
}

// PanelNetwork 面板统一容器网络：商店应用/运行环境/数据库/nginx 全部接入，容器名互通。
const PanelNetwork = "ypanel_default"

// ensurePanelNetwork 确保统一网络存在（幂等）。
func ensurePanelNetwork(ctx context.Context, ac *agentclient.Client) error {
	_, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "docker network create " + PanelNetwork + " 2>/dev/null; true", TimeoutSecs: 60})
	return err
}

// StoreService 商店服务。
type StoreService struct {
	db    *gorm.DB
	nodes *NodeService
	sites *SiteService
	tasks *TaskService
	dbs   *DatabaseService
	http  *http.Client
	mu    sync.Mutex
	dir   string // 源缓存目录根 <data>/store-sources

	hostIPOnce sync.Once
	hostIP     string
	hostIPErr  error
}

// NewStoreService 创建（dataDir 为面板数据目录；sites 用于一键反代、tasks 用于任务中心，均可为 nil）。
func NewStoreService(db *gorm.DB, nodes *NodeService, sites *SiteService, tasks *TaskService, dbs *DatabaseService, dataDir string) *StoreService {
	return &StoreService{
		db: db, nodes: nodes, sites: sites, tasks: tasks, dbs: dbs,
		http: &http.Client{Timeout: 5 * time.Minute},
		dir:  filepath.Join(dataDir, storeSourceDirName),
	}
}

func (s *StoreService) client() (*agentclient.Client, error) {
	return s.clientFor("local")
}

// clientFor 按节点路由 agent 客户端（M55 商店节点安装：空/local=本机）。
func (s *StoreService) clientFor(nodeId string) (*agentclient.Client, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// ---------- 源管理 ----------

// Sources 源列表。
func (s *StoreService) Sources() []model.AppStoreSource {
	out := []model.AppStoreSource{}
	_ = s.db.Order("builtin desc, id").Find(&out).Error
	return out
}

// StoreSourceInput 源新增/编辑入参。
type StoreSourceInput struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"` // onepanel / yp-url / yp-git
	URL       string `json:"url"`
	Branch    string `json:"branch"`
	AuthToken string `json:"authToken"`
	Remark    string `json:"remark"`
}

var sourceTypes = map[string]bool{"onepanel": true, "yp-url": true, "yp-git": true}

// CreateSource 添加自定义源。
func (s *StoreService) CreateSource(in StoreSourceInput) (*model.AppStoreSource, error) {
	if !sourceTypes[in.Type] {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的源类型: "+in.Type)
	}
	if in.URL == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "源地址不能为空")
	}
	if in.Type == "yp-git" {
		if err := validateGitURL(in.URL); err != nil {
			return nil, err
		}
	}
	var count int64
	_ = s.db.Model(&model.AppStoreSource{}).Where("name = ?", in.Name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.sourceExists", "源名称已存在")
	}
	row := &model.AppStoreSource{
		Name: in.Name, Type: in.Type, URL: strings.TrimSpace(in.URL), Branch: in.Branch,
		AuthToken: in.AuthToken, Enabled: true, Remark: in.Remark,
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateSource 编辑源（内置源不可改名/类型/删除，可改地址/凭据/启停）。
func (s *StoreService) UpdateSource(id uint, in StoreSourceInput) (*model.AppStoreSource, error) {
	var row model.AppStoreSource
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.sourceNotFound", "源不存在")
	}
	if in.Type != "" && !sourceTypes[in.Type] {
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的源类型: "+in.Type)
	}
	updates := map[string]any{}
	if !row.Builtin {
		if in.Name != "" {
			updates["name"] = in.Name
		}
		if in.Type != "" {
			updates["type"] = in.Type
		}
	}
	if in.Type == "yp-git" && in.URL != "" {
		if err := validateGitURL(in.URL); err != nil {
			return nil, err
		}
	}
	updates["url"] = strings.TrimSpace(in.URL)
	if in.Branch != "" || in.AuthToken != "" {
		updates["branch"] = in.Branch
		updates["auth_token"] = in.AuthToken
	}
	updates["remark"] = in.Remark
	if err := s.db.Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// SetSourceEnabled 启停源（停用后列表不展示、同步跳过）。
func (s *StoreService) SetSourceEnabled(id uint, enabled bool) error {
	var row model.AppStoreSource
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.sourceNotFound", "源不存在")
	}
	return s.db.Model(&row).Update("enabled", enabled).Error
}

// DeleteSource 删除源（内置拒绝；级联清理其应用，安装记录保留）。
func (s *StoreService) DeleteSource(id uint) error {
	var row model.AppStoreSource
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.sourceNotFound", "源不存在")
	}
	if row.Builtin {
		return errs.New(errs.CodeConflict, "error.sourceBuiltin", "内置源不可删除（可停用）")
	}
	_ = s.db.Where("source_id = ?", id).Delete(&model.AppStoreApp{}).Error
	return s.db.Delete(&model.AppStoreSource{}, id).Error
}

// ---------- 同步 ----------

// SyncAll 同步全部启用源（TTL 内跳过；force 强制）。
func (s *StoreService) SyncAll(ctx context.Context, force bool) (map[string]any, error) {
	srcs := s.Sources()
	results := map[string]any{}
	anyErr := error(nil)
	for _, src := range srcs {
		if !src.Enabled {
			continue
		}
		if !force && src.LastSyncAt != nil && time.Since(*src.LastSyncAt) < storeSyncTTL {
			results[src.Name] = "skipped"
			continue
		}
		if _, err := s.SyncSource(ctx, src.ID, true); err != nil {
			results[src.Name] = err.Error()
			anyErr = err
			continue
		}
		results[src.Name] = "ok"
	}
	if anyErr != nil {
		return results, anyErr
	}
	return results, nil
}

// SyncSource 同步单个源。
func (s *StoreService) SyncSource(ctx context.Context, id uint, force bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var src model.AppStoreSource
	if err := s.db.First(&src, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.sourceNotFound", "源不存在")
	}
	if !src.Enabled {
		return nil, errs.Wrap(errs.ErrBadRequest, "源已停用")
	}
	if !force && src.LastSyncAt != nil && time.Since(*src.LastSyncAt) < storeSyncTTL {
		return map[string]any{"skipped": true}, nil
	}
	var (
		count int
		err   error
	)
	switch src.Type {
	case "onepanel":
		count, err = s.syncOnePanel(ctx, &src)
	case "yp-url":
		count, err = s.syncYpURL(ctx, &src)
	case "yp-git":
		count, err = s.syncYpGit(ctx, &src)
	default:
		return nil, errs.Wrap(errs.ErrBadRequest, "不支持的源类型: "+src.Type)
	}
	now := time.Now()
	updates := map[string]any{"last_sync_at": now, "app_count": count}
	if err != nil {
		updates["status"] = "error"
		updates["message"] = truncStr(err.Error(), 500)
		_ = s.db.Model(&src).Updates(updates).Error
		return nil, err
	}
	updates["status"] = "ok"
	updates["message"] = ""
	_ = s.db.Model(&src).Updates(updates).Error
	return map[string]any{"source": src.Name, "total": count}, nil
}

// upsertApps 清除源内旧应用后整体写入（简单可靠，源同步为低频操作）。
func (s *StoreService) upsertApps(srcID uint, rows []model.AppStoreApp) (int, error) {
	if err := s.db.Where("source_id = ?", srcID).Delete(&model.AppStoreApp{}).Error; err != nil {
		return 0, err
	}
	// 已安装应用可能被引用删除破坏 join——安装记录独立存在，仅商店条目消失，可接受
	for i := range rows {
		rows[i].SourceID = srcID
		rows[i].SyncedAt = time.Now()
		if err := s.db.Create(&rows[i]).Error; err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}

// syncOnePanel 1Panel 官方 zip 清单。
func (s *StoreService) syncOnePanel(ctx context.Context, src *model.AppStoreSource) (int, error) {
	url := src.URL
	if url == "" {
		url = "https://apps-assets.fit2cloud.com/dev/1panel.json.zip"
	}
	body, err := s.httpGet(ctx, url, 64<<20)
	if err != nil {
		return 0, errs.Wrap(errs.ErrAgentUnreach, "下载应用源失败: "+err.Error())
	}
	list, err := unzipEntry(body, "1panel.json")
	if err != nil {
		return 0, errs.Wrap(errs.ErrBadRequest, "应用源包解析失败: "+err.Error())
	}
	var listDTO struct {
		Apps []struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			ReadMe       string   `json:"readMe"`
			Icon         string   `json:"icon"`
			Tags         []string `json:"tags"`
			LastModified int64    `json:"lastModified"`
			AdditionalProperties struct {
				Website       string   `json:"website"`
				GitHub        string   `json:"github"`
				Document      string   `json:"document"`
				Architectures []string `json:"architectures"`
			} `json:"additionalProperties"`
			Versions []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				DownloadURL string `json:"downloadUrl"`
				AdditionalProperties struct {
					FormFields []StoreFormField `json:"formFields"`
				} `json:"additionalProperties"`
			} `json:"versions"`
		} `json:"apps"`
	}
	if err := json.Unmarshal(list, &listDTO); err != nil {
		return 0, errs.Wrap(errs.ErrBadRequest, "应用清单解析失败: "+err.Error())
	}
	rows := make([]model.AppStoreApp, 0, len(listDTO.Apps))
	for _, a := range listDTO.Apps {
		if a.ID == "" {
			continue
		}
		versions := make([]StoreVersion, 0, len(a.Versions))
		for _, v := range a.Versions {
			versions = append(versions, StoreVersion{ID: v.ID, Name: v.Name, DownloadURL: v.DownloadURL, FormFields: v.AdditionalProperties.FormFields})
		}
		rows = append(rows, model.AppStoreApp{
			Key: a.ID, Name: a.Name, Title: a.Title,
			Description: truncStr(a.Description, 500), ReadMe: a.ReadMe,
			IconURL: a.Icon, Tags: strings.Join(a.Tags, ","),
			Kind: "app",
			Website: truncStr(a.AdditionalProperties.Website, 500),
			SourceURL: truncStr(a.AdditionalProperties.GitHub, 500),
			Document: truncStr(a.AdditionalProperties.Document, 500),
			Arch: strings.Join(a.AdditionalProperties.Architectures, ","),
			VersionsJSON: marshalJSON(versions), LatestVersion: latestVersionOf(versions),
			LastModified: a.LastModified,
		})
	}
	return s.upsertApps(src.ID, rows)
}

// syncYpURL yp 格式 index.json（远端）。
func (s *StoreService) syncYpURL(ctx context.Context, src *model.AppStoreSource) (int, error) {
	body, err := s.httpGet(ctx, src.URL, 16<<20)
	if err != nil {
		return 0, errs.Wrap(errs.ErrAgentUnreach, "下载源清单失败: "+err.Error())
	}
	return s.syncYpManifest(src, body, "")
}

// syncYpGit yp 格式 git 仓库（clone/pull 后读 index.json 与包目录）。
func (s *StoreService) syncYpGit(ctx context.Context, src *model.AppStoreSource) (int, error) {
	dir, err := s.gitCheckout(ctx, src)
	if err != nil {
		return 0, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return 0, errs.Wrap(errs.ErrBadRequest, "源仓库缺少 index.json: "+err.Error())
	}
	return s.syncYpManifest(src, raw, dir)
}

// syncYpManifest 解析 yp 清单入库（localRoot 非空时为 yp-git，readme/logo 从本地读）。
func (s *StoreService) syncYpManifest(src *model.AppStoreSource, raw []byte, localRoot string) (int, error) {
	var list ypListDTO
	if err := json.Unmarshal(raw, &list); err != nil {
		return 0, errs.Wrap(errs.ErrBadRequest, "源清单解析失败: "+err.Error())
	}
	base := strings.TrimSuffix(src.URL, "/index.json")
	rows := make([]model.AppStoreApp, 0, len(list.Apps))
	for _, a := range list.Apps {
		if a.ID == "" || len(a.Versions) == 0 {
			continue
		}
		kind := a.Kind
		if kind == "" {
			kind = "app"
		}
		readme := ""
		if localRoot != "" && a.Readme != "" {
			if b, err := os.ReadFile(filepath.Join(localRoot, filepath.FromSlash(a.Readme))); err == nil {
				readme = string(b)
			}
		}
		iconURL := a.Logo
		if localRoot != "" {
			iconURL = fmt.Sprintf("/api/v1/store/apps/%d/%s/icon", src.ID, a.ID) // 经图标接口读本地文件
		} else if a.Logo != "" && !strings.HasPrefix(a.Logo, "http") {
			iconURL = base + "/" + a.Logo
		}
		versions := make([]StoreVersion, 0, len(a.Versions))
		for _, v := range a.Versions {
			if v.ID == "" || v.Package == "" {
				continue
			}
			sv := StoreVersion{ID: v.ID, Name: v.ID, ReleaseNotes: v.ReleaseNotes}
			switch {
			case localRoot != "":
				sv.LocalDir = v.Package
			case strings.HasPrefix(v.Package, "http://"), strings.HasPrefix(v.Package, "https://"):
				sv.DownloadURL = v.Package
			default:
				sv.DownloadURL = base + "/" + v.Package
			}
			fields := make([]StoreFormField, 0, len(v.Env)+len(v.Ports))
			for _, e := range v.Env {
				fields = append(fields, StoreFormField{
					EnvKey: e.Key, Label: map[string]string{"zh": e.Label}, Default: e.Default,
					Type: e.Type, Required: e.Required, Rule: e.Rule,
				})
			}
			for _, p := range v.Ports {
				if p.EnvKey == "" {
					continue
				}
				fields = append(fields, StoreFormField{
					EnvKey: p.EnvKey, Label: map[string]string{"zh": "端口"}, Default: p.Default, Type: "number",
				})
			}
			sv.FormFields = fields
			versions = append(versions, sv)
		}
		if len(versions) == 0 {
			continue
		}
		rows = append(rows, model.AppStoreApp{
			Key: a.ID, Name: a.Name, Title: a.Name,
			Description: truncStr(a.Description, 500), ReadMe: readme,
			IconURL: iconURL, Tags: a.Category, Kind: kind, Author: a.Author,
			Arch: strings.Join(a.Arch, ","), ReverseProxy: a.ReverseProxy,
			Website: truncStr(a.Website, 500), SourceURL: truncStr(a.SourceURL, 500), Document: truncStr(a.Document, 500),
			VersionsJSON: marshalJSON(versions), LatestVersion: latestVersionOf(versions),
		})
	}
	return s.upsertApps(src.ID, rows)
}

// ---------- git 检出 ----------

func validateGitURL(u string) error {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(u, "-") || strings.HasPrefix(u, "ext::") {
		return errs.Wrap(errs.ErrBadRequest, "非法的 git 地址")
	}
	if !gitURLPattern.MatchString(u) {
		return errs.Wrap(errs.ErrBadRequest, "git 地址需为 http(s)/ssh/file 形式")
	}
	return nil
}

// gitCheckout clone / 更新源仓库，返回本地目录。
func (s *StoreService) gitCheckout(ctx context.Context, src *model.AppStoreSource) (string, error) {
	if err := validateGitURL(src.URL); err != nil {
		return "", err
	}
	dir := filepath.Join(s.dir, fmt.Sprintf("src-%d", src.ID))
	url := injectGitToken(src.URL, src.AuthToken)
	branch := src.Branch
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		ref := "HEAD"
		if branch != "" {
			ref = branch
		}
		// 源地址/凭据可能已编辑：先同步 origin 指向（clone 时缓存的旧地址/旧 token），否则 fetch 仍走旧源
		if err := s.gitRun(ctx, dir, "remote", "set-url", "origin", url); err != nil {
			return "", errs.Wrap(errs.ErrAgentUnreach, "源仓库 remote 更新失败: "+err.Error())
		}
		if err := s.gitRun(ctx, dir, "fetch", "--depth", "1", "origin", ref); err != nil {
			return "", errs.Wrap(errs.ErrAgentUnreach, "源仓库 fetch 失败: "+err.Error())
		}
		if err := s.gitRun(ctx, dir, "reset", "--hard", "FETCH_HEAD"); err != nil {
			return "", errs.Wrap(errs.ErrAgentUnreach, "源仓库更新失败: "+err.Error())
		}
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, url, dir)
	if err := s.gitRun(ctx, "", args...); err != nil {
		return "", errs.Wrap(errs.ErrAgentUnreach, "源仓库 clone 失败: "+err.Error())
	}
	return dir, nil
}

func (s *StoreService) gitRun(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", truncStr(msg, 300))
	}
	return nil
}

// injectGitToken https 地址内嵌访问 token（不落日志）。
func injectGitToken(raw, token string) string {
	if token == "" || !strings.HasPrefix(raw, "http") {
		return raw
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	scheme := "https://"
	if !strings.HasPrefix(raw, "https://") {
		scheme = "http://"
	}
	return scheme + "ypanel:" + token + "@" + rest
}

// ---------- 列表 / 详情 ----------

// Apps 分页列表（内存状态过滤；列表不携带 readMe/versions 大字段）。
func (s *StoreService) Apps(q StoreListQuery) (map[string]any, error) {
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	// 启用源白名单
	enabled := map[uint]bool{}
	for _, src := range s.Sources() {
		enabled[src.ID] = src.Enabled
	}
	var all []model.AppStoreApp
	dq := s.db.Omit("read_me", "versions_json")
	if q.SourceID > 0 {
		dq = dq.Where("source_id = ?", q.SourceID)
	}
	if err := dq.Find(&all).Error; err != nil {
		return nil, err
	}
	installs := s.Installed()
	installByKey := make(map[string]model.AppStoreInstall, len(installs))
	for _, i := range installs {
		installByKey[fmt.Sprintf("%d/%s", i.SourceID, i.Key)] = i
	}
	kw := strings.ToLower(q.Search)
	items := make([]StoreAppItem, 0, len(all))
	for _, a := range all {
		if !enabled[a.SourceID] {
			continue
		}
		if q.Kind != "" && a.Kind != q.Kind {
			continue
		}
		if q.Tag != "" && !strings.Contains(strings.ToLower(a.Tags), strings.ToLower(q.Tag)) {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(a.Name+a.Title+a.Description+a.Key), kw) {
			continue
		}
		item := StoreAppItem{AppStoreApp: a, LatestVer: a.LatestVersion}
		if inst, ok := installByKey[fmt.Sprintf("%d/%s", a.SourceID, a.Key)]; ok {
			item.Installed = true
			item.InstallInfo = &inst
			item.Upgradable = inst.Version != a.LatestVersion
		}
		switch q.Status {
		case "installed":
			if !item.Installed {
				continue
			}
		case "notInstalled":
			if item.Installed {
				continue
			}
		case "upgradable":
			if !item.Upgradable {
				continue
			}
		}
		items = append(items, item)
	}
	// 排序
	asc := q.Order != "desc"
	sort.Slice(items, func(i, j int) bool {
		switch q.OrderBy {
		case "lastModified":
			if items[i].LastModified != items[j].LastModified {
				less := items[i].LastModified > items[j].LastModified // 新在前
				return less == asc
			}
			return items[i].ID < items[j].ID
		case "name":
			if items[i].Name != items[j].Name {
				less := items[i].Name < items[j].Name
				return less == asc
			}
			return items[i].ID < items[j].ID
		default:
			return items[i].ID < items[j].ID
		}
	})
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return map[string]any{
		"total": total, "page": page, "pageSize": pageSize, "items": items[start:end],
	}, nil
}

// Tags 启用源的分类聚合。
func (s *StoreService) Tags() []map[string]any {
	enabled := map[uint]bool{}
	for _, src := range s.Sources() {
		enabled[src.ID] = src.Enabled
	}
	var rows []model.AppStoreApp
	// source_id 必须一并查出，否则启用源过滤会把全部行（SourceID=0）丢弃
	_ = s.db.Select("source_id", "tags").Find(&rows).Error
	counts := map[string]int{}
	for _, r := range rows {
		if !enabled[r.SourceID] {
			continue
		}
		for _, t := range strings.Split(r.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				counts[t]++
			}
		}
	}
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		out = append(out, map[string]any{"name": n, "count": counts[n]})
	}
	return out
}

// App 详情（含解析后的版本）。
func (s *StoreService) App(sourceID uint, key string) (map[string]any, error) {
	var row model.AppStoreApp
	if err := s.db.Where("source_id = ? AND key = ?", sourceID, key).First(&row).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.appNotFound", "应用不存在（可能未同步）")
	}
	versions := []StoreVersion{}
	_ = json.Unmarshal([]byte(row.VersionsJSON), &versions)
	var inst model.AppStoreInstall
	installed := s.db.Where("source_id = ? AND key = ?", sourceID, key).First(&inst).Error == nil
	return map[string]any{
		"app": row, "versions": versions,
		"installed": installed, "install": inst,
	}, nil
}

// AppIcon yp-git 源应用图标（本地文件直读）。
func (s *StoreService) AppIcon(sourceID uint, key string) ([]byte, string, error) {
	var src model.AppStoreSource
	if err := s.db.First(&src, sourceID).Error; err != nil || src.Type != "yp-git" {
		return nil, "", errs.ErrNotFound
	}
	dir := filepath.Join(s.dir, fmt.Sprintf("src-%d", sourceID))
	var list ypListDTO
	raw, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, "", err
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, "", err
	}
	for _, a := range list.Apps {
		if a.ID != key || a.Logo == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(a.Logo)))
		if err != nil {
			return nil, "", err
		}
		ct := "image/png"
		switch strings.ToLower(path.Ext(a.Logo)) {
		case ".svg":
			ct = "image/svg+xml"
		case ".jpg", ".jpeg":
			ct = "image/jpeg"
		case ".webp":
			ct = "image/webp"
		}
		return b, ct, nil
	}
	return nil, "", errs.ErrNotFound
}

// ---------- 安装 / 卸载 ----------

// StoreInstallInput 安装入参。
type StoreInstallInput struct {
	SourceID uint           `json:"sourceId" binding:"required"`
	NodeID   string         `json:"nodeId"`          // M55 目标节点（空=本机）
	Key      string         `json:"key" binding:"required"`
	Version  string         `json:"version"`
	Name     string         `json:"name" binding:"required"`
	Params   map[string]any `json:"params"` // 宽容类型：数字型字段（端口等）可能以 number 提交
	Domain   string         `json:"domain"` // 可选：安装后一键反代域名
	// 高级选项
	Network        string   `json:"network"`        // 接入网络；空 = ypanel_default
	CreateNetwork  bool     `json:"createNetwork"`  // network 不存在时创建
	Timezone       string   `json:"timezone"`       // 非空注入 TZ 环境变量（compose override）
	ExtraHosts     []string `json:"extraHosts"`     // 额外 hosts 映射（host:ip，compose override extra_hosts）
	MountHostsFile bool     `json:"mountHostsFile"` // 挂载宿主机 /etc/hosts 到容器（实时同步）

	// 使用已有数据库实例（M32）：识别到应用包的数据库 host 参数后，把应用装到指定纳管实例上
	ExternalDB *StoreExternalDB `json:"externalDB,omitempty"`

	OwnerID uint `json:"-"` // M54-P3 创建归属（assigned 调用者 → 自己；all → 公共），由 API 层填
}

// StoreExternalDB 安装时外接数据库实例选项。
type StoreExternalDB struct {
	InstanceID      uint   `json:"instanceId" binding:"required"`
	Database        string `json:"database"`        // 目标库名；空 = 应用 key
	User            string `json:"user"`            // 应用账号名；空 = <key>_user
	CreateIfMissing bool   `json:"createIfMissing"` // 库/账号不存在时自动创建
}

// dbFieldSet 应用包数据库参数键集（由 formFields 识别）。
type dbFieldSet struct {
	Host, Port, Name, User, Password string
}

var networkNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

var tzPattern = regexp.MustCompile(`^[A-Za-z0-9_+\-/]{1,64}$`)

var extraHostPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,253}:[0-9a-fA-F.:]{1,45}$`)

// resolveInstallNetwork 解析安装目标网络（校验/按需创建），返回网络名。
func (s *StoreService) resolveInstallNetwork(ctx context.Context, ac *agentclient.Client, in StoreInstallInput) (string, error) {
	name := strings.TrimSpace(in.Network)
	if name == "" {
		name = PanelNetwork
	}
	if !networkNamePattern.MatchString(name) {
		return "", errs.Wrap(errs.ErrBadRequest, "网络名不合法（字母/数字/中划线/下划线）")
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "docker network inspect " + name + " >/dev/null 2>&1 && echo ok || echo missing", TimeoutSecs: 60})
	if err != nil {
		return "", err
	}
	if name == "host" {
		return name, nil // host 网络模式由 compose 改写处理，不参与 network connect
	}
	exists := strings.Contains(out.Output, "ok")
	if !exists {
		if name == PanelNetwork || in.CreateNetwork {
			if _, cerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
				&dto.ExecReq{Command: "docker network create " + name, TimeoutSecs: 60}); cerr != nil {
				return "", errs.Wrap(errs.ErrBadRequest, "创建网络失败: "+cerr.Error())
			}
		} else {
			return "", errs.Wrap(errs.ErrBadRequest, "网络不存在: "+name+"（可勾选创建）")
		}
	}
	return name, nil
}

// writeComposeOverride 生成 docker-compose.override.yml（TZ / extra_hosts / 挂载主机 hosts，compose up 自动合并全部服务）。
func (s *StoreService) writeComposeOverride(ctx context.Context, logf TaskLogf, ac *agentclient.Client, dir, tz string, hosts []string, mountHostsFile bool) error {
	if tz == "" && len(hosts) == 0 && !mountHostsFile {
		return nil
	}
	// 服务名由 agent 端 docker compose config --services 动态获取，逐服务注入 override
	var script strings.Builder
	script.WriteString(`svcs=$(docker compose config --services) && { echo "services:"; for s in $svcs; do echo "  $s:";`)
	if tz != "" {
		script.WriteString(` echo "    environment:"; echo "      TZ: ` + tz + `";`)
	}
	if len(hosts) > 0 {
		script.WriteString(` echo "    extra_hosts:";`)
		for _, h := range hosts {
			script.WriteString(` printf '      - "%s"\n' "` + h + `";`)
		}
	}
	if mountHostsFile {
		// 只读挂载宿主机 /etc/hosts（compose volumes 列表与原服务挂载合并，容器内与主机 hosts 实时同步）
		script.WriteString(` echo "    volumes:"; echo "      - /etc/hosts:/etc/hosts:ro";`)
	}
	script.WriteString(` done; } > docker-compose.override.yml`)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("cd %s && %s", dir, script.String()), TimeoutSecs: 120})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "生成 override 失败: "+tailOutput(out.Output, 400))
	}
	logf("info", "已注入 compose override（时区/hosts）")
	return nil
}

// rewriteComposeHostNetwork 把 compose 改写为 host 网络模式：全部服务移除 networks、注入 network_mode: host。
// 1p 包 compose 结构规整（机械生成），标准 YAML 解析改写可靠。
func (s *StoreService) rewriteComposeHostNetwork(ctx context.Context, ac *agentclient.Client, composePath string) error {
	out, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(composePath))
	if err != nil {
		return errs.Wrap(errs.ErrBadRequest, "读取 compose 失败: "+err.Error())
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(out.Content), &doc); err != nil {
		return errs.Wrap(errs.ErrBadRequest, "compose 解析失败: "+err.Error())
	}
	services, _ := doc["services"].(map[string]any)
	if len(services) == 0 {
		return errs.Wrap(errs.ErrBadRequest, "compose 内无 services 定义")
	}
	for name, raw := range services {
		svc, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		delete(svc, "networks")
		svc["network_mode"] = "host"
		services[name] = svc
	}
	delete(doc, "networks")
	rewritten, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: composePath, Content: string(rewritten)})
	return err
}

// stringifyParam 参数值统一转字符串（JSON number 是 float64，fmt.Sprint 会出科学计数法）。
func stringifyParam(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(n)
	case nil:
		return ""
	default:
		return fmt.Sprint(n)
	}
}

// Install 安装商店应用（异步任务：立即返回任务 ID，日志在任务中心/向导内轮询）。
func (s *StoreService) Install(ctx context.Context, in StoreInstallInput) (map[string]any, error) {
	if !storeAppNamePattern.MatchString(in.Name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "应用实例名不合法（小写字母/数字/中划线）")
	}
	var row model.AppStoreApp
	if err := s.db.Where("source_id = ? AND key = ?", in.SourceID, in.Key).First(&row).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.appNotFound", "应用不存在（可能未同步）")
	}
	versions := []StoreVersion{}
	_ = json.Unmarshal([]byte(row.VersionsJSON), &versions)
	ver := resolveStoreVersion(versions, in.Version)
	if ver == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "版本不存在")
	}
	// 参数合并：定义 default ∪ 用户输入；random 字段缺省时后端兜底随机
	finalParams := map[string]string{}
	for _, f := range ver.FormFields {
		if f.EnvKey == "" {
			continue
		}
		finalParams[f.EnvKey] = sanitizeParam(fmt.Sprint(f.Default))
		if f.Random && f.Type == "password" && (in.Params == nil || stringifyParam(in.Params[f.EnvKey]) == "") {
			finalParams[f.EnvKey] = randomHex(12)
		}
	}
	for k, v := range in.Params {
		if !paramKeyPattern.MatchString(k) {
			return nil, errs.Wrap(errs.ErrBadRequest, "参数名不合法: "+k)
		}
		if sv := sanitizeParam(stringifyParam(v)); sv != "" {
			finalParams[k] = sv
		}
	}
	// 密码类字段兜底：用户/ AI 未提供非空值且无 random 声明时，拒绝落固定 default 之外的空值——
	// envKey 含 PASSWORD 且最终为空一律随机生成（防 1Panel 包弱 default 泄漏后的空密码）。
	for _, f := range ver.FormFields {
		if f.Type == "password" && strings.Contains(strings.ToUpper(f.EnvKey), "PASSWORD") && finalParams[f.EnvKey] == "" {
			finalParams[f.EnvKey] = randomHex(12)
		}
	}
	project := "app-" + in.Name
	finalParams["CONTAINER_NAME"] = project
	finalParams["CONTAINER_NAME1"] = project + "-1"


	// 端口占用预检在任务内执行（需 agent exec）
	input := in
	task, err := s.tasks.StartTask(TaskStoreInstall, fmt.Sprintf("安装 %s（%s）", row.Name, project), project, 30*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			return s.runInstall(tctx, logf, row, *ver, input, project, finalParams)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID, "project": project}, nil
}

// runInstall 任务化安装主体：重装清理 → 端口预检 → 部署 → 记录 → 一键反代。
func (s *StoreService) runInstall(ctx context.Context, logf TaskLogf, app model.AppStoreApp, ver StoreVersion, in StoreInstallInput, project string, params map[string]string) error {
	ac, err := s.clientFor(in.NodeID)
	if err != nil {
		return err
	}
	logf("info", "开始安装 %s 版本 %s → 项目 %s%s", app.Name, ver.ID, project,
		mapStr(normalizeNodeID(in.NodeID) != "local", "（节点 "+normalizeNodeID(in.NodeID)+"）", ""))

	// 重装场景：先 down 同名项目释放端口与容器
	if out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("docker compose -p %s ps --format '{{.Name}}' 2>/dev/null | grep -q . && cd /opt/ypanel/compose/%s 2>/dev/null && docker compose -p %s down || true", project, project, project), TimeoutSecs: 120}); err == nil {
		if strings.Contains(out.Output, project) || out.ExitCode == 0 {
			logf("info", "已停止同名旧项目（重装）")
		}
	}

	// 端口占用预检（目标节点）
	if err := s.precheckPorts(ctx, logf, ac, params); err != nil {
		return err
	}

	// 高级选项校验：网络（不存在可创建）、时区、hosts
	network, err := s.resolveInstallNetwork(ctx, ac, in)
	if err != nil {
		return err
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz != "" && !tzPattern.MatchString(tz) {
		return errs.Wrap(errs.ErrBadRequest, "时区格式不合法（如 Asia/Shanghai）")
	}
	hosts := make([]string, 0, len(in.ExtraHosts))
	for _, h := range in.ExtraHosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if !extraHostPattern.MatchString(h) {
			return errs.Wrap(errs.ErrBadRequest, "hosts 映射格式不合法（需 域名:IP）: "+h)
		}
		hosts = append(hosts, h)
	}

	logs, err := s.deployCompose(ctx, logf, app, ver, in, project, params, network, tz, hosts, in.MountHostsFile)
	if err != nil {
		return err
	}
	_ = logs

	// 已装记录 upsert（同名重装=换版本，参数同步落库供"参数"查看）
	var exist model.AppStoreInstall
	if err := s.db.Where("compose_project = ?", project).First(&exist).Error; err == nil {
		_ = s.db.Model(&exist).Updates(map[string]any{
			"source_id": app.SourceID, "key": app.Key, "name": in.Name, "version": ver.ID,
			"params_json": marshalJSON(params), "node_id": normalizeNodeID(in.NodeID),
		}).Error
	} else {
		_ = s.db.Create(&model.AppStoreInstall{
			SourceID: app.SourceID, Key: app.Key, Name: in.Name, Version: ver.ID,
			ComposeProject: project, ParamsJSON: marshalJSON(params), OwnerID: in.OwnerID,
			NodeID: normalizeNodeID(in.NodeID),
		}).Error
	}
	logf("info", "安装完成，项目 %s 已启动", project)

	// 数据库类应用：等待初始化就绪后自动接管至数据库模块（对齐 1Panel「装完即可管理」）
	if dbServiceKeys[app.Key] && s.dbs != nil {
		logf("info", "检测到数据库类应用，等待初始化后自动接管（最多 90 秒）…")
		actx, cancel := context.WithTimeout(ctx, 90*time.Second)
		if _, aerr := s.dbs.AdoptWithRetry(actx, project, normalizeNodeID(in.NodeID), 12, 5*time.Second); aerr != nil {
			logf("warn", "自动接管未完成: %s（可稍后在数据库页手动接管）", aerr.Error())
		} else {
			logf("info", "已自动接管至数据库模块，可直接管理库/用户/备份")
		}
		cancel()
	}

	// 一键反代（失败不回滚安装）
	if in.Domain != "" && app.ReverseProxy != "" && s.sites != nil {
		port := params[app.ReverseProxy]
		if port == "" {
			port = "80"
		}
		logf("info", "创建反代站点 %s → http://%s:%s", in.Domain, project, port)
		site, perr := s.sites.Create(ctx, SiteCreateInput{
			Name: "app-" + in.Name, Type: "proxy", Domain: in.Domain,
			ProxyPass: "http://" + project + ":" + port,
			Remark:    "商店应用 " + app.Name + " 反代",
		})
		if perr != nil {
			logf("warn", "反代创建失败（不影响安装）: %s", perr.Error())
		} else if site != nil {
			logf("info", "反代站点已创建：%s", site.Domain)
		}
	}
	return nil
}

// precheckPorts 端口占用预检：宿主已监听端口即冲突（compose down 后检测，本项目旧端口已释放）。
func (s *StoreService) precheckPorts(ctx context.Context, logf TaskLogf, ac *agentclient.Client, params map[string]string) error {
	ports := []int{}
	for k, v := range params {
		if !strings.Contains(strings.ToUpper(k), "PORT") || v == "" {
			continue
		}
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 {
			ports = append(ports, n)
		}
	}
	if len(ports) == 0 {
		return nil
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "ss -tlnH | awk '{print $4}' | grep -oE '[0-9]+$' | sort -un", TimeoutSecs: 30})
	if err != nil {
		logf("warn", "端口预检跳过（ss 不可用）: %s", err.Error())
		return nil
	}
	listening := map[int]bool{}
	for _, l := range strings.Fields(out.Output) {
		if n, err := strconv.Atoi(l); err == nil {
			listening[n] = true
		}
	}
	conflicts := []int{}
	for _, p := range ports {
		if listening[p] {
			conflicts = append(conflicts, p)
		}
	}
	if len(conflicts) > 0 {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("端口已被占用: %v（请修改安装参数中的端口后重试）", conflicts))
	}
	logf("info", "端口预检通过: %v", ports)
	return nil
}

// deployCompose 两种包形态的统一部署（步骤日志写任务；network 为解析后的目标网络）。
func (s *StoreService) deployCompose(ctx context.Context, logf TaskLogf, app model.AppStoreApp, ver StoreVersion, in StoreInstallInput, project string, params map[string]string, network, tz string, hosts []string, mountHostsFile bool) (string, error) {
	ac, err := s.clientFor(in.NodeID)
	if err != nil {
		return "", err
	}
	dir := "/opt/ypanel/compose/" + project
	var logs []string
	step := func(desc, cmd string, timeoutSecs int) error {
		out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs})
		if err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, desc+" 失败: "+err.Error())
		}
		if out.ExitCode != 0 {
			// compose up/pull 的真实错误常在输出尾部（首行多为 Pulling 进度）
			return errs.Wrapc(errs.CodeFileOpFailed, desc+" 失败: "+tailOutput(out.Output, 1200))
		}
		logs = append(logs, desc+" ✓")
		logf("info", "%s ✓", desc)
		return nil
	}
	if err := step("创建目录", fmt.Sprintf("mkdir -p %s", dir), 60); err != nil {
		return strings.Join(logs, "\n"), err
	}
	// 清理上次部署的遗留（docker 会为缺失的挂载源自动建同名目录，残留会导致整包拷贝类型冲突）；数据卷目录 data 保留
	if err := step("清理旧文件", fmt.Sprintf("cd %s && find . -mindepth 1 -maxdepth 1 ! -name 'data' -exec rm -rf {} +", dir), 120); err != nil {
		return strings.Join(logs, "\n"), err
	}
	switch {
	case ver.DownloadURL != "":
		// 远端包：agent 端下载解压
		logf("info", "下载应用包: %s", ver.DownloadURL)
		if err := step("下载应用包", fmt.Sprintf("curl -sSL --connect-timeout 20 -o %s/pkg.tar.gz '%s'", dir, ver.DownloadURL), 600); err != nil {
			return strings.Join(logs, "\n"), err
		}
		if err := step("解压", fmt.Sprintf("cd %s && tar xzf pkg.tar.gz", dir), 300); err != nil {
			return strings.Join(logs, "\n"), err
		}
		// 整包内容上移到项目根：1p 包常带 conf/ 等附属文件（compose 相对挂载 ./conf/my.cnf），只拷 compose 会导致挂载失败；
		// 同时把包内写死的 1panel-network 替换为目标网络
		tidy := `p=$(find . -name 'docker-compose.y*ml' -o -name 'compose.y*ml' | head -1) && cp "$p" ./docker-compose.yml && d=$(dirname "$p") && if [ "$d" != "." ]; then cp -rf "$d"/. ./; fi && rm -f pkg.tar.gz && sed -i 's/1panel-network/` + network + `/g' docker-compose.yml`
		if err := step("整理应用包", fmt.Sprintf("cd %s && %s", dir, tidy), 120); err != nil {
			return strings.Join(logs, "\n"), err
		}
		if network == "host" {
			if rerr := s.rewriteComposeHostNetwork(ctx, ac, dir+"/docker-compose.yml"); rerr != nil {
				return strings.Join(logs, "\n"), rerr
			}
			logf("info", "已改写为 host 网络模式")
		}
	case ver.LocalDir != "":
		// 本地包（yp-git）：core 读缓存目录写入 agent（compose 内网络名替换为目标网络）
		n, werr := s.writeLocalPackage(ctx, ac, app.SourceID, ver.LocalDir, dir, network)
		if werr != nil {
			return strings.Join(logs, "\n"), errs.Wrapc(errs.CodeFileOpFailed, "写入应用包失败: "+werr.Error())
		}
		if network == "host" {
			if rerr := s.rewriteComposeHostNetwork(ctx, ac, dir+"/docker-compose.yml"); rerr != nil {
				return strings.Join(logs, "\n"), rerr
			}
			logf("info", "已改写为 host 网络模式")
		}
		logs = append(logs, fmt.Sprintf("写入应用包 %d 个文件 ✓", n))
		logf("info", "写入应用包 %d 个文件 ✓", n)
	default:
		return strings.Join(logs, "\n"), errs.Wrap(errs.ErrBadRequest, "版本包地址缺失")
	}
	if network != "host" {
		if err := step("接入网络", "docker network inspect "+network+" >/dev/null 2>&1 || docker network create "+network, 60); err != nil {
			return strings.Join(logs, "\n"), err
		}
	}
	// M32：外接数据库实例——compose 已写盘，读模板识别数据库参数键集后建库建号并注入（赶在 .env 生成前）
	if in.ExternalDB != nil {
		composeText := ""
		if res, rerr := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(dir+"/docker-compose.yml")); rerr == nil && res != nil {
			composeText = res.Content
		}
		if err := s.applyExternalDB(ctx, logf, ver, in, params, composeText); err != nil {
			return strings.Join(logs, "\n"), err
		}
	}
	// .env 渲染
	var envBuf strings.Builder
	envBuf.WriteString("CONTAINER_NAME=" + project + "\n")
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		envBuf.WriteString(k + "=" + params[k] + "\n")
	}
	if err := s.writeViaAgent(ctx, ac, dir+"/.env", envBuf.String()); err != nil {
		return strings.Join(logs, "\n"), err
	}
	logs = append(logs, "写入 .env ✓")
	logf("info", "写入 .env ✓")
	// 时区 / hosts 映射：compose override（up 自动合并全部服务）
	if oerr := s.writeComposeOverride(ctx, logf, ac, dir, tz, hosts, mountHostsFile); oerr != nil {
		return strings.Join(logs, "\n"), oerr
	}
	// 镜像本地齐全时跳过 pull（compose pull 会连 registry 校验 digest，网络受限时干等）；
	// 有缺失才 pull（大镜像耗时长，进度/错误完整可读）
	pullCmd := `missing=0; for img in $(grep -E '^[[:space:]]*image:' docker-compose.yml | awk '{print $2}' | tr -d '"'); do docker image inspect "$img" >/dev/null 2>&1 || missing=1; done; if [ "$missing" -ne 0 ]; then docker compose -p ` + project + ` pull --quiet 2>&1 | tail -5; test ${PIPESTATUS[0]} -eq 0; fi`
	_ = step("检查镜像", fmt.Sprintf("cd %s && %s", dir, pullCmd), 1800)
	if err := step("compose up", fmt.Sprintf("cd %s && docker compose -p %s up -d", dir, project), 600); err != nil {
		return strings.Join(logs, "\n"), err
	}
	return strings.Join(logs, "\n"), nil
}

// writeLocalPackage 把仓库内包目录的文本文件写入 agent 目标目录（compose 文件统一命名 docker-compose.yml）。
func (s *StoreService) writeLocalPackage(ctx context.Context, ac *agentclient.Client, sourceID uint, pkgRel, remoteDir, network string) (int, error) {
	root := filepath.Join(s.dir, fmt.Sprintf("src-%d", sourceID))
	pkgDir := filepath.Join(root, filepath.FromSlash(pkgRel))
	if !strings.HasPrefix(pkgDir, filepath.Clean(root)+string(os.PathSeparator)) {
		return 0, fmt.Errorf("包目录越界")
	}
	count := 0
	err := filepath.WalkDir(pkgDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if count >= ypPkgMaxFileCount {
			return fmt.Errorf("包文件数超限")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > ypPkgMaxFileSize {
			return fmt.Errorf("文件过大: %s", d.Name())
		}
		rel, err := filepath.Rel(pkgDir, p)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		base := path.Base(name)
		for _, cn := range []string{"compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"} {
			if base == cn {
				name = "docker-compose.yml"
			}
		}
		// 包内容物约定在 package/ 子目录一层：附属文件落到项目根（compose 相对路径挂载可达），
		// 否则只处理 compose 的旧逻辑会把附属文件埋进 package/ 导致挂载源缺失（dockerd 自动建同名目录）
		name = strings.TrimPrefix(name, "package/")
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		content := string(b)
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			// compose 内网络统一为所选安装网络
			content = strings.ReplaceAll(content, "1panel-network", network)
		}
		if err := s.writeViaAgent(ctx, ac, remoteDir+"/"+name, content); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return count, err
	}
	if count == 0 {
		return 0, fmt.Errorf("包目录为空")
	}
	return count, nil
}

// StoreUninstallOptions 卸载选项（级联资源勾选）。
type StoreUninstallOptions struct {
	NodeID      string // M55 目标节点（空=本机）
	PurgeData   bool // 删除应用数据（compose 目录含数据卷/数据库文件）
	RemoveImage bool // 删除应用镜像（compose 内全部 image）
	CascadeDB   bool // 级联移除关联的数据库纳管记录与备份
}

// Uninstall 卸载（异步任务）：compose down；数据/镜像/级联按选项执行，默认保留数据目录。
func (s *StoreService) Uninstall(ctx context.Context, project string, opts StoreUninstallOptions) (map[string]any, error) {
	if !storeAppNamePattern.MatchString(strings.TrimPrefix(project, "app-")) {
		return nil, errs.ErrBadRequest
	}
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	nodeId := normalizeNodeID(opts.NodeID)
	task, err := s.tasks.StartTask(TaskStoreUninstall, fmt.Sprintf("卸载 %s", project), project, 15*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			ac, aerr := s.clientFor(nodeId)
			if aerr != nil {
				return aerr
			}
			dir := "/opt/ypanel/compose/" + project
			logf("info", "停止并移除容器")
			// down 失败不阻塞（项目目录/容器可能已被手动清理，僵尸记录照样可卸载）
			out, oerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, tctx, "POST", "/agent/v1/exec",
				&dto.ExecReq{Command: fmt.Sprintf("cd %s 2>/dev/null && docker compose -p %s down", dir, project), TimeoutSecs: 300})
			if oerr != nil || out.ExitCode != 0 {
				logf("warn", "停止容器未成功（可能已被移除），继续清理")
			}
			if opts.PurgeData {
				logf("info", "删除应用数据（compose 目录，含数据卷/数据库文件）")
				_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, tctx, "POST", "/agent/v1/files/delete",
					&dto.FileDeleteReq{Paths: []string{dir}})
			} else {
				logf("info", "应用数据已保留在 %s（重装同名应用可复用）", dir)
			}
			if opts.RemoveImage {
				logf("info", "删除应用镜像")
				images := `for img in $(grep -E "^[[:space:]]*image:" docker-compose.yml | awk '{print $2}' | tr -d '"'); do docker rmi -f "$img" 2>/dev/null && echo "removed $img"; done`
				out2, _ := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, tctx, "POST", "/agent/v1/exec",
					&dto.ExecReq{Command: fmt.Sprintf("cd %s && %s", dir, images), TimeoutSecs: 300})
				if out2.ExitCode == 0 && strings.TrimSpace(out2.Output) != "" {
					for _, l := range strings.Split(strings.TrimSpace(out2.Output), "\n") {
						logf("info", "%s", l)
					}
				}
			}
			if opts.CascadeDB {
				var linked []model.DatabaseInstance
				_ = s.db.Where("compose_project = ?", project).Find(&linked).Error
				for _, d := range linked {
					logf("info", "级联移除数据库纳管记录 %s（%s）及其备份", d.Name, d.Type)
					_, _ = agentclient.DoJSON[dto.FileDeleteReq, struct{}](ac, tctx, "POST", "/agent/v1/files/delete",
						&dto.FileDeleteReq{Paths: []string{"/opt/ypanel/backups/" + d.Type + "/" + d.Name}})
					_ = s.db.Delete(&model.DatabaseInstance{}, d.ID).Error
				}
			}
			_ = s.db.Where("compose_project = ?", project).Delete(&model.AppStoreInstall{}).Error
			logf("info", "卸载完成")
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID, "project": project}, nil
}

// Installed 已装列表。
// InstalledByProject 按编排项目名取安装记录（M54-P3 属主断言用）。
func (s *StoreService) InstalledByProject(project string) (*model.AppStoreInstall, error) {
	var inst model.AppStoreInstall
	if err := s.db.Where("compose_project = ?", project).First(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

// ownerFilterInstalls assigned 数据范围下仅保留 owner∈{uid,0} 的安装记录。
func ownerFilterInstalls(ctx context.Context, installs []model.AppStoreInstall) []model.AppStoreInstall {
	caller, ok := rbac.CallerFrom(ctx)
	if !ok || !caller.Assigned() {
		return installs
	}
	out := make([]model.AppStoreInstall, 0, len(installs))
	for _, i := range installs {
		if i.OwnerID == 0 || i.OwnerID == caller.UserID {
			out = append(out, i)
		}
	}
	return out
}

// SetInstallOwner 属主再分配（0=公共；仅 all 数据范围调用方可达，由 API 层把关）。
func (s *StoreService) SetInstallOwner(project string, ownerID uint) error {
	return s.db.Model(&model.AppStoreInstall{}).Where("compose_project = ?", project).Update("owner_id", ownerID).Error
}

func (s *StoreService) Installed() []model.AppStoreInstall {
	out := []model.AppStoreInstall{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
}

// StoreInstallInfo 已安装应用详情（列表卡片聚合：状态/端口/图标/可升级/参数）。
type StoreInstallInfo struct {
	ID             uint              `json:"id"`
	SourceID       uint              `json:"sourceId"`
	Key            string            `json:"key"`
	Name           string            `json:"name"`
	Remark         string            `json:"remark"`
	AppName        string            `json:"appName"`
	IconURL        string            `json:"iconUrl"`
	Version        string            `json:"version"`
	LatestVersion  string            `json:"latestVersion"`
	Upgradable     bool              `json:"upgradable"`
	ComposeProject string            `json:"composeProject"`
	Running        bool              `json:"running"`
	Ports          []int             `json:"ports"`
	Params         map[string]string `json:"params"` // 密码类值已打码
	OwnerID        uint              `json:"ownerId"` // M54-P3 数据范围属主（0=公共）
	CreatedAt      time.Time         `json:"createdAt"`
	NodeID string `json:"nodeId"`
}

// InstalledDetailed 已安装详情聚合（含 compose 运行状态、应用元数据与安装参数）。
func (s *StoreService) InstalledDetailed(ctx context.Context) ([]StoreInstallInfo, error) {
	installs := s.Installed()
	installs = ownerFilterInstalls(ctx, installs)
	if len(installs) == 0 {
		return []StoreInstallInfo{}, nil
	}
	// 应用元数据（icon/名称/最新版本）
	type appMeta struct {
		name, icon, latest string
	}
	metas := map[string]appMeta{}
	for _, i := range installs {
		mk := fmt.Sprintf("%d/%s", i.SourceID, i.Key)
		if _, ok := metas[mk]; ok {
			continue
		}
		var a model.AppStoreApp
		if err := s.db.Select("name", "icon_url", "latest_version").Where("source_id = ? AND key = ?", i.SourceID, i.Key).First(&a).Error; err == nil {
			metas[mk] = appMeta{name: a.Name, icon: a.IconURL, latest: a.LatestVersion}
		}
	}
	// compose 运行状态（M55：按节点分组路由查询）
	running := map[string]bool{}
	nodeClients := map[string]*agentclient.Client{}
	for _, i := range installs {
		nid := normalizeNodeID(i.NodeID)
		if _, ok := nodeClients[nid]; ok {
			continue
		}
		if ac, err := s.clientFor(nid); err == nil {
			nodeClients[nid] = ac
			if projects, perr := agentclient.GetJSON[[]dto.ComposeProject](ac, ctx, "/agent/v1/compose/projects"); perr == nil {
				for _, p := range projectsSafe(projects) {
					running[p.Name] = running[p.Name] || p.Running > 0
				}
			}
		}
	}
	out := make([]StoreInstallInfo, 0, len(installs))
	for _, i := range installs {
		info := StoreInstallInfo{
			ID: i.ID, SourceID: i.SourceID, Key: i.Key, Name: i.Name, Remark: i.Remark, Version: i.Version,
			ComposeProject: i.ComposeProject, Running: running[i.ComposeProject],
			OwnerID: i.OwnerID, CreatedAt: i.CreatedAt, Params: map[string]string{}, Ports: []int{},
			NodeID: normalizeNodeID(i.NodeID),
		}
		if m, ok := metas[fmt.Sprintf("%d/%s", i.SourceID, i.Key)]; ok {
			info.AppName = m.name
			info.IconURL = m.icon
			info.LatestVersion = m.latest
			info.Upgradable = m.latest != "" && i.Version != m.latest
		}
		var params map[string]string
		if i.ParamsJSON != "" {
			_ = json.Unmarshal([]byte(i.ParamsJSON), &params)
		}
		for k, v := range params {
			lk := strings.ToUpper(k)
			if strings.Contains(lk, "PASSWORD") || strings.Contains(lk, "SECRET") || strings.Contains(lk, "PASSWD") {
				info.Params[k] = "******"
				continue
			}
			info.Params[k] = v
			if strings.Contains(lk, "PORT") {
				if n, perr := strconv.Atoi(v); perr == nil && n > 0 && n < 65536 {
					info.Ports = append(info.Ports, n)
				}
			}
		}
		sort.Ints(info.Ports)
		out = append(out, info)
	}
	return out, nil
}

// InstalledAction 已安装应用操作：start / stop / restart / rebuild。
func (s *StoreService) InstalledAction(ctx context.Context, project, action, nodeId string) error {
	if !storeAppNamePattern.MatchString(strings.TrimPrefix(project, "app-")) {
		return errs.ErrBadRequest
	}
	var count int64
	_ = s.db.Model(&model.AppStoreInstall{}).Where("compose_project = ?", project).Count(&count).Error
	if count == 0 {
		return errs.New(errs.CodeNotFound, "error.installNotFound", "安装记录不存在")
	}
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return err
	}
	var cmd string
	switch action {
	case "start":
		_, err = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up",
			&dto.ComposeActionReq{Name: project})
		return err
	case "stop":
		_, err = agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/down",
			&dto.ComposeActionReq{Name: project})
		return err
	case "restart":
		cmd = fmt.Sprintf("docker compose -p %s restart", project)
	case "rebuild":
		cmd = fmt.Sprintf("docker compose -p %s up -d --force-recreate", project)
	default:
		return errs.Wrap(errs.ErrBadRequest, "不支持的操作: "+action)
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: 300})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, action+" 失败: "+tailOutput(out.Output, 800))
	}
	return nil
}

// InstallEnv 读取安装参数 .env（admin；含密码明文）。
func (s *StoreService) InstallEnv(ctx context.Context, project, nodeId string) (map[string]string, error) {
	if !storeAppNamePattern.MatchString(strings.TrimPrefix(project, "app-")) {
		return nil, errs.ErrBadRequest
	}
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return nil, err
	}
	out, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2("/opt/ypanel/compose/"+project+"/.env"))
	if err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "未找到参数文件（.env）")
	}
	env := map[string]string{}
	for _, line := range strings.Split(out.Content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			env[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return env, nil
}

// SaveInstallEnv 保存参数 .env 并重建容器使其生效（admin）。
func (s *StoreService) SaveInstallEnv(ctx context.Context, project, nodeId, content string) error {
	if !storeAppNamePattern.MatchString(strings.TrimPrefix(project, "app-")) {
		return errs.ErrBadRequest
	}
	// 行格式校验：仅允许 KEY=VALUE / 注释 / 空行，防注入 compose 上下文
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		k, _, ok := strings.Cut(t, "=")
		if !ok || !paramKeyPattern.MatchString(strings.TrimSpace(k)) {
			return errs.Wrap(errs.ErrBadRequest, "参数行不合法（需 KEY=VALUE 格式）: "+truncStr(t, 60))
		}
	}
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return err
	}
	if _, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: "/opt/ypanel/compose/" + project + "/.env", Content: content}); err != nil {
		return err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("docker compose -p %s up -d --force-recreate", project), TimeoutSecs: 300})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "参数已保存，但重建容器失败: "+tailOutput(out.Output, 800))
	}
	return nil
}

// ---------- 工具 ----------

func (s *StoreService) httpGet(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func (s *StoreService) writeViaAgent(ctx context.Context, ac *agentclient.Client, p, content string) error {
	_, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: p, Content: content})
	return err
}

// resolveStoreVersion 选版本（空=第一条，约定新版本在前）。
func resolveStoreVersion(versions []StoreVersion, versionID string) *StoreVersion {
	if versionID == "" && len(versions) > 0 {
		return &versions[0]
	}
	for i := range versions {
		if versions[i].ID == versionID || versions[i].Name == versionID {
			return &versions[i]
		}
	}
	if len(versions) > 0 {
		return &versions[0]
	}
	return nil
}

// latestVersionOf 最新版本（约定数组第一条）。
func latestVersionOf(versions []StoreVersion) string {
	if len(versions) > 0 {
		return versions[0].ID
	}
	return ""
}

func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// unzipEntry 从内存 zip 提取指定文件。
func unzipEntry(body []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("缺少 %s", name)
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// tailOutput 取输出尾部（compose 错误摘要在尾部而非首行）。
func tailOutput(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

// sanitizeParam 参数值白名单过滤（防注入 .env/shell）。
func sanitizeParam(v string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\'', '"', '`', '\\', '$', ';', '&', '|', '<', '>', '\n', '\r':
			return -1
		}
		return r
	}, strings.TrimSpace(v))
}

var paramKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

// escapeURL2 query 转义（files/list 等通道复用）。
func escapeURL2(s string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "?", "%3F", "#", "%23", "&", "%26", "+", "%2B", "/", "%2F")
	return r.Replace(s)
}

// ---- M32：安装外接数据库实例 ----

var (
	dbFieldHostStem = regexp.MustCompile(`(DB|MYSQL|SQL|MONGO|DATABASE|MARIA)`)
	dbFieldHostRe   = regexp.MustCompile(`^[A-Z0-9_]*HOST$`)
)

// dbFieldSetOf 从 formFields 与 compose 模板变量引用（${VAR}）的并集识别数据库参数键集
// （*_HOST 结尾且含 DB/SQL 等词干，同前缀推导 PORT/NAME/USER/PASSWORD）。
// 1Panel 包的 PANEL_DB_HOST/PORT 常为模板引用而非表单字段（系统按所选数据库自动注入），必须扫模板。
func dbFieldSetOf(fields []StoreFormField, composeText string) *dbFieldSet {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.EnvKey != "" {
			keys = append(keys, strings.ToUpper(f.EnvKey))
		}
	}
	for _, m := range regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`).FindAllStringSubmatch(composeText, -1) {
		keys = append(keys, m[1])
	}
	for _, k := range keys {
		if !dbFieldHostRe.MatchString(k) || !dbFieldHostStem.MatchString(strings.TrimSuffix(strings.TrimPrefix(k, "PANEL_"), "_HOST")) {
			continue
		}
		prefix := strings.TrimSuffix(k, "HOST")
		fs := &dbFieldSet{Host: k}
		if hasKey(keys, prefix+"PORT") {
			fs.Port = prefix + "PORT"
		}
		for _, cand := range []string{prefix + "NAME", prefix + "DATABASE"} {
			if hasKey(keys, cand) {
				fs.Name = cand
				break
			}
		}
		for _, cand := range []string{prefix + "USER", prefix + "USERNAME"} {
			if hasKey(keys, cand) {
				fs.User = cand
				break
			}
		}
		// 密码优先 USER_PASSWORD（应用账号），回退 PASSWORD/ROOT_PASSWORD
		for _, cand := range []string{prefix + "USER_PASSWORD", prefix + "PASSWORD", prefix + "ROOT_PASSWORD"} {
			if hasKey(keys, cand) {
				fs.Password = cand
				break
			}
		}
		return fs
	}
	return nil
}

func hasKey(keys []string, k string) bool {
	for _, x := range keys {
		if x == k {
			return true
		}
	}
	return false
}

// applyExternalDB 外接实例落地：reveal 凭据 → 按需建库建号（幂等）→ 注入连接参数。
// 连接地址：面板容器实例用宿主内网 IP + 映射端口（应用容器→宿主可达）；外部实例用 host 原样。
func (s *StoreService) applyExternalDB(ctx context.Context, logf TaskLogf, ver StoreVersion, in StoreInstallInput, params map[string]string, composeText string) error {
	ext := in.ExternalDB
	fs := dbFieldSetOf(ver.FormFields, composeText)
	if fs == nil || fs.Host == "" {
		return errs.Wrap(errs.ErrBadRequest, "未识别到数据库连接参数（*_HOST）")
	}
	inst, err := s.dbs.ByID(ext.InstanceID)
	if err != nil {
		return err
	}
	if inst.Type != "mysql" && inst.Type != "postgres" {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("暂仅支持 mysql/postgres 实例外接，实例类型: %s", inst.Type))
	}
	info, err := s.dbs.ConnectInfo(ctx, ext.InstanceID)
	if err != nil {
		return errs.Wrapc(errs.CodeInternal, "读取实例连接信息失败: "+err.Error())
	}

	dbName := ext.Database
	if dbName == "" {
		dbName = in.Key
	}
	dbUser := ext.User
	if dbUser == "" {
		dbUser = in.Key + "_user"
	}
	dbPass := randomHex(12)

	// 幂等建库建号（已存在即复用）
	if ext.CreateIfMissing {
		existing, err := s.dbs.Databases(ctx, ext.InstanceID)
		if err != nil {
			return errs.Wrapc(errs.CodeInternal, "列出实例数据库失败: "+err.Error())
		}
		found := false
		for _, d := range existing {
			if d.Name == dbName {
				found = true
				break
			}
		}
		if found {
			// 生产约束：不允许覆盖已有数据库——换名新建（追加序号）直到可用
			base := dbName
			for i := 2; i <= 50; i++ {
				cand := fmt.Sprintf("%s%d", base, i)
				candExists := false
				for _, d := range existing {
					if d.Name == cand {
						candExists = true
						break
					}
				}
				if !candExists {
					dbName = cand
					found = false
					break
				}
			}
			logf("info", "数据库 %s 已存在（不允许覆盖），已改用新库名 %s", base, dbName)
		}
		if !found {
			if err := s.dbs.CreateDatabase(ctx, ext.InstanceID, dbName, ""); err != nil {
				return errs.Wrapc(errs.CodeInternal, "创建数据库 "+dbName+" 失败: "+err.Error())
			}
			logf("info", "已在实例 %s 创建数据库 %s", inst.Name, dbName)
		}
		users, err := s.dbs.Users(ctx, ext.InstanceID)
		if err != nil {
			return errs.Wrapc(errs.CodeInternal, "列出实例账号失败: "+err.Error())
		}
		userFound := false
		for _, u := range users {
			if u.Name == dbUser {
				userFound = true
				break
			}
		}
		if !userFound {
			if err := s.dbs.CreateUser(ctx, ext.InstanceID, dbUser, "%", dbPass); err != nil {
				return errs.Wrapc(errs.CodeInternal, "创建账号 "+dbUser+" 失败: "+err.Error())
			}
			logf("info", "已在实例 %s 创建账号 %s", inst.Name, dbUser)
		} else {
			// 已存在账号不允许覆盖密码（用户约束）：换名新建（追加序号）直到可用
			base := dbUser
			for i := 2; i <= 50; i++ {
				cand := fmt.Sprintf("%s%d", base, i)
				if len(cand) > 32 {
					cand = cand[:28] + fmt.Sprintf("%d", i)
				}
				candExists := false
				users2, uerr := s.dbs.Users(ctx, ext.InstanceID)
				if uerr == nil {
					for _, u := range users2 {
						if u.Name == cand {
							candExists = true
							break
						}
					}
				}
				if !candExists {
					dbUser = cand
					break
				}
			}
			if err := s.dbs.CreateUser(ctx, ext.InstanceID, dbUser, "%", dbPass); err != nil {
				return errs.Wrapc(errs.CodeInternal, "创建账号 "+dbUser+" 失败: "+err.Error())
			}
			logf("info", "账号 %s 已存在（不覆盖密码），已新建账号 %s", base, dbUser)
		}
		if err := s.dbs.GrantUserDatabase(ctx, ext.InstanceID, dbName, dbUser, "%"); err != nil {
			return errs.Wrapc(errs.CodeInternal, "授权 "+dbUser+" 访问 "+dbName+" 失败: "+err.Error())
		}
		logf("info", "已授权 %s 访问 %s", dbUser, dbName)
	} else {
		logf("info", "跳过建库建号（未勾选自动创建），请确保 %s/%s 已存在", dbName, dbUser)
	}

	// 连接地址策略（M32）：应用默认接入统一网络，与实例容器同网络时用「容器名:内部端口」直连
	// （docker DNS，不依赖宿主端口映射）；不同网络或外部实例用宿主内网 IP + 映射端口
	// （实例凭据里的回环地址在应用容器内指向容器自身，必须改写）。
	appNet := in.Network
	if appNet == "" {
		appNet = PanelNetwork
	}
	// M55 跨节点：容器名直连仅当应用与实例同节点（不同节点 docker 网络隔离），否则走宿主内网 IP:映射端口
	sameNode := normalizeNodeID(in.NodeID) == "local"
	host := info.LanIP
	port := info.MapPort
	if info.Container != "" && sameNode && strings.Contains(info.Networks, appNet) {
		host = info.Container
		port = info.InnerPort
		logf("info", "实例容器与应用同在 %s 网络，连接地址使用 %s:%s（容器名直连）", appNet, host, port)
	} else {
		logf("info", "实例与应用不同网络或为外部实例，连接地址使用宿主内网 %s:%s", host, port)
	}

	if fs.Host != "" {
		params[fs.Host] = host
	}
	if fs.Port != "" {
		params[fs.Port] = port
	}
	if fs.Name != "" {
		params[fs.Name] = dbName
	}
	if fs.User != "" {
		params[fs.User] = dbUser
	}
	if fs.Password != "" {
		params[fs.Password] = dbPass
	}
	logf("info", "已注入外接数据库参数（实例 %s，库 %s，账号 %s）", inst.Name, dbName, dbUser)
	return nil
}

// hostLanIP 宿主内网 IP（agent hostname -I 首个地址，进程内缓存）。
func (s *StoreService) hostLanIP(ctx context.Context) (string, error) {
	return s.hostLanIPOf(ctx, "local")
}

// hostLanIPOf 目标节点内网 IP（agent hostname -I 首个地址；M55 跨节点外接数据库连接信息用）。
func (s *StoreService) hostLanIPOf(ctx context.Context, nodeId string) (string, error) {
	s.hostIPOnce.Do(func() {
		ac, err := s.client()
		if err != nil {
			s.hostIPErr = err
			return
		}
		res, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: "hostname -I | awk '{print $1}'", TimeoutSecs: 15})
		if err != nil {
			s.hostIPErr = err
			return
		}
		ip := strings.TrimSpace(res.Output)
		if ip == "" {
			s.hostIPErr = fmt.Errorf("宿主未返回内网 IP")
			return
		}
		s.hostIP = ip
	})
	return s.hostIP, s.hostIPErr
}
