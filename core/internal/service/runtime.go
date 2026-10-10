// RuntimeService 运行环境服务：容器化运行时（模板构建式，任务中心驱动）+ 本机 fastcgi 接管。
//
// 目录约定：/opt/ypanel/runtime/<type>/<name>/（docker-compose.yml、Dockerfile、conf/、log/）。
// compose 项目名 rt-<type>-<name>，php 容器名 php-<name>（站点 fastcgi_pass 依赖该命名）。
package service

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

//go:embed runtimedata
var runtimeDataFS embed.FS

// 运行环境远端目录与命名。
const (
	runtimeRootDir     = "/opt/ypanel/runtime"
	legacyComposeDir   = "/opt/ypanel/compose"
	runtimeTZ          = "Asia/Shanghai"
	runtimeImagePrefix = "ypanel-rt"
)

// phpVersions 支持的 PHP 版本 → 基础镜像。
var phpVersions = map[string]string{
	"7.4": "php:7.4-fpm-alpine",
	"8.0": "php:8.0-fpm-alpine",
	"8.1": "php:8.1-fpm-alpine",
	"8.2": "php:8.2-fpm-alpine",
	"8.3": "php:8.3-fpm-alpine",
	"8.4": "php:8.4-fpm-alpine",
}

// codeVersions 代码运行时支持版本（type → version → 镜像）。
var codeVersions = map[string]map[string]string{
	"node": {
		"18": "node:18-alpine",
		"20": "node:20-alpine",
		"22": "node:22-alpine",
	},
	"python": {
		"3.10": "python:3.10-alpine",
		"3.11": "python:3.11-alpine",
		"3.12": "python:3.12-alpine",
		"3.13": "python:3.13-alpine",
	},
	"java": {
		"8":  "eclipse-temurin:8-jre-alpine",
		"11": "eclipse-temurin:11-jre-alpine",
		"17": "eclipse-temurin:17-jre-alpine",
		"21": "eclipse-temurin:21-jre-alpine",
	},
	"go": {
		"1.22": "golang:1.22-alpine",
		"1.23": "golang:1.23-alpine",
		"1.24": "golang:1.24-alpine",
	},
}

// codeTypes 代码运行时类型集合。
var codeTypes = map[string]bool{"node": true, "python": true, "java": true, "go": true}

// RuntimePort 代码运行时端口映射。
type RuntimePort struct {
	Host      int    `json:"host"`
	Container int    `json:"container"`
	Protocol  string `json:"protocol,omitempty"` // tcp（默认）/ udp
}

// 运行时状态。
const (
	RuntimeStatusRunning  = "running"
	RuntimeStatusStopped  = "stopped"
	RuntimeStatusBuilding = "building"
	RuntimeStatusCreating = "creating"
	RuntimeStatusError    = "error"
)

// 任务类型。
const (
	TaskRuntimeCreate       = "runtime-create"
	TaskRuntimeExtInstall   = "runtime-ext-install"
	TaskRuntimeExtUninstall = "runtime-ext-uninstall"
)

// runtimeEnv EnvJSON 结构（运行参数）。
type runtimeEnv struct {
	Extensions  []string      `json:"extensions,omitempty"` // PHP 扩展清单（创建勾选 + 热装追加）
	StartCmd    string        `json:"startCmd,omitempty"`   // 代码运行时启动命令
	AutoInstall bool          `json:"autoInstall,omitempty"`
	PkgMgr      string        `json:"pkgMgr,omitempty"` // node: auto/npm/yarn/pnpm
	Ports       []RuntimePort `json:"ports,omitempty"`  // 代码运行时端口映射
}

// RuntimeCreateInput 创建运行时入参。
type RuntimeCreateInput struct {
	NodeID      string        `json:"nodeId"` // M57 目标节点（空=local）
	Name        string        `json:"name"`
	Type        string        `json:"type"` // php（默认）/ node / python / java / go
	Version     string        `json:"version"`
	Extensions  []string      `json:"extensions"`
	Remark      string        `json:"remark"`
	CodeDir     string        `json:"codeDir"`
	StartCmd    string        `json:"startCmd"`
	AutoInstall bool          `json:"autoInstall"`
	PkgMgr      string        `json:"pkgMgr"`
	Ports       []RuntimePort `json:"ports"`
}

// RuntimeService 运行环境服务。
type RuntimeService struct {
	db    *gorm.DB
	nodes *NodeService
	tasks *TaskService
	nodeClient *agentclient.Client // WithNode 绑定（M57 节点化）
}

// NewRuntimeService 创建。
func NewRuntimeService(db *gorm.DB, nodes *NodeService, tasks *TaskService) *RuntimeService {
	return &RuntimeService{db: db, nodes: nodes, tasks: tasks}
}

// WithNode 返回绑定目标节点的副本（M57 runtime 节点化：实例操作按归属节点路由）。
func (s *RuntimeService) WithNode(nodeId string) (*RuntimeService, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	cp := *s
	cp.nodeClient = agentclient.New(node.BaseURL, node.Token)
	return &cp, nil
}

// clientFor 节点客户端（空/local=本机；nodeClient 非空时短路返回副本绑定的客户端）。
func (s *RuntimeService) clientFor(nodeId string) (*agentclient.Client, error) {
	if s.nodeClient != nil {
		return s.nodeClient, nil
	}
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

func (s *RuntimeService) client() (*agentclient.Client, error) {
	if s.nodeClient != nil {
		return s.nodeClient, nil
	}
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// exec 通过 agent 受控命令通道执行。
func (s *RuntimeService) exec(ctx context.Context, timeoutSecs int, format string, args ...any) (dto.ExecResp, error) {
	ac, err := s.client()
	if err != nil {
		return dto.ExecResp{}, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: fmt.Sprintf(format, args...), TimeoutSecs: timeoutSecs})
	if err != nil {
		return dto.ExecResp{}, err
	}
	return *out, nil
}

// writeRemote 写文本文件到 agent 主机（自动创建父目录）。
func (s *RuntimeService) writeRemote(ctx context.Context, p, content string) error {
	ac, err := s.client()
	if err != nil {
		return err
	}
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: p, Content: content})
	return err
}

// readRemote 读 agent 主机文本文件。
func (s *RuntimeService) readRemote(ctx context.Context, p string) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	q := url.Values{"path": []string{p}}
	resp, err := agentclient.GetJSON[dto.FileReadResp](ac, ctx, "/agent/v1/files/read?"+q.Encode())
	if err != nil {
		return "", err
	}
	if resp.Truncated {
		return "", errs.Wrapc(errs.CodeFileOpFailed, "文件过大无法读取: "+p)
	}
	return resp.Content, nil
}

// mustEmbed 读取内嵌模板文件。
func mustEmbed(rel string) string {
	b, err := runtimeDataFS.ReadFile(rel)
	if err != nil {
		panic(fmt.Sprintf("内嵌模板缺失 %s: %v", rel, err))
	}
	return string(b)
}

func (s *RuntimeService) dir(r *model.Runtime) string {
	return runtimeRootDir + "/" + r.Type + "/" + r.Name
}

// containerNameOr 兼容 M16 旧行（ContainerName 为空）。
func containerNameOr(r *model.Runtime) string {
	if r.ContainerName != "" {
		return r.ContainerName
	}
	return "php-" + r.Name
}

// locateCompose 运行时编排文件路径（兼容旧 M16 行：编排目录在受管 compose 目录下）。
func (s *RuntimeService) locateCompose(ctx context.Context, r *model.Runtime) (string, error) {
	candidates := []string{
		s.dir(r) + "/docker-compose.yml",
		legacyComposeDir + "/" + r.ComposeProject + "/docker-compose.yml",
	}
	out, err := s.exec(ctx, 15, "for p in %s %s; do [ -f \"$p\" ] && echo \"$p\" && break; done", candidates[0], candidates[1])
	if err == nil && out.ExitCode == 0 {
		if line := strings.TrimSpace(out.Output); line != "" {
			return line, nil
		}
	}
	return "", errs.New(errs.CodeNotFound, "error.notFound", "运行环境编排文件缺失（可能已被手动移除）")
}

func composeCmd(file, project, op string) string {
	return fmt.Sprintf("docker compose -f %s -p %s %s", file, project, op)
}

// EnsureHostGateway 确保运行时容器可解析 host.docker.internal（商店 php 应用同节点不同网络
// 外接数据库时，连接参数用宿主代指而非搬 IP）。新建运行时的模板已自带该 extra_hosts；
// 复用的存量运行时缺时补写主 compose（面板模板渲染的标准结构，environment: 前插入安全）
// 并 up -d 重建容器——php-fpm 闪断数秒，共享该运行时的其他站点会短暂中断。
// 返回是否发生了重建。
func (s *RuntimeService) EnsureHostGateway(ctx context.Context, r *model.Runtime) (bool, error) {
	out, err := s.exec(ctx, 20, "docker inspect -f '{{range .HostConfig.ExtraHosts}}{{.}} {{end}}' %s 2>/dev/null || true", r.ContainerName)
	if err != nil {
		return false, err
	}
	if strings.Contains(out.Output, "host.docker.internal") {
		return false, nil
	}
	file, err := s.locateCompose(ctx, r)
	if err != nil {
		return false, err
	}
	src, err := s.readRemote(ctx, file)
	if err != nil {
		return false, err
	}
	if strings.Contains(src, "extra_hosts:") {
		// 编排文件已带（新模板），容器尚未重建——交给下次 up，不做文本改写
		return false, nil
	}
	const inject = "    extra_hosts:\n      - \"host.docker.internal:host-gateway\"\n"
	idx := strings.Index(src, "    environment:")
	if idx < 0 {
		return false, fmt.Errorf("运行时 compose 结构异常，未找到注入点（environment 段缺失）")
	}
	if err := s.writeRemote(ctx, file, src[:idx]+inject+src[idx:]); err != nil {
		return false, err
	}
	if _, err := s.exec(ctx, 300, "%s", composeCmd(file, r.ComposeProject, "up -d")); err != nil {
		return false, err
	}
	return true, nil
}

// envJSON 解析运行参数。
func envJSON(row *model.Runtime) runtimeEnv {
	var env runtimeEnv
	if row.EnvJSON != "" {
		_ = json.Unmarshal([]byte(row.EnvJSON), &env)
	}
	return env
}

// fcgiAddrPattern 外部 fastcgi 地址（host:port 或 unix:/path/socket.sock）。
var fcgiAddrPattern = regexp.MustCompile(`^([a-zA-Z0-9._-]+:[0-9]{1,5}|unix:/[^\s]{1,200})$`)

// Create 创建运行时（PHP：模板渲染 → docker compose build → up；代码型：run.sh → pull → up）。
func (s *RuntimeService) Create(ctx context.Context, in RuntimeCreateInput) (map[string]any, error) {
	if in.Type == "" {
		in.Type = "php"
	}
	if in.Type != "php" && !codeTypes[in.Type] {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的运行时类型")
	}
	if !siteNamePattern.MatchString(in.Name) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "名称不合法（小写字母/数字/中划线）")
	}
	var count int64
	_ = s.db.Model(&model.Runtime{}).Where("name = ?", in.Name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.runtimeExists", "运行环境名已存在")
	}

	if in.Type == "php" {
		return s.createPHP(ctx, in)
	}
	return s.createCode(ctx, in)
}

// createPHP PHP 运行时（构建式）。
func (s *RuntimeService) createPHP(ctx context.Context, in RuntimeCreateInput) (map[string]any, error) {
	baseImage, ok := phpVersions[in.Version]
	if !ok {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的 PHP 版本（7.4/8.0-8.4）")
	}
	exts, err := normalizePHPExtensions(in.Extensions)
	if err != nil {
		return nil, err
	}
	env := runtimeEnv{Extensions: exts}
	envBytes, _ := json.Marshal(env)
	row := &model.Runtime{
		Name: in.Name, Type: "php", Version: in.Version, Origin: "container",
		Image:          fmt.Sprintf("%s/php-%s:%s", runtimeImagePrefix, in.Name, in.Version),
		ContainerName:  "php-" + in.Name,
		ComposeProject: "rt-php-" + in.Name,
		NodeID:         normalizeNodeID(in.NodeID),
		Status:         RuntimeStatusBuilding,
		EnvJSON:        string(envBytes),
		Remark:         truncStr(in.Remark, 255),
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	task, err := s.tasks.StartTask(TaskRuntimeCreate, fmt.Sprintf("创建 PHP %s 运行环境 %s", in.Version, in.Name), row.ComposeProject, 30*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			return s.buildPHP(tctx, logf, row, baseImage)
		})
	if err != nil {
		row.Status = RuntimeStatusError
		row.Message = err.Error()
		_ = s.db.Save(row)
		return nil, err
	}
	return map[string]any{"runtime": row, "taskId": task.ID}, nil
}

// createCode 代码运行时（node/python/java/go：代码目录挂载 + run.sh 启动）。
func (s *RuntimeService) createCode(ctx context.Context, in RuntimeCreateInput) (map[string]any, error) {
	image, ok := codeVersions[in.Type][in.Version]
	if !ok {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的 "+in.Type+" 版本")
	}
	if !pathAbsRe.MatchString(in.CodeDir) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "代码目录需为绝对路径")
	}
	if strings.HasPrefix(filepath.Clean(in.CodeDir), runtimeRootDir+"/") {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "代码目录不能位于运行环境目录内")
	}
	startCmd := strings.TrimSpace(in.StartCmd)
	if startCmd == "" {
		switch in.Type {
		case "node":
			startCmd = "npm start"
		case "python":
			startCmd = "python main.py"
		case "java":
			startCmd = "java -jar app.jar"
		case "go":
			startCmd = "go run ."
		}
	}
	if len(startCmd) > 500 {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "启动命令过长")
	}
	if in.PkgMgr != "" && !map[string]bool{"auto": true, "npm": true, "yarn": true, "pnpm": true}[in.PkgMgr] {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的包管理器")
	}
	if len(in.Ports) > 10 {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "端口映射数量超限（≤10）")
	}
	for _, p := range in.Ports {
		if p.Host < 1 || p.Host > 65535 || p.Container < 1 || p.Container > 65535 {
			return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "端口号需在 1-65535")
		}
	}
	if err := s.checkPortsFree(ctx, in.Ports, 0); err != nil {
		return nil, err
	}
	// 代码目录必须已存在
	out, err := s.exec(ctx, 15, "test -d %s && echo ok", in.CodeDir)
	if err != nil || out.ExitCode != 0 {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "代码目录不存在: "+in.CodeDir)
	}
	env := runtimeEnv{StartCmd: startCmd, AutoInstall: in.AutoInstall, PkgMgr: in.PkgMgr, Ports: in.Ports}
	envBytes, _ := json.Marshal(env)
	row := &model.Runtime{
		Name: in.Name, Type: in.Type, Version: in.Version, Origin: "container",
		Image:          image,
		CodeDir:        filepath.Clean(in.CodeDir),
		ContainerName:  "rt-" + in.Type + "-" + in.Name,
		ComposeProject: "rt-" + in.Type + "-" + in.Name,
		NodeID:         normalizeNodeID(in.NodeID),
		Status:         RuntimeStatusCreating,
		EnvJSON:        string(envBytes),
		Port:           joinHostPorts(in.Ports),
		Remark:         truncStr(in.Remark, 255),
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	task, err := s.tasks.StartTask(TaskRuntimeCreate, fmt.Sprintf("创建 %s %s 运行环境 %s", in.Type, in.Version, in.Name), row.ComposeProject, 20*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			return s.startCode(tctx, logf, row)
		})
	if err != nil {
		row.Status = RuntimeStatusError
		row.Message = err.Error()
		_ = s.db.Save(row)
		return nil, err
	}
	return map[string]any{"runtime": row, "taskId": task.ID}, nil
}

// startCode 写 compose/run.sh 并拉镜像启动（任务内执行）。
func (s *RuntimeService) startCode(ctx context.Context, logf TaskLogf, row *model.Runtime) error {
	dir := s.dir(row)
	env := envJSON(row)
	files := map[string]string{
		dir + "/run.sh":             mustEmbed("runtimedata/" + row.Type + "/run.sh"),
		dir + "/docker-compose.yml": s.renderCodeCompose(row, env),
	}
	names := make([]string, 0, len(files))
	for p := range files {
		names = append(names, p)
	}
	sort.Strings(names)
	for _, p := range names {
		logf("info", "写入 %s", p)
		if err := s.writeRemote(ctx, p, files[p]); err != nil {
			return err
		}
	}
	logf("info", "拉取镜像 %s", row.Image)
	pullOut, err := s.exec(ctx, 1800, "%s 2>&1", composeCmd(dir+"/docker-compose.yml", row.ComposeProject, "pull"))
	if err != nil {
		return err
	}
	if pullOut.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "镜像拉取失败: "+tailOutput(pullOut.Output, 600))
	}
	logf("info", "启动容器 %s", containerNameOr(row))
	upOut, err := s.exec(ctx, 600, "%s", composeCmd(dir+"/docker-compose.yml", row.ComposeProject, "up -d"))
	if err != nil {
		return err
	}
	if upOut.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "容器启动失败: "+tailOutput(upOut.Output, 600))
	}
	state, err := s.containerState(ctx, containerNameOr(row))
	if err != nil || state != "running" {
		return errs.Wrapc(errs.CodeFileOpFailed, "容器未进入运行状态（启动命令可能异常，请查看容器日志）: "+state)
	}
	row.Status = RuntimeStatusRunning
	row.Message = ""
	return s.db.Save(row).Error
}

// renderCodeCompose 渲染代码运行时 compose。
func (s *RuntimeService) renderCodeCompose(row *model.Runtime, env runtimeEnv) string {
	extraVolumes := ""
	if row.Type == "go" {
		extraVolumes = "      - " + s.dir(row) + "/gomod:/go/pkg/mod"
	}
	portsBlock := ""
	if len(env.Ports) > 0 {
		var b strings.Builder
		b.WriteString("    ports:\n")
		for _, p := range env.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			b.WriteString(fmt.Sprintf("      - \"%d:%d/%s\"\n", p.Host, p.Container, proto))
		}
		portsBlock = strings.TrimSuffix(b.String(), "\n")
	}
	// YAML 双引号标量转义
	escape := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	replacer := strings.NewReplacer(
		"__IMAGE__", row.Image,
		"__TYPE__", row.Type,
		"__NAME__", row.Name,
		"__TZ__", runtimeTZ,
		"__START_CMD__", escape.Replace(env.StartCmd),
		"__RUN_INSTALL__", map[bool]string{true: "1", false: "0"}[env.AutoInstall],
		"__PKG_MGR__", escape.Replace(env.PkgMgr),
		"__CODE_DIR__", row.CodeDir,
		"__EXTRA_VOLUMES__", extraVolumes,
		"__PORTS__", portsBlock,
	)
	return replacer.Replace(mustEmbed("runtimedata/code/docker-compose.yml"))
}

func joinHostPorts(ports []RuntimePort) string {
	if len(ports) == 0 {
		return ""
	}
	strs := make([]string, 0, len(ports))
	for _, p := range ports {
		strs = append(strs, strconv.Itoa(p.Host))
	}
	return strings.Join(strs, ",")
}

// pathAbsRe 绝对路径形态。
var pathAbsRe = regexp.MustCompile(`^/[^\s]{1,200}$`)

// checkPortsFree 端口映射占用校验：与其他运行时互斥 + 宿主监听占用。
func (s *RuntimeService) checkPortsFree(ctx context.Context, ports []RuntimePort, selfID uint) error {
	if len(ports) == 0 {
		return nil
	}
	var rows []model.Runtime
	_ = s.db.Find(&rows).Error
	for _, r := range rows {
		if r.ID == selfID {
			continue
		}
		for _, p := range envJSON(&r).Ports {
			for _, np := range ports {
				if p.Host == np.Host && strings.ToLower(cmpOr(p.Protocol, "tcp")) == strings.ToLower(cmpOr(np.Protocol, "tcp")) {
					return errs.New(errs.CodeConflict, "error.conflict", fmt.Sprintf("宿主端口 %d 已被运行环境 %s 占用", np.Host, r.Name))
				}
			}
		}
	}
	out, err := s.exec(ctx, 20, "ss -H -tln")
	if err != nil {
		return nil // 探测失败不阻断创建（up 时仍会暴露占用）
	}
	listening := map[string]bool{}
	for _, line := range strings.Split(out.Output, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		addr := f[len(f)-3]
		if i := strings.LastIndex(addr, ":"); i >= 0 {
			listening[addr[i+1:]] = true
		}
	}
	for _, p := range ports {
		if p.Protocol == "udp" {
			continue
		}
		if listening[strconv.Itoa(p.Host)] {
			return errs.New(errs.CodeConflict, "error.conflict", fmt.Sprintf("宿主端口 %d 已被监听占用", p.Host))
		}
	}
	return nil
}

func cmpOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// buildPHP 写模板文件并构建、启动（任务内执行；重建时用户配置仅在缺失时补写，避免覆盖手工修改）。
func (s *RuntimeService) buildPHP(ctx context.Context, logf TaskLogf, row *model.Runtime, baseImage string) error {
	dir := s.dir(row)
	exts := envJSON(row).Extensions
	files := map[string]string{
		dir + "/Dockerfile":                          mustEmbed("runtimedata/php/Dockerfile"),
		dir + "/install-ext.sh":                      mustEmbed("runtimedata/php/install-ext.sh"),
		dir + "/supervisor/supervisord.conf":         mustEmbed("runtimedata/php/supervisord.conf"),
		dir + "/supervisor/supervisor.d/php-fpm.ini": mustEmbed("runtimedata/php/php-fpm.ini"),
		dir + "/docker-compose.yml":                  renderPHPCompose(row.Name, row.Image, baseImage, exts, os.Getenv("PANEL_APK_MIRROR")),
		dir + "/conf/php.ini":                        mustEmbed("runtimedata/php/php.ini"),
		dir + "/conf/php-fpm.conf":                   mustEmbed("runtimedata/php/php-fpm.conf"),
	}
	// 用户可编辑文件：重建时已存在则跳过（php.ini / php-fpm.conf / supervisor 配置）
	keep, err := s.existingFiles(ctx,
		dir+"/conf/php.ini", dir+"/conf/php-fpm.conf",
		dir+"/supervisor/supervisord.conf", dir+"/supervisor/supervisor.d/php-fpm.ini")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for p := range files {
		names = append(names, p)
	}
	sort.Strings(names)
	for _, p := range names {
		if keep[p] {
			logf("info", "保留已有 %s", p)
			continue
		}
		logf("info", "写入 %s", p)
		if err := s.writeRemote(ctx, p, files[p]); err != nil {
			return err
		}
	}
	logf("info", "准备日志目录")
	if _, err := s.exec(ctx, 30, "mkdir -p %s/log %s/supervisor/log %s/supervisor/supervisor.d && chmod -R 0777 %s/log %s/supervisor/log", dir, dir, dir, dir, dir); err != nil {
		return err
	}
	// 兼容重装：清掉同名旧容器
	logf("info", "清理同名旧容器（如有）")
	_, _ = s.exec(ctx, 120, "%s 2>/dev/null; true", composeCmd(dir+"/docker-compose.yml", row.ComposeProject, "down --remove-orphans"))

	logf("info", "构建镜像 %s（首次构建含扩展编译，可能耗时数分钟）", row.Image)
	buildOut, err := s.exec(ctx, 1800, "%s", composeCmd(dir+"/docker-compose.yml", row.ComposeProject, "build --progress=plain"))
	logf("info", "%s", tailOutput(buildOut.Output, 1500))
	if err != nil {
		return err
	}
	if buildOut.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "镜像构建失败: "+tailOutput(buildOut.Output, 600))
	}

	logf("info", "启动容器 %s", containerNameOr(row))
	upOut, err := s.exec(ctx, 600, "%s", composeCmd(dir+"/docker-compose.yml", row.ComposeProject, "up -d"))
	if err != nil {
		return err
	}
	if upOut.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "容器启动失败: "+tailOutput(upOut.Output, 600))
	}
	state, err := s.containerState(ctx, containerNameOr(row))
	if err != nil || state != "running" {
		return errs.Wrapc(errs.CodeFileOpFailed, "容器未进入运行状态: "+state)
	}
	row.Status = RuntimeStatusRunning
	row.Message = ""
	return s.db.Save(row).Error
}

// renderPHPCompose 渲染 php 运行时 compose（模板占位符 __XXX__）。
func renderPHPCompose(name, image, baseImage string, exts []string, apkMirror string) string {
	replacer := strings.NewReplacer(
		"__NAME__", name,
		"__IMAGE__", image,
		"__BASE_IMAGE__", baseImage,
		"__EXTENSIONS__", strings.Join(exts, ","),
		"__APK_MIRROR__", apkMirror,
		"__TZ__", runtimeTZ,
	)
	return replacer.Replace(mustEmbed("runtimedata/php/docker-compose.yml"))
}

// containerState 查询容器状态（未找到返回空串）。
func (s *RuntimeService) containerState(ctx context.Context, container string) (string, error) {
	out, err := s.exec(ctx, 20, "docker inspect -f '{{.State.Status}}' %s 2>/dev/null || true", container)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Output), nil
}

// existingFiles 批量探测远端文件是否存在。
func (s *RuntimeService) existingFiles(ctx context.Context, paths ...string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(paths) == 0 {
		return out, nil
	}
	quoted := make([]string, 0, len(paths))
	for _, p := range paths {
		quoted = append(quoted, "'"+p+"'")
	}
	cmd := fmt.Sprintf("for f in %s; do [ -f \"$f\" ] && echo \"$f\"; done", strings.Join(quoted, " "))
	resp, err := s.exec(ctx, 15, "%s", cmd)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(resp.Output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out[line] = true
		}
	}
	return out, nil
}

// Rebuild 重建运行时（php：用当前面板模板重新构建镜像；配置文件保留用户修改）。
func (s *RuntimeService) Rebuild(ctx context.Context, id uint) (map[string]any, error) {
	row, err := s.getRuntimeRow(id)
	if err != nil {
		return nil, err
	}
	if row.Origin == "external" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "外部运行环境不支持重建")
	}
	if s.extTaskRunning(id) {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "该运行环境已有任务进行中，请稍后再试")
	}
	var title string
	run := func(tctx context.Context, logf TaskLogf) error { return nil }
	if row.Type == "php" {
		baseImage, ok := phpVersions[row.Version]
		if !ok {
			return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的 PHP 版本: "+row.Version)
		}
		row.Status = RuntimeStatusBuilding
		if err := s.db.Save(row).Error; err != nil {
			return nil, err
		}
		title = fmt.Sprintf("重建 PHP 运行环境 %s", row.Name)
		run = func(tctx context.Context, logf TaskLogf) error { return s.buildPHP(tctx, logf, row, baseImage) }
	} else {
		row.Status = RuntimeStatusCreating
		if err := s.db.Save(row).Error; err != nil {
			return nil, err
		}
		title = fmt.Sprintf("重建 %s 运行环境 %s", row.Type, row.Name)
		run = func(tctx context.Context, logf TaskLogf) error { return s.startCode(tctx, logf, row) }
	}
	task, err := s.tasks.StartTask("runtime-rebuild", title, row.ComposeProject, 30*time.Minute, run)
	if err != nil {
		row.Status = RuntimeStatusError
		row.Message = err.Error()
		_ = s.db.Save(row)
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// AttachExternal 接管本机已有 php-fpm（fastcgi 直连，不创建容器）。
func (s *RuntimeService) AttachExternal(name, version, fcgiAddr, remark string) (*model.Runtime, error) {
	if !siteNamePattern.MatchString(name) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "名称不合法（小写字母/数字/中划线）")
	}
	if version == "" {
		version = "unknown"
	}
	if !fcgiAddrPattern.MatchString(fcgiAddr) {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "FastCGI 地址需为 host:port 或 unix:/path 形式")
	}
	var count int64
	_ = s.db.Model(&model.Runtime{}).Where("name = ?", name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.runtimeExists", "运行环境名已存在")
	}
	row := &model.Runtime{
		Name: name, Type: "php", Version: version, Origin: "external", FCGIAddr: fcgiAddr,
		Status: RuntimeStatusRunning, Remark: truncStr(remark, 255), ComposeProject: "-",
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// List 运行时列表（含容器实时状态）。
func (s *RuntimeService) List(ctx context.Context) ([]map[string]any, error) {
	var rows []model.Runtime
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	// M57：容器状态按实例归属节点分组查询（跨节点聚合）
	stateMap := map[string]string{}
	byNode := map[string][]string{}
	for i := range rows {
		if rows[i].Origin == "container" {
			nid := normalizeNodeID(rows[i].NodeID)
			byNode[nid] = append(byNode[nid], containerNameOr(&rows[i]))
		}
	}
	for nid, names := range byNode {
		ac, err := s.clientFor(nid)
		if err != nil {
			continue
		}
		out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: fmt.Sprintf("docker inspect -f '{{.Name}}|{{.State.Status}}' %s 2>/dev/null || true", strings.Join(names, " ")), TimeoutSecs: 20})
		if err != nil {
			continue
		}
		for _, line := range strings.Split(out.Output, "\n") {
			name, status, ok := strings.Cut(strings.TrimSpace(line), "|")
			if ok {
				stateMap[strings.TrimPrefix(name, "/")] = status
			}
		}
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		status := r.Status
		if r.Origin == "external" {
			status = RuntimeStatusRunning
		} else if st, ok := stateMap[containerNameOr(r)]; ok && status != RuntimeStatusBuilding && status != RuntimeStatusCreating {
			switch st {
			case "running":
				status = RuntimeStatusRunning
			case "paused":
				status = RuntimeStatusStopped
			default:
				status = RuntimeStatusStopped
			}
		} else if !ok && status != RuntimeStatusBuilding && status != RuntimeStatusCreating {
			status = RuntimeStatusStopped
		}
		env := envJSON(r)
		item := map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "version": r.Version,
			"origin": r.Origin, "fcgiAddr": r.FCGIAddr, "remark": r.Remark,
			"nodeId": normalizeNodeID(r.NodeID),
			"image": r.Image, "containerName": containerNameOr(r),
			"composeProject": r.ComposeProject,
			"status":         status, "message": r.Message,
			"extensions": env.Extensions,
			"running":    status == RuntimeStatusRunning,
			"createdAt":  r.CreatedAt,
		}
		if r.Type != "php" {
			item["codeDir"] = r.CodeDir
			item["startCmd"] = env.StartCmd
			item["pkgMgr"] = env.PkgMgr
			item["autoInstall"] = env.AutoInstall
			item["ports"] = env.Ports
		}
		out = append(out, item)
	}
	return out, nil
}

// Detail 运行时详情。
func (s *RuntimeService) Detail(ctx context.Context, id uint) (map[string]any, error) {
	var row model.Runtime
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.runtimeNotFound", "运行环境不存在")
	}
	items, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it["id"].(uint) == id {
			return it, nil
		}
	}
	return nil, errs.New(errs.CodeNotFound, "error.runtimeNotFound", "运行环境不存在")
}

// Delete 删除运行时（站点绑定校验；容器 down + 清理编排目录，www 数据保留）。
func (s *RuntimeService) Delete(ctx context.Context, id uint) error {
	var row model.Runtime
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.runtimeNotFound", "运行环境不存在")
	}
	if row.Origin == "external" {
		return s.db.Delete(&model.Runtime{}, id).Error
	}
	var bound int64
	_ = s.db.Model(&model.Site{}).Where("runtime_id = ?", id).Count(&bound).Error
	if bound > 0 {
		return errs.New(errs.CodeConflict, "error.runtimeInUse", "存在绑定该运行环境的站点，请先改绑或删除站点")
	}
	if file, err := s.locateCompose(ctx, &row); err == nil {
		_, _ = s.exec(ctx, 300, "%s 2>/dev/null; rm -rf %s", composeCmd(file, row.ComposeProject, "down --remove-orphans"), s.dir(&row))
	}
	return s.db.Delete(&model.Runtime{}, id).Error
}

// Operate 启动/停止/重启。
func (s *RuntimeService) Operate(ctx context.Context, id uint, action string) error {
	var row model.Runtime
	if err := s.db.First(&row, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.runtimeNotFound", "运行环境不存在")
	}
	if row.Origin == "external" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "外部运行环境由其所在主机管理，面板不支持启停")
	}
	var (
		out     dto.ExecResp
		err     error
		timeout int
		cmdStr  string
	)
	switch action {
	case "start":
		file, ferr := s.locateCompose(ctx, &row)
		if ferr != nil {
			return ferr
		}
		cmdStr, timeout = composeCmd(file, row.ComposeProject, "up -d"), 600
	case "stop":
		file, ferr := s.locateCompose(ctx, &row)
		if ferr != nil {
			return ferr
		}
		cmdStr, timeout = composeCmd(file, row.ComposeProject, "down"), 300
	case "restart":
		cmdStr, timeout = fmt.Sprintf("docker restart %s", containerNameOr(&row)), 300
	default:
		return errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的操作")
	}
	out, err = s.exec(ctx, timeout, "%s", cmdStr)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		err = errs.Wrapc(errs.CodeFileOpFailed, action+" 失败: "+tailOutput(out.Output, 600))
		row.Status = RuntimeStatusError
		row.Message = err.Error()
		_ = s.db.Save(&row)
		return err
	}
	switch action {
	case "start", "restart":
		row.Status = RuntimeStatusRunning
	case "stop":
		row.Status = RuntimeStatusStopped
	}
	row.Message = ""
	return s.db.Save(&row).Error
}
