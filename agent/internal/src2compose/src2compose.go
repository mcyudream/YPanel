// Package src2compose 源码构建支撑：git 克隆（安全：URL 白名单、凭据走 argv/env、禁交互）
// 与语言标记检测（根目录 + 深度≤2 子目录，monorepo 多候选）。
package src2compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// markPriority 目录内标记优先级（同目录多标记只报最高：Dockerfile > go.mod > ...）。
var markPriority = []string{
	"Dockerfile", "go.mod", "pom.xml", "build.gradle", "build.gradle.kts",
	"package.json", "requirements.txt", "pyproject.toml", "composer.json", "index.php", "index.html",
}

// skipDirs 检测遍历跳过的目录（依赖/产物，避免 node_modules 里翻出海量假候选）。
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true, "dist": true,
	"target": true, "build": true, ".next": true, "__pycache__": true,
}

var (
	gitURLPattern   = regexp.MustCompile(`^(https?://|ssh://|git@|file:///)[^\s]+$`)
	goModRe         = regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
	pomJavaVerRe    = regexp.MustCompile(`<(?:java\.version|maven\.compiler\.(?:release|target))>(\d+)</`)
	pomPackagingRe  = regexp.MustCompile(`<packaging>\s*pom\s*</packaging>`)
	pomModuleRe     = regexp.MustCompile(`<module>([^<]+)</module>`)
	maxScanFilesize = 1 << 20 // 标记文件内容提取上限 1MB
	maxItems        = 32      // 候选目录上限
	maxDepth        = 2       // 子目录扫描深度
)

// ValidateGitURL http(s)/ssh/git@/file 白名单；禁 ext:: 传输协议与 "-" 开头参数形态。
func ValidateGitURL(u string) error {
	u = strings.TrimSpace(u)
	if u == "" || strings.HasPrefix(u, "-") || strings.HasPrefix(u, "ext::") {
		return errs.Wrap(errs.ErrBadRequest, "非法的 git 地址")
	}
	if !gitURLPattern.MatchString(u) {
		return errs.Wrap(errs.ErrBadRequest, "git 地址需为 https:// 或 git@ 形式")
	}
	return nil
}

// GitAuth 克隆凭据：https 账密/token 或 ssh 私钥（core 凭据库匹配后下发，不落日志）。
type GitAuth struct {
	Token      string // https：作为密码注入 URL（用户名取 Username，空则 ypanel）
	Username   string
	PrivateKey string // ssh：写临时 key 文件，GIT_SSH_COMMAND 指定
}

// env 组装 git 环境变量（禁交互 + ssh key）。
func (a GitAuth) env() ([]string, func(), error) {
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if strings.TrimSpace(a.PrivateKey) == "" {
		return env, func() {}, nil
	}
	f, err := os.CreateTemp("", "yp-gitkey-*")
	if err != nil {
		return nil, nil, errs.Wrapc(errs.CodeFileOpFailed, "写入临时 ssh key 失败: "+err.Error())
	}
	if err := os.WriteFile(f.Name(), []byte(a.PrivateKey), 0o600); err != nil {
		_ = os.Remove(f.Name())
		return nil, nil, errs.Wrapc(errs.CodeFileOpFailed, "写入临时 ssh key 失败: "+err.Error())
	}
	cleanup := func() { _ = os.Remove(f.Name()) }
	ssh := "ssh -i " + f.Name() + " -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new"
	return append(env, "GIT_SSH_COMMAND="+ssh), cleanup, nil
}

// InjectToken https 地址内嵌访问凭据（argv 直传不经 shell，不落日志）。
func InjectToken(raw, username, token string) string {
	if token == "" || !strings.HasPrefix(raw, "http") {
		return raw
	}
	if username == "" {
		username = "ypanel"
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	scheme := "https://"
	if !strings.HasPrefix(raw, "https://") {
		scheme = "http://"
	}
	return scheme + username + ":" + token + "@" + rest
}

// maskToken 报错信息脱敏：proto://user:***@host。
func maskToken(s string) string {
	if i := strings.Index(s, "@"); i >= 0 {
		if j := strings.LastIndex(s[:i], ":"); j >= 0 && strings.Contains(s[:j], "//") {
			return s[:j+1] + "***" + s[i:]
		}
	}
	return s
}

// gitRun 参数数组直调 git，禁交互；返回 stderr 便于报错。
func gitRun(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = env
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", maskToken(msg))
	}
	return nil
}

// checkout 克隆或更新仓库到 dest，返回 commit。已存在 .git 走 fetch+reset。
func checkout(ctx context.Context, gitURL, branch string, a GitAuth, dest string) (string, error) {
	if err := ValidateGitURL(gitURL); err != nil {
		return "", err
	}
	url := InjectToken(gitURL, a.Username, a.Token)
	env, cleanup, err := a.env()
	if err != nil {
		return "", err
	}
	defer cleanup()

	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		ref := branch
		if ref == "" {
			ref = "HEAD"
		}
		if err := gitRun(ctx, dest, env, "fetch", "--depth", "1", "origin", ref); err != nil {
			return "", errs.Wrap(errs.ErrBadRequest, "源仓库 fetch 失败: "+err.Error())
		}
		if err := gitRun(ctx, dest, env, "reset", "--hard", "FETCH_HEAD"); err != nil {
			return "", errs.Wrap(errs.ErrBadRequest, "源仓库更新失败: "+err.Error())
		}
	} else {
		args := []string{"clone", "--depth", "1"}
		if branch != "" {
			args = append(args, "-b", branch)
		}
		args = append(args, url, dest)
		if err := gitRun(ctx, "", env, args...); err != nil {
			return "", errs.Wrap(errs.ErrBadRequest, "源仓库 clone 失败: "+err.Error())
		}
	}
	var out strings.Builder
	cmd := exec.CommandContext(ctx, "git", "-C", dest, "rev-parse", "--short=12", "HEAD")
	cmd.Stdout = &out
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return "", nil
	}
	return strings.TrimSpace(out.String()), nil
}

// CloneTmp 克隆到新建临时目录（/tmp/yp-src2-*），返回目录与 commit；清理由调用方负责
//（core 流式预检：clone → detect → exec rm，分步回传进度）。
func CloneTmp(ctx context.Context, gitURL, branch string, a GitAuth) (dir, commit string, err error) {
	tmp, err := os.MkdirTemp("", "yp-src2-*")
	if err != nil {
		return "", "", err
	}
	commit, err = checkout(ctx, gitURL, branch, a, tmp)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return "", "", err
	}
	return tmp, commit, nil
}

// CloneTo 克隆到指定项目 src 目录（宿主通道：compose 托管目录）。
func CloneTo(ctx context.Context, gitURL, branch string, a GitAuth, dest string) (commit string, err error) {
	return checkout(ctx, gitURL, branch, a, dest)
}

// Detect 扫描仓库根 + 深度≤2 子目录，每个目录报最高优先级标记与内容线索。
func Detect(repoDir string) []dto.Src2ComposeDetectItem {
	items := make([]dto.Src2ComposeDetectItem, 0, 4)
	seen := map[string]bool{}
	var walk func(dir string, rel string, depth int)
	walk = func(dir string, rel string, depth int) {
		if len(items) >= maxItems {
			return
		}
		if m, f, ok := detectDir(dir); ok {
			if !seen[rel] {
				seen[rel] = true
				items = append(items, dto.Src2ComposeDetectItem{Dir: rel, Marker: m, Facts: f})
			}
		}
		if depth >= maxDepth {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || skipDirs[name] || strings.HasPrefix(name, ".") {
				continue
			}
			childRel := name
			if rel != "" {
				childRel = rel + "/" + name
			}
			walk(filepath.Join(dir, name), childRel, depth+1)
		}
	}
	walk(repoDir, "", 0)
	return items
}

// detectDir 单目录：按优先级找标记并提取线索。
func detectDir(dir string) (marker string, facts dto.Src2ComposeFacts, ok bool) {
	for _, m := range markPriority {
		st, err := os.Stat(filepath.Join(dir, m))
		if err != nil || st.IsDir() {
			continue
		}
		return m, extractFacts(dir, m), true
	}
	return "", facts, false
}

func readSmall(path string) []byte {
	st, err := os.Stat(path)
	if err != nil || st.Size() > int64(maxScanFilesize) {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func extractFacts(dir, marker string) dto.Src2ComposeFacts {
	var f dto.Src2ComposeFacts
	switch marker {
	case "go.mod":
		if m := goModRe.FindSubmatch(readSmall(filepath.Join(dir, "go.mod"))); m != nil {
			f.Version = string(m[1])
		}
	case "pom.xml":
		b := readSmall(filepath.Join(dir, "pom.xml"))
		if m := pomJavaVerRe.FindSubmatch(b); m != nil {
			f.Version = string(m[1])
		}
		// 多模块判定：packaging=pom + <module> 列表；boot 判定看 spring-boot 插件
		if pomPackagingRe.FindSubmatch(b) != nil {
			f.Packaging = "pom"
			for _, mm := range pomModuleRe.FindAllSubmatch(b, -1) {
				f.Modules = append(f.Modules, string(mm[1]))
			}
		} else {
			f.Packaging = "jar"
		}
		f.Boot = strings.Contains(string(b), "spring-boot-maven-plugin")
	case "package.json":
		var pkg struct {
			Engines struct {
				Node string `json:"node"`
			} `json:"engines"`
			Scripts struct {
				Start string `json:"start"`
				Build string `json:"build"`
			} `json:"scripts"`
		}
		if b := readSmall(filepath.Join(dir, "package.json")); b != nil && json.Unmarshal(b, &pkg) == nil {
			f.Version = pkg.Engines.Node
			f.StartCmd = pkg.Scripts.Start
			f.BuildCmd = pkg.Scripts.Build
		}
		// 包管理器按 lockfile 识别
		switch {
		case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
			f.PkgMgr = "pnpm"
		case fileExists(filepath.Join(dir, "yarn.lock")):
			f.PkgMgr = "yarn"
		default:
			f.PkgMgr = "npm"
		}
	}
	return f
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
