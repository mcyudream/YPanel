package compose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// 用命令替身捕获真正传给 CLI 的 argv，避免测试触碰已有容器。
func TestServiceActionProjectConfigAndFailure(t *testing.T) {
	base, external, configDir, bin := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	capture := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$YPANEL_TEST_ARGS\"\nprintf '%s' 'service output'\nexit \"${YPANEL_TEST_EXIT:-0}\"\n"
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("YPANEL_TEST_ARGS", capture)
	managed := filepath.Join(base, "demo")
	if err := os.Mkdir(managed, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{managed, external} {
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	main, override := filepath.Join(configDir, "custom main.yaml"), filepath.Join(configDir, "override.yaml")
	for _, file := range []string{main, override} {
		if err := os.WriteFile(file, []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, dir, workDir string
		files              []string
		labels             bool
	}{
		{name: "托管项目", workDir: managed, files: []string{filepath.Join(managed, "compose.yaml")}},
		{name: "外部项目", dir: external, workDir: external, files: []string{filepath.Join(external, "compose.yaml")}},
		{name: "外部多文件", dir: external, workDir: external, labels: true, files: []string{main, override}},
		{name: "旧请求按标签反查", workDir: external, labels: true, files: []string{main, override}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var lister DockerLister
			if tc.labels {
				lister = fakeLister{items: []dto.ContainerItem{{Labels: map[string]string{
					composeProjectLabel: "demo", composeWorkdirLabel: external, composeConfigFilesLabel: main + ", " + override,
				}}}}
			}
			m, err := New(base, lister)
			if err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{"start", "stop", "restart", "pull", "up"} {
				out, err := m.ServiceAction(context.Background(), "demo", "web", action, tc.dir)
				if err != nil || out != "service output" {
					t.Fatalf("操作失败: output=%q, err=%v", out, err)
				}
				want := []string{"compose", "--project-name", "demo", "--project-directory", tc.workDir}
				for _, file := range tc.files {
					want = append(want, "--file", file)
				}
				want = append(want, action)
				if action == "up" {
					want = append(want, "-d")
				}
				want = append(want, "web")
				args, err := os.ReadFile(capture)
				if err != nil || !reflect.DeepEqual(strings.Split(strings.TrimSuffix(string(args), "\n"), "\n"), want) {
					t.Fatalf("项目/目录/文件顺序错误: %s, want=%v, err=%v", args, want, err)
				}
			}
		})
	}
	m, err := New(base, fakeLister{items: []dto.ContainerItem{{Labels: map[string]string{
		composeProjectLabel: "demo", composeWorkdirLabel: external, composeConfigFilesLabel: main + ", " + override,
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(override); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(capture); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ServiceAction(context.Background(), "demo", "web", "restart", external); err == nil {
		t.Fatal("缺少 override 不得使用部分配置执行")
	}
	if _, err := os.Stat(capture); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("配置不完整时仍调用了 Docker")
	}
	m, err = New(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("YPANEL_TEST_EXIT", "17")
	out, err := m.ServiceAction(context.Background(), "demo", "web", "restart", "")
	var businessErr *errs.Error
	if out != "service output" || !errors.As(err, &businessErr) || businessErr.Code != errs.CodeFileOpFailed {
		t.Fatalf("非零退出码丢失: output=%q, err=%v", out, err)
	}
}
