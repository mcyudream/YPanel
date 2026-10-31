package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
)

func parseStoreEnv(raw string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
			v = strings.ReplaceAll(v[1:len(v)-1], `\'`, "'")
		} else if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			if decoded, err := strconv.Unquote(v); err == nil {
				v = strings.ReplaceAll(decoded, "$$", "$")
			}
		}
		values[k] = v
	}
	return values
}

func quoteStoreEnv(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "$", "$$")
	return `"` + value + `"`
}

func mergeStoreKeyValues(raw any, additions map[string]any) (map[string]any, error) {
	values := map[string]any{}
	switch old := raw.(type) {
	case nil:
	case map[string]any:
		for k, v := range old {
			values[k] = v
		}
	case []any:
		for _, item := range old {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("应用包的 labels/environment 格式不合法")
			}
			k, v, hasValue := strings.Cut(text, "=")
			if hasValue {
				values[k] = v
			} else {
				values[k] = nil
			}
		}
	default:
		return nil, fmt.Errorf("应用包的 labels/environment 格式不合法")
	}
	for k, v := range additions {
		values[k] = v
	}
	return values, nil
}

func storeInstallQuery(db *gorm.DB, project, node string) *gorm.DB {
	q := db.Where("compose_project = ?", project)
	if normalizeNodeID(node) == "local" {
		return q.Where("node_id IN ?", []string{"", "local"})
	}
	return q.Where("node_id = ?", normalizeNodeID(node))
}

func storeOwnerAllowed(ctx context.Context, inst model.AppStoreInstall) bool {
	caller, ok := rbac.CallerFrom(ctx)
	return !ok || !caller.Assigned() || inst.OwnerID == 0 || inst.OwnerID == caller.UserID
}

// restoreStoreInstall 从目标节点的真实 .env 沿用凭据；原文保留，避免重新插值或改写密码。
func (s *StoreService) restoreStoreInstall(ctx context.Context, project string, ver StoreVersion, in *StoreInstallInput) error {
	in.providedParams = map[string]bool{}
	for k, v := range in.Params {
		if stringifyParam(v) != "" && stringifyParam(v) != "******" {
			in.providedParams[k] = true
		}
	}
	var inst model.AppStoreInstall
	if err := storeInstallQuery(s.db, project, in.NodeID).Limit(1).Find(&inst).Error; err != nil {
		return err
	}
	if inst.ID == 0 {
		return nil
	}
	if !storeOwnerAllowed(ctx, inst) {
		return errs.ErrNotFound
	}
	if inst.Key != in.Key || inst.SourceID != in.SourceID {
		return fmt.Errorf("同名项目已属于其他应用，请使用新的实例名")
	}
	in.existing = &inst
	params := map[string]any{}
	if inst.ParamsJSON != "" {
		if err := json.Unmarshal([]byte(inst.ParamsJSON), &params); err != nil {
			return fmt.Errorf("原安装参数无法读取，已中止部署")
		}
	}
	ac, err := s.clientFor(inst.NodeID)
	if err != nil {
		return err
	}
	if inst.Version != "" {
		var app model.AppStoreApp
		if err := s.db.Select("kind").Where("source_id = ? AND key = ?", inst.SourceID, inst.Key).First(&app).Error; err != nil {
			return err
		}
		if app.Kind != "php" {
			env, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2("/opt/ypanel/compose/"+project+"/.env"))
			if err != nil || env.Truncated {
				return fmt.Errorf("无法完整读取原应用 .env，已中止部署以保留原参数")
			}
			in.rawEnv = env.Content
			for k, v := range parseStoreEnv(env.Content) {
				params[k] = v
			}
			var options storeInstallOptions
			if inst.OptionsJSON != "" {
				if err := json.Unmarshal([]byte(inst.OptionsJSON), &options); err != nil {
					return fmt.Errorf("原安装选项无法读取")
				}
			} else {
				cfg, err := readStoreComposeConfig(ctx, ac, "/opt/ypanel/compose/"+project, project)
				if err != nil {
					return err
				}
				options = storeOptionsFromConfig(cfg, ver)
			}
			if in.Network == "" {
				in.Network = options.Network
			}
			if in.Timezone == "" {
				in.Timezone = options.Timezone
			}
			if in.ExtraHosts == nil {
				in.ExtraHosts = options.ExtraHosts
			}
			if in.upgrade {
				in.MountHostsFile = options.MountHostsFile
			}
			if in.ExternalDB == nil {
				in.ExternalDB = &StoreExternalDB{OptionalServices: options.OptionalServices}
			}
		}
	}
	for k, v := range in.Params {
		if in.providedParams[k] {
			params[k] = v
		}
	}
	in.Params = params
	return nil
}

func storeOptionsFromConfig(cfg *storeComposeConfig, ver StoreVersion) storeInstallOptions {
	o := storeInstallOptions{}
	networkKeys := make([]string, 0, len(cfg.Networks))
	for key := range cfg.Networks {
		networkKeys = append(networkKeys, key)
	}
	sort.Strings(networkKeys)
	for _, key := range networkKeys {
		n := cfg.Networks[key]
		if n.External {
			o.Network = n.Name
			if n.Name == PanelNetwork {
				break
			}
		}
	}
	hosts := map[string]bool{}
	for _, svc := range cfg.Services {
		if svc.NetworkMode == "host" {
			o.Network = "host"
		}
		if svc.Environment["TZ"] != "" {
			o.Timezone = svc.Environment["TZ"]
		}
		for _, host := range svc.ExtraHosts {
			hosts[strings.Replace(host, "=", ":", 1)] = true
		}
		for _, v := range svc.Volumes {
			if v.Source == "/etc/hosts" && v.Target == "/etc/hosts" {
				o.MountHostsFile = true
			}
		}
	}
	for h := range hosts {
		o.ExtraHosts = append(o.ExtraHosts, h)
	}
	sort.Strings(o.ExtraHosts)
	for _, opt := range ver.OptionalServices {
		if _, ok := cfg.Services[opt.ID]; ok {
			o.OptionalServices = append(o.OptionalServices, opt.ID)
		}
	}
	return o
}

// StoreVolumeRecovery 在旧容器缺少挂载时恢复持久化文件；路径均由商店包显式声明。
type StoreVolumeRecovery struct {
	Service string `json:"service"`
	Source  string `json:"source"`
	Target  string `json:"target"`
}

type storeInstallOptions struct {
	Network          string   `json:"network"`
	Timezone         string   `json:"timezone"`
	ExtraHosts       []string `json:"extraHosts"`
	MountHostsFile   bool     `json:"mountHostsFile"`
	OptionalServices []string `json:"optionalServices"`
}

type storeComposeConfig struct {
	Services map[string]struct {
		ContainerName string                                  `json:"container_name"`
		NetworkMode   string                                  `json:"network_mode"`
		Environment   map[string]string                       `json:"environment"`
		ExtraHosts    []string                                `json:"extra_hosts"`
		Volumes       []struct{ Type, Source, Target string } `json:"volumes"`
		Ports         []struct {
			Published string `json:"published"`
		} `json:"ports"`
	} `json:"services"`
	Networks map[string]struct {
		Name     string
		External bool
	} `json:"networks"`
}

func storeExec(ctx context.Context, ac *agentclient.Client, command string, seconds int) (string, error) {
	r, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec", &dto.ExecReq{Command: command, TimeoutSecs: seconds})
	if err != nil {
		return "", err
	}
	if r.TimedOut {
		return "", fmt.Errorf("应用部署命令超时（%d 秒）", seconds)
	}
	if r.ExitCode != 0 {
		return "", fmt.Errorf("应用部署命令失败（exit %d）: %s", r.ExitCode, tailOutput(r.Output, 800))
	}
	return r.Output, nil
}

func readStoreComposeConfig(ctx context.Context, ac *agentclient.Client, dir, project string) (*storeComposeConfig, error) {
	out, err := storeExec(ctx, ac, "cd "+shellQuote(dir)+" && docker compose -p "+shellQuote(project)+" config --format json", 60)
	if err != nil {
		return nil, err
	}
	var cfg storeComposeConfig
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return nil, fmt.Errorf("无法解析应用编排配置")
	}
	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("应用编排中没有服务")
	}
	return &cfg, nil
}

func safeStoreRelative(p string) bool {
	return p != "" && p != "." && !path.IsAbs(p) && path.Clean(p) == p && p != ".." && !strings.HasPrefix(p, "../") && !strings.ContainsAny(p, "\x00\r\n")
}

// 配置回滚只覆盖本次替换的文件；挂载数据与目录内的其他用户文件始终保留。
type storeDeployment struct {
	actual, stage, backup, project string
	files                          []string
	hadOld                         bool
	started                        bool
}

func newStoreDeployment(ctx context.Context, ac *agentclient.Client, project string) (*storeDeployment, error) {
	d := &storeDeployment{actual: "/opt/ypanel/compose/" + project, project: project}
	d.stage = "/opt/ypanel/compose/.prepare-" + project + "-" + randomHex(6)
	d.backup = d.stage + ".backup"
	_, err := storeExec(ctx, ac, "umask 077; mkdir -p "+shellQuote(d.stage)+" "+shellQuote(d.backup), 30)
	return d, err
}

// protectedStorePaths 同时保护新旧编排的相对 bind 源；named volume 由 Docker 自身保留。
func protectedStorePaths(cfg *storeComposeConfig, root string) ([]string, error) {
	set := map[string]bool{"data": true}
	for _, svc := range cfg.Services {
		for _, vol := range svc.Volumes {
			if vol.Type != "bind" {
				continue
			}
			if vol.Source == root {
				return nil, fmt.Errorf("应用挂载了整个项目目录，需先确认配置文件迁移方式")
			}
			if !strings.HasPrefix(vol.Source, root+"/") {
				continue
			}
			rel := strings.TrimPrefix(vol.Source, root+"/")
			if !safeStoreRelative(rel) {
				return nil, fmt.Errorf("应用持久化路径不合法")
			}
			for _, name := range []string{".env", "docker-compose.yml", "docker-compose.override.yml"} {
				if rel == name {
					return nil, fmt.Errorf("应用将部署配置 %s 用作数据挂载，需先迁移该挂载", name)
				}
			}
			set[rel] = true
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func storePathGuard(root, rel string) string {
	check := func(p string) string {
		return "if [ -L " + shellQuote(p) + " ]; then printf '%s\\n' " + shellQuote("部署路径包含符号链接，已中止: "+p) + "; exit 1; fi\n"
	}
	guard := check(root)
	for p := rel; p != "."; p = path.Dir(p) {
		guard += check(root + "/" + p)
	}
	return guard
}

func storePathProtected(rel string, protected []string) bool {
	for _, keep := range protected {
		if rel == keep || strings.HasPrefix(rel, keep+"/") {
			return true
		}
	}
	return false
}

func (d *storeDeployment) activate(ctx context.Context, ac *agentclient.Client, cfg *storeComposeConfig, recoveries []StoreVolumeRecovery, logf TaskLogf) error {
	protected, err := protectedStorePaths(cfg, d.stage)
	if err != nil {
		return err
	}
	absolutePaths, err := protectedStorePaths(cfg, d.actual)
	if err != nil {
		return err
	}
	protected = append(protected, absolutePaths...)
	probe, err := storeExec(ctx, ac, "if [ -f "+shellQuote(d.actual+"/docker-compose.yml")+" ]; then printf old; fi", 30)
	if err != nil {
		return err
	}
	d.hadOld = probe == "old"
	if d.hadOld {
		old, err := readStoreComposeConfig(ctx, ac, d.actual, d.project)
		if err != nil {
			return fmt.Errorf("原编排配置校验失败，已保留原部署: %w", err)
		}
		paths, err := protectedStorePaths(old, d.actual)
		if err != nil {
			return err
		}
		protected = append(protected, paths...)
	}
	for _, recovery := range recoveries {
		if !src2NameRe.MatchString(recovery.Service) || !safeStoreRelative(recovery.Target) || !path.IsAbs(recovery.Source) || recovery.Source == "/" || path.Clean(recovery.Source) != recovery.Source || strings.ContainsAny(recovery.Source, "\x00\r\n") || !storePathProtected(recovery.Target, protected) {
			return fmt.Errorf("应用包的持久化恢复声明不合法")
		}
		if !d.hadOld {
			continue
		}
		// 已挂载的数据直接保留。缺少挂载的旧容器（包括停止的容器）使用 docker cp 导出；
		// 仅明确的不存在可跳过，权限/daemon/传输等错误一律阻止部署。
		cmd := "set -e\n" + storePathGuard(d.actual, recovery.Target) + storePathGuard(d.stage, recovery.Target) +
			"if [ ! -e " + shellQuote(d.actual+"/"+recovery.Target) + " ]; then\n" +
			"ids=$(docker ps -aq --filter " + shellQuote("label=com.docker.compose.project="+d.project) + " --filter " + shellQuote("label=com.docker.compose.service="+recovery.Service) + ") || exit $?\n" +
			"for id in $ids; do\nmkdir -p " + shellQuote(d.stage+"/"+recovery.Target) + "\n" +
			"if docker cp \"$id\":" + shellQuote(recovery.Source+"/.") + " " + shellQuote(d.stage+"/"+recovery.Target) + " 2>" + shellQuote(d.backup+"/recovery-error") + "; then break; else\n" +
			"if ! grep -Eq 'Could not find the file|No such file or directory|does not exist' " + shellQuote(d.backup+"/recovery-error") + "; then cat " + shellQuote(d.backup+"/recovery-error") + "; exit 1; fi\nfi\ndone\nfi"
		if _, err := storeExec(ctx, ac, cmd, 120); err != nil {
			return fmt.Errorf("恢复旧容器持久化配置失败: %w", err)
		}
	}
	listing, err := storeExec(ctx, ac, "cd "+shellQuote(d.stage)+" && if find . -type l | head -1 | grep -q .; then echo '应用包包含符号链接'; exit 1; fi; find . -type f -print", 30)
	if err != nil {
		return err
	}
	for _, file := range strings.Split(strings.TrimSpace(listing), "\n") {
		rel := strings.TrimPrefix(file, "./")
		if !safeStoreRelative(rel) {
			return fmt.Errorf("应用包文件路径不合法")
		}
		d.files = append(d.files, rel)
	}
	// 即使新配置不再需要 override，旧 override 也应备份并撤下，避免旧配置继续参与合并。
	if !containsStorePath(d.files, "docker-compose.override.yml") {
		d.files = append(d.files, "docker-compose.override.yml")
	}
	var script strings.Builder
	script.WriteString("set -e\numask 077\nmkdir -p " + shellQuote(d.actual) + "\n")
	// 整个挂载源一次性初始化；不列入配置回滚，恢复出的引导配置属于用户数据。
	sort.Strings(protected)
	kept := make([]string, 0, len(protected))
	for _, keep := range protected {
		if storePathProtected(keep, kept) {
			continue
		}
		kept = append(kept, keep)
		target, source := d.actual+"/"+keep, d.stage+"/"+keep
		script.WriteString(storePathGuard(d.actual, keep))
		script.WriteString("if [ -f " + shellQuote(source) + " ] && [ -d " + shellQuote(target) + " ]; then rmdir -- " + shellQuote(target) + "; fi\n")
		script.WriteString("if [ ! -e " + shellQuote(target) + " ] && [ -e " + shellQuote(source) + " ]; then mkdir -p " + shellQuote(path.Dir(target)) + "; cp -a -- " + shellQuote(source) + " " + shellQuote(target) + "; fi\n")
	}
	for _, rel := range d.files {
		if storePathProtected(rel, protected) {
			continue
		}
		target := d.actual + "/" + rel
		source := d.stage + "/" + rel
		script.WriteString(storePathGuard(d.actual, rel))
		script.WriteString("test ! -d " + shellQuote(target) + "\nmkdir -p " + shellQuote(path.Dir(target)) + " " + shellQuote(path.Dir(d.backup+"/old/"+rel)) + " " + shellQuote(path.Dir(d.backup+"/touched/"+rel)) + "\n")
		script.WriteString("if [ -f " + shellQuote(target) + " ]; then cp -p -- " + shellQuote(target) + " " + shellQuote(d.backup+"/old/"+rel) + "; fi\n")
		script.WriteString(": > " + shellQuote(d.backup+"/touched/"+rel) + "\n")
		script.WriteString("if [ -f " + shellQuote(source) + " ]; then cp -p -- " + shellQuote(source) + " " + shellQuote(target) + "; else rm -f -- " + shellQuote(target) + "; fi\n")
	}
	d.started = true
	if _, err = storeExec(ctx, ac, script.String(), 120); err != nil {
		return err
	}
	logf("info", "新包配置已校验；保留数据卷、配置及插件目录，替换部署文件")
	return nil
}

func containsStorePath(paths []string, p string) bool {
	for _, v := range paths {
		if v == p {
			return true
		}
	}
	return false
}

func (d *storeDeployment) finish(ac *agentclient.Client, deployErr *error, logf TaskLogf) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if *deployErr != nil && d.started {
		var script strings.Builder
		script.WriteString("set -e\n")
		if !d.hadOld {
			if _, err := storeExec(ctx, ac, "cd "+shellQuote(d.actual)+" && docker compose -p "+shellQuote(d.project)+" down", 120); err != nil {
				logf("warn", "新建容器清理未完成，继续恢复部署文件；请检查项目残留容器: %s", err)
			}
		}
		for _, rel := range d.files {
			script.WriteString("if [ -f " + shellQuote(d.backup+"/touched/"+rel) + " ]; then\nif [ -f " + shellQuote(d.backup+"/old/"+rel) + " ]; then cp -p -- " + shellQuote(d.backup+"/old/"+rel) + " " + shellQuote(d.actual+"/"+rel) + "; else rm -f -- " + shellQuote(d.actual+"/"+rel) + "; fi\nfi\n")
		}
		if d.hadOld {
			script.WriteString("cd " + shellQuote(d.actual) + " && docker compose -p " + shellQuote(d.project) + " up -d --no-build --pull never --remove-orphans\n")
		}
		if _, err := storeExec(ctx, ac, script.String(), 150); err != nil {
			*deployErr = fmt.Errorf("%w；恢复原部署失败: %v（配置备份保留在 %s）", *deployErr, err, d.backup)
			return
		}
		logf("warn", "部署失败，已恢复原部署配置；数据卷保持原位，应用自身的数据迁移不会自动撤销")
	}
	if _, err := storeExec(ctx, ac, "rm -rf -- "+shellQuote(d.stage)+" "+shellQuote(d.backup), 30); err != nil {
		logf("warn", "临时部署目录清理失败: %s", err)
	}
}

func renderStoreEnv(in StoreInstallInput, ver StoreVersion, params map[string]string) string {
	var envBuf strings.Builder
	envBuf.WriteString(in.rawEnv)
	envBuf.WriteString("\n")
	oldEnv := parseStoreEnv(in.rawEnv)
	if in.providedParams == nil {
		in.providedParams = map[string]bool{}
	}
	for k, v := range params {
		if before, ok := in.Params[k]; !ok || stringifyParam(before) != v {
			in.providedParams[k] = true
		}
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, hasOld := oldEnv[k]
		_, managed := ver.VersionParams[k]
		if hasOld && !in.providedParams[k] && !managed && k != "CONTAINER_NAME" && k != "CONTAINER_NAME1" {
			continue
		}
		envBuf.WriteString(k + "=" + quoteStoreEnv(params[k]) + "\n")
	}
	return envBuf.String()
}
