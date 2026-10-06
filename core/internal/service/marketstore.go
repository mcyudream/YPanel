// AppStore 应用商店（接 1Panel 默认源）。
// 源机制（逆向自 reference/1panel）：{repo}/{mode}/1panel.json.zip，内含 1panel.json
// （apps: [{id,name,title,description,readMe,icon,tags,versions:[{id,name,downloadUrl,additionalProperties.formFields}]]}）。
// 安装 = 下载版本 tar.gz（docker-compose.yml+data.yml）→ 参数写 .env → compose up。
package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 默认源。
const (
	appStoreRepo   = "https://apps-assets.fit2cloud.com"
	appStoreMode   = "dev"
	appStoreSyncTTL = 24 * time.Hour
)

// AppStoreApp 源应用。
type AppStoreApp struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Key          string `gorm:"uniqueIndex;size:64;not null" json:"key"`
	Name         string `gorm:"size:128;not null" json:"name"`
	Title        string `gorm:"size:255" json:"title"`
	Description  string `gorm:"type:text" json:"description"`
	ReadMe       string `gorm:"type:text" json:"readMe"`
	IconURL      string `gorm:"size:512" json:"iconUrl"`
	Tags         string `gorm:"size:255" json:"tags"` // 逗号分隔
	VersionsJSON string `gorm:"type:text" json:"versionsJson"`
	LastModified int64  `json:"lastModified"`
	SyncedAt     time.Time `json:"syncedAt"`
}

// AppStoreVersion 版本（versions_json 内）。
type AppStoreVersion struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DownloadURL string `json:"downloadUrl"`
	FormFields  []AppStoreFormField `json:"formFields"`
}

// AppStoreFormField 版本参数定义（1Panel data.yml formFields 同构）。
type AppStoreFormField struct {
	EnvKey   string `json:"envKey"`
	Label    map[string]string `json:"label"`
	Default  interface{}       `json:"default"`
	Type     string `json:"type"`
	Rule     string `json:"rule"`
	Required bool   `json:"required"`
}

// AppStoreInstall 已安装的商店应用。
type AppStoreInstall struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	Key            string `gorm:"size:64;not null" json:"key"`
	Name           string `gorm:"size:64;not null" json:"name"`
	Version        string `gorm:"size:32;not null" json:"version"`
	ComposeProject string `gorm:"size:64;not null;uniqueIndex" json:"composeProject"`
	CreatedAt      time.Time `json:"createdAt"`
}

func init() {
	// 表注册由 db.go AutoMigrate 统一处理（见 db.go）
	_ = AppStoreFormField{}
}

var tableNameFix = func() {} // 占位：db.go 中已迁移 AppStoreApp/AppStoreInstall

// MarketService2 商店服务（与内置 MarketService 并存，前端区分来源）。
type MarketStoreService struct {
	db    *gorm.DB
	nodes *NodeService
	http  *http.Client
	mu    sync.Mutex
	lastSync time.Time
}

// client agent 通道客户端。
func (s *MarketStoreService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// NewMarketStoreService 创建。
func NewMarketStoreService(db *gorm.DB, nodes *NodeService) *MarketStoreService {
	return &MarketStoreService{db: db, nodes: nodes, http: &http.Client{Timeout: 5 * time.Minute}}
}

var appNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)

// storeListDTO 源清单结构（仅所需字段）。
type storeListDTO struct {
	Apps []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		ReadMe       string `json:"readMe"`
		Icon         string `json:"icon"`
		Tags         []string `json:"tags"`
		LastModified int64  `json:"lastModified"`
		Versions     []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DownloadURL string `json:"downloadUrl"`
			AdditionalProperties struct {
				FormFields []AppStoreFormField `json:"formFields"`
			} `json:"additionalProperties"`
		} `json:"versions"`
	} `json:"apps"`
}

// Sync 从默认源同步应用到本地库（24h 内不重复；force 可强制）。
func (s *MarketStoreService) Sync(ctx context.Context, force bool) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !force && time.Since(s.lastSync) < appStoreSyncTTL {
		return map[string]any{"skipped": true}, nil
	}
	url := appStoreRepo + "/" + appStoreMode + "/1panel.json.zip"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, errs.Wrap(errs.ErrAgentUnreach, "下载应用源失败: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errs.Wrap(errs.ErrAgentUnreach, fmt.Sprintf("应用源 HTTP %d", resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "应用源包解析失败: "+err.Error())
	}
	var listFile *zip.File
	for _, f := range zr.File {
		if f.Name == "1panel.json" {
			listFile = f
			break
		}
	}
	if listFile == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "应用源包缺少 1panel.json")
	}
	rc, err := listFile.Open()
	if err != nil {
		return nil, err
	}
	var list storeListDTO
	if err := json.NewDecoder(rc).Decode(&list); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "应用清单解析失败: "+err.Error())
	}
	_ = rc.Close()

	count := 0
	for _, a := range list.Apps {
		if a.ID == "" {
			continue
		}
		versions := []AppStoreVersion{}
		for _, v := range a.Versions {
			fields := v.AdditionalProperties.FormFields
			versions = append(versions, AppStoreVersion{ID: v.ID, Name: v.Name, DownloadURL: v.DownloadURL, FormFields: fields})
		}
		vj, _ := json.Marshal(versions)
		tags := strings.Join(a.Tags, ",")
		row := model.AppStoreApp{
			Key: a.ID, Name: a.Name, Title: a.Title,
			Description: truncStr(a.Description, 500), ReadMe: a.ReadMe,
			IconURL: a.Icon, Tags: tags, VersionsJSON: string(vj),
			LastModified: a.LastModified, SyncedAt: time.Now(),
		}
		// upsert by key
		var exist model.AppStoreApp
		if err := s.db.Where("key = ?", a.ID).First(&exist).Error; err == nil {
			_ = s.db.Model(&exist).Updates(map[string]any{
				"name": row.Name, "title": row.Title, "description": row.Description,
				"read_me": row.ReadMe, "icon_url": row.IconURL, "tags": row.Tags,
				"versions_json": row.VersionsJSON, "last_modified": row.LastModified, "synced_at": row.SyncedAt,
			}).Error
		} else {
			if err := s.db.Create(&row).Error; err != nil {
				continue
			}
			count++
		}
	}
	s.lastSync = time.Now()
	return map[string]any{"total": len(list.Apps), "created": count}, nil
}

// List 市场列表（search/tag 过滤）。
func (s *MarketStoreService) List(search, tag string) ([]model.AppStoreApp, error) {
	out := []model.AppStoreApp{}
	q := s.db.Model(&model.AppStoreApp{})
	if search != "" {
		kw := "%" + strings.ToLower(search) + "%"
		q = q.Where("lower(name) LIKE ? OR lower(title) LIKE ? OR lower(description) LIKE ? OR lower(key) LIKE ?", kw, kw, kw, kw)
	}
	if tag != "" && tag != "全部" {
		q = q.Where("tags LIKE ?", "%"+tag+"%")
	}
	if err := q.Order("name").Limit(500).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Get 单个应用详情。
func (s *MarketStoreService) Get(key string) (*model.AppStoreApp, []AppStoreVersion, error) {
	var row model.AppStoreApp
	if err := s.db.Where("key = ?", key).First(&row).Error; err != nil {
		return nil, nil, errs.New(errs.CodeNotFound, "error.appNotFound", "应用不存在（可能未同步）")
	}
	versions := []AppStoreVersion{}
	_ = json.Unmarshal([]byte(row.VersionsJSON), &versions)
	return &row, versions, nil
}

// resolveVersion 选版本（空=第一条，1P 源按新到旧排列）。
func resolveVersion(versions []AppStoreVersion, versionID string) *AppStoreVersion {
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

// Install 安装商店应用：下载包 → 参数渲染 .env → compose up。
func (s *MarketStoreService) Install(ctx context.Context, key, versionID, name string, params map[string]string) (map[string]any, error) {
	if !appNamePattern.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "应用实例名不合法（小写字母/数字/中划线）")
	}
	_, versions, err := s.Get(key)
	if err != nil {
		return nil, err
	}
	ver := resolveVersion(versions, versionID)
	if ver == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "版本不存在")
	}

	// 合并参数：formFields default ∪ 用户 params；值白名单过滤
	finalParams := map[string]string{}
	for _, f := range ver.FormFields {
		if f.EnvKey == "" {
			continue
		}
		finalParams[f.EnvKey] = sanitizeParam(fmt.Sprint(f.Default))
	}
	for k, v := range params {
		if !paramKeyPattern.MatchString(k) {
			return nil, errs.Wrap(errs.ErrBadRequest, "参数名不合法: "+k)
		}
		finalParams[k] = sanitizeParam(v)
	}
	finalParams["CONTAINER_NAME"] = "app-" + name
	finalParams["CONTAINER_NAME1"] = "app-" + name + "-1"

	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	project := "app-" + name
	dir := "/opt/ypanel/compose/" + project

	// 编排命令序列（每条独立 exec；文件内容走 files write 通道避免转义）
	steps := []struct {
		desc, cmd string
	}{
		{"创建目录", fmt.Sprintf("mkdir -p %s", dir)},
		{"下载应用包", fmt.Sprintf("curl -sL -o %s/pkg.tar.gz '%s'", dir, ver.DownloadURL)},
		{"解压", fmt.Sprintf("cd %s && tar xzf pkg.tar.gz", dir)},
		{"定位 compose", fmt.Sprintf("cd %s && find . -name docker-compose.yml | head -1 | xargs -I{} cp {} ./docker-compose.yml", dir)},
		{"创建网络", "docker network create 1panel-network 2>/dev/null; true"},
	}
	var logs []string
	for _, st := range steps {
		out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: st.cmd, TimeoutSecs: 600})
		if err != nil {
			return nil, errs.Wrapc(errs.CodeFileOpFailed, st.desc+" 失败: "+err.Error())
		}
		if out.ExitCode != 0 {
			return map[string]any{"logs": logs}, errs.Wrapc(errs.CodeFileOpFailed, st.desc+" 失败: "+firstLine(out.Output))
		}
		logs = append(logs, st.desc+" ✓")
	}

	// .env 渲染
	var envBuf strings.Builder
	envBuf.WriteString("CONTAINER_NAME=" + project + "\n")
	keys := make([]string, 0, len(finalParams))
	for k := range finalParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		envBuf.WriteString(k + "=" + finalParams[k] + "\n")
	}
	if err := s.writeViaAgent(ctx, dir+"/.env", envBuf.String()); err != nil {
		return nil, err
	}

	// compose up
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("cd %s && docker compose -p %s up -d", dir, project), TimeoutSecs: 600})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return map[string]any{"logs": append(logs, out.Output)}, errs.Wrapc(errs.CodeFileOpFailed, "compose up 失败: "+firstLine(out.Output))
	}
	logs = append(logs, "compose up ✓")

	// 已装记录
	var exist model.AppStoreInstall
	if err := s.db.Where("compose_project = ?", project).First(&exist).Error; err == nil {
		_ = s.db.Model(&exist).Updates(map[string]any{"version": ver.Name}).Error
	} else {
		_ = s.db.Create(&model.AppStoreInstall{Key: key, Name: name, Version: ver.Name, ComposeProject: project}).Error
	}
	return map[string]any{"project": project, "logs": strings.Join(logs, "\n")}, nil
}

// Uninstall 卸载（compose down + 清目录，保留数据卷）。
func (s *MarketStoreService) Uninstall(ctx context.Context, project string) error {
	if !appNamePattern.MatchString(strings.TrimPrefix(project, "app-")) {
		return errs.ErrBadRequest
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	dir := "/opt/ypanel/compose/" + project
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf("cd %s 2>/dev/null && docker compose -p %s down; rm -rf %s", dir, project, dir), TimeoutSecs: 300})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "卸载失败: "+firstLine(out.Output))
	}
	_ = s.db.Where("compose_project = ?", project).Delete(&model.AppStoreInstall{}).Error
	return nil
}

// Installed 已装列表。
func (s *MarketStoreService) Installed() []model.AppStoreInstall {
	out := []model.AppStoreInstall{}
	_ = s.db.Order("id desc").Find(&out).Error
	return out
}

// writeViaAgent 经 agent files 通道写文件。
func (s *MarketStoreService) writeViaAgent(ctx context.Context, p, content string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: p, Content: content})
	return err
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

// truncStr 截断。
func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
