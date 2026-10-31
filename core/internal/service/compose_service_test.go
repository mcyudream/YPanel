package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestComposeServiceActionRouting(t *testing.T) {
	requests := 0
	fail := false
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		q := r.URL.Query()
		if r.Method != http.MethodPost || r.URL.Path != "/agent/v1/compose/service-action" || r.Header.Get("Authorization") != "Bearer remote-agent" || q.Get("project") != "demo" || q.Get("service") != "web" || q.Get("dir") != "/srv/external app & config" {
			t.Errorf("服务操作目标或目录丢失: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		if fail {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": errs.CodeFileOpFailed, "message": "compose restart 失败: missing service"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": dto.ComposeServiceActionResp{Output: q.Get("action") + " web"}})
	}))
	defer agent.Close()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "nodes.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	if err := db.AutoMigrate(&model.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Node{ID: 7, Name: "remote", Addr: agent.URL, Token: "remote-agent"}).Error; err != nil {
		t.Fatal(err)
	}
	nodes := &NodeService{db: db}
	ctx := rbac.WithCaller(context.Background(), rbac.Caller{NodeSet: map[string]struct{}{"7": {}}})
	req := dto.ComposeServiceActionReq{Project: "demo", Service: "web", Dir: "/srv/external app & config"}
	for _, action := range []string{"start", "stop", "restart", "pull", "up"} {
		req.Action = action
		out, err := RunComposeServiceAction(ctx, nodes, "7", req)
		if err != nil || out.Output != action+" web" {
			t.Fatalf("操作 %s 未保留输出: %+v, %v", action, out, err)
		}
	}
	req.Project, req.Name, req.Action = "", "demo", "restart"
	if _, err := RunComposeServiceAction(ctx, nodes, "7", req); err != nil {
		t.Fatalf("旧 name 参数没有兼容: %v", err)
	}
	before := requests
	if _, err := RunComposeServiceAction(ctx, nodes, "8", req); err == nil || requests != before {
		t.Fatalf("越权请求到达 agent: requests=%d, err=%v", requests, err)
	}
	if _, err := RunComposeServiceAction(context.Background(), nodes, "999", req); err == nil || requests != before {
		t.Fatalf("未知节点没有返回错误: requests=%d, err=%v", requests, err)
	}
	fail = true
	_, err = RunComposeServiceAction(ctx, nodes, "7", req)
	var businessErr *errs.Error
	if !errors.As(err, &businessErr) || businessErr.Code != errs.CodeFileOpFailed {
		t.Fatalf("agent 操作失败被误报为成功: %v", err)
	}
}
