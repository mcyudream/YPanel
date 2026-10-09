// 商店收尾（B25 顺延项）：本地应用包扫描入库 + 已装应用升级检查。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// localStoreDir 本地应用包扫描根（agent 主机侧，1Panel 包目录：每个子目录一个应用）。
const localStoreDir = "/opt/ypanel/store-local"

// ScanLocalStore 扫描本地应用包目录，把 1Panel 格式包（子目录含 1panel.json 或 app.ini + data/) 入库为
// 「本地源」（source type=local，key 前缀 local-）。幂等：重复扫描按 (source,key) upsert。
func (s *StoreService) ScanLocalStore(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	list, err := agentclient.GetJSON[dto.FileListResp](ac, ctx, "/agent/v1/files/list?path="+escapeURL(localStoreDir))
	if err != nil {
		return nil, errs.Wrap(errs.ErrNotFound, "本地包目录不存在（"+localStoreDir+"），请先放置应用包目录")
	}
	// 本地源（固定单例，name=本地应用包）
	var src model.AppStoreSource
	if err := s.db.Where("type = 'local'").First(&src).Error; err != nil {
		src = model.AppStoreSource{
			Name: "本地应用包", Type: "local", Enabled: true, Builtin: true,
			Remark: "服务器 " + localStoreDir + " 目录扫描入库（只读对照 1Panel 包格式）",
		}
		if err := s.db.Create(&src).Error; err != nil {
			return nil, err
		}
	}
	scanned, imported := 0, 0
	rows := make([]model.AppStoreApp, 0, 4)
	for _, e := range list.Entries {
		if !e.IsDir {
			continue
		}
		scanned++
		// 读包清单（1Panel 格式 1panel.json；缺失则跳过）
		manifestPath := path.Join(e.Path, "1panel.json")
		out, rerr := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL(manifestPath))
		if rerr != nil || out.Truncated {
			continue
		}
		var mf struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			ReadMe      string   `json:"readMe"`
			Icon        string   `json:"icon"`
			Tags        []string `json:"tags"`
			Versions    []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"versions"`
		}
		if json.Unmarshal([]byte(out.Content), &mf) != nil || mf.ID == "" {
			continue
		}
		key := "local-" + mf.ID
		versions := make([]StoreVersion, 0, len(mf.Versions))
		// 本地包版本目录 = <appDir>/<version>/（compose 在版本目录内）；本地路径直装
		for _, v := range mf.Versions {
			versions = append(versions, StoreVersion{ID: v.ID, Name: v.Name, DownloadURL: path.Join(e.Path, v.ID)})
		}
		if len(versions) == 0 {
			versions = append(versions, StoreVersion{ID: "1.0.0", Name: "1.0.0", DownloadURL: e.Path})
		}
		rows = append(rows, model.AppStoreApp{
			Key: key, Name: mf.Name, Title: mf.Title,
			Description: truncStr(mf.Description, 500), ReadMe: mf.ReadMe,
			IconURL: mf.Icon, Tags: strings.Join(mf.Tags, ","),
			Kind: "app", Arch: "amd64,arm64",
			VersionsJSON: marshalJSON(versions), LatestVersion: latestVersionOf(versions),
			LastModified: e.ModTime.Unix(), SyncedAt: time.Now(),
		})
		imported++
	}
	if len(rows) > 0 {
		if _, err := s.upsertApps(src.ID, rows); err != nil {
			return nil, err
		}
	}
	now := time.Now()
	_ = s.db.Model(&model.AppStoreSource{}).Where("id = ?", src.ID).
		Updates(map[string]any{"status": "ok", "last_sync_at": &now, "message": fmt.Sprintf("扫描 %d 个目录，入库 %d 个应用", scanned, imported)}).Error
	return map[string]any{"sourceId": src.ID, "scanned": scanned, "imported": imported}, nil
}

// CheckUpgrades 已装应用升级检查：与源内最新版本比对，标记 upgradable。
// 已装记录有 SourceID（安装时来源），源应用被删除时跳过。
func (s *StoreService) CheckUpgrades(ctx context.Context) (map[string]any, error) {
	installs := s.Installed()
	checked, upgradable := 0, 0
	details := make([]map[string]any, 0, len(installs))
	for _, inst := range installs {
		if inst.SourceID == 0 {
			continue
		}
		var app model.AppStoreApp
		if err := s.db.Where("source_id = ? AND key = ?", inst.SourceID, inst.Key).First(&app).Error; err != nil {
			details = append(details, map[string]any{"project": inst.ComposeProject, "key": inst.Key, "status": "源应用不存在"})
			continue
		}
		checked++
		has := app.LatestVersion != "" && app.LatestVersion != inst.Version
		if has {
			upgradable++
			_ = s.db.Model(&model.AppStoreInstall{}).Where("id = ?", inst.ID).Update("version", inst.Version).Error
		}
		details = append(details, map[string]any{
			"project": inst.ComposeProject, "key": inst.Key, "current": inst.Version,
			"latest": app.LatestVersion, "upgradable": has,
		})
	}
	sort.Slice(details, func(i, j int) bool { return details[i]["project"].(string) < details[j]["project"].(string) })
	return map[string]any{"checked": checked, "upgradable": upgradable, "items": details}, nil
}

// UpgradeApp 已装应用升级：按安装参数重装到最新版本（复用重装管线，保留 data 目录）。
func (s *StoreService) UpgradeApp(ctx context.Context, project string) (map[string]any, error) {
	var inst model.AppStoreInstall
	if err := s.db.Where("compose_project = ?", project).First(&inst).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.installNotFound", "安装记录不存在")
	}
	var app model.AppStoreApp
	if err := s.db.Where("source_id = ? AND key = ?", inst.SourceID, inst.Key).First(&app).Error; err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "源应用不存在，无法升级")
	}
	if app.LatestVersion == "" || app.LatestVersion == inst.Version {
		return map[string]any{"upgraded": false, "version": inst.Version}, nil
	}
	var params map[string]any
	_ = json.Unmarshal([]byte(inst.ParamsJSON), &params)
	// Install 对同名项目自带重装语义（down 同名项目→端口预检→部署→记录 upsert）
	out, err := s.Install(ctx, StoreInstallInput{
		SourceID: inst.SourceID, Key: inst.Key, Name: inst.Name, Version: app.LatestVersion,
		Params: params, NodeID: inst.NodeID, // M55：升级保持在原节点
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"upgraded": true, "from": inst.Version, "to": app.LatestVersion, "task": out}, nil
}
