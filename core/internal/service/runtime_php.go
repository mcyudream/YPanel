// runtime_php.go PHP 运行时管理：扩展装/卸、php.ini 快捷设置、FPM 进程池配置、FPM 状态。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// PHPExtension PHP 扩展目录项（catalog/php_extensions.json）。
type PHPExtension struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Check string `json:"check"` // php -m 输出中的模块名（比对用）
}

var phpExtCatalog = func() []PHPExtension {
	var out []PHPExtension
	raw := mustEmbed("runtimedata/catalog/php_extensions.json")
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		panic("php_extensions.json 解析失败: " + err.Error())
	}
	return out
}()

// normalizePHPExtensions 校验并去重扩展清单。
func normalizePHPExtensions(in []string) ([]string, error) {
	known := map[string]bool{}
	for _, e := range phpExtCatalog {
		known[e.Name] = true
	}
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !known[e] {
			return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的 PHP 扩展: "+e)
		}
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out, nil
}

// getRuntimeRow 按 id 取运行时记录。
func (s *RuntimeService) getRuntimeRow(id uint) (*model.Runtime, error) {
	var row model.Runtime
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.runtimeNotFound", "运行环境不存在")
	}
	return &row, nil
}

// PHPExtensionItem 目录项 + 安装态。
type PHPExtensionItem struct {
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Installed bool   `json:"installed"`
}

// PHPExtensionsResp 扩展清单响应。
type PHPExtensionsResp struct {
	Catalog   []PHPExtensionItem `json:"catalog"`
	Installed []string           `json:"installed"` // php -m 全量（小写）
}

// PHPExtensionTemplate 扩展模板包（一键勾选一组扩展）。
type PHPExtensionTemplate struct {
	Name       string   `json:"name"`
	Desc       string   `json:"desc"`
	Extensions []string `json:"extensions"`
}

var phpExtTemplates = []PHPExtensionTemplate{
	{Name: "通用 Web", Desc: "常见 Web 应用基础扩展", Extensions: []string{"opcache", "bcmath", "gd", "zip", "intl", "sockets"}},
	{Name: "WordPress", Desc: "WordPress 推荐全家桶", Extensions: []string{"opcache", "imagick", "gd", "zip", "bcmath", "intl", "exif", "gettext", "mysqli", "pdo_mysql"}},
	{Name: "Laravel", Desc: "Laravel 推荐扩展", Extensions: []string{"opcache", "bcmath", "gd", "zip", "intl", "pcntl", "redis", "exif", "pdo_mysql"}},
	{Name: "高性能缓存", Desc: "OPcache + Redis/Memcached", Extensions: []string{"opcache", "redis", "memcached"}},
	{Name: "调试开发", Desc: "Xdebug 调试（仅开发环境）", Extensions: []string{"xdebug"}},
}

// PHPExtensionCatalogResp 扩展目录响应（目录 + 模板包）。
type PHPExtensionCatalogResp struct {
	Catalog   []PHPExtensionItem     `json:"catalog"`
	Templates []PHPExtensionTemplate `json:"templates"`
}

// PHPExtensionCatalog 扩展目录（创建向导用，不含安装态）。
func (s *RuntimeService) PHPExtensionCatalog() *PHPExtensionCatalogResp {
	items := make([]PHPExtensionItem, 0, len(phpExtCatalog))
	for _, e := range phpExtCatalog {
		items = append(items, PHPExtensionItem{Name: e.Name, Desc: e.Desc, Installed: false})
	}
	return &PHPExtensionCatalogResp{Catalog: items, Templates: phpExtTemplates}
}

// PHPExtensions 扩展清单（php -m 实测 + 面板目录比对）。
func (s *RuntimeService) PHPExtensions(ctx context.Context, id uint) (*PHPExtensionsResp, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持面板扩展管理")
	}
	out, err := s.exec(ctx, 30, "docker exec -i %s php -m", containerNameOr(row))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "php -m 失败（容器未运行？）: "+tailOutput(out.Output, 300))
	}
	installedSet := map[string]bool{}
	installed := make([]string, 0, 32)
	for _, line := range strings.Split(out.Output, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasPrefix(name, "[") {
			continue
		}
		lower := strings.ToLower(name)
		if !installedSet[lower] {
			installedSet[lower] = true
			installed = append(installed, lower)
		}
	}
	resp := &PHPExtensionsResp{Catalog: make([]PHPExtensionItem, 0, len(phpExtCatalog)), Installed: installed}
	for _, e := range phpExtCatalog {
		resp.Catalog = append(resp.Catalog, PHPExtensionItem{
			Name: e.Name, Desc: e.Desc, Installed: installedSet[strings.ToLower(e.Check)],
		})
	}
	return resp, nil
}

// extTaskRunning 同一运行时是否有扩展任务进行中。
func (s *RuntimeService) extTaskRunning(id uint) bool {
	var count int64
	_ = s.db.Model(&model.AppTask{}).
		Where("type IN ? AND ref = ? AND status = ?", []string{TaskRuntimeExtInstall, TaskRuntimeExtUninstall}, fmt.Sprintf("runtime-%d", id), TaskRunning).
		Count(&count).Error
	return count > 0
}

// InstallPHPExtension 在线安装扩展（install-ext + docker commit 持久化 + 重启）。
func (s *RuntimeService) InstallPHPExtension(ctx context.Context, id uint, name string) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持面板扩展管理")
	}
	if _, err := normalizePHPExtensions([]string{name}); err != nil {
		return nil, err
	}
	if s.extTaskRunning(id) {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "该运行环境已有扩展任务进行中，请稍后再试")
	}
	task, err := s.tasks.StartTask(TaskRuntimeExtInstall, fmt.Sprintf("安装 PHP 扩展 %s（%s）", name, row.Name), fmt.Sprintf("runtime-%d", id), 20*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			logf("info", "编译安装扩展 %s", name)
			out, err := s.exec(tctx, 1200, "docker exec -i %s install-ext %s", containerNameOr(row), name)
			if err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "扩展安装失败: "+tailOutput(out.Output, 600))
			}
			logf("info", "提交镜像层（持久化扩展）")
			if out, err = s.exec(tctx, 600, "docker commit %s %s", containerNameOr(row), row.Image); err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "镜像提交失败: "+tailOutput(out.Output, 600))
			}
			logf("info", "重启容器使扩展生效")
			if out, err = s.exec(tctx, 300, "docker restart %s", containerNameOr(row)); err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "容器重启失败: "+tailOutput(out.Output, 600))
			}
			env := envJSON(row)
			exists := false
			for _, e := range env.Extensions {
				if e == name {
					exists = true
					break
				}
			}
			if !exists {
				env.Extensions = append(env.Extensions, name)
			}
			if err := s.saveEnv(row, env); err != nil {
				return err
			}
			logf("info", "扩展 %s 安装完成", name)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// UninstallPHPExtension 卸载扩展（删 conf.d ini + commit + 重启）。
func (s *RuntimeService) UninstallPHPExtension(ctx context.Context, id uint, name string) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持面板扩展管理")
	}
	if _, err := normalizePHPExtensions([]string{name}); err != nil {
		return nil, err
	}
	if s.extTaskRunning(id) {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "该运行环境已有扩展任务进行中，请稍后再试")
	}
	task, err := s.tasks.StartTask(TaskRuntimeExtUninstall, fmt.Sprintf("卸载 PHP 扩展 %s（%s）", name, row.Name), fmt.Sprintf("runtime-%d", id), 10*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			// ext 标识符已被 normalizePHPExtensions 白名单校验，可安全进 shell
			logf("info", "移除扩展加载项 %s", name)
			out, err := s.exec(tctx, 60, "docker exec -i %s sh -c 'rm -f /usr/local/etc/php/conf.d/docker-php-ext-%s.ini'", containerNameOr(row), name)
			if err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "移除扩展失败: "+tailOutput(out.Output, 600))
			}
			logf("info", "提交镜像层")
			if out, err = s.exec(tctx, 600, "docker commit %s %s", containerNameOr(row), row.Image); err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "镜像提交失败: "+tailOutput(out.Output, 600))
			}
			logf("info", "重启容器")
			if out, err = s.exec(tctx, 300, "docker restart %s", containerNameOr(row)); err != nil {
				return err
			}
			if out.ExitCode != 0 {
				return errs.Wrapc(errs.CodeFileOpFailed, "容器重启失败: "+tailOutput(out.Output, 600))
			}
			env := envJSON(row)
			kept := env.Extensions[:0]
			for _, e := range env.Extensions {
				if e != name {
					kept = append(kept, e)
				}
			}
			env.Extensions = kept
			if err := s.saveEnv(row, env); err != nil {
				return err
			}
			logf("info", "扩展 %s 已卸载", name)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

func (s *RuntimeService) saveEnv(row *model.Runtime, env runtimeEnv) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	row.EnvJSON = string(b)
	return s.db.Save(row).Error
}

// ---- php.ini 快捷设置 ----

// PHPConfig php.ini 快捷设置视图。
type PHPConfig struct {
	MemoryLimit      string   `json:"memoryLimit"`
	UploadMaxSize    string   `json:"uploadMaxSize"`
	MaxExecutionTime string   `json:"maxExecutionTime"`
	DisableFunctions []string `json:"disableFunctions"`
}

var (
	phpSizeRe  = regexp.MustCompile(`^\d{1,5}[KMG]?$`)
	phpIntRe   = regexp.MustCompile(`^\d{1,6}$`)
	phpIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
)

// phpIniPath 运行时 php.ini 远端路径。
func (s *RuntimeService) phpIniPath(row *model.Runtime) string {
	return s.dir(row) + "/conf/php.ini"
}

// GetPHPConfig 读取快捷设置。
func (s *RuntimeService) GetPHPConfig(ctx context.Context, id uint) (*PHPConfig, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	content, err := s.readRemote(ctx, s.phpIniPath(row))
	if err != nil {
		return nil, err
	}
	vals := parseIniKeys(content, []string{"memory_limit", "upload_max_filesize", "max_execution_time", "disable_functions"})
	cfg := &PHPConfig{
		MemoryLimit:      vals["memory_limit"],
		UploadMaxSize:    vals["upload_max_filesize"],
		MaxExecutionTime: vals["max_execution_time"],
		DisableFunctions: []string{},
	}
	if vals["disable_functions"] != "" {
		for _, f := range strings.Split(vals["disable_functions"], ",") {
			if f = strings.TrimSpace(f); f != "" {
				cfg.DisableFunctions = append(cfg.DisableFunctions, f)
			}
		}
	}
	return cfg, nil
}

// PHPConfigUpdate 快捷设置更新（nil = 不修改）。
type PHPConfigUpdate struct {
	MemoryLimit      *string   `json:"memoryLimit"`
	UploadMaxSize    *string   `json:"uploadMaxSize"`
	MaxExecutionTime *string   `json:"maxExecutionTime"`
	DisableFunctions *[]string `json:"disableFunctions"`
}

// UpdatePHPConfig 更新快捷设置（写 php.ini + 重启；失败回滚）。
func (s *RuntimeService) UpdatePHPConfig(ctx context.Context, id uint, in PHPConfigUpdate) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持面板配置管理")
	}
	kv := map[string]string{}
	if in.MemoryLimit != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.MemoryLimit))
		if !phpSizeRe.MatchString(v) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "memory_limit 格式不合法（如 256M / 1G）")
		}
		kv["memory_limit"] = v
	}
	if in.UploadMaxSize != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.UploadMaxSize))
		if !phpSizeRe.MatchString(v) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "上传上限格式不合法（如 50M / 1G）")
		}
		kv["upload_max_filesize"] = v
		kv["post_max_size"] = v
	}
	if in.MaxExecutionTime != nil {
		v := strings.TrimSpace(*in.MaxExecutionTime)
		if !phpIntRe.MatchString(v) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "执行超时需为正整数秒")
		}
		kv["max_execution_time"] = v
		kv["max_input_time"] = v
	}
	if in.DisableFunctions != nil {
		fns := make([]string, 0, len(*in.DisableFunctions))
		for _, f := range *in.DisableFunctions {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if !phpIdentRe.MatchString(f) {
				return errs.New(errs.CodeBadRequest, "error.badRequest", "禁用函数名不合法: "+f)
			}
			fns = append(fns, f)
		}
		kv["disable_functions"] = strings.Join(fns, ",")
	}
	if len(kv) == 0 {
		return nil
	}
	path := s.phpIniPath(row)
	content, err := s.readRemote(ctx, path)
	if err != nil {
		return err
	}
	updated := replaceIniKeys(content, kv)
	if err := s.writeRemote(ctx, path, updated); err != nil {
		return err
	}
	if err := s.restartRuntime(ctx, row); err != nil {
		_ = s.writeRemote(ctx, path, content)
		return err
	}
	return nil
}

// restartRuntime 重启运行时容器。
func (s *RuntimeService) restartRuntime(ctx context.Context, row *model.Runtime) error {
	out, err := s.exec(ctx, 300, "docker restart %s", containerNameOr(row))
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "容器重启失败: "+tailOutput(out.Output, 400))
	}
	return nil
}

// parseIniKeys 从 ini 文本提取指定键的值（跳过注释行）。
func parseIniKeys(content string, keys []string) map[string]string {
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	out := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if want[k] {
			out[k] = strings.TrimSpace(v)
		}
	}
	return out
}

// replaceIniKeys 行级替换 ini 键值；文件中不存在的键追加到末尾。
func replaceIniKeys(content string, kv map[string]string) string {
	lines := strings.Split(content, "\n")
	done := map[string]bool{}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, ";") || strings.HasPrefix(trim, "#") {
			continue
		}
		k, _, ok := strings.Cut(trim, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if v, hit := kv[k]; hit {
			lines[i] = fmt.Sprintf("%s = %s", k, v)
			done[k] = true
		}
	}
	if len(done) < len(kv) {
		var extra []string
		for k, v := range kv {
			if !done[k] {
				extra = append(extra, fmt.Sprintf("%s = %s", k, v))
			}
		}
		if len(extra) > 0 {
			content := strings.Join(lines, "\n")
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			return content + strings.Join(extra, "\n") + "\n"
		}
	}
	return strings.Join(lines, "\n")
}

// ---- FPM 进程池配置 ----

// fpmPoolKeys 可在面板修改的 [www] 池参数。
var fpmPoolKeys = []string{"pm", "pm.max_children", "pm.start_servers", "pm.min_spare_servers", "pm.max_spare_servers", "pm.max_requests"}

// GetFPMConfig 读取进程池参数。
func (s *RuntimeService) GetFPMConfig(ctx context.Context, id uint) (map[string]string, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	content, err := s.readRemote(ctx, s.dir(row)+"/conf/php-fpm.conf")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range parseIniKeys(content, fpmPoolKeys) {
		out[k] = v
	}
	return out, nil
}

// UpdateFPMConfig 更新进程池参数（写 php-fpm.conf + 重启；失败回滚）。
func (s *RuntimeService) UpdateFPMConfig(ctx context.Context, id uint, params map[string]string) error {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return err
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 不支持面板配置管理")
	}
	allowed := map[string]bool{}
	for _, k := range fpmPoolKeys {
		allowed[k] = true
	}
	kv := map[string]string{}
	for k, v := range params {
		if !allowed[k] {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的 FPM 参数: "+k)
		}
		v = strings.TrimSpace(v)
		if k == "pm" {
			if v != "dynamic" && v != "static" && v != "ondemand" {
				return errs.New(errs.CodeBadRequest, "error.badRequest", "pm 仅支持 dynamic/static/ondemand")
			}
		} else if !phpIntRe.MatchString(v) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", k+" 需为正整数")
		}
		kv[k] = v
	}
	if len(kv) == 0 {
		return nil
	}
	path := s.dir(row) + "/conf/php-fpm.conf"
	content, err := s.readRemote(ctx, path)
	if err != nil {
		return err
	}
	if err := s.writeRemote(ctx, path, replaceIniKeys(content, kv)); err != nil {
		return err
	}
	if err := s.restartRuntime(ctx, row); err != nil {
		_ = s.writeRemote(ctx, path, content)
		return err
	}
	return nil
}

// FPMStatus FPM 状态页（agent FastCGI 直拨容器 pm.status_path）。
func (s *RuntimeService) FPMStatus(ctx context.Context, id uint) (*dto.FpmStatusResp, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部接管的 PHP 请直接访问其自带状态页")
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	q := url.Values{"container": []string{containerNameOr(row)}}
	resp, err := agentclient.GetJSON[dto.FpmStatusResp](ac, ctx, "/agent/v1/runtime/php/fpm-status?"+q.Encode())
	if err != nil {
		return nil, err
	}
	if resp.Items == nil {
		resp.Items = []dto.FpmStatusItem{}
	}
	return resp, nil
}
