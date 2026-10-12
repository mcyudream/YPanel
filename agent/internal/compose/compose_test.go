package compose

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ypanel/shared/dto"
)

type fakeLister struct{ items []dto.ContainerItem }

func (f fakeLister) List(context.Context) ([]dto.ContainerItem, error) { return f.items, nil }

// 目录无白名单文件时，从容器 config_files label 反查真实编排文件（-f 别处/文件被挪走场景）。
func TestResolveProjectConfigFilesLabelFallback(t *testing.T) {
	base := t.TempDir()
	realDir := t.TempDir()
	realFile := filepath.Join(realDir, "my-compose.custom.yaml")
	if err := os.WriteFile(realFile, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 外部目录存在但里面没有任何白名单文件
	extDir := t.TempDir()

	m, err := New(base, fakeLister{items: []dto.ContainerItem{{
		Name: "ext-app-1",
		Labels: map[string]string{
			composeProjectLabel:     "ext-app",
			composeWorkdirLabel:     extDir,
			composeConfigFilesLabel: realFile + ", " + realDir + "/override.yaml",
		},
	}}})
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := m.resolveProject(context.Background(), "ext-app", extDir)
	if err != nil {
		t.Fatalf("期望 label 反查命中，实际报错: %v", err)
	}
	if cfg != realFile {
		t.Fatalf("期望取主文件 %s，实际 %s", realFile, cfg)
	}
}

// label 缺失/文件不存在时保持原报错（1002 资源不存在语义不变）。
func TestResolveProjectStillNotFoundWithoutLabel(t *testing.T) {
	base := t.TempDir()
	extDir := t.TempDir()

	m, err := New(base, fakeLister{items: []dto.ContainerItem{{
		Name:   "broken-1",
		Labels: map[string]string{composeProjectLabel: "broken", composeWorkdirLabel: extDir},
	}}})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.resolveProject(context.Background(), "broken", extDir); err == nil {
		t.Fatal("期望报错（label 无 config_files），实际命中")
	}
}
