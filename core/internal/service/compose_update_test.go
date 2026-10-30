package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newComposeUpdateTestService(t *testing.T, execFn func(dto.ExecReq) dto.ExecResp) (*Src2ComposeService, func() []string) {
	t.Helper()
	var mu sync.Mutex
	commands := make([]string, 0)
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer test-agent" {
			t.Error("更新请求未使用目标节点的认证信息")
		}
		var data any
		switch r.URL.Path {
		case "/agent/v1/compose/config":
			if r.URL.Query().Get("name") != "demo" {
				t.Errorf("配置查询项目错误: %s", r.URL.RawQuery)
			}
			data = dto.ComposeConfigResp{File: "/srv/demo/compose.yaml", Dir: "/srv/demo"}
		case "/agent/v1/exec":
			var req dto.ExecReq
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("执行参数解析失败: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			mu.Lock()
			command := req.Command
			if hook, ok := req.Env["YPANEL_COMPOSE_UPDATE_HOOK"]; ok {
				command = hook
				if req.Command != `printf '%s\n' "$YPANEL_COMPOSE_UPDATE_HOOK" | sh -e` || req.TimeoutSecs != 300 {
					t.Errorf("脚本应独立执行、失败立即退出且不能出现在命令日志中: %+v", req)
				}
			}
			commands = append(commands, command)
			mu.Unlock()
			data = execFn(req)
		default:
			t.Errorf("意外 agent 请求: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
	}))
	t.Cleanup(agent.Close)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "updates.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.ComposeAutoUpdate{}, &model.AppTask{}, &model.Node{}); err != nil {
		t.Fatal(err)
	}
	nodes := &NodeService{db: db, local: &Node{ID: "local", BaseURL: agent.URL, Token: "test-agent"}}
	return &Src2ComposeService{DB: db, Nodes: nodes, Tasks: NewTaskService(db)}, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), commands...)
	}
}

// 信封成功但命令失败、未变更镜像、脚本失败策略都必须影响实际重建行为。
func TestComposeUpdateExecution(t *testing.T) {
	for _, tc := range []struct {
		name, policy, failAt, wantErr                             string
		noRunning, unchanged, timeout, wantUp, wantPost, wantWarn bool
	}{
		{name: "停止项目跳过", noRunning: true},
		{name: "镜像未变化", unchanged: true},
		{name: "只重建变化服务", wantUp: true, wantPost: true},
		{name: "拉取非零退出", failAt: "pull", wantErr: "镜像拉取失败"},
		{name: "拉取超时", failAt: "pull", timeout: true, wantErr: "超时"},
		{name: "前脚本默认中止", failAt: "before-update", wantErr: "更新前脚本失败"},
		{name: "前脚本失败继续", policy: "continue", failAt: "before-update", wantUp: true, wantPost: true, wantWarn: true},
		{name: "重建失败不执行后脚本", failAt: "up", wantUp: true, wantErr: "容器重建失败"},
		{name: "后脚本失败不宣称回滚", failAt: "after-update", wantUp: true, wantPost: true, wantErr: "不会自动回滚"},
		{name: "后脚本失败继续", policy: "continue", failAt: "after-update", wantUp: true, wantPost: true, wantWarn: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			images := []composeUpdateImage{
				{Service: "web", Image: "web:stable", ID: "sha256:old", ConfigFiles: "/srv/demo/compose.yaml, /srv/demo/override.yaml", WorkDir: "/srv/demo"},
				{Service: "db", Image: "db:16", ID: "sha256:db"},
				{Service: "migration", Image: "web:stable", ID: "sha256:old", OneOff: "True"},
			}
			var snapshot strings.Builder
			if !tc.noRunning {
				for _, image := range images {
					b, _ := json.Marshal(image)
					snapshot.Write(b)
					snapshot.WriteByte('\n')
				}
			}
			s, commands := newComposeUpdateTestService(t, func(req dto.ExecReq) dto.ExecResp {
				kind := ""
				switch {
				case strings.HasPrefix(req.Command, "ids=$(docker ps"):
					return dto.ExecResp{Output: snapshot.String()}
				case strings.Contains(req.Command, " pull "):
					kind = "pull"
				case strings.Contains(req.Command, " up -d "):
					kind = "up"
				case strings.HasPrefix(req.Command, "docker image inspect"):
					id := "sha256:new"
					if tc.unchanged {
						id = "sha256:old"
					}
					return dto.ExecResp{Output: id + "\nsha256:db\n"}
				default:
					kind = req.Command
					if hook, ok := req.Env["YPANEL_COMPOSE_UPDATE_HOOK"]; ok {
						kind = hook
					}
				}
				if kind == tc.failAt {
					return dto.ExecResp{Output: "failed step", ExitCode: 1, TimedOut: tc.timeout}
				}
				return dto.ExecResp{}
			})
			warned := false
			result, err := s.runComposeUpdate(t.Context(), dto.ComposePullUpdateReq{NodeID: "local", Project: "demo", PreHook: "before-update", PostHook: "after-update", HookFailurePolicy: tc.policy},
				func(level, _ string, _ ...any) { warned = warned || level == "warn" })
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %s", err, tc.wantErr)
				}
			} else if err != nil || result == "" {
				t.Fatalf("result=%q, err=%v", result, err)
			}
			var up, post, pre bool
			for _, command := range commands() {
				if strings.Contains(command, " pull ") {
					for _, want := range []string{"pull --ignore-buildable --policy always", "'--file' '/srv/demo/override.yaml'", "'--project-name' 'demo'"} {
						if !strings.Contains(command, want) {
							t.Errorf("拉取命令缺少 %q: %s", want, command)
						}
					}
					if strings.Contains(command, "migration") {
						t.Error("一次性容器不应参与更新")
					}
				}
				if strings.Contains(command, " up -d ") {
					up = true
					if !strings.HasSuffix(command, "--no-deps --no-build --pull never 'web'") {
						t.Errorf("未变化或停止的服务不应被重建: %s", command)
					}
				}
				pre = pre || command == "before-update"
				post = post || command == "after-update"
			}
			if up != tc.wantUp || post != tc.wantPost || warned != tc.wantWarn {
				t.Fatalf("up/post/warn = %v/%v/%v, want %v/%v/%v", up, post, warned, tc.wantUp, tc.wantPost, tc.wantWarn)
			}
			if (tc.noRunning || tc.unchanged) && (pre || len(commands()) > 3) {
				t.Error("未发现更新时不应执行脚本或重建")
			}
		})
	}
}

func TestComposeAutoUpdatePersistence(t *testing.T) {
	s, _ := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp { return dto.ExecResp{} })
	config, err := s.GetAutoUpdate(t.Context(), "", "demo")
	if err != nil || config.Exists || config.Enabled || config.IntervalMin != 60 || config.HookFailurePolicy != "stop" {
		t.Fatalf("新配置默认值错误: %+v, %v", config, err)
	}
	req := dto.ComposeAutoUpdateSaveReq{ComposePullUpdateReq: dto.ComposePullUpdateReq{Project: "demo", PreHook: "echo before\necho backup"}, IntervalMin: 360}
	if err := s.SaveAutoUpdate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	config, err = s.GetAutoUpdate(t.Context(), "local", "demo")
	if err != nil || config.Enabled || !config.Exists || config.IntervalMin != 360 || config.PreHook != req.PreHook || config.HookFailurePolicy != "stop" {
		t.Fatalf("首次关闭配置保存失败: %+v, %v", config, err)
	}
	now := time.Now().Truncate(time.Second)
	if err := s.DB.Model(&model.ComposeAutoUpdate{}).Where("project = ?", "demo").Updates(map[string]any{"last_checked_at": now, "last_result": "existing result"}).Error; err != nil {
		t.Fatal(err)
	}
	req.Enabled, req.PreHook, req.PostHook, req.IntervalMin = true, "", "echo done", 17
	if err := s.SaveAutoUpdate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	config, err = s.GetAutoUpdate(t.Context(), "local", "demo")
	if err != nil || !config.Enabled || config.PreHook != "" || config.PostHook != "echo done" || config.IntervalMin != 17 || config.LastResult != "existing result" || !config.LastCheckedAt.Equal(now) || config.NextCheckAt == nil {
		t.Fatalf("更新配置覆盖了执行状态或未清空脚本: %+v, %v", config, err)
	}
	for _, interval := range []int{0, 4, 1441} {
		req.IntervalMin = interval
		if err := s.SaveAutoUpdate(t.Context(), req); errs.From(err).Code != errs.CodeBadRequest {
			t.Errorf("无效间隔 %d 应被拒绝: %v", interval, err)
		}
	}
	req.IntervalMin, req.HookFailurePolicy = 60, "invalid"
	if err := s.SaveAutoUpdate(t.Context(), req); err == nil {
		t.Error("无效脚本策略应被拒绝")
	}
}

func waitComposeUpdateTask(t *testing.T, s *Src2ComposeService, id uint) model.AppTask {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		var task model.AppTask
		if err := s.DB.First(&task, id).Error; err != nil {
			t.Fatal(err)
		}
		if task.Status != TaskRunning {
			return task
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("等待更新任务完成超时")
	return model.AppTask{}
}

func TestComposeAutoUpdateTaskLifecycle(t *testing.T) {
	entered, release := make(chan struct{}, 1), make(chan struct{})
	s, _ := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp {
		entered <- struct{}{}
		<-release
		return dto.ExecResp{Output: "registry unavailable", ExitCode: 1}
	})
	req := dto.ComposeAutoUpdateSaveReq{ComposePullUpdateReq: dto.ComposePullUpdateReq{Project: "demo"}, Enabled: true, IntervalMin: 60}
	if err := s.SaveAutoUpdate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	first, err := s.PullUpdateTask(t.Context(), req.ComposePullUpdateReq)
	if err != nil {
		t.Fatal(err)
	}
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		_ = waitComposeUpdateTask(t, s, first.TaskID)
	})
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("任务未进入执行")
	}
	if _, err := s.startComposeUpdateTask(t.Context(), req.ComposePullUpdateReq, composeUpdateAuto); errs.From(err).Code != errs.CodeConflict {
		t.Fatalf("手动/自动更新必须互斥: %v", err)
	}
	config, err := s.GetAutoUpdate(t.Context(), "local", "demo")
	if err != nil || config.LastTaskID != first.TaskID || config.LastStatus != TaskRunning || config.LastCheckedAt == nil {
		t.Fatalf("执行中状态未回显: %+v, %v", config, err)
	}
	releaseOnce.Do(func() { close(release) })
	failed := waitComposeUpdateTask(t, s, first.TaskID)
	config, err = s.GetAutoUpdate(t.Context(), "local", "demo")
	if err != nil || failed.Status != TaskFailed || config.LastStatus != TaskFailed || !strings.Contains(config.LastResult, "registry unavailable") || !strings.Contains(failed.LogText, "registry unavailable") {
		t.Fatalf("失败结果未记录: task=%+v, config=%+v, err=%v", failed, config, err)
	}
}

func TestComposeAutoUpdateSchedulerAndResult(t *testing.T) {
	s, commands := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp { return dto.ExecResp{} })
	req := dto.ComposeAutoUpdateSaveReq{ComposePullUpdateReq: dto.ComposePullUpdateReq{Project: "demo"}, Enabled: true, IntervalMin: 60}
	if err := s.SaveAutoUpdate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	s.autoUpdateTick(t.Context())
	config, err := s.GetAutoUpdate(t.Context(), "local", "demo")
	if err != nil || config.LastTaskID == 0 {
		t.Fatalf("到期项目未调度: %+v, %v", config, err)
	}
	task := waitComposeUpdateTask(t, s, config.LastTaskID)
	config, err = s.GetAutoUpdate(t.Context(), "local", "demo")
	if err != nil || task.Type != composeUpdateAuto || config.LastStatus != TaskSuccess || !strings.Contains(config.LastResult, "没有运行中的服务") || !strings.Contains(task.LogText, config.LastResult) {
		t.Fatalf("任务实际结果未写回: task=%+v, config=%+v, err=%v", task, config, err)
	}
	s.autoUpdateTick(t.Context())
	if len(commands()) != 1 {
		t.Error("间隔未到不应再次更新")
	}
	req.Enabled = false
	if err := s.SaveAutoUpdate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	// 即便已到期，关闭的配置也不能创建新任务。
	if err := s.DB.Model(&model.ComposeAutoUpdate{}).Where("project = ?", "demo").Update("last_checked_at", nil).Error; err != nil {
		t.Fatal(err)
	}
	s.autoUpdateTick(t.Context())
	var count int64
	if err := s.DB.Model(&model.AppTask{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Error("关闭配置不应调度")
	}
}

func TestComposeUpdateTargetValidationAndNode(t *testing.T) {
	ctx := rbac.WithCaller(context.Background(), rbac.Caller{NodeSet: map[string]struct{}{"7": {}}})
	for _, tc := range []struct {
		req  dto.ComposePullUpdateReq
		code int
	}{
		{dto.ComposePullUpdateReq{NodeID: "7", Name: "demo"}, 0},
		{dto.ComposePullUpdateReq{Name: "demo"}, errs.CodeForbidden},
		{dto.ComposePullUpdateReq{NodeID: "8", Project: "demo"}, errs.CodeForbidden},
		{dto.ComposePullUpdateReq{NodeID: "7", Project: "demo", Name: "other"}, errs.CodeBadRequest},
		{dto.ComposePullUpdateReq{NodeID: "7", Project: "demo;echo"}, errs.CodeBadRequest},
		{dto.ComposePullUpdateReq{NodeID: "7", Project: "demo", Dir: "relative"}, errs.CodeBadRequest},
	} {
		err := NormalizeComposeUpdateTarget(ctx, &tc.req)
		if tc.code == 0 {
			if err != nil || tc.req.Project != "demo" {
				t.Errorf("旧请求未归一化: %+v, %v", tc.req, err)
			}
		} else {
			var be *errs.Error
			if !errors.As(err, &be) || be.Code != tc.code {
				t.Errorf("目标节点/参数未拦截: %+v, %v", tc.req, err)
			}
		}
	}
	s, commands := newComposeUpdateTestService(t, func(dto.ExecReq) dto.ExecResp { return dto.ExecResp{} })
	remote := model.Node{ID: 7, Name: "remote", Addr: s.Nodes.local.BaseURL, Token: "test-agent"}
	if err := s.DB.Create(&remote).Error; err != nil {
		t.Fatal(err)
	}
	s.Nodes.local.BaseURL = "http://127.0.0.1:1"
	task, err := s.PullUpdateTask(ctx, dto.ComposePullUpdateReq{NodeID: "7", Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	completed := waitComposeUpdateTask(t, s, task.TaskID)
	if completed.Status != TaskSuccess || completed.Ref != "7:demo" || len(commands()) != 1 {
		t.Fatalf("未在实际远程节点执行: %+v", completed)
	}
	config, err := s.GetAutoUpdate(ctx, "7", "demo")
	if err != nil || config.Exists || config.LastResult != "项目没有运行中的服务，已跳过" {
		t.Fatalf("未保存配置的手动更新也应返回实际结果: %+v, %v", config, err)
	}
}
