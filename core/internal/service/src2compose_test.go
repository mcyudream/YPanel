package service

import (
	"strings"
	"testing"

	"github.com/ypanel/shared/dto"
)

func TestRenderComposeYAML(t *testing.T) {
	out := renderComposeYAML("my-app", "abc123def456", []dto.Src2ComposeServiceSpec{
		{Name: "web", Dir: "", Lang: "node", HostPort: 38080, ContainerPort: 3000, StartCmd: "npm start"},
		{Name: "api", Dir: "server", Lang: "java-maven", HostPort: 38081, ContainerPort: 8080},
		{Name: "web-static", Dir: "client", Lang: "static", HostPort: 38082},
	})
	for _, want := range []string{
		"name: my-app",
		"  web:",
		"context: ./src",
		"dockerfile: Dockerfile.ypanel",
		"image: ypanel-apps/my-app-web:abc123def456",
		"\"38080:3000\"",
		"context: ./src/server",
		"ypanel-apps/my-app-api:abc123def456",
		"\"38082:80\"", // static 容器端口默认 80
		"- PORT=3000",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("compose 输出缺少 %q\n---\n%s", want, out)
		}
	}
	// php/static 不应有 PORT 环境变量注入（容器固定 80 由镜像决定）
	if strings.Contains(out, "PORT=80\n") && strings.Count(out, "PORT=") != 1 {
		t.Fatalf("PORT 环境变量仅应注入代码类服务:\n%s", out)
	}
}

func TestRenderDockerfileVersions(t *testing.T) {
	out, err := renderSrc2Dockerfile(dto.Src2ComposeServiceSpec{Name: "a", Lang: "go", Version: "1.22", ContainerPort: 9000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "golang:1.22-alpine") || !strings.Contains(out, "EXPOSE 9000") {
		t.Fatalf("go 模板版本/端口未生效:\n%s", out)
	}
	// 非法版本回退默认（防 FROM 行注入）
	out, err = renderSrc2Dockerfile(dto.Src2ComposeServiceSpec{Name: "a", Lang: "go", Version: "1.22;rm -rf /", ContainerPort: 8080})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "golang:1.23-alpine") {
		t.Fatalf("非法版本应回退默认:\n%s", out)
	}
	// node engines ">=18" 取主版本
	out, _ = renderSrc2Dockerfile(dto.Src2ComposeServiceSpec{Name: "a", Lang: "node", Version: ">=18", ContainerPort: 3000, StartCmd: "npm run dev"})
	if !strings.Contains(out, "node:18-alpine") || !strings.Contains(out, "npm run dev") {
		t.Fatalf("node 版本提取/启动命令未生效:\n%s", out)
	}
}

func TestRenderMavenModuleAndNodeBuild(t *testing.T) {
	// maven 多模块：-pl 定向构建 + 从模块目录取 jar
	out, err := renderSrc2Dockerfile(dto.Src2ComposeServiceSpec{Name: "a", Lang: "java-maven", ContainerPort: 8080, Module: "yudream-application/yudream-bootstrap"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "mvn -q -DskipTests package -pl yudream-application/yudream-bootstrap -am") ||
		!strings.Contains(out, "/src/yudream-application/yudream-bootstrap/target/*.jar") {
		t.Fatalf("maven 多模块渲染不符:\n%s", out)
	}
	// node-build：默认构建命令按 lockfile 推导，产物目录默认 dist
	out, err = renderSrc2Dockerfile(dto.Src2ComposeServiceSpec{Name: "a", Lang: "node-build", ContainerPort: 80})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pnpm build") || !strings.Contains(out, "/app/dist/") {
		t.Fatalf("node-build 默认构建命令/产物目录不符:\n%s", out)
	}
	out, _ = renderSrc2Dockerfile(dto.Src2ComposeServiceSpec{Name: "a", Lang: "node-build", ContainerPort: 80, BuildCmd: "pnpm run build:prod", DistDir: "build"})
	if !strings.Contains(out, "pnpm run build:prod") || !strings.Contains(out, "/app/build/") {
		t.Fatalf("node-build 自定义命令/产物目录未生效:\n%s", out)
	}
	// compose：node-build 容器端口默认 80 且不注入 PORT 环境变量
	yaml := renderComposeYAML("web", "abc123def456", []dto.Src2ComposeServiceSpec{
		{Name: "fe", Lang: "node-build", HostPort: 38083},
	})
	if !strings.Contains(yaml, `"38083:80"`) || strings.Contains(yaml, "PORT=") {
		t.Fatalf("node-build compose 渲染不符:\n%s", yaml)
	}
}

func TestCheckRelDir(t *testing.T) {
	for _, ok := range []string{"", "server", "apps/web", "a.b-c_d"} {
		if err := checkRelDir(ok); err != nil {
			t.Fatalf("%q 应合法: %v", ok, err)
		}
	}
	for _, bad := range []string{"/abs", "..", "a/../b", `a\b`, "a b", ".hidden"} {
		if err := checkRelDir(bad); err == nil {
			t.Fatalf("%q 应被拒绝", bad)
		}
	}
}

func TestValidateReq(t *testing.T) {
	s := &Src2ComposeService{}
	base := dto.Src2ComposeBuildReq{
		Name: "demo", GitURL: "https://github.com/x/y.git",
		Services: []dto.Src2ComposeServiceSpec{{Name: "web", Lang: "node", HostPort: 30000, ContainerPort: 3000}},
	}
	if err := s.validateReq(base); err != nil {
		t.Fatalf("合法请求不应报错: %v", err)
	}
	bad := []dto.Src2ComposeBuildReq{
		{Name: "-bad", GitURL: base.GitURL, Services: base.Services},
		{Name: "demo", GitURL: "ext::/bin/sh", Services: base.Services},
		{Name: "demo", GitURL: base.GitURL, Services: []dto.Src2ComposeServiceSpec{}},
		{Name: "demo", GitURL: base.GitURL, Services: []dto.Src2ComposeServiceSpec{
			{Name: "a", Lang: "node"}, {Name: "a", Lang: "go"},
		}},
		{Name: "demo", GitURL: base.GitURL, Services: []dto.Src2ComposeServiceSpec{{Name: "a", Lang: "cobol"}}},
		{Name: "demo", GitURL: base.GitURL, Services: []dto.Src2ComposeServiceSpec{{Name: "a", Lang: "go", Dir: "../esc", HostPort: 1, ContainerPort: 2}}},
		{Name: "demo", GitURL: base.GitURL, Services: []dto.Src2ComposeServiceSpec{{Name: "a", Lang: "go", HostPort: 30000}}},
	}
	for i, b := range bad {
		if err := s.validateReq(b); err == nil {
			t.Fatalf("用例 %d 应被拒绝", i)
		}
	}
}

func TestHostOfGitURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/a/b.git":          "github.com",
		"https://u:p@gitlab.com/x/y":          "gitlab.com",
		"git@github.com:a/b.git":              "github.com",
		"ssh://git@gitee.com:2222/a/b.git":    "gitee.com",
		"http://192.168.1.10:3000/a/b.git":    "192.168.1.10",
	}
	for u, want := range cases {
		if got := HostOfGitURL(u); got != want {
			t.Fatalf("%s → %q, 期望 %q", u, got, want)
		}
	}
}
