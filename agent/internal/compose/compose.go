// Package compose 基于 docker compose CLI 的项目编排管理。
// 安全：项目名白名单校验；docker 以参数数组直调、不经 shell；外部项目目录须存在 compose 配置方可操作。
package compose

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 托管目录下认可的配置文件命名。
var managedFileNames = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

const (
	composeProjectLabel     = "com.docker.compose.project"
	composeWorkdirLabel     = "com.docker.compose.project.working_dir"
	composeServiceLabel     = "com.docker.compose.service"
)

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// ValidateName 项目名/服务名白名单。
func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return errs.Wrap(errs.ErrBadRequest, "名称不合法（仅字母/数字/中划线/下划线，≤64 位）")
	}
	return nil
}

// DockerLister 容器列表依赖（dockerx.Manager 满足；用于外部项目识别与服务状态聚合）。
type DockerLister interface {
	List(ctx context.Context) ([]dto.ContainerItem, error)
}

// Manager compose 管理器。
type Manager struct {
	baseDir string
	docker  DockerLister
}

// New 创建管理器并确保托管目录存在。
func New(baseDir string, docker DockerLister) (*Manager, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建 compose 托管目录失败: %w", err)
	}
	return &Manager{baseDir: baseDir, docker: docker}, nil
}

// BaseDir 托管目录。
func (m *Manager) BaseDir() string { return m.baseDir }

// findConfig 定位目录下的 compose 配置文件。
func findConfig(dir string) (string, bool) {
	for _, n := range managedFileNames {
		p := filepath.Join(dir, n)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// resolveProject 解析项目配置文件：dir 非空视为外部项目目录，否则 baseDir/name（托管）。
func (m *Manager) resolveProject(name, dir string) (configFile string, err error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	workDir := filepath.Join(m.baseDir, name)
	if dir != "" {
		workDir = filepath.Clean(dir)
	}
	cfg, ok := findConfig(workDir)
	if !ok {
		return "", errs.Wrap(errs.ErrNotFound, "未在 "+workDir+" 找到 compose 配置文件（compose.yaml/docker-compose.yml 等）")
	}
	return cfg, nil
}

// composeArgs 组装 docker compose 基础参数。
func composeArgs(configFile string, args ...string) []string {
	return append([]string{"compose", "--file", configFile}, args...)
}

// runDocker 执行 docker 子命令（120s 超时），返回合并输出与退出码。
func runDocker(ctx context.Context, args ...string) (string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 120_000_000_000)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if len(out) > 1<<20 {
		out = out[:1<<20]
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode(), nil
		}
		return string(out), -1, err
	}
	return string(out), 0, nil
}


// ServiceAction 单服务操作（start/stop/restart/pull；up=按当前编排定义重建该服务）。
func (m *Manager) ServiceAction(ctx context.Context, project, service, action string) (string, error) {
	if err := ValidateName(project); err != nil {
		return "", err
	}
	if err := ValidateName(service); err != nil {
		return "", err
	}
	switch action {
	case "start", "stop", "restart", "pull", "up":
	default:
		return "", errs.ErrBadRequest
	}
	workDir := filepath.Join(m.baseDir, project)
	cfg, ok := findConfig(workDir)
	if !ok {
		return "", errs.Wrap(errs.ErrNotFound, "未在 "+workDir+" 找到 compose 配置")
	}
	args := []string{"--project-name", project}
	if action == "up" {
		args = append(args, "up", "-d", service)
	} else {
		args = append(args, action, service)
	}
	out, code, err := runDocker(ctx, composeArgs(cfg, args...)...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, errs.Wrapc(errs.CodeFileOpFailed, "compose "+action+" 失败: "+out)
	}
	return out, nil
}

// ProjectDelete 删除托管项目：先 down（忽略失败）再移除编排目录（含其下全部数据，调用方须已确认）。
func (m *Manager) ProjectDelete(ctx context.Context, project string) error {
	if err := ValidateName(project); err != nil {
		return err
	}
	dir := filepath.Join(m.baseDir, project)
	if dir == m.baseDir || !strings.HasPrefix(dir, m.baseDir+string(filepath.Separator)) {
		return errs.ErrPathInvalid
	}
	if _, err := os.Stat(dir); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "项目目录不存在: "+dir)
	}
	// down 失败不阻塞（容器可能已不存在）
	if cfg, ok := findConfig(dir); ok {
		_, _, _ = runDocker(ctx, composeArgs(cfg, "--project-name", project, "down")...)
	}
	if err := os.RemoveAll(dir); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, "删除项目目录失败: "+err.Error())
	}
	return nil
}

func (m *Manager) List(ctx context.Context) ([]dto.ComposeProject, error) {
	type agg struct {
		dir      string
		services map[string]dto.ComposeServiceState
	}
	groups := map[string]*agg{}

	if containers, err := m.docker.List(ctx); err == nil {
		for _, c := range containers {
			name := c.Labels[composeProjectLabel]
			if name == "" {
				continue
			}
			g, ok := groups[name]
			if !ok {
				g = &agg{dir: c.Labels[composeWorkdirLabel], services: map[string]dto.ComposeServiceState{}}
				groups[name] = g
			}
			// service 名以 label 为准（com.docker.compose.service）；容器名砍项目名前缀仅在
			// label 缺失时兜底——项目名与容器名相同（如 container_name 显式同名）时会砍成空串
			svc := c.Labels[composeServiceLabel]
			if svc == "" {
				svc = strings.TrimPrefix(strings.TrimPrefix(c.Name, name), "-")
			}
			if svc == "" {
				svc = c.Name
			}
			g.services[c.Name] = dto.ComposeServiceState{Name: svc, Image: c.Image, State: c.State}
		}
	}

	if err := os.MkdirAll(m.baseDir, 0o755); err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}

	managed := map[string]bool{}
	projects := make([]dto.ComposeProject, 0, len(groups)+len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(m.baseDir, e.Name())
		if _, ok := findConfig(dir); !ok {
			continue
		}
		managed[e.Name()] = true
		pv := dto.ComposeProject{Name: e.Name(), Dir: dir, Managed: true}
		if g, ok := groups[e.Name()]; ok {
			pv.Services = toSlice(g.services)
		}
		pv.Running, pv.Total = countStates(pv.Services)
		projects = append(projects, pv)
	}
	names := make([]string, 0, len(groups))
	for n := range groups {
		if !managed[n] {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		g := groups[n]
		pv := dto.ComposeProject{Name: n, Dir: g.dir, Managed: false, Services: toSlice(g.services)}
		pv.Running, pv.Total = countStates(pv.Services)
		projects = append(projects, pv)
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	return projects, nil
}

// Config 读取项目 yaml。
func (m *Manager) Config(name, dir string) (dto.ComposeConfigResp, error) {
	cfg, err := m.resolveProject(name, dir)
	if err != nil {
		return dto.ComposeConfigResp{}, err
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		return dto.ComposeConfigResp{}, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return dto.ComposeConfigResp{Name: name, Dir: filepath.Dir(cfg), File: cfg, Content: string(b)}, nil
}

// WriteConfig 写托管项目 yaml（外部项目拒绝）。
func (m *Manager) WriteConfig(name, content string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	dir := filepath.Join(m.baseDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	cfg, ok := findConfig(dir)
	if !ok {
		cfg = filepath.Join(dir, "compose.yaml")
	}
	if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return nil
}

// Up 启动项目（up -d）。
func (m *Manager) Up(ctx context.Context, name, dir string) (string, error) {
	cfg, err := m.resolveProject(name, dir)
	if err != nil {
		return "", err
	}
	out, code, err := runDocker(ctx, composeArgs(cfg, "--project-name", name, "up", "-d")...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, errs.Wrapc(errs.CodeFileOpFailed, "compose up 失败")
	}
	return out, nil
}

// Down 停止并移除项目。
func (m *Manager) Down(ctx context.Context, name, dir string) (string, error) {
	cfg, err := m.resolveProject(name, dir)
	if err != nil {
		return "", err
	}
	out, code, err := runDocker(ctx, composeArgs(cfg, "--project-name", name, "down")...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, errs.Wrapc(errs.CodeFileOpFailed, "compose down 失败")
	}
	return out, nil
}

// Logs 输出项目日志到 w；follow 时持续直到 ctx 取消。
func (m *Manager) Logs(ctx context.Context, name, dir, tail, service string, w io.Writer) error {
	cfg, err := m.resolveProject(name, dir)
	if err != nil {
		return err
	}
	if service != "" {
		if err := ValidateName(service); err != nil {
			return err
		}
	}
	if tail == "" {
		tail = "500"
	}
	// 项目名必须显式传入：compose 默认取编排文件所在目录名，运行时目录（/opt/ypanel/runtime/<type>/<name>）
	// 的目录名与项目名（rt-<type>-<name>）不一致时会静默查不到任何容器日志
	args := composeArgs(cfg, "--project-name", name, "logs", "--tail", tail, "--no-log-prefix")
	if service != "" {
		args = append(args, service)
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()
	buf := make([]byte, 8192)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		n, rerr := stdout.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return nil
			}
		}
		if rerr != nil {
			return nil
		}
	}
}

func toSlice(m map[string]dto.ComposeServiceState) []dto.ComposeServiceState {
	out := make([]dto.ComposeServiceState, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func countStates(services []dto.ComposeServiceState) (running, total int) {
	total = len(services)
	for _, s := range services {
		if s.State == "running" {
			running++
		}
	}
	return
}

// ScanProjects B12：扫描父目录下含 compose 文件的项目（排除托管目录自身）。
func ScanProjects(parentDir string) ([]map[string]any, error) {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(parentDir, e.Name())
		composeFile := ""
		for _, f := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
			if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
				composeFile = f
				break
			}
		}
		if composeFile == "" {
			continue
		}
		out = append(out, map[string]any{"name": e.Name(), "dir": dir, "file": composeFile})
	}
	return out, nil
}
