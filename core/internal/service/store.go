// StoreService 统一应用商店：多源驱动（1Panel zip / yp-url index.json / yp-git 仓库）。
//
// yp 源格式（index.json）：
//
//	{ "name": "...", "apps": [{ "id","name","category","kind","description","author","arch",
//	  "logo","readme","reverseProxy","versions":[{ "id","package","releaseNotes","env":[{key,label,type,default,required,rule}] }]}]}
//
// kind：app 普通应用 / service 面板功能依赖的环境服务 / middleware 可共享复用中间件。
// 版本包形态：downloadUrl（远端 tar.gz，agent 端下载解压）或 package（yp-git 仓库内目录，core 读文件写入）。
// 安装 = compose 目录 + .env 渲染 + docker compose up（接入 1panel-network，与站点反代互通）。
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

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
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

// StoreService 商店服务。
type StoreService struct {
	db    *gorm.DB
	nodes *NodeService
	sites *SiteService
	tasks *TaskService
	http  *http.Client
	mu    sync.Mutex
	dir   string // 源缓存目录根 <data>/store-sources
}

// NewStoreService 创建（dataDir 为面板数据目录；sites 用于一键反代、tasks 用于任务中心，均可为 nil）。
func NewStoreService(db *gorm.DB, nodes *NodeService, sites *SiteService, tasks *TaskService, dataDir string) *StoreService {
	return &StoreService{
		db: db, nodes: nodes, sites: sites, tasks: tasks,
		http: &http.Client{Timeout: 5 * time.Minute},
		dir:  filepath.Join(dataDir, storeSourceDirName),
	}
}

func (s *StoreService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
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
			Versions     []struct {
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
			Kind: "app", VersionsJSON: marshalJSON(versions), LatestVersion: latestVersionOf(versions),
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
	_ = s.db.Select("tags").Find(&rows).Error
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
	SourceID uint              `json:"sourceId" binding:"required"`
	Key      string            `json:"key" binding:"required"`
	Version  string            `json:"version"`
	Name     string            `json:"name" binding:"required"`
	Params   map[string]string `json:"params"`
	Domain   string            `json:"domain"` // 可选：安装后一键反代域名
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
		if f.Random && f.Type == "password" && (in.Params == nil || in.Params[f.EnvKey] == "") {
			finalParams[f.EnvKey] = randomHex(12)
		}
	}
	for k, v := range in.Params {
		if !paramKeyPattern.MatchString(k) {
			return nil, errs.Wrap(errs.ErrBadRequest, "参数名不合法: "+k)
		}
		if v != "" {
			finalParams[k] = sanitizeParam(v)
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
	ac, err := s.client()
	if err != nil {
		return err
	}
	logf("info", "开始安装 %s 版本 %s → 项目 %s", app.Name, ver.ID, project)

	// 重装场景：先 down 同名项目释放端口与容器
	if out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("docker compose -p %s ps --format '{{.Name}}' 2>/dev/null | grep -q . && cd /opt/ypanel/compose/%s 2>/dev/null && docker compose -p %s down || true", project, project, project), TimeoutSecs: 120}); err == nil {
		if strings.Contains(out.Output, project) || out.ExitCode == 0 {
			logf("info", "已停止同名旧项目（重装）")
		}
	}

	// 端口占用预检
	if err := s.precheckPorts(ctx, logf, params); err != nil {
		return err
	}

	logs, err := s.deployCompose(ctx, logf, app, ver, project, params)
	if err != nil {
		return err
	}
	_ = logs

	// 已装记录 upsert（同名重装=换版本）
	var exist model.AppStoreInstall
	if err := s.db.Where("compose_project = ?", project).First(&exist).Error; err == nil {
		_ = s.db.Model(&exist).Updates(map[string]any{
			"source_id": app.SourceID, "key": app.Key, "name": in.Name, "version": ver.ID,
		}).Error
	} else {
		_ = s.db.Create(&model.AppStoreInstall{
			SourceID: app.SourceID, Key: app.Key, Name: in.Name, Version: ver.ID, ComposeProject: project,
		}).Error
	}
	logf("info", "安装完成，项目 %s 已启动", project)

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
func (s *StoreService) precheckPorts(ctx context.Context, logf TaskLogf, params map[string]string) error {
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
	ac, err := s.client()
	if err != nil {
		return err
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

// deployCompose 两种包形态的统一部署（步骤日志写任务）。
func (s *StoreService) deployCompose(ctx context.Context, logf TaskLogf, app model.AppStoreApp, ver StoreVersion, project string, params map[string]string) (string, error) {
	ac, err := s.client()
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
		if err := step("定位 compose", fmt.Sprintf("cd %s && find . -name 'docker-compose.y*ml' -o -name 'compose.y*ml' | head -1 | xargs -I{} cp {} ./docker-compose.yml", dir), 60); err != nil {
			return strings.Join(logs, "\n"), err
		}
	case ver.LocalDir != "":
		// 本地包（yp-git）：core 读缓存目录写入 agent
		n, werr := s.writeLocalPackage(ctx, app.SourceID, ver.LocalDir, dir)
		if werr != nil {
			return strings.Join(logs, "\n"), errs.Wrapc(errs.CodeFileOpFailed, "写入应用包失败: "+werr.Error())
		}
		logs = append(logs, fmt.Sprintf("写入应用包 %d 个文件 ✓", n))
		logf("info", "写入应用包 %d 个文件 ✓", n)
	default:
		return strings.Join(logs, "\n"), errs.Wrap(errs.ErrBadRequest, "版本包地址缺失")
	}
	if err := step("创建网络", "docker network create 1panel-network 2>/dev/null; true", 60); err != nil {
		return strings.Join(logs, "\n"), err
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
	if err := s.writeViaAgent(ctx, dir+"/.env", envBuf.String()); err != nil {
		return strings.Join(logs, "\n"), err
	}
	logs = append(logs, "写入 .env ✓")
	logf("info", "写入 .env ✓")
	// 先 pull（大镜像耗时长，进度/错误完整可读）再 up
	_ = step("拉取镜像", fmt.Sprintf("cd %s && docker compose -p %s pull --quiet 2>&1 | tail -5; test ${PIPESTATUS[0]} -eq 0", dir, project), 1800)
	if err := step("compose up", fmt.Sprintf("cd %s && docker compose -p %s up -d", dir, project), 600); err != nil {
		return strings.Join(logs, "\n"), err
	}
	return strings.Join(logs, "\n"), nil
}

// writeLocalPackage 把仓库内包目录的文本文件写入 agent 目标目录（compose 文件统一命名 docker-compose.yml）。
func (s *StoreService) writeLocalPackage(ctx context.Context, sourceID uint, pkgRel, remoteDir string) (int, error) {
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
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := s.writeViaAgent(ctx, remoteDir+"/"+name, string(b)); err != nil {
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

// Uninstall 卸载（异步任务：compose down + 清目录，保留数据卷）。
func (s *StoreService) Uninstall(ctx context.Context, project string) (map[string]any, error) {
	if !storeAppNamePattern.MatchString(strings.TrimPrefix(project, "app-")) {
		return nil, errs.ErrBadRequest
	}
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	task, err := s.tasks.StartTask(TaskStoreUninstall, fmt.Sprintf("卸载 %s", project), project, 10*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			ac, aerr := s.client()
			if aerr != nil {
				return aerr
			}
			dir := "/opt/ypanel/compose/" + project
			logf("info", "停止并移除容器（数据卷保留在项目目录）")
			out, oerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, tctx, "POST", "/agent/v1/exec",
				&dto.ExecReq{Command: fmt.Sprintf("cd %s 2>/dev/null && docker compose -p %s down; rm -rf %s", dir, project, dir), TimeoutSecs: 300})
			if oerr != nil {
				return oerr
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "卸载失败: "+tailOutput(out.Output, 800))
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
func (s *StoreService) Installed() []model.AppStoreInstall {
	out := []model.AppStoreInstall{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
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

func (s *StoreService) writeViaAgent(ctx context.Context, p, content string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
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
