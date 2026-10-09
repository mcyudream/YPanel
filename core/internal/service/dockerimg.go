// DockerImgService Docker 镜像生命周期与容器增强（M35）：build/save/load/tag/更新检查、
// 容器 commit/卷备份、Swarm 状态。全部走 agent 受控 exec 通道（零 agent 改动）；
// 长输出按 exp 教训「落文件 + BUILD_EXIT 标记 + 尾部回读」，杜绝进度输出丢尾。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// dockerTmpBase 镜像/构建临时产物目录（agent 主机侧）。
const dockerTmpBase = "/opt/ypanel/tmp"

// containerBackupBase 容器卷备份目录。
const containerBackupBase = "/opt/ypanel/backups/containers"

// imageRefPattern 镜像引用（name:tag / name@digest / registry 前缀）。
var imageRefPattern = regexp.MustCompile(`^[a-zA-Z0-9._/:@-]{1,250}$`)

// containerNamePattern 容器名。
var containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$`)

// tagRefPattern 新 tag 引用（不含 digest）。
var tagRefPattern = regexp.MustCompile(`^[a-zA-Z0-9._/: -]{1,250}$`)

// pathSafePattern 绝对路径且无穿越。
var pathSafePattern = regexp.MustCompile(`^/[a-zA-Z0-9._/@:+ -]+$`)

// DockerImgService 镜像生命周期。
type DockerImgService struct {
	nodes *NodeService
}

// NewDockerImgService 创建。
func NewDockerImgService(nodes *NodeService) *DockerImgService {
	return &DockerImgService{nodes: nodes}
}

func (s *DockerImgService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// execRun 执行命令（超时可控），exit!=0 时返回带尾部输出的错误。
func (s *DockerImgService) execRun(ctx context.Context, cmd string, timeout int) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 || out.TimedOut {
		return out.Output, errs.Wrapc(errs.CodeFileOpFailed, "执行失败: "+firstLine(tail(out.Output, 1200)))
	}
	return out.Output, nil
}

func validateRef(s string, what string) error {
	if !imageRefPattern.MatchString(s) {
		return errs.Wrap(errs.ErrBadRequest, what+"不合法")
	}
	return nil
}

// Build 构建镜像：contextDir 内以 dockerfile 构建为 tag；输出落 build.log（失败保留并返回路径）。
func (s *DockerImgService) Build(ctx context.Context, contextDir, dockerfile, tag string) (map[string]any, error) {
	contextDir = strings.TrimSpace(contextDir)
	if !pathSafePattern.MatchString(contextDir) || strings.Contains(contextDir, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "构建上下文目录不合法（需绝对路径）")
	}
	if !tagRefPattern.MatchString(tag) || strings.Contains(tag, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "镜像 tag 不合法")
	}
	df := dockerfile
	if df == "" {
		df = "Dockerfile"
	}
	if strings.Contains(df, "..") || strings.ContainsAny(df, "&|;$`") {
		return nil, errs.Wrap(errs.ErrBadRequest, "Dockerfile 路径不合法")
	}
	ts := time.Now().Format("20060102-150405")
	logFile := path.Join(dockerTmpBase, "build-"+ts+".log")
	cmd := fmt.Sprintf(
		`mkdir -p %s && cd '%s' && docker build -f '%s' -t '%s' . > %s 2>&1; ec=$?; tail -c 1200 %s; echo BUILD_EXIT=$ec; [ $ec -eq 0 ] && echo OK || exit 1`,
		dockerTmpBase, contextDir, df, tag, logFile, logFile)
	out, err := s.execRun(ctx, cmd, 1800)
	if err != nil {
		return map[string]any{"logFile": logFile, "output": out},
			errs.Wrapc(errs.CodeFileOpFailed, fmt.Sprintf("构建失败（完整日志: %s）: %s", logFile, firstLine(tail(out, 400))))
	}
	return map[string]any{"tag": tag, "logFile": logFile}, nil
}

// Save 导出镜像为 tar（多镜像单包），返回文件路径供下载。
func (s *DockerImgService) Save(ctx context.Context, images []string, name string) (map[string]any, error) {
	if len(images) == 0 || len(images) > 20 {
		return nil, errs.Wrap(errs.ErrBadRequest, "镜像列表为空或超限")
	}
	for _, im := range images {
		if err := validateRef(im, "镜像引用"); err != nil {
			return nil, err
		}
	}
	if name == "" {
		name = "images-" + time.Now().Format("20060102-150405")
	}
	if !containerNamePattern.MatchString(name) {
		return nil, errs.Wrap(errs.ErrBadRequest, "导出名不合法")
	}
	target := path.Join(dockerTmpBase, name+".tar")
	cmd := fmt.Sprintf("mkdir -p %s && docker save -o '%s' %s", dockerTmpBase, target,
		strings.Join(quoteArgs(images), " "))
	if _, err := s.execRun(ctx, cmd, 1800); err != nil {
		return nil, err
	}
	return map[string]any{"file": target, "name": name + ".tar"}, nil
}

// Load 导入镜像（tar 已由调用方上传到 agentPath）。
func (s *DockerImgService) Load(ctx context.Context, agentPath string) (map[string]any, error) {
	if !pathSafePattern.MatchString(agentPath) || strings.Contains(agentPath, "..") {
		return nil, errs.Wrap(errs.ErrBadRequest, "导入文件路径不合法")
	}
	out, err := s.execRun(ctx, fmt.Sprintf("docker load -i '%s'", agentPath), 1800)
	if err != nil {
		return nil, err
	}
	return map[string]any{"output": tail(out, 1200)}, nil
}

// Tag 为镜像打新 tag。
func (s *DockerImgService) Tag(ctx context.Context, src, dst string) error {
	if err := validateRef(src, "源镜像"); err != nil {
		return err
	}
	if !tagRefPattern.MatchString(dst) || strings.Contains(dst, "..") {
		return errs.Wrap(errs.ErrBadRequest, "新 tag 不合法")
	}
	_, err := s.execRun(ctx, fmt.Sprintf("docker tag '%s' '%s'", src, dst), 60)
	return err
}

// Commit 容器提交为镜像。
func (s *DockerImgService) Commit(ctx context.Context, container, image string) error {
	if !containerNamePattern.MatchString(container) {
		return errs.Wrap(errs.ErrBadRequest, "容器名不合法")
	}
	if !tagRefPattern.MatchString(image) || strings.Contains(image, "..") {
		return errs.Wrap(errs.ErrBadRequest, "镜像名不合法")
	}
	_, err := s.execRun(ctx, fmt.Sprintf("docker commit '%s' '%s'", container, image), 300)
	return err
}

// CheckUpdates 镜像更新检查（本地 RepoDigests vs registry 远端 manifest digest）。
func (s *DockerImgService) CheckUpdates(ctx context.Context, images []string) ([]map[string]any, error) {
	if len(images) == 0 || len(images) > 30 {
		return nil, errs.Wrap(errs.ErrBadRequest, "镜像列表为空或超限（≤30）")
	}
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(images))
	for _, im := range images {
		if err := validateRef(im, "镜像引用"); err != nil {
			return nil, err
		}
		row := map[string]any{"image": im}
		// 本地 digest（同名 tag 的 RepoDigests）
		lo, lerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: fmt.Sprintf("docker image inspect '%s' --format '{{join .RepoDigests \",\"}}'", im), TimeoutSecs: 30})
		if lerr == nil && lo.ExitCode == 0 {
			row["localDigests"] = splitDigests(lo.Output)
		}
		reg, repo, tag := splitImageRef(im)
		remote, rerr := remoteManifestDigest(ctx, reg, repo, tag)
		if rerr != nil {
			row["ok"] = false
			row["error"] = rerr.Error()
		} else {
			row["ok"] = true
			row["remoteDigest"] = remote
			row["hasUpdate"] = !hasDigest(row["localDigests"], remote)
		}
		out = append(out, row)
	}
	return out, nil
}

func splitDigests(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "," {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hasDigest(v any, digest string) bool {
	list, ok := v.([]string)
	if !ok {
		return false
	}
	for _, d := range list {
		if d == digest || strings.HasSuffix(d, "@"+digest) {
			return true
		}
	}
	return false
}

// splitImageRef 拆 registry/repo:tag（无 registry 前缀视为 docker.io，无 library/ 前缀补齐）。
func splitImageRef(ref string) (registry, repo, tag string) {
	tag = "latest"
	if i := strings.LastIndex(ref, "@"); i >= 0 {
		tag = ref[i+1:]
		ref = ref[:i]
	} else if i := strings.LastIndex(ref, ":"); i >= 0 && !strings.Contains(ref[i:], "/") {
		tag = ref[i+1:]
		ref = ref[:i]
	}
	registry = "docker.io"
	if seg := strings.SplitN(ref, "/", 2); len(seg) == 2 && (strings.ContainsAny(seg[0], ".:") || seg[0] == "localhost") {
		registry, repo = seg[0], seg[1]
	} else {
		repo = ref
		if !strings.Contains(repo, "/") {
			repo = "library/" + repo
		}
	}
	return registry, repo, tag
}

// manifestAccept registry v2 支持的清单类型。
const manifestAccept = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"

// remoteManifestDigest 查询远端 tag 的 manifest digest（Docker Hub 匿名 token 流；其余 registry 直连）。
func remoteManifestDigest(ctx context.Context, registry, repo, ref string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	base := "https://" + registry
	if registry == "docker.io" {
		base = "https://registry-1.docker.io"
	}
	get := func(url string, tok string) (*http.Response, error) {
		req, err := httpRequest(cctx, http.MethodGet, url, "", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", manifestAccept)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		return defaultHTTPClient.Do(req)
	}
	// token 预取（401 时按 WWW-Authenticate 提示的 realm 重试；Docker Hub 直接先取）
	var token string
	if registry == "docker.io" {
		tr, terr := httpRequest(cctx, http.MethodGet, "https://auth.docker.io/token?service=registry.docker.io&scope=repository:"+repo+":pull", "", nil)
		if terr == nil {
			if resp, err := defaultHTTPClient.Do(tr); err == nil {
				var body struct {
					Token string `json:"token"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&body)
				_ = resp.Body.Close()
				token = body.Token
			}
		}
	}
	resp, err := get(base+"/v2/"+repo+"/manifests/"+ref, token)
	if err != nil {
		return "", errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "registry 不可达: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized && token == "" {
		return "", fmt.Errorf("registry 需要认证，暂不支持该镜像的更新检查（%s）", registry)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry 返回 HTTP %d", resp.StatusCode)
	}
	dg := resp.Header.Get("Docker-Content-Digest")
	if dg == "" {
		return "", fmt.Errorf("registry 未返回 Docker-Content-Digest")
	}
	return dg, nil
}

// ContainerVolumes 备份容器的 named volumes（逐卷 tar.gz 到 /opt/ypanel/backups/containers）+ inspect 落盘。
func (s *DockerImgService) ContainerVolumes(ctx context.Context, container string) (map[string]any, error) {
	if !containerNamePattern.MatchString(container) {
		return nil, errs.Wrap(errs.ErrBadRequest, "容器名不合法")
	}
	ts := time.Now().Format("20060102-150405")
	// 1. inspect 取 named volumes 与配置
	inspCmd := fmt.Sprintf(`mkdir -p %s && docker inspect '%s' > '%s/%s-%s.json' && docker inspect -f '{{range .Mounts}}{{.Type}}|{{.Name}}|{{.Destination}}{{"\n"}}{{end}}' '%s'`,
		containerBackupBase, container, containerBackupBase, container, ts, container)
	out, err := s.execRun(ctx, inspCmd, 120)
	if err != nil {
		return nil, err
	}
	files := []string{fmt.Sprintf("%s-%s.json", container, ts)}
	volumes := []string{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "|")
		if len(parts) != 3 || parts[0] != "volume" || parts[1] == "" {
			continue
		}
		vol := parts[1]
		if !containerNamePattern.MatchString(vol) {
			continue
		}
		volumes = append(volumes, vol)
		f := fmt.Sprintf("%s-%s-%s.tar.gz", container, vol, ts)
		// 2. 逐卷导出（临时挂载卷打 tar）
		cmd := fmt.Sprintf(`docker run --rm -v '%s':/from:ro -v '%s':/to alpine tar czf '/to/%s' -C /from .`,
			vol, containerBackupBase, f)
		if _, verr := s.execRun(ctx, cmd, 900); verr != nil {
			return map[string]any{"files": files, "volumes": volumes, "error": "卷 " + vol + " 导出失败: " + verr.Error()},
				errs.Wrapc(errs.CodeFileOpFailed, "卷 "+vol+" 导出失败")
		}
		files = append(files, f)
	}
	return map[string]any{"files": files, "volumes": volumes, "dir": containerBackupBase}, nil
}

// SwarmStatus 本机 Swarm 状态（docker info 部分环境下 exit!=0 但输出有效，按输出解析）。
func (s *DockerImgService) SwarmStatus(ctx context.Context) (map[string]any, error) {
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: `docker info --format '{{.Swarm.LocalNodeState}}|{{.Swarm.ControlAvailable}}|{{if .Swarm.Cluster}}{{.Swarm.Cluster.ID}}{{end}}'`, TimeoutSecs: 60})
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(out.Output), "|")
	if out.Output == "" || len(parts) < 2 {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "swarm 状态读取失败: "+firstLine(out.Output))
	}
	row := map[string]any{"state": parts[0], "controlAvailable": len(parts) > 1 && parts[1] == "true"}
	if len(parts) > 2 {
		row["clusterId"] = parts[2]
	}
	return row, nil
}

// quoteArgs 白名单校验后的参数加单引号（值已过 imageRefPattern，天然无引号）。
func quoteArgs(vals []string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, "'"+v+"'")
	}
	return out
}
