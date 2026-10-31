package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

func storeTestWrite(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func storeTestRead(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// 执行真实 shell 文件操作，只以替身代替 Docker；不访问宿主部署目录和现有容器。
func newStoreShellAgent(t *testing.T, root, oldConfig string, invalidConfig bool) *agentclient.Client {
	t.Helper()
	bin := filepath.Join(root, "bin")
	storeTestWrite(t, filepath.Join(bin, "docker"), `#!/bin/sh
case "$*" in
  *"config --services"*) printf 'backend\n' ;;
  *"config --format json"*)
    if [ "$STORE_INVALID_CONFIG" = 1 ]; then echo 'invalid template' >&2; exit 1; fi
    cat "$STORE_OLD_CONFIG" ;;
  ps*) printf 'old-container\n' ;;
  cp*) mkdir -p "$3"; cp -a "$STORE_RECOVERY/." "$3/" ;;
  *" down"*) exit 1 ;;
  *" up "*) printf '%s\n' "$*" >> "$STORE_COMMANDS" ;;
esac
`)
	if err := os.Chmod(filepath.Join(bin, "docker"), 0700); err != nil {
		t.Fatal(err)
	}
	storeTestWrite(t, filepath.Join(root, "old-config.json"), oldConfig)
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data any
		translate := func(p string) string {
			return strings.ReplaceAll(p, "/opt/ypanel/compose", filepath.Join(root, "compose"))
		}
		switch r.URL.Path {
		case "/agent/v1/exec":
			var req dto.ExecReq
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				http.Error(w, "bad request", 400)
				return
			}
			command := exec.CommandContext(r.Context(), "sh", "-c", translate(req.Command))
			command.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "STORE_OLD_CONFIG="+filepath.Join(root, "old-config.json"), "STORE_RECOVERY="+filepath.Join(root, "recovery"), "STORE_COMMANDS="+filepath.Join(root, "commands"))
			if invalidConfig {
				command.Env = append(command.Env, "STORE_INVALID_CONFIG=1")
			}
			out, err := command.CombinedOutput()
			code := 0
			if err != nil {
				code = 1
				var ee *exec.ExitError
				if errors.As(err, &ee) {
					code = ee.ExitCode()
				}
			}
			data = dto.ExecResp{Output: string(out), ExitCode: code}
		case "/agent/v1/files/write":
			var req dto.FileWriteReq
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				return
			}
			storeTestWrite(t, translate(req.Path), req.Content)
			data = struct{}{}
		case "/agent/v1/files/read":
			b, err := os.ReadFile(translate(r.URL.Query().Get("path")))
			if err != nil {
				http.Error(w, "file unavailable", 404)
				return
			}
			data = dto.FileReadResp{Content: string(b)}
		case "/agent/v1/files/mkdir":
			var req struct{ Path string }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				return
			}
			if err := os.MkdirAll(translate(req.Path), 0700); err != nil {
				t.Error(err)
			}
			data = struct{}{}
		default:
			t.Errorf("意外 agent 请求: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
	}))
	t.Cleanup(agent.Close)
	return agentclient.New(agent.URL, "test")
}

func storeTestConfig(root string) string {
	return `{"services":{"backend":{"volumes":[{"type":"bind","source":` + strconvQuote(root+"/plugins") + `,"target":"/app/plugins"},{"type":"bind","source":` + strconvQuote(root+"/market-source") + `,"target":"/app/market-source"},{"type":"bind","source":` + strconvQuote(root+"/config") + `,"target":"/app/config"},{"type":"bind","source":` + strconvQuote(root+"/data") + `,"target":"/app/data"}]}}}`
}

func strconvQuote(s string) string { b, _ := json.Marshal(s); return string(b) }

func TestStoreDeploymentPreservesData(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[rollback], func(t *testing.T) {
			root := t.TempDir()
			d := &storeDeployment{actual: filepath.Join(root, "actual"), stage: filepath.Join(root, "stage"), backup: filepath.Join(root, "backup"), project: "app-demo"}
			for _, dir := range []string{d.stage, d.backup} {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			storeTestWrite(t, d.actual+"/docker-compose.yml", "old compose")
			storeTestWrite(t, d.actual+"/.env", "SECRET=old\n")
			for _, rel := range []string{"plugins/a.jar", "plugins/nested/b.jar", "market-source/custom.json", "data/db.bin", "user-notes.txt"} {
				storeTestWrite(t, d.actual+"/"+rel, "original "+rel)
			}
			storeTestWrite(t, d.stage+"/docker-compose.yml", "new compose")
			storeTestWrite(t, d.stage+"/.env", "SECRET=new\n")
			storeTestWrite(t, d.stage+"/new-package.txt", "new")
			storeTestWrite(t, d.stage+"/plugins/a.jar", "bundled default")
			storeTestWrite(t, root+"/recovery/installer.yaml", "persisted installer")
			storeTestWrite(t, root+"/recovery/nested/credentials.yaml", "persisted credentials")
			ac := newStoreShellAgent(t, root, storeTestConfig(d.actual), false)
			var cfg storeComposeConfig
			if err := json.Unmarshal([]byte(storeTestConfig(d.stage)), &cfg); err != nil {
				t.Fatal(err)
			}
			logf := TaskLogf(func(string, string, ...any) {})
			if err := d.activate(t.Context(), ac, &cfg, []StoreVolumeRecovery{{Service: "backend", Source: "/app/config", Target: "config"}}, logf); err != nil {
				t.Fatal(err)
			}
			if storeTestRead(t, d.actual+"/docker-compose.yml") != "new compose" {
				t.Fatal("未应用新配置")
			}
			var deployErr error
			if rollback {
				deployErr = errors.New("up failed")
			}
			d.finish(ac, &deployErr, logf)
			want := "new compose"
			if rollback {
				want = "old compose"
			}
			if storeTestRead(t, d.actual+"/docker-compose.yml") != want {
				t.Fatal("部署配置未正确恢复")
			}
			for _, rel := range []string{"plugins/a.jar", "plugins/nested/b.jar", "market-source/custom.json", "data/db.bin", "user-notes.txt"} {
				if storeTestRead(t, d.actual+"/"+rel) != "original "+rel {
					t.Fatalf("用户文件被覆盖: %s", rel)
				}
			}
			if storeTestRead(t, d.actual+"/config/installer.yaml") != "persisted installer" || storeTestRead(t, d.actual+"/config/nested/credentials.yaml") != "persisted credentials" {
				t.Fatal("恢复出的多文件配置必须完整保留，包括失败回滚")
			}
			if rollback {
				if storeTestRead(t, d.actual+"/.env") != "SECRET=old\n" {
					t.Fatal("原凭据文件未恢复")
				}
				if _, err := os.Stat(d.actual + "/new-package.txt"); !os.IsNotExist(err) {
					t.Fatal("回滚未撤下新配置文件")
				}
			}
		})
	}
}

func TestStoreDeploymentProtectsBindSources(t *testing.T) {
	for _, scenario := range []string{"empty-file-directory", "nonempty-file-directory", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			d := &storeDeployment{actual: root + "/actual", stage: root + "/stage", backup: root + "/backup", project: "app-demo"}
			storeTestWrite(t, d.stage+"/docker-compose.yml", "new compose")
			storeTestWrite(t, d.stage+"/conf/app.ini", "seed config")
			if err := os.MkdirAll(d.backup, 0700); err != nil {
				t.Fatal(err)
			}
			if scenario == "symlink" {
				if err := os.MkdirAll(d.actual, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(root+"/outside", d.actual+"/conf"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(d.actual+"/conf/app.ini", 0700); err != nil {
					t.Fatal(err)
				}
				if scenario == "nonempty-file-directory" {
					storeTestWrite(t, d.actual+"/conf/app.ini/user-data", "keep")
				}
			}
			ac := newStoreShellAgent(t, root, "{}", false)
			var cfg storeComposeConfig
			if err := json.Unmarshal([]byte(`{"services":{"backend":{"volumes":[{"type":"bind","source":`+strconvQuote(d.stage+"/conf/app.ini")+`,"target":"/etc/app.ini"}]}}}`), &cfg); err != nil {
				t.Fatal(err)
			}
			err := d.activate(t.Context(), ac, &cfg, nil, func(string, string, ...any) {})
			if scenario == "empty-file-directory" {
				if err != nil || storeTestRead(t, d.actual+"/conf/app.ini") != "seed config" {
					t.Fatalf("空的遗留目录修复失败: %v", err)
				}
			} else if err == nil {
				t.Fatal("非空目录或符号链接必须阻止部署")
			}
			if scenario == "nonempty-file-directory" && storeTestRead(t, d.actual+"/conf/app.ini/user-data") != "keep" {
				t.Fatal("不能删除用户目录")
			}
			if scenario == "symlink" {
				if _, err := os.Stat(root + "/outside"); !os.IsNotExist(err) {
					t.Fatal("不能写入符号链接外的目录")
				}
			}
		})
	}
}

func TestStoreInvalidPackageLeavesDeploymentRunning(t *testing.T) {
	root := t.TempDir()
	ac := newStoreShellAgent(t, root, "{}", true)
	pkg := root + "/package"
	storeTestWrite(t, pkg+"/docker-compose.yml", "services:\n  backend:\n    image: example:1\n")
	storeTestWrite(t, root+"/compose/app-demo/docker-compose.yml", "old compose")
	storeTestWrite(t, root+"/compose/app-demo/.env", "SECRET=keep\n")
	s := &StoreService{nodes: &NodeService{local: &Node{ID: "local", BaseURL: ac.BaseURL, Token: ac.Token}}}
	_, err := s.deployCompose(t.Context(), func(string, string, ...any) {}, model.AppStoreApp{Key: "demo"}, StoreVersion{DownloadURL: pkg}, StoreInstallInput{Name: "demo"}, "app-demo", map[string]string{"CONTAINER_NAME": "app-demo"}, PanelNetwork, "", nil, false)
	if err == nil || !strings.Contains(err.Error(), "新应用包配置校验失败") {
		t.Fatalf("未在停止旧部署前中止: %v", err)
	}
	if storeTestRead(t, root+"/compose/app-demo/docker-compose.yml") != "old compose" || storeTestRead(t, root+"/compose/app-demo/.env") != "SECRET=keep\n" {
		t.Fatal("无效新包触碰了旧配置")
	}
	if _, err := os.Stat(root + "/commands"); !os.IsNotExist(err) {
		t.Fatal("校验失败不应执行 up")
	}
}

func TestStoreFreshFailureKeepsDataAndRestoresFiles(t *testing.T) {
	root := t.TempDir()
	d := &storeDeployment{actual: root + "/actual", stage: root + "/stage", backup: root + "/backup", project: "app-demo"}
	storeTestWrite(t, d.stage+"/docker-compose.yml", "new")
	storeTestWrite(t, d.stage+"/.env", "SECRET=new\n")
	storeTestWrite(t, d.stage+"/data/seed", "persistent")
	if err := os.MkdirAll(d.backup, 0700); err != nil {
		t.Fatal(err)
	}
	ac := newStoreShellAgent(t, root, "{}", false)
	var cfg storeComposeConfig
	if err := json.Unmarshal([]byte(storeTestConfig(d.stage)), &cfg); err != nil {
		t.Fatal(err)
	}
	logf := TaskLogf(func(string, string, ...any) {})
	if err := d.activate(t.Context(), ac, &cfg, nil, logf); err != nil {
		t.Fatal(err)
	}
	err := errors.New("deployment failed")
	d.finish(ac, &err, logf) // docker down 在替身中失败，文件恢复仍必须执行。
	if storeTestRead(t, d.actual+"/data/seed") != "persistent" {
		t.Fatal("首次部署失败也不能删掉持久化数据")
	}
	for _, rel := range []string{"docker-compose.yml", ".env"} {
		if _, err := os.Stat(d.actual + "/" + rel); !os.IsNotExist(err) {
			t.Fatalf("首次部署失败未撤下本次配置 %s", rel)
		}
	}
}

func TestStoreParametersAndEnvContract(t *testing.T) {
	src, _ := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp { return dto.ExecResp{} })
	if err := src.DB.AutoMigrate(&model.AppStoreInstall{}, &model.AppStoreApp{}); err != nil {
		t.Fatal(err)
	}
	s := &StoreService{db: src.DB}
	ver := StoreVersion{FormFields: []StoreFormField{{EnvKey: "KEY", Type: "password", Random: true, RandomFormat: "base64-32"}, {EnvKey: "REDIS_PASSWORD", Type: "password"}}, VersionParams: map[string]string{"TAG": "1.12.0"}}
	in := StoreInstallInput{Params: map[string]any{"TAG": "latest"}}
	params, err := s.prepareStoreInstall(t.Context(), "app-demo", ver, &in)
	if err != nil {
		t.Fatal(err)
	}
	key, err := base64.StdEncoding.DecodeString(params["KEY"])
	if err != nil || len(key) != 32 || params["TAG"] != "1.12.0" || params["REDIS_PASSWORD"] != "" {
		t.Fatalf("随机密钥/版本/可选空密码契约错误: key bytes=%d, err=%v", len(key), err)
	}
	invalid := StoreInstallInput{Params: map[string]any{"KEY": "incorrect-format"}}
	if _, err := s.prepareStoreInstall(t.Context(), "app-other", ver, &invalid); err == nil {
		t.Fatal("显式提供的错误格式密钥应在部署前拒绝")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker Compose 不可用，跳过真实 .env 解析")
	}
	for _, value := range []string{"quote'and\"double", `trailing\`, `slash\'quote`, "$UNKNOWN; & $(printf unsafe)", `literal\n`, "space # comment"} {
		root := t.TempDir()
		storeTestWrite(t, root+"/compose.yaml", "services:\n  test:\n    image: example:1\n    environment:\n      VALUE: ${VALUE}\n")
		storeTestWrite(t, root+"/.env", renderStoreEnv(StoreInstallInput{}, StoreVersion{}, map[string]string{"VALUE": value}))
		cmd := exec.CommandContext(t.Context(), "docker", "compose", "-p", "ypanel-env-test", "-f", root+"/compose.yaml", "config", "--environment")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Compose 解析 .env 失败: %s", out)
		}
		if !strings.Contains(string(out), "\nVALUE="+value+"\n") && !strings.HasPrefix(string(out), "VALUE="+value+"\n") {
			t.Fatalf("Compose 未保留参数值: %q", value)
		}
		if parseStoreEnv("VALUE=" + quoteStoreEnv(value))["VALUE"] != value {
			t.Fatalf("面板参数读取未还原值: %q", value)
		}
	}
}

func TestStoreUpgradeRestoresCredentialsAndOptions(t *testing.T) {
	src, _ := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp { return dto.ExecResp{} })
	if err := src.DB.AutoMigrate(&model.AppStoreInstall{}, &model.AppStoreApp{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	ac := newStoreShellAgent(t, root, storeTestConfig(root+"/compose/app-demo"), false)
	s := &StoreService{db: src.DB, nodes: &NodeService{local: &Node{ID: "local", BaseURL: ac.BaseURL, Token: ac.Token}}}
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	rawEnv := "# hand maintained\nTAG=latest\nKEY=" + quoteStoreEnv(key) + "\nPASSWORD=" + quoteStoreEnv(`dollar$;quote'backslash\`) + "\n"
	storeTestWrite(t, root+"/compose/app-demo/.env", rawEnv)
	app := model.AppStoreApp{SourceID: 1, Key: "demo", Kind: "app"}
	if err := src.DB.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	inst := model.AppStoreInstall{SourceID: 1, Key: "demo", Name: "demo", Version: "1.0", ComposeProject: "app-demo", NodeID: "", OwnerID: 9, ParamsJSON: `{"KEY":"stale-db-value"}`, OptionsJSON: `{"network":"custom-net","timezone":"Asia/Shanghai","extraHosts":["host.docker.internal:host-gateway"],"mountHostsFile":true,"optionalServices":["render-server"]}`}
	if err := src.DB.Create(&inst).Error; err != nil {
		t.Fatal(err)
	}
	ver := StoreVersion{VersionParams: map[string]string{"TAG": "1.12.0"}, FormFields: []StoreFormField{{EnvKey: "KEY", Type: "password", Random: true, RandomFormat: "base64-32"}}}
	in := StoreInstallInput{SourceID: 1, Key: "demo", Name: "demo", NodeID: "local", Params: map[string]any{"KEY": "******", "USER": "edited"}, upgrade: true}
	params, err := s.prepareStoreInstall(t.Context(), "app-demo", ver, &in)
	if err != nil {
		t.Fatal(err)
	}
	if params["KEY"] != key || params["TAG"] != "1.12.0" || params["USER"] != "edited" || params["PASSWORD"] != `dollar$;quote'backslash\` {
		t.Fatal("必须沿用真实凭据、应用显式修改并使用目标版本标签")
	}
	if in.rawEnv != rawEnv || in.Network != "custom-net" || in.Timezone != "Asia/Shanghai" || !in.MountHostsFile || len(in.ExtraHosts) != 1 || in.ExternalDB == nil || len(in.ExternalDB.OptionalServices) != 1 {
		t.Fatalf("升级选项未保留: %+v", in)
	}
	rendered := renderStoreEnv(in, ver, params)
	if !strings.HasPrefix(rendered, rawEnv) || strings.Count(rendered, "KEY=") != 1 || !strings.Contains(rendered, `TAG="1.12.0"`) {
		t.Fatal("原 .env 必须保留原文，版本参数必须覆盖")
	}
	ctx := rbac.WithCaller(t.Context(), rbac.Caller{DataScope: "assigned", UserID: 10, AllNodes: true})
	other := StoreInstallInput{SourceID: 1, Key: "demo"}
	if err := s.restoreStoreInstall(ctx, "app-demo", ver, &other); errs.From(err).Code != errs.CodeNotFound {
		t.Fatal("不能覆盖其他属主的应用")
	}
	if _, err := s.UpgradeApp(t.Context(), "app-demo", "7"); errs.From(err).Code != errs.CodeNotFound {
		t.Fatal("不能用其他节点请求升级本机同名项目")
	}
}

func TestStorePackageOverrideIsPreserved(t *testing.T) {
	root := t.TempDir()
	ac := newStoreShellAgent(t, root, "{}", false)
	dir := root + "/stage"
	storeTestWrite(t, dir+"/docker-compose.override.yml", "services:\n  backend:\n    command: [serve, --custom]\n    labels:\n      - vendor.label=keep\n    environment:\n      - APP_SETTING=keep\n    extra_hosts:\n      old.internal: 192.0.2.1\n    volumes:\n      - ./custom:/custom\n")
	s := &StoreService{}
	if err := s.writeComposeOverride(t.Context(), func(string, string, ...any) {}, ac, dir, "Asia/Shanghai", []string{"host.docker.internal:host-gateway"}, true, map[string]string{"io.ypanel.app.key": "demo"}); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(storeTestRead(t, dir+"/docker-compose.override.yml")), &doc); err != nil {
		t.Fatal(err)
	}
	svc := doc["services"].(map[string]any)["backend"].(map[string]any)
	if len(svc["command"].([]any)) != 2 || len(svc["volumes"].([]any)) != 2 || svc["labels"].(map[string]any)["vendor.label"] != "keep" || svc["environment"].(map[string]any)["APP_SETTING"] != "keep" || len(svc["extra_hosts"].(map[string]any)) != 2 {
		t.Fatal("注入应用标签不能丢失包内 override 的设置")
	}
}

func TestStoreAndImageUpdateTasksAreMutuallyExclusive(t *testing.T) {
	s, _ := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp { return dto.ExecResp{} })
	release := make(chan struct{})
	var task *model.AppTask
	defer func() {
		close(release)
		if task != nil {
			_ = waitComposeUpdateTask(t, s, task.ID)
		}
	}()
	var err error
	task, err = s.Tasks.StartProjectTask(TaskStoreInstall, "deploy", "local", "demo", time.Minute, func(context.Context, TaskLogf) error { <-release; return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []string{composeUpdateManual, composeUpdateAuto, TaskStoreInstall, TaskStoreUninstall} {
		if _, err := s.Tasks.StartProjectTask(typ, "duplicate", "local", "demo", time.Minute, func(context.Context, TaskLogf) error { return nil }); errs.From(err).Code != errs.CodeConflict {
			t.Fatalf("部署与 %s 未互斥: %v", typ, err)
		}
	}
	remote, err := s.Tasks.StartProjectTask(TaskStoreInstall, "remote", "7", "demo", time.Minute, func(context.Context, TaskLogf) error { return nil })
	if err != nil {
		t.Fatalf("不同节点不应互斥: %v", err)
	}
	_ = waitComposeUpdateTask(t, s, remote.ID)
	caller := rbac.WithCaller(t.Context(), rbac.Caller{AllNodes: false, NodeSet: map[string]struct{}{"7": {}}, DataScope: "assigned", UserID: 5})
	installs := ownerFilterInstalls(caller, []model.AppStoreInstall{{NodeID: "local", OwnerID: 5}, {NodeID: "7", OwnerID: 6}, {NodeID: "7", OwnerID: 5}})
	if len(installs) != 1 || installs[0].NodeID != "7" || installs[0].OwnerID != 5 {
		t.Fatal("已安装列表必须同时遵守节点与属主范围")
	}
}
