package service

// PHP 应用类型（kind: php）：源码型 PHP Web 应用一键部署。
// 安装 = （P2：自动建库）→ 复用/创建 PHP 运行时 → 创建站点绑定运行时 → 落源码 → 执行安装命令。
// 运行时复用策略：同节点同 PHP 版本、且现有扩展 ⊇ 应用必需扩展即复用；否则创建独立运行时。
// 卸载保留运行时（同节点共享基建，RuntimeService.Delete 自带站点绑定保护）。
// 计划：docs/plan/store-php-app-kind.md

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"github.com/ypanel/core/internal/model"
)

// phpAppManifest PHP 应用清单（kind:php 版本包内 app.json，地位等同 compose 应用的 compose.yml）。
type phpAppManifest struct {
	PHP struct {
		Versions   []string `json:"versions"` // 兼容的 PHP 版本（安装向导下拉）
		Default    string   `json:"default"`
		Extensions struct {
			Required []string `json:"required"` // 必需扩展（不满足则新建运行时）
			Optional []string `json:"optional"` // 可选扩展（P2：向导勾选）
		} `json:"extensions"`
		INI map[string]string `json:"ini"` // php.ini 覆盖（P2：写入运行时 conf）
	} `json:"php"`
	Web struct {
		Rewrite string `json:"rewrite"` // 伪静态模板（P2：laravel 等）
		Root    string `json:"root"`    // 站点入口子目录（如 public，P2）
	} `json:"web"`
	Database struct {
		Engine string `json:"engine"` // mysql / pg（P2：自动建库建号）
		Create bool   `json:"create"`
	} `json:"database"`
	Install []string `json:"install"` // 安装命令序列（运行时容器内、站点目录下执行）
	Source  string   `json:"source"`  // 包内源码目录（相对 package，默认 source/）
}

func loadPHPManifest(path string) (*phpAppManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m phpAppManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("app.json 解析失败: %w", err)
	}
	if len(m.PHP.Versions) == 0 {
		return nil, fmt.Errorf("app.json 缺少 php.versions")
	}
	if m.PHP.Default == "" {
		m.PHP.Default = m.PHP.Versions[0]
	}
	if m.Source == "" {
		m.Source = "source"
	}
	return &m, nil
}

// phpSyntheticFields php 应用的向导合成字段（PHP 版本下拉 / 站点地址 / 端口），前端零改动渲染。
func phpSyntheticFields(m *phpAppManifest) []StoreFormField {
	values := make([]StoreFormValue, 0, len(m.PHP.Versions))
	for _, v := range m.PHP.Versions {
		values = append(values, StoreFormValue{Label: "PHP " + v, Value: v})
	}
	return []StoreFormField{
		{EnvKey: "PHP_VERSION", Label: map[string]string{"zh": "PHP 版本"}, Type: "select", Values: values, Default: m.PHP.Default, Required: true},
		{EnvKey: "SITE_DOMAIN", Label: map[string]string{"zh": "站点地址（IP 或域名）"}, Type: "text", Required: true,
			Description: FlexString("浏览器访问的主机地址；填本机 IP 可 http://IP:端口 直访")},
		{EnvKey: "SITE_PORT", Label: map[string]string{"zh": "站点端口"}, Type: "number", Rule: "paramPort", Default: 38090, Required: true},
	}
}

// phpExtSuperset 现有运行时扩展是否覆盖应用必需扩展（复用判定）。
func phpExtSuperset(envJSON string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	var env struct {
		Extensions []string `json:"extensions"`
	}
	if json.Unmarshal([]byte(envJSON), &env) != nil {
		return false
	}
	have := map[string]bool{}
	for _, e := range env.Extensions {
		have[strings.ToLower(e)] = true
	}
	for _, e := range required {
		if !have[strings.ToLower(e)] {
			return false
		}
	}
	return true
}

// runInstallPHP php 应用安装主体（异步任务闭包）。
func (s *StoreService) runInstallPHP(ctx context.Context, logf TaskLogf, app model.AppStoreApp, ver StoreVersion, in StoreInstallInput, project string, params map[string]string) error {
	nodeId := normalizeNodeID(in.NodeID)

	// 重装场景：移除同名旧站点（配置与源码随后重建）
	var oldSite model.Site
	if err := s.db.Where("name = ?", project).First(&oldSite).Error; err == nil {
		logf("info", "已移除同名旧站点（重装）")
		if err := s.sites.Delete(ctx, oldSite.ID, SiteDeleteOptions{PurgeFiles: true}); err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, "移除旧站点失败: "+err.Error())
		}
	}

	// 1. 解析 app.json（yp-git 源检出目录）
	var src model.AppStoreSource
	if err := s.db.First(&src, app.SourceID).Error; err != nil || src.Type != "yp-git" {
		return errs.Wrap(errs.ErrBadRequest, "php 应用当前仅支持 yp-git 源")
	}
	pkgDir := filepath.Join(s.dir, fmt.Sprintf("src-%d", app.SourceID), filepath.FromSlash(ver.LocalDir))
	mf, err := loadPHPManifest(filepath.Join(pkgDir, "app.json"))
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "读取应用清单失败: "+err.Error())
	}

	// 2. PHP 版本与必需扩展
	phpVer := params["PHP_VERSION"]
	if phpVer == "" {
		phpVer = mf.PHP.Default
	}
	verOk := false
	for _, v := range mf.PHP.Versions {
		if v == phpVer {
			verOk = true
			break
		}
	}
	if !verOk {
		return errs.Wrap(errs.ErrBadRequest, "PHP 版本不在应用声明的兼容列表内: "+phpVer)
	}
	need := mf.PHP.Extensions.Required

	// 3. 运行时复用/创建（同节点同版本扩展超集即复用）
	var rt model.Runtime
	_ = s.db.Where("type = ? AND version = ? AND node_id = ? AND origin = ?", "php", phpVer, nodeId, "container").
		Order("id asc").First(&rt).Error
	if rt.ID != 0 && !phpExtSuperset(rt.EnvJSON, need) {
		logf("info", "已有 PHP %s 运行时扩展不满足（需 %v），改建独立运行时", phpVer, need)
		rt = model.Runtime{}
	}
	if rt.ID == 0 {
		extDesc := "无额外扩展"
		if len(need) > 0 {
			extDesc = strings.Join(need, "、")
		}
		logf("info", "创建 PHP %s 运行时（扩展：%s），构建可能需要数分钟…", phpVer, extDesc)
		created, cerr := s.runtimes.Create(ctx, RuntimeCreateInput{
			Name: project, Type: "php", Version: phpVer, Extensions: need,
			NodeID: in.NodeID, Remark: "商店应用 " + app.Name,
		})
		if cerr != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, "创建运行时失败: "+cerr.Error())
		}
		row := created["runtime"].(*model.Runtime)
		deadline := time.Now().Add(12 * time.Minute)
		tick := 0
		for {
			time.Sleep(3 * time.Second)
			var cur model.Runtime
			if err := s.db.First(&cur, row.ID).Error; err != nil {
				return err
			}
			if cur.Status == RuntimeStatusRunning {
				rt = cur
				logf("info", "运行时就绪（%s）", cur.ContainerName)
				break
			}
			if cur.Status == RuntimeStatusError {
				return fmt.Errorf("运行时构建失败: %s", cur.Message)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("运行时构建超时（状态 %s）", cur.Status)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			tick++
			if tick%10 == 0 {
				logf("info", "运行时构建中…（%d 秒）", tick*3)
			}
		}
	} else {
		logf("info", "复用 PHP %s 运行时 %s（同节点共享）", phpVer, rt.Name)
	}

	// 4. 创建站点绑定运行时（nginx fastcgi → 运行时容器:9000；端口映射/防火墙由站点模块处理）
	port := 0
	_, _ = fmt.Sscanf(params["SITE_PORT"], "%d", &port)
	site, serr := s.sites.Create(ctx, SiteCreateInput{
		Name: project, Type: "php", Domain: params["SITE_DOMAIN"], Port: port,
		RuntimeID: rt.ID, IndexFiles: "index.php", NodeID: in.NodeID,
		Remark: "商店应用 " + app.Name,
	})
	if serr != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "创建站点失败: "+serr.Error())
	}
	logf("info", "站点已创建：%s（%s:%d）", site.Name, params["SITE_DOMAIN"], port)

	// 5. 源码落盘：站点根 /opt/ypanel/nginx/www/sites/<name>/（nginx 与 php 容器共享挂载 /var/www）
	ac, err := s.clientFor(in.NodeID)
	if err != nil {
		return err
	}
	siteDir := "/opt/ypanel/nginx/www/sites/" + project
	srcRoot := filepath.Join(pkgDir, filepath.FromSlash(mf.Source))
	count := 0
	werr := filepath.WalkDir(srcRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if count >= ypPkgMaxFileCount {
			return fmt.Errorf("源码文件数超限")
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if info.Size() > ypPkgMaxFileSize {
			return fmt.Errorf("源码文件过大: %s", d.Name())
		}
		rel, rerr := filepath.Rel(srcRoot, p)
		if rerr != nil {
			return rerr
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		if werr := s.writeViaAgent(ctx, ac, siteDir+"/"+filepath.ToSlash(rel), string(b)); werr != nil {
			return werr
		}
		count++
		return nil
	})
	if werr != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "源码落盘失败: "+werr.Error())
	}
	logf("info", "源码已落盘 %d 个文件 → %s", count, siteDir)

	// 6. 应用安装命令（运行时容器内、站点目录下逐条执行）
	for _, cmd := range mf.Install {
		full := fmt.Sprintf("docker exec %s sh -c 'cd /var/www/sites/%s && %s'", rt.ContainerName, project, cmd)
		out, oerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: full, TimeoutSecs: 600})
		if oerr != nil || out.ExitCode != 0 {
			detail := ""
			if oerr != nil {
				detail = oerr.Error()
			} else {
				detail = tailOutput(out.Output, 600)
			}
			return errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("安装命令失败（%s）: %s", cmd, detail))
		}
		logf("info", "install ✓ %s", cmd)
	}

	// 7. 已装记录 upsert（与 compose 安装同模式）
	params["PHP_RUNTIME"] = rt.Name
	var exist model.AppStoreInstall
	if err := s.db.Where("compose_project = ? AND node_id = ?", project, nodeId).First(&exist).Error; err == nil {
		_ = s.db.Model(&exist).Updates(map[string]any{
			"source_id": app.SourceID, "key": app.Key, "name": in.Name, "version": ver.ID,
			"params_json": marshalJSON(params), "node_id": nodeId,
		}).Error
	} else {
		_ = s.db.Create(&model.AppStoreInstall{
			SourceID: app.SourceID, Key: app.Key, Name: in.Name, Version: ver.ID,
			ComposeProject: project, ParamsJSON: marshalJSON(params), OwnerID: in.OwnerID,
			NodeID: nodeId,
		}).Error
	}
	logf("info", "安装完成，访问 http://%s:%d", params["SITE_DOMAIN"], port)
	return nil
}

// uninstallPHP php 应用卸载：删站点（源码随站点删除）+ 保留共享运行时。
func (s *StoreService) uninstallPHP(ctx context.Context, logf TaskLogf, project string, purgeData bool) error {
	var site model.Site
	if err := s.db.Where("name = ?", project).First(&site).Error; err == nil {
		if err := s.sites.Delete(ctx, site.ID, SiteDeleteOptions{PurgeFiles: purgeData}); err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, "移除站点失败: "+err.Error())
		}
		logf("info", "站点 %s 已移除", site.Name)
	}
	logf("info", "PHP 运行时为同节点共享基建，已保留（可在「运行环境」管理）")
	return nil
}
