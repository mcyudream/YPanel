// Src2ComposeService 源码构建生成 compose 项目（M26 P2）。
// 编排：core 渲染模板与 compose.yaml（embed 模板，同 B20 范式）→ agent 执行 clone/build/up（任务化，30 分钟）。
// 凭据：项目级 credentialId 优先，否则按 git 地址 host 自动匹配凭据库；secret 仅内存传递不落日志。
package service

import (
	"context"
	"embed"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

//go:embed src2composetpl
var src2composeFS embed.FS

const Src2ComposeTaskType = "src2compose-build"

// src2Root compose 托管根（与 agent compose.Manager 默认一致）。
const src2Root = "/opt/ypanel/compose"

var (
	src2NameRe  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	src2VerRe   = regexp.MustCompile(`^[0-9][0-9a-zA-Z.\-]{0,15}$`)
	src2DirRe   = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._\-]*$`)
	buildExitRe = regexp.MustCompile(`BUILD_EXIT=(\d+)`)
)

// SrcLangs 支持的语言栈（向导展示与校验）。
var SrcLangs = []map[string]string{
	{"lang": "dockerfile", "label": "Dockerfile（仓库自带）"},
	{"lang": "go", "label": "Go"},
	{"lang": "node", "label": "Node.js（启动服务）"},
	{"lang": "node-build", "label": "Node.js（构建静态站）"},
	{"lang": "java-maven", "label": "Java (Maven)"},
	{"lang": "java-gradle", "label": "Java (Gradle)"},
	{"lang": "python", "label": "Python"},
	{"lang": "php", "label": "PHP"},
	{"lang": "static", "label": "静态站点"},
}

var srcLangSet = func() map[string]bool {
	m := map[string]bool{}
	for _, l := range SrcLangs {
		m[l["lang"]] = true
	}
	return m
}()

// srcLangByMarker 标记 → 语言建议（与 agent markPriority 对应，标记同名同义）。
var srcLangByMarker = map[string]string{
	"Dockerfile":       "dockerfile",
	"go.mod":           "go",
	"pom.xml":          "java-maven",
	"build.gradle":     "java-gradle",
	"build.gradle.kts": "java-gradle",
	"package.json":     "node",
	"requirements.txt": "python",
	"pyproject.toml":   "python",
	"composer.json":    "php",
	"index.php":        "php",
	"index.html":       "static",
}

type src2LangDefault struct {
	port     int
	startCmd string
	version  string
}

// src2Defaults 各语言默认容器端口/启动命令/版本。
var src2Defaults = map[string]src2LangDefault{
	"go":          {8080, "", "1.23"},
	"node":        {3000, "npm start", "22"},
	"node-build":  {80, "", "22"},
	"java-maven":  {8080, "", "21"},
	"java-gradle": {8080, "", "21"},
	"python":      {8000, "python main.py", "3.12"},
	"php":         {80, "", ""},
	"static":      {80, "", ""},
	"dockerfile":  {8080, "", ""},
}

type Src2ComposeService struct {
	Nodes *NodeService
	Tasks *TaskService
	Creds *GitCredService
	nodeClient *agentclient.Client // WithNode 绑定（M57）
}

// srcSuggestion preview 候选（lang/默认值已补全，向导直接可用）。
type srcSuggestion struct {
	Dir       string   `json:"dir"`
	Marker    string   `json:"marker"`
	Lang      string   `json:"lang"`
	Version   string   `json:"version,omitempty"`
	StartCmd  string   `json:"startCmd,omitempty"`
	BuildCmd  string   `json:"buildCmd,omitempty"` // package.json scripts.build
	PkgMgr    string   `json:"pkgMgr,omitempty"`   // pnpm/yarn/npm
	Packaging string   `json:"packaging,omitempty"` // maven pom=多模块父/jar
	Modules   []string `json:"modules,omitempty"`   // maven 模块列表
	Port      int      `json:"port"`                // 建议容器端口
	HostPort  int      `json:"hostPort"`            // 建议宿主端口（随机高位，避免撞车）
	Service   string   `json:"service"`             // 建议服务名（目录名/app）
}

func (s *Src2ComposeService) agentClient() (*agentclient.Client, error) {
	return s.agentClientFor("local")
}

// agentClientFor 按节点路由（M57：clone/build 在目标节点执行）。
func (s *Src2ComposeService) agentClientFor(nodeId string) (*agentclient.Client, error) {
	node, err := s.Nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

// PreviewStream 流式预检：分步回传进度（clone → 检出 → 检测 → 清理），logf 由 SSE handler 提供。
// 克隆期间每 5s 上报已用时，消除"黑箱"等待。
func (s *Src2ComposeService) PreviewStream(ctx context.Context, req dto.Src2ComposePreviewReq, logf func(text string)) (dto.Src2ComposeDetectResp, []srcSuggestion, error) {
	if err := checkGitURLLight(req.GitURL); err != nil {
		return dto.Src2ComposeDetectResp{}, nil, err
	}
	ac, err := s.agentClientFor(normalizeNodeID(req.NodeID))
	if err != nil {
		return dto.Src2ComposeDetectResp{}, nil, err
	}
	logf("校验地址：" + req.GitURL)
	a, credName, err := s.resolveAuth(ctx, req.GitURL, req.CredentialID)
	if err != nil {
		return dto.Src2ComposeDetectResp{}, nil, err
	}
	if credName != "" {
		logf("匹配到凭据：" + credName)
	} else {
		logf("未匹配到凭据，匿名克隆")
	}
	req.Token, req.Username, req.PrivateKey = a.token, a.username, a.key

	logf("开始克隆（--depth 1" + orDefaultStr(req.Branch, "，分支 默认") + "）…")
	cloneDone := make(chan error, 1)
	var tmpDir, commit string
	go func() {
		out, cerr := agentclient.DoJSON[dto.Src2ComposeCloneTmpReq, dto.Src2ComposeCloneTmpResp](ac, ctx, "POST", "/agent/v1/src2compose/clone-tmp", &dto.Src2ComposeCloneTmpReq{
			GitURL: req.GitURL, Branch: req.Branch, Token: a.token, Username: a.username, PrivateKey: a.key,
		})
		if out != nil {
			tmpDir, commit = out.Dir, out.Commit
		}
		cloneDone <- cerr
	}()
	start := time.Now()
	tk := time.NewTicker(5 * time.Second)
	defer tk.Stop()
	for done := false; !done; {
		select {
		case err := <-cloneDone:
			if err != nil {
				return dto.Src2ComposeDetectResp{}, nil, errs.Wrapc(errs.CodeBadRequest, "克隆失败: "+err.Error())
			}
			done = true
		case <-tk.C:
			logf(fmt.Sprintf("克隆中… 已用时 %ds", int(time.Since(start).Seconds())))
		case <-ctx.Done():
			return dto.Src2ComposeDetectResp{}, nil, ctx.Err()
		}
	}
	logf("已检出 commit " + orDefaultStr(commit, "未知"))

	// 无论后续成败都清理临时克隆（exec 通道；目录前缀白名单校验后执行）
	defer func() {
		if tmpDir == "" || !strings.HasPrefix(tmpDir, "/tmp/yp-src2-") {
			return
		}
		_, _ = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, context.Background(), "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: "rm -rf " + tmpDir, TimeoutSecs: 30})
	}()

	logf("扫描语言标记…")
	v := url.Values{}
	v.Set("dir", tmpDir)
	det, err := agentclient.GetJSON[dto.Src2ComposeDetectResp](ac, ctx, "/agent/v1/src2compose/detect?"+v.Encode())
	if err != nil {
		return dto.Src2ComposeDetectResp{}, nil, err
	}
	for _, it := range det.Items {
		dir := it.Dir
		if dir == "" {
			dir = "."
		}
		logf(fmt.Sprintf("  %s → %s（%s）", dir, it.Marker, srcLangByMarker[it.Marker]))
	}
	if len(det.Items) == 0 {
		logf("未发现可构建的语言标记（Dockerfile/go.mod/package.json 等）")
	}
	sugg := s.suggest(det.Items)
	logf(fmt.Sprintf("检测完成，共 %d 个候选", len(sugg)))
	// commit 取本地克隆结果（detect 端点响应不含 commit）
	return dto.Src2ComposeDetectResp{Commit: commit, Items: det.Items}, sugg, nil
}

// Create 校验并启动构建任务。
func (s *Src2ComposeService) Create(ctx context.Context, req dto.Src2ComposeBuildReq) (dto.Src2ComposeBuildResp, error) {
	if err := s.validateReq(req); err != nil {
		return dto.Src2ComposeBuildResp{}, err
	}
	ac, err := s.agentClientFor(normalizeNodeID(req.NodeID))
	if err != nil {
		return dto.Src2ComposeBuildResp{}, err
	}
	a, _, err := s.resolveAuth(ctx, req.GitURL, req.CredentialID)
	if err != nil {
		return dto.Src2ComposeBuildResp{}, err
	}
	task, err := s.Tasks.StartTask(Src2ComposeTaskType, "源码构建 "+req.Name, req.Name, 30*time.Minute,
		func(tctx context.Context, logf TaskLogf) error {
			return s.runBuild(tctx, logf, ac, req, a)
		})
	if err != nil {
		return dto.Src2ComposeBuildResp{}, err
	}
	return dto.Src2ComposeBuildResp{TaskID: task.ID}, nil
}

func (s *Src2ComposeService) validateReq(req dto.Src2ComposeBuildReq) error {
	if !src2NameRe.MatchString(req.Name) {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "项目名不合法（字母/数字/中划线/下划线，字母或数字开头，≤64 位）")
	}
	if err := checkGitURLLight(req.GitURL); err != nil {
		return err
	}
	if len(req.Services) == 0 || len(req.Services) > 8 {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "服务数量需为 1~8 个")
	}
	seenSvc := map[string]bool{}
	for _, svc := range req.Services {
		if !src2NameRe.MatchString(svc.Name) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "服务名不合法: "+svc.Name)
		}
		if seenSvc[svc.Name] {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "服务名重复: "+svc.Name)
		}
		seenSvc[svc.Name] = true
		if !srcLangSet[svc.Lang] {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的语言栈: "+svc.Lang)
		}
		if err := checkRelDir(svc.Dir); err != nil {
			return err
		}
		if svc.HostPort < 0 || svc.HostPort > 65535 || svc.ContainerPort < 0 || svc.ContainerPort > 65535 {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "端口需在 0~65535 范围内")
		}
		// static/php/node-build 容器端口由镜像固定（80），缺省自动补全；代码类服务发布端口时必须明确容器端口
		if svc.HostPort > 0 && svc.ContainerPort == 0 && svc.Lang != "static" && svc.Lang != "php" && svc.Lang != "node-build" {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "发布端口时容器端口不能为 0（服务 "+svc.Name+"）")
		}
		if len(svc.StartCmd) > 500 {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "启动命令过长（服务 "+svc.Name+"）")
		}
		if len(svc.BuildCmd) > 500 {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "构建命令过长（服务 "+svc.Name+"）")
		}
		if err := checkRelDir(svc.Module); err != nil {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "maven 模块路径不合法（服务 "+svc.Name+"）：相对构建目录的模块路径")
		}
		if err := checkRelDir(svc.DistDir); err != nil {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "产物目录不合法（服务 "+svc.Name+"）")
		}
	}
	return nil
}

// checkGitURLLight core 侧轻校验（agent 为权威校验：白名单 + 禁 ext::/前导 -）。
func checkGitURLLight(u string) error {
	u = strings.TrimSpace(u)
	if u == "" || strings.HasPrefix(u, "-") || strings.HasPrefix(u, "ext::") {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "非法的 git 地址")
	}
	if HostOfGitURL(u) == "" {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "无法从 git 地址解析主机名")
	}
	return nil
}

// checkRelDir 相对构建目录白名单：禁绝对路径/穿越/反斜杠/空白，段内字符受限。
func checkRelDir(d string) error {
	if d == "" {
		return nil
	}
	if strings.HasPrefix(d, "/") || strings.Contains(d, "\\") || strings.Contains(d, "..") {
		return errs.New(errs.CodeBadRequest, "error.badRequest", "构建目录需为仓库内相对路径: "+d)
	}
	for _, seg := range strings.Split(d, "/") {
		if seg == "" || !src2DirRe.MatchString(seg) {
			return errs.New(errs.CodeBadRequest, "error.badRequest", "构建目录段不合法: "+seg)
		}
	}
	return nil
}

// resolvedAuth 凭据解析结果（secret 仅内存传递，不进日志；credName 仅供进度提示）。
type resolvedAuth struct {
	token, username, key, credName string
}

// resolveAuth 项目级 credentialId 优先，其次按 host 自动匹配；两者皆无则匿名。
// credName 仅供进度提示（不含 secret）。
func (s *Src2ComposeService) resolveAuth(ctx context.Context, gitURL string, credentialID uint) (resolvedAuth, string, error) {
	var cred *model.GitCredential
	var err error
	if credentialID > 0 {
		cred, err = s.Creds.GetByID(ctx, credentialID)
		if err != nil {
			return resolvedAuth{}, "", err
		}
	} else {
		cred, err = s.Creds.Match(ctx, gitURL)
		if err != nil {
			return resolvedAuth{}, "", err
		}
	}
	if cred == nil {
		return resolvedAuth{}, "", nil
	}
	if cred.Type == "ssh" {
		return resolvedAuth{key: cred.Secret}, cred.Name + "（ssh）", nil
	}
	return resolvedAuth{token: cred.Secret, username: cred.Username}, cred.Name + "（host " + cred.Host + "）", nil
}

func (s *Src2ComposeService) suggest(items []dto.Src2ComposeDetectItem) []srcSuggestion {
	out := make([]srcSuggestion, 0, len(items))
	usedPort := map[int]bool{}
	for _, it := range items {
		lang := srcLangByMarker[it.Marker]
		def := src2Defaults[lang]
		svc := it.Dir
		if svc == "" {
			svc = "app"
		}
		// Maven 多模块父 POM / 带构建脚本的前端：给出默认构建路径
		s := srcSuggestion{
			Dir: it.Dir, Marker: it.Marker, Lang: lang,
			Version: it.Facts.Version, StartCmd: it.Facts.StartCmd,
			BuildCmd: it.Facts.BuildCmd, PkgMgr: it.Facts.PkgMgr,
			Packaging: it.Facts.Packaging, Modules: it.Facts.Modules,
			Port: def.port, HostPort: randomHighPort(usedPort), Service: sanitizeServiceName(svc),
		}
		if lang == "node" && it.Facts.BuildCmd != "" && it.Facts.StartCmd == "" {
			// 只有构建脚本：默认按静态站构建
			s.Lang = "node-build"
			s.Port = 80
		}
		out = append(out, s)
	}
	return out
}

func randomHighPort(used map[int]bool) int {
	for i := 0; i < 20; i++ {
		p := 32768 + rand.Intn(61000-32768)
		if !used[p] {
			used[p] = true
			return p
		}
	}
	return 32768
}

func sanitizeServiceName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, s)
	if !src2NameRe.MatchString(s) {
		return "app"
	}
	return s
}

// runWithHeartbeat 执行 fn，期间每 10s 通过 logf 上报已用时（消除长步骤的黑箱等待）。
func runWithHeartbeat(ctx context.Context, logf TaskLogf, text string, fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	start := time.Now()
	tk := time.NewTicker(10 * time.Second)
	defer tk.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-tk.C:
			logf("info", "%s… 已用时 %ds", text, int(time.Since(start).Seconds()))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runBuild 任务主体：克隆 → 检测复核 → 生成编排 → 构建 → 启动。
func (s *Src2ComposeService) runBuild(ctx context.Context, logf TaskLogf, ac *agentclient.Client, req dto.Src2ComposeBuildReq, a resolvedAuth) error {
	projDir := src2Root + "/" + req.Name

	logf("step", "克隆源码 %s（分支 %s）", req.GitURL, orDefaultStr(req.Branch, "默认"))
	var cloneOut *struct {
		Commit string `json:"commit"`
	}
	err := runWithHeartbeat(ctx, logf, "克隆中", func() error {
		out, cerr := agentclient.DoJSON[dto.Src2ComposeCloneReq, struct {
			Commit string `json:"commit"`
		}](ac, ctx, "POST", "/agent/v1/src2compose/clone", &dto.Src2ComposeCloneReq{
			Name: req.Name, GitURL: req.GitURL, Branch: req.Branch,
			Token: a.token, Username: a.username, PrivateKey: a.key,
		})
		cloneOut = out
		return cerr
	})
	if err != nil {
		return errs.Wrapc(errs.CodeInternal, "克隆失败: "+err.Error())
	}
	logf("info", "已检出 commit %s", orDefaultStr(cloneOut.Commit, "未知"))

	logf("step", "语言栈复核")
	v := url.Values{}
	v.Set("dir", projDir+"/src")
	if det, derr := agentclient.GetJSON[dto.Src2ComposeDetectResp](ac, ctx, "/agent/v1/src2compose/detect?"+v.Encode()); derr == nil {
		for _, it := range det.Items {
			dir := it.Dir
			if dir == "" {
				dir = "."
			}
			logf("info", "  %s → %s（%s）", dir, it.Marker, srcLangByMarker[it.Marker])
		}
	} else {
		logf("warn", "  复核跳过：%v", derr)
	}

	logf("step", "生成编排（Dockerfile 模板 + compose.yaml）")
	for _, svc := range req.Services {
		if svc.Lang == "dockerfile" {
			logf("info", "  %s：使用仓库自带 Dockerfile", svc.Name)
			continue
		}
		content, rerr := renderSrc2Dockerfile(svc)
		if rerr != nil {
			return rerr
		}
		p := projDir + "/src"
		if svc.Dir != "" {
			p += "/" + svc.Dir
		}
		p += "/Dockerfile.ypanel"
		if werr := s.writeRemote(ctx, ac, p, content); werr != nil {
			return errs.Wrapc(errs.CodeInternal, "写入 "+p+" 失败: "+werr.Error())
		}
		logf("info", "  写入 %s", p)
	}
	if werr := s.writeRemote(ctx, ac, projDir+"/compose.yaml", renderComposeYAML(req.Name, cloneOut.Commit, req.Services)); werr != nil {
		return errs.Wrapc(errs.CodeInternal, "写入 compose.yaml 失败: "+werr.Error())
	}
	logf("info", "  写入 %s/compose.yaml", projDir)

	logf("step", "构建镜像（可能耗时较长，超时 25 分钟）")
	// 构建走 agent exec 通道（商店安装/B20 构建同款路径），输出先落文件再回读尾部：
	// 此 compose 版本在部分父进程环境下进度缓冲会在退出时丢尾部（daemon 侧 solve 实际
	// 完整、exit 1）。失败时保留 build.log 供完整排查。
	var execOut *dto.ExecResp
	err = runWithHeartbeat(ctx, logf, "构建中", func() error {
		out, cerr := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{
				Command: fmt.Sprintf("cd %s && docker compose --progress plain --file compose.yaml --project-name %s build > build.log 2>&1; ec=$?; echo BUILD_EXIT=$ec; tail -c 1600 build.log",
					projDir, req.Name),
				TimeoutSecs: 1500,
			})
		execOut = out
		return cerr
	})
	if err != nil {
		return errs.Wrapc(errs.CodeInternal, "构建执行失败: "+err.Error())
	}
	// 构建退出码取 BUILD_EXIT 标记（整条 sh 的退出码是 tail 的，不可用）
	m := buildExitRe.FindStringSubmatch(execOut.Output)
	buildExit := 1
	if m != nil {
		buildExit, _ = strconv.Atoi(m[1])
	}
	if buildExit != 0 {
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed",
			"构建失败（exit "+strconv.Itoa(buildExit)+"），完整日志见 "+projDir+"/build.log：\n"+tailStr(execOut.Output, 1600))
	}
	// 成功后清理构建日志（保留 compose.yaml/Dockerfile.ypanel 供重建）
	_, _ = agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: "rm -f " + projDir + "/build.log", TimeoutSecs: 15})
	logf("info", "%s", tailStr(execOut.Output, 700))

	logf("step", "启动服务")
	upOut, err := agentclient.DoJSON[dto.ComposeActionReq, map[string]string](ac, ctx, "POST", "/agent/v1/compose/up", &dto.ComposeActionReq{Name: req.Name})
	if err != nil {
		return errs.Wrapc(errs.CodeInternal, "启动失败: "+err.Error())
	}
	if upOut != nil {
		logf("info", "%s", tailStr((*upOut)["output"], 600))
	}
	logf("step", "完成：项目 %s 已创建，可进入应用详情查看服务拓扑", req.Name)
	return nil
}

func (s *Src2ComposeService) writeRemote(ctx context.Context, ac *agentclient.Client, p, content string) error {
	_, err := agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: p, Content: content})
	return err
}

// renderSrc2Dockerfile 渲染语言模板（{{VAR}} 替换；版本值过白名单防注入）。
func renderSrc2Dockerfile(svc dto.Src2ComposeServiceSpec) (string, error) {
	b, err := src2composeFS.ReadFile("src2composetpl/" + svc.Lang + "/Dockerfile.tpl")
	if err != nil {
		return "", errs.New(errs.CodeBadRequest, "error.badRequest", "不支持的语言栈: "+svc.Lang)
	}
	def := src2Defaults[svc.Lang]
	port := svc.ContainerPort
	if port <= 0 {
		port = def.port
	}
	// maven 多模块：-pl <module> -am 只构建目标模块及依赖；jar 从模块目录取
	module := strings.Trim(svc.Module, "/")
	mavenArgs := ""
	jarDir := "."
	if module != "" {
		mavenArgs = " -pl " + module + " -am"
		jarDir = module
	}
	vars := map[string]string{
		"PORT":              strconv.Itoa(port),
		"GO_VERSION":        safeVersion(svc.Version, def.version),
		"NODE_VERSION":      safeVersion(nodeMajor(svc.Version), def.version),
		"JAVA_VERSION":      safeVersion(svc.Version, def.version),
		"PY_VERSION":        safeVersion(svc.Version, def.version),
		"START_CMD":         orDefaultStr(svc.StartCmd, def.startCmd),
		"MAVEN_MODULE_ARGS": mavenArgs,
		"JAR_DIR":           jarDir,
		"BUILD_CMD":         orDefaultStr(svc.BuildCmd, nodeBuildDefaultCmd),
		"DIST_DIR":          orDefaultStr(strings.Trim(svc.DistDir, "/"), "dist"),
	}
	tpl, err := template.New("d").Parse(string(b))
	if err != nil {
		return "", errs.Wrapc(errs.CodeInternal, "模板解析失败: "+err.Error())
	}
	var sb strings.Builder
	if err := tpl.Execute(&sb, vars); err != nil {
		return "", errs.Wrapc(errs.CodeInternal, "模板渲染失败: "+err.Error())
	}
	return sb.String(), nil
}

// nodeBuildDefaultCmd 未指定构建命令时按 lockfile 推导（与安装步骤同套判定）。
const nodeBuildDefaultCmd = `if [ -f pnpm-lock.yaml ]; then pnpm build; elif [ -f yarn.lock ]; then yarn build; else npm run build; fi`

// safeVersion 版本白名单（进 Dockerfile 的 FROM 行，防注入）；不合法回退默认。
func safeVersion(v, def string) string {
	if src2VerRe.MatchString(v) {
		return v
	}
	return def
}

// nodeMajor engines.node 常为 ">=18"、"20.x"，取首个数字段。
func nodeMajor(v string) string {
	m := regexp.MustCompile(`\d+`).FindString(v)
	if m == "" {
		return "22"
	}
	return m
}

// renderComposeYAML 生成多服务 compose（镜像 tag=commit，天然支持后续回滚）。
func renderComposeYAML(name, commit string, svcs []dto.Src2ComposeServiceSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\nservices:\n", name)
	tag := commit
	if tag == "" {
		tag = "latest"
	}
	for _, svc := range svcs {
		fmt.Fprintf(&b, "  %s:\n", svc.Name)
		ctxDir := "./src"
		if svc.Dir != "" {
			ctxDir = "./src/" + svc.Dir
		}
		df := "Dockerfile.ypanel"
		if svc.Lang == "dockerfile" {
			df = "Dockerfile"
		}
		fmt.Fprintf(&b, "    build:\n      context: %s\n      dockerfile: %s\n", ctxDir, df)
		fmt.Fprintf(&b, "    image: ypanel-apps/%s-%s:%s\n", name, svc.Name, tag)
		fmt.Fprintf(&b, "    restart: unless-stopped\n")
		cport := svc.ContainerPort
		if cport <= 0 {
			cport = src2Defaults[svc.Lang].port
		}
		if svc.HostPort > 0 {
			fmt.Fprintf(&b, "    ports:\n      - \"%d:%d\"\n", svc.HostPort, cport)
		}
		switch svc.Lang {
		case "go", "node", "python", "java-maven", "java-gradle":
			fmt.Fprintf(&b, "    environment:\n      - PORT=%d\n", cport)
		}
	}
	return b.String()
}

func tailStr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

func orDefaultStr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
