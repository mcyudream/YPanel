package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ypanel/core/internal/model"
)

// gitCheckoutFollowsURLChange 回归：yp-git 源编辑 URL 后复用克隆缓存，
// gitCheckout 必须先同步 origin 指向再 fetch，否则永远拉旧源。
func TestGitCheckoutFollowsURLChange(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoA := makeFileRepo(t, filepath.Join(root, "repoA"), "A")
	repoB := makeFileRepo(t, filepath.Join(root, "repoB"), "B")

	s := &StoreService{dir: filepath.Join(root, "cache")}
	src := &model.AppStoreSource{ID: 1, Type: "yp-git", URL: repoA, Branch: "main"}
	if _, err := s.gitCheckout(ctx, src); err != nil {
		t.Fatalf("首次检出失败: %v", err)
	}

	src.URL = repoB
	dir, err := s.gitCheckout(ctx, src)
	if err != nil {
		t.Fatalf("地址变更后再检出失败: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "B" {
		t.Fatalf("检出内容仍来自旧源: got %q, want %q", got, "B")
	}
}

// makeFileRepo 建一个只含 index.json（内容=marker）的本地 git 仓库，返回 file:// URL。
func makeFileRepo(t *testing.T, dir, marker string) string {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	run("init", "-b", "main", dir)
	run("-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "add", "-A")
	run("-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-m", "init")
	p := filepath.ToSlash(dir)
	if strings.HasPrefix(p, "/") {
		return "file://" + p
	}
	return "file:///" + p
}
