// 商店可复用中间件与可选伴生服务（M68）。
// 声明进 index.json 版本项（middleware / optionalServices），安装向导呈现「中间件复用」与
// 「可选组件」；复用只查找注入不创建——数据库域类型（s3/neo4j）走纳管实例，registry 类型走
// MiddlewareInstance；无实例时回退应用自带参数由用户手填。
package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// StoreMiddlewareDecl 版本中间件复用声明。
type StoreMiddlewareDecl struct {
	Type       string `json:"type"`            // s3 / neo4j（数据库域）或 middlewareRegistry 注册类型（rabbitmq/memcached/…）
	Prefix     string `json:"prefix"`          // env 键前缀（如 S3、RABBITMQ）；注入时填充该前缀下应用实际引用的键
	Required   bool   `json:"required"`        // 向导标注必选（仍可回退手填）
	Label      string `json:"label"`           // 向导展示名（缺省用类型 label）
	MinVersion string `json:"minVersion"`      // 最低可复用版本（缺省用类型基线）；低于则候选禁选
	Endpoint   string `json:"endpoint"`        // 可选：地址注入形态提示（uri=单键 URI；缺省 host/port 分键）
}

// StoreOptionalSvc 版本可选伴生服务声明。
type StoreOptionalSvc struct {
	ID     string            `json:"id"`     // 勾选标识
	Label  string            `json:"label"`  // 向导展示名
	Compose string           `json:"compose"` // 包内附加 compose 文件（相对包根，安装时合并进主文件）
	Env    map[string]string `json:"env"`    // 勾选后注入的开关 env（如 PLATFORM_X_ENABLED=true）
}

// middlewareDeclsWithDefaults 声明规范化：label/minVersion 缺省回退类型描述符。
func middlewareDeclsWithDefaults(decls []StoreMiddlewareDecl) []StoreMiddlewareDecl {
	out := make([]StoreMiddlewareDecl, 0, len(decls))
	for _, d := range decls {
		if d.Type == "" || d.Prefix == "" {
			continue
		}
		d.Type = strings.ToLower(strings.TrimSpace(d.Type))
		d.Prefix = strings.ToUpper(strings.TrimSpace(d.Prefix))
		if d.Label == "" {
			if desc, ok := middlewareDescriptorOf(d.Type); ok {
				d.Label = desc.Label
			} else if d.Type == "s3" {
				d.Label = "对象存储(S3)"
			} else if d.Type == "neo4j" {
				d.Label = "Neo4j"
			} else {
				d.Label = d.Type
			}
		}
		if d.MinVersion == "" {
			if desc, ok := middlewareDescriptorOf(d.Type); ok {
				d.MinVersion = desc.MinReuseVersion
			}
		}
		out = append(out, d)
	}
	return out
}

// applyExternalMiddleware registry 类型中间件复用注入：按声明前缀填充应用实际引用的 env 键。
// 键值语义按后缀映射：HOST/ADDR/ENDPOINT/URI → 地址（容器名/IP），PORT → 端口，
// USER/USERNAME → 用户，PASSWORD/PASS → 密码（实例凭据 reveal，不落表单）。
func (s *StoreService) applyExternalMiddleware(ctx context.Context, logf TaskLogf, decl StoreMiddlewareDecl, instanceID uint, in StoreInstallInput, params map[string]string, composeText string, formFields []StoreFormField) error {
	inst, err := s.mws.ByID(instanceID)
	if err != nil {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("中间件实例不存在（%s）", decl.Prefix))
	}
	if inst.Type != decl.Type {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("%s 键集需选择 %s 实例，实例 %s 类型为 %s", decl.Prefix, decl.Type, inst.Name, inst.Type))
	}
	if !middlewareVersionOK(inst.Version, decl.MinVersion) {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("实例 %s 版本 %s 低于最低可复用版本 %s", inst.Name, inst.Version, decl.MinVersion))
	}
	pass, err := s.mws.credentialOf(inst)
	if err != nil {
		return err
	}
	port := fmt.Sprint(middlewarePortOf(inst.Type, inst.Port))
	host := inst.Host
	if inst.Origin == "container" && inst.ContainerName != "" {
		host = inst.ContainerName // 同网络容器名直连（商店应用统一接入 PanelNetwork）
		logf("info", "中间件 %s（%s）为容器实例，连接地址使用 %s:%s（容器名直连）", inst.Name, inst.Type, host, port)
	}
	// 收集该前缀下应用实际引用的键（表单字段 + compose 变量引用）
	declared := map[string]bool{}
	for _, f := range formFields {
		if f.EnvKey != "" && strings.HasPrefix(strings.ToUpper(f.EnvKey), decl.Prefix+"_") {
			declared[strings.ToUpper(f.EnvKey)] = true
		}
	}
	for _, m := range composeVarPattern.FindAllStringSubmatch(composeText, -1) {
		if len(m) >= 2 && strings.HasPrefix(strings.ToUpper(m[1]), decl.Prefix+"_") {
			declared[strings.ToUpper(m[1])] = true
		}
	}
	if len(declared) == 0 {
		declared[decl.Prefix+"_HOST"] = true
	}
	injected := 0
	for k := range declared {
		stem := strings.TrimPrefix(k, decl.Prefix+"_")
		switch {
		case stem == "PORT":
			params[k] = port
			injected++
		case stem == "USER" || stem == "USERNAME":
			if inst.User != "" {
				params[k] = inst.User
				injected++
			}
		case stem == "PASSWORD" || stem == "PASS" || stem == "SECRET":
			if pass != "" {
				params[k] = pass
				injected++
			}
		case stem == "HOST" || stem == "ADDR" || stem == "ADDRESS":
			params[k] = host
			injected++
		case stem == "ENDPOINT":
			params[k] = "http://" + host + ":" + port
			injected++
		}
	}
	logf("info", "已复用中间件实例 %s（%s，版本 %s）注入 %s 前缀 %d 个参数", inst.Name, inst.Type, inst.Version, decl.Prefix, injected)
	return nil
}

// composeVarPattern compose 文本变量引用（${VAR}）提取。
var composeVarPattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)

// ApplyExternalDeclarations M68 声明注入主入口：数据库域（s3/neo4j）+ registry 类型 + optionalServices env。
// composeText 为项目目录主 compose 内容（键集收集用）。
func (s *StoreService) ApplyExternalDeclarations(ctx context.Context, logf TaskLogf, ver StoreVersion, in StoreInstallInput, params map[string]string, composeText string) error {
	ext := in.ExternalDB
	if ext == nil {
		return nil
	}
	decls := middlewareDeclsWithDefaults(ver.Middleware)
	for _, d := range decls {
		switch d.Type {
		case "s3":
			id := ext.S3InstanceID
			if id == 0 {
				continue
			}
			if err := s.applyExternalS3(ctx, logf, d, id, in, params); err != nil {
				return err
			}
		case "neo4j":
			id := ext.Neo4jInstanceID
			if id == 0 {
				continue
			}
			if err := s.applyExternalNeo4j(ctx, logf, d, id, in, params); err != nil {
				return err
			}
		default:
			id := uint(0)
			if ext.MiddlewareIDs != nil {
				id = ext.MiddlewareIDs[d.Type]
			}
			if id == 0 {
				continue
			}
			if err := s.applyExternalMiddleware(ctx, logf, d, id, in, params, composeText, ver.FormFields); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyExternalS3 数据库域 S3 复用：三层地址 → 建桶 → 注入前缀键集。
func (s *StoreService) applyExternalS3(ctx context.Context, logf TaskLogf, decl StoreMiddlewareDecl, instanceID uint, in StoreInstallInput, params map[string]string) error {
	host, port, inst, extraHosts, err := s.dbEndpoint(ctx, logf, instanceID, in)
	if err != nil {
		return err
	}
	if inst.Type != "s3" {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("S3 键集需选择对象存储(s3)实例，实例 %s 类型为 %s", inst.Name, inst.Type))
	}
	secret, rerr := s.dbs.Reveal(instanceID)
	if rerr != nil {
		return rerr
	}
	pwd, _ := secret["password"].(string)
	bucket := dbIdentSanitize(in.Key) + "-app"
	// 建桶幂等：已存在跳过（复用既有 bucket 语义对齐建库）
	drv, err := dbdriver.NewS3(inst.Host, inst.Port, inst.RootUser, pwd, inst.Secure)
	if err != nil {
		return errs.Wrapc(errs.CodeInternal, "连接对象存储实例失败: "+err.Error())
	}
	defer drv.Close()
	if err := drv.CreateDatabase(ctx, bucket, ""); err != nil {
		if !strings.Contains(err.Error(), "already") {
			logf("warn", "建桶 %s 未完成（可能已存在）: %s", bucket, err.Error())
		}
	} else {
		logf("info", "已在实例 %s 创建 bucket %s", inst.Name, bucket)
	}
	scheme := "http"
	if inst.Secure {
		scheme = "https"
	}
	endpoint := scheme + "://" + host + ":" + port
	injected := 0
	put := func(k, v string) {
		if k != "" {
			params[k] = v
			injected++
		}
	}
	stem := decl.Prefix + "_"
	put(stem+"ENDPOINT", endpoint)
	put(stem+"HOST", host)
	put(stem+"PORT", port)
	put(stem+"ACCESS_KEY", inst.RootUser)
	put(stem+"ACCESSKEY", inst.RootUser)
	put(stem+"SECRET_KEY", pwd)
	put(stem+"SECRETKEY", pwd)
	put(stem+"BUCKET", bucket)
	put(stem+"REGION", "us-east-1")
	put(stem+"PATH_STYLE_ACCESS", "true")
	_ = extraHosts
	logf("info", "已复用对象存储实例 %s（endpoint %s，bucket %s）注入 %s 前缀 %d 个参数", inst.Name, endpoint, bucket, decl.Prefix, injected)
	return nil
}

// applyExternalNeo4j 数据库域 Neo4j 复用：注入 bolt URI 与凭据（连接校验在纳管时已做）。
func (s *StoreService) applyExternalNeo4j(ctx context.Context, logf TaskLogf, decl StoreMiddlewareDecl, instanceID uint, in StoreInstallInput, params map[string]string) error {
	host, port, inst, _, err := s.dbEndpoint(ctx, logf, instanceID, in)
	if err != nil {
		return err
	}
	if inst.Type != "neo4j" {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("Neo4j 键集需选择 Neo4j 实例，实例 %s 类型为 %s", inst.Name, inst.Type))
	}
	secret, rerr := s.dbs.Reveal(instanceID)
	if rerr != nil {
		return rerr
	}
	pwd, _ := secret["password"].(string)
	stem := decl.Prefix + "_"
	params[stem+"URI"] = "bolt://" + host + ":" + port
	params[stem+"HOST"] = host
	params[stem+"PORT"] = port
	params[stem+"USERNAME"] = inst.RootUser
	params[stem+"USER"] = inst.RootUser
	params[stem+"PASSWORD"] = pwd
	logf("info", "已复用 Neo4j 实例 %s（bolt://%s:%s）注入 %s 前缀参数", inst.Name, host, port, decl.Prefix)
	return nil
}

// OptionalServiceEnv 勾选的 optionalServices 的开关 env 合并进安装参数（.env 渲染前调用）。
func optionalServiceEnv(ver StoreVersion, selected []string, params map[string]string) []string {
	if len(selected) == 0 {
		return nil
	}
	applied := make([]string, 0, len(selected))
	for _, svc := range ver.OptionalServices {
		hit := false
		for _, id := range selected {
			if id == svc.ID {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		applied = append(applied, svc.ID)
		keys := make([]string, 0, len(svc.Env))
		for k := range svc.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			params[k] = svc.Env[k]
		}
	}
	return applied
}

// MergeOptionalCompose 把勾选的 optional compose 文件合并进项目主 docker-compose.yml
// （services/volumes/networks 段深合并；主文件键优先）。合并后 up/ps/logs/init 全链零特判。
func (s *StoreService) MergeOptionalCompose(ctx context.Context, ac *agentclient.Client, dir string, ver StoreVersion, selected []string) error {
	if len(selected) == 0 {
		return nil
	}
	files := make([]string, 0, len(selected))
	for _, svc := range ver.OptionalServices {
		hit := false
		for _, id := range selected {
			if id == svc.ID {
				hit = true
				break
			}
		}
		if hit && svc.Compose != "" {
			files = append(files, svc.Compose)
		}
	}
	if len(files) == 0 {
		return nil
	}
	main, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(dir+"/docker-compose.yml"))
	if err != nil || main == nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "读取主 compose 失败: "+fmt.Sprint(err))
	}
	var base map[string]any
	if err := yaml.Unmarshal([]byte(main.Content), &base); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "解析主 compose 失败: "+err.Error())
	}
	for _, f := range files {
		res, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?path="+escapeURL2(dir+"/"+f))
		if err != nil || res == nil {
			return errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("读取可选组件 compose %s 失败: %v", f, err))
		}
		var extra map[string]any
		if err := yaml.Unmarshal([]byte(res.Content), &extra); err != nil {
			return errs.Wrapc(errs.CodeFileOpFailed, "解析 "+f+" 失败: "+err.Error())
		}
		for _, section := range []string{"services", "volumes", "networks", "secrets", "configs"} {
			em, ok := extra[section].(map[string]any)
			if !ok || len(em) == 0 {
				continue
			}
			bm, _ := base[section].(map[string]any)
			if bm == nil {
				bm = map[string]any{}
			}
			for k, v := range em {
				if _, exists := bm[k]; !exists {
					bm[k] = v
				}
			}
			base[section] = bm
		}
	}
	out, err := yaml.Marshal(base)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "序列化合并 compose 失败: "+err.Error())
	}
	if err := s.writeViaAgent(ctx, ac, dir+"/docker-compose.yml", string(out)); err != nil {
		return err
	}
	return nil
}

// ---- 键集型包装（dbFieldSetsOf 检测出的 s3/neo4j 键集 → 前缀复用声明注入）----

// dbFieldStemMatch 键词干任一子串命中。
func dbFieldStemMatch(stem string, subs ...string) bool {
	up := strings.ToUpper(stem)
	for _, sub := range subs {
		if strings.Contains(up, sub) {
			return true
		}
	}
	return false
}

// prefixOfHostKey 连接锚键 → 注入前缀（S3_HOST → S3；S3_ENDPOINT → S3；NEO4J_URI → NEO4J）。
func prefixOfHostKey(key string) string {
	up := strings.ToUpper(key)
	for _, suffix := range []string{"_HOST", "_ENDPOINT", "_URI"} {
		if i := strings.LastIndex(up, suffix); i > 0 {
			return up[:i]
		}
	}
	return up
}

// applyExternalS3KeySet 键集型 S3 注入（fs.Host 形态）。
func (s *StoreService) applyExternalS3KeySet(ctx context.Context, logf TaskLogf, fs dbFieldSet, instanceID uint, in StoreInstallInput, params map[string]string) error {
	return s.applyExternalS3(ctx, logf, StoreMiddlewareDecl{Prefix: prefixOfHostKey(fs.Host)}, instanceID, in, params)
}

// applyExternalNeo4jKeySet 键集型 Neo4j 注入（fs.Host 形态）。
func (s *StoreService) applyExternalNeo4jKeySet(ctx context.Context, logf TaskLogf, fs dbFieldSet, instanceID uint, in StoreInstallInput, params map[string]string) error {
	return s.applyExternalNeo4j(ctx, logf, StoreMiddlewareDecl{Prefix: prefixOfHostKey(fs.Host)}, instanceID, in, params)
}

// ensure model 引用不悬空（MiddlewareInstance 由 middleware.go 域使用）。
var _ = model.MiddlewareInstance{}
