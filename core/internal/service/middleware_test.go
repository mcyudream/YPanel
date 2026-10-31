package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMiddlewareTestService(t *testing.T, execFn func(string, dto.ExecReq) dto.ExecResp) (*MiddlewareService, func() []string) {
	t.Helper()
	var mu sync.Mutex
	commands := make([]string, 0)
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node := "local"
		if r.Header.Get("Authorization") == "Bearer remote-test" {
			node = "7"
		} else if r.Header.Get("Authorization") != "Bearer local-test" {
			t.Error("缺少目标节点凭据")
		}
		if r.URL.Path != "/agent/v1/exec" {
			t.Errorf("意外请求 %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var req dto.ExecReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			http.Error(w, "invalid", 400)
			return
		}
		mu.Lock()
		commands = append(commands, node+":"+req.Command)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": execFn(node, req)})
	}))
	t.Cleanup(agent.Close)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "middleware.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.MiddlewareInstance{}, &model.DatabaseInstance{}, &model.AppStoreInstall{}, &model.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Node{ID: 7, Name: "remote-test", Addr: agent.URL, Token: "remote-test"}).Error; err != nil {
		t.Fatal(err)
	}
	nodes := &NodeService{db: db, local: &Node{ID: "local", BaseURL: agent.URL, Token: "local-test"}}
	return NewMiddlewareService(db, NewDatabaseService(db, nodes, nil, "test-secret"), nodes), func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), commands...) }
}

func middlewareSnapshot(items ...middlewareContainer) string {
	var b strings.Builder
	for _, item := range items {
		raw, _ := json.Marshal(item)
		b.Write(raw)
		b.WriteByte('\n')
	}
	return b.String()
}
func mustCreateMiddlewareRows(t *testing.T, s *MiddlewareService, rows ...any) {
	t.Helper()
	for _, row := range rows {
		if err := s.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestMiddlewareImageRecognition(t *testing.T) {
	for _, tc := range []struct{ image, typ, version string }{
		{"rabbitmq:4.3.2-management-alpine", "rabbitmq", "4.3.2"},
		{"docker.io/library/rabbitmq:3.13", "rabbitmq", "3.13"},
		{"registry.test:5000/proxy/bitnami/rabbitmq:4.2@sha256:abcdef", "rabbitmq", "4.2"},
		{"registry.test:5000/library/memcached", "memcached", ""},
		{"swr.example/proxy/docker.io/neo4j:5.26.7", "neo4j", "5.26.7"},
		{"docker.io/rustfs/rustfs:latest", "s3", ""},
		{"quay.io/minio/minio:RELEASE.2025-04-22T22-12-26Z", "s3", ""},
		{"redis:7.2-alpine@sha256:12.34", "redis", "7.2"},
		{"registry.test:5000/library/mongo@sha256:12.34", "mongo", ""},
		{"bitnami/postgresql:16.2", "postgres", "16.2"},
		{"mariadb:11.4", "mysql", "11.4"},
		{"rabbitmq-exporter:1.6", "", "1.6"},
		{"redis/redisinsight:2.6", "", "2.6"},
		{"rustfs/console:1.2", "", "1.2"},
	} {
		t.Run(tc.image, func(t *testing.T) {
			typ, _, ok := middlewareTypeByImage(tc.image)
			if typ != tc.typ || ok != (tc.typ != "") {
				t.Fatalf("类型 %q, %v; want %q", typ, ok, tc.typ)
			}
			if v := versionFromImage(tc.image); v != tc.version {
				t.Fatalf("版本 %q; want %q", v, tc.version)
			}
		})
	}
}

func TestMiddlewareListCombinesSourcesWithoutCredentials(t *testing.T) {
	snapshot := middlewareSnapshot(
		middlewareContainer{ContainerName: "queue", Image: "docker.io/library/rabbitmq:4.3.2", State: "running", Networks: "z-net,ypanel_default", ComposeProject: "app-queue"},
		middlewareContainer{ContainerName: "stopped-cache", Image: "registry:5000/bitnami/memcached:1.6", State: "exited"},
		middlewareContainer{ContainerName: "rustfs", Image: "rustfs/rustfs:latest", State: "running", Ports: "0.0.0.0:9009->9000/tcp, [::]:9009->9000/tcp"},
		middlewareContainer{ContainerName: "graph", Image: "proxy/docker.io/neo4j:5.26.7", State: "running", ComposeProject: "app-graph"},
		middlewareContainer{ContainerName: "redis", Image: "redis:7.2", State: "running", Ports: "0.0.0.0:6379->6379/tcp"},
		middlewareContainer{ContainerName: "mongo", Image: "mongo:7.0", State: "running"},
		middlewareContainer{ContainerName: "unrelated", Image: "my-app:1.0", State: "running"},
	)
	s, commands := newMiddlewareTestService(t, func(_ string, req dto.ExecReq) dto.ExecResp {
		if !strings.HasPrefix(req.Command, "docker ps -a ") || strings.Contains(req.Command, ".Config.Env") {
			t.Errorf("列表必须只读元数据: %s", req.Command)
		}
		return dto.ExecResp{Output: snapshot}
	})
	mustCreateMiddlewareRows(t, s,
		&model.MiddlewareInstance{Name: "queue-managed", Type: "rabbitmq", Origin: "container", Host: "queue", Port: 5672, ContainerName: "queue", PasswordEnc: "encrypted-middleware-secret"},
		&model.MiddlewareInstance{Name: "external-cache", Type: "memcached", Origin: "external", Host: "cache.test", Port: 11211},
		&model.MiddlewareInstance{Name: "remote-queue", Type: "rabbitmq", Origin: "container", Host: "queue", Port: 5672, ContainerName: "queue", NodeID: "7"},
		&model.DatabaseInstance{Name: "objects", Type: "s3", Origin: "external", Host: "127.0.0.1", Port: 9009, ComposeProject: "-", PasswordEnc: "encrypted-db-secret"},
		&model.DatabaseInstance{Name: "managed-graph", Type: "neo4j", Origin: "container", Host: "127.0.0.1", Port: 7687, ComposeProject: "app-graph"},
		&model.DatabaseInstance{Name: "managed-redis", Type: "redis", Origin: "external", Host: "127.0.0.1", Port: 6379, ComposeProject: "-"},
	)
	list, err := s.List(t.Context(), "local")
	if err != nil {
		t.Fatal(err)
	}
	if list.DiscoveryError != "" || list.Total != 7 {
		t.Fatalf("清单未正确合并: %+v", list)
	}
	seen := map[string]MiddlewareListItem{}
	keys := map[string]bool{}
	for _, row := range list.Items {
		seen[row.Name] = row
		if keys[row.Key] {
			t.Fatalf("来源 ID 冲突 %s", row.Key)
		}
		keys[row.Key] = true
	}
	if row := seen["queue-managed"]; row.State != "running" || row.NodeID != "local" || row.Version != "4.3.2" || len(row.Networks) != 2 {
		t.Fatalf("存量记录未关联: %+v", row)
	}
	if row := seen["objects"]; row.Source != "database" || row.ContainerName != "rustfs" {
		t.Fatalf("S3 未关联去重: %+v", row)
	}
	if row := seen["managed-graph"]; row.Source != "database" || row.ContainerName != "graph" || row.Version != "5.26.7" {
		t.Fatalf("Neo4j 未关联: %+v", row)
	}
	if row := seen["stopped-cache"]; row.State != "exited" || !row.Adoptable || row.Source != "discovered" {
		t.Fatalf("停止容器应可见: %+v", row)
	}
	if row := seen["mongo"]; row.ManagedBy != "database" || row.Adoptable {
		t.Fatalf("数据库容器不应走中间件纳管: %+v", row)
	}
	if seen["remote-queue"].Name != "" {
		t.Fatal("跨节点同名登记混入本机")
	}
	raw, _ := json.Marshal(list)
	if strings.Contains(strings.ToLower(string(raw)), "password") || strings.Contains(string(raw), "secret") {
		t.Fatal("清单泄漏凭据")
	}
	scan, err := s.Scan(t.Context(), "local")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(scan)
	if !strings.Contains(string(raw), `"adoptable":true`) || strings.Contains(string(raw), "admittable") || strings.Contains(string(raw), "password") {
		t.Fatal("扫描响应契约错误")
	}
	var count int64
	if err := s.db.Model(&model.MiddlewareInstance{}).Count(&count).Error; err != nil || count != 3 {
		t.Fatal("列表不应自动写入纳管记录")
	}
	if len(commands()) != 2 {
		t.Fatalf("列表和扫描每次只需一次查询: %v", commands())
	}
}

func TestMiddlewareDiscoveryFailureKeepsRegistered(t *testing.T) {
	for _, tc := range []struct {
		name string
		resp dto.ExecResp
	}{
		{"Docker 非零退出", dto.ExecResp{ExitCode: 1, Output: "private diagnostic"}},
		{"节点命令超时", dto.ExecResp{TimedOut: true}},
		{"元数据异常", dto.ExecResp{Output: "not-json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newMiddlewareTestService(t, func(_ string, _ dto.ExecReq) dto.ExecResp { return tc.resp })
			mustCreateMiddlewareRows(t, s, &model.MiddlewareInstance{Name: "known-queue", Type: "rabbitmq", Origin: "container", Host: "queue", Port: 5672, ContainerName: "queue"})
			list, err := s.List(t.Context(), "local")
			if err != nil || list.Total != 1 || list.DiscoveryError == "" || list.Items[0].State != "unknown" {
				t.Fatalf("失败应保留登记且报告错误: %+v, %v", list, err)
			}
			if strings.Contains(list.DiscoveryError, "private diagnostic") {
				t.Fatal("不应回传潜在敏感输出")
			}
			if _, err := s.Scan(t.Context(), "local"); err == nil {
				t.Fatal("扫描失败不能假装空结果")
			}
		})
	}
}

func TestMiddlewareRemoteAdoptAndScope(t *testing.T) {
	s, commands := newMiddlewareTestService(t, func(node string, req dto.ExecReq) dto.ExecResp {
		if node != "7" {
			t.Error("远程实例错误地落到本机")
		}
		switch {
		case strings.HasPrefix(req.Command, "docker ps -a "):
			return dto.ExecResp{Output: middlewareSnapshot(middlewareContainer{ContainerName: "queue", Image: "registry:5000/bitnami/rabbitmq:4.2", State: "running", Networks: PanelNetwork})}
		case strings.Contains(req.Command, ".Config.Env"):
			return dto.ExecResp{Output: `["RABBITMQ_USERNAME=admin","RABBITMQ_PASSWORD=test-adopt-secret"]`}
		case strings.Contains(req.Command, ".Config.Image"):
			return dto.ExecResp{Output: "rabbitmq:4.3.2-management"}
		default:
			t.Errorf("意外命令 %s", req.Command)
			return dto.ExecResp{ExitCode: 1}
		}
	})
	mustCreateMiddlewareRows(t, s, &model.MiddlewareInstance{Name: "rabbitmq-queue", Type: "rabbitmq", Origin: "container", NodeID: "local", ContainerName: "queue", Host: "queue", Port: 5672})
	ctx := rbac.WithCaller(t.Context(), rbac.Caller{UserID: 42, DataScope: "assigned", NodeSet: map[string]struct{}{"7": {}}})
	inst, err := s.Adopt(ctx, "7", "queue", "")
	if err != nil {
		t.Fatal(err)
	}
	if inst.NodeID != "7" || inst.OwnerID != 42 || inst.Name != "rabbitmq-queue-n7" || inst.PasswordEnc == "test-adopt-secret" {
		t.Fatal("未保存节点/属主/加密凭据")
	}
	secret, err := s.Reveal(ctx, inst.ID)
	if err != nil || secret["password"] != "test-adopt-secret" || secret["user"] != "admin" {
		t.Fatal("纳管未提取凭据")
	}
	if _, err := s.Adopt(ctx, "7", "queue", "other-name"); err == nil {
		t.Fatal("同节点容器不可重复纳管")
	}
	v, err := s.RefreshVersion(ctx, inst.ID, "")
	if err != nil || v != "4.3.2" {
		t.Fatalf("应按实例节点探测: %s, %v", v, err)
	}
	before := len(commands())
	if _, err := s.RefreshVersion(ctx, inst.ID, "local"); err == nil {
		t.Fatal("节点不一致应拒绝")
	}
	if _, err := s.List(ctx, "local"); err == nil {
		t.Fatal("节点范围应生效")
	}
	other := rbac.WithCaller(context.Background(), rbac.Caller{UserID: 99, DataScope: "assigned", AllNodes: true})
	if _, err := s.Reveal(other, inst.ID); err == nil {
		t.Fatal("他人实例不得 reveal")
	}
	if err := s.Delete(other, inst.ID); err == nil {
		t.Fatal("他人实例不得移除")
	}
	if err := s.Test(other, inst.ID); err == nil {
		t.Fatal("他人实例不得探测")
	}
	if len(commands()) != before {
		t.Fatal("越权请求不应执行节点命令")
	}
}

func TestMiddlewareDatabaseIdentityAndHiddenResources(t *testing.T) {
	snapshot := middlewareSnapshot(
		middlewareContainer{ContainerName: "graph-a", Image: "neo4j:5.26", State: "running", ComposeProject: "graphs"},
		middlewareContainer{ContainerName: "graph-b", Image: "neo4j:5.26", State: "running", ComposeProject: "graphs"},
		middlewareContainer{ContainerName: "private-queue", Image: "rabbitmq:4.2", State: "running"},
		middlewareContainer{ContainerName: "private-s3", Image: "minio/minio:latest", State: "running", Ports: "127.0.0.1:9001->9000/tcp"},
		middlewareContainer{ContainerName: "private-app", Image: "memcached:1.6", State: "running", ComposeProject: "private-app"},
	)
	s, _ := newMiddlewareTestService(t, func(_ string, _ dto.ExecReq) dto.ExecResp { return dto.ExecResp{Output: snapshot} })
	mustCreateMiddlewareRows(t, s,
		&model.DatabaseInstance{Name: "graphs-managed", Type: "neo4j", Origin: "container", Host: "127.0.0.1", Port: 7687, ComposeProject: "graphs"},
		&model.DatabaseInstance{Name: "graph-a", Type: "neo4j", Origin: "external", Host: "remote.example", Port: 7687, ComposeProject: "-"},
		&model.MiddlewareInstance{Name: "private-queue", Type: "rabbitmq", Origin: "container", Host: "private-queue", Port: 5672, ContainerName: "private-queue", OwnerID: 99},
		&model.DatabaseInstance{Name: "private-storage", Type: "s3", Origin: "external", Host: "127.0.0.1", Port: 9001, ComposeProject: "-", OwnerID: 99},
		&model.AppStoreInstall{Key: "memcached", Name: "private-app", ComposeProject: "private-app", OwnerID: 99},
	)
	ctx := rbac.WithCaller(t.Context(), rbac.Caller{UserID: 42, DataScope: "assigned", AllNodes: true})
	list, err := s.List(ctx, "local")
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 4 {
		t.Fatalf("不应泄漏其他属主或合并歧义容器: %+v", list.Items)
	}
	for _, row := range list.Items {
		if row.Source == "database" && row.ContainerName != "" {
			t.Fatalf("歧义或远端记录不能关联本机容器: %+v", row)
		}
		if strings.Contains(row.Name, "private") {
			t.Fatal("受限实例被重新发现为未纳管")
		}
	}
	if _, err := s.Adopt(ctx, "local", "graph-a", ""); err == nil || !strings.Contains(err.Error(), "数据库") {
		t.Fatal("数据库类型必须走数据库域接入")
	}
	if _, err := s.Adopt(ctx, "local", "private-queue", ""); err == nil {
		t.Fatal("不得从扫描入口接管他人容器")
	}
}

func TestMiddlewareDatabaseNodeAndEmptyList(t *testing.T) {
	s, _ := newMiddlewareTestService(t, func(node string, _ dto.ExecReq) dto.ExecResp {
		if node == "local" {
			return dto.ExecResp{}
		}
		return dto.ExecResp{Output: middlewareSnapshot(middlewareContainer{ContainerName: "objects", Image: "rustfs/rustfs:latest", State: "running", ComposeProject: "app-objects"})}
	})
	list, err := s.List(t.Context(), "local")
	if err != nil || list.Total != 0 || list.Items == nil {
		t.Fatalf("空清单必须是数组: %+v, %v", list, err)
	}
	mustCreateMiddlewareRows(t, s,
		&model.DatabaseInstance{Name: "objects-managed", Type: "s3", Origin: "container", Host: "remote.test", Port: 9000, ComposeProject: "app-objects"},
		&model.AppStoreInstall{Key: "rustfs", Name: "objects", ComposeProject: "app-objects", NodeID: "7"},
	)
	list, err = s.List(t.Context(), "7")
	if err != nil || list.Total != 1 || list.Items[0].NodeID != "7" || list.Items[0].Source != "database" || list.Items[0].ContainerName != "objects" {
		t.Fatalf("远程数据库项目未关联: %+v, %v", list, err)
	}
	list, err = s.List(t.Context(), "local")
	if err != nil || list.Total != 0 {
		t.Fatalf("远程数据库记录不应混入本机: %+v, %v", list, err)
	}
}

func TestMiddlewareReuseHonorsNodeNetworkAndOwner(t *testing.T) {
	for _, tc := range []struct {
		name, appNode, network, state, wantError string
		restricted                               bool
	}{
		{name: "同节点同网络", appNode: "7", network: PanelNetwork, state: "running"},
		{name: "跨节点阻止注入容器名", appNode: "local", network: PanelNetwork, state: "running", wantError: "同一节点"},
		{name: "跨网络阻止注入容器名", appNode: "7", network: "other-network", state: "running", wantError: "同一网络"},
		{name: "停止容器不能复用", appNode: "7", network: PanelNetwork, state: "exited", wantError: "未运行"},
		{name: "host 网络不注入容器 DNS", appNode: "7", network: "host", state: "running", wantError: "外部连接地址"},
		{name: "他人实例不可复用", appNode: "7", network: PanelNetwork, state: "running", restricted: true, wantError: "实例不存在"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newMiddlewareTestService(t, func(node string, req dto.ExecReq) dto.ExecResp {
				if node != "7" || !strings.HasPrefix(req.Command, "docker ps -a ") {
					t.Error("复用必须检查实例所在节点网络")
				}
				return dto.ExecResp{Output: middlewareSnapshot(middlewareContainer{ContainerName: "queue", Image: "rabbitmq:4.2", State: tc.state, Networks: PanelNetwork})}
			})
			inst := &model.MiddlewareInstance{Name: "queue-managed", Type: "rabbitmq", Origin: "container", Host: "queue", ContainerName: "queue", Port: 5672, NodeID: "7", OwnerID: 99}
			mustCreateMiddlewareRows(t, s, inst)
			ctx := t.Context()
			if tc.restricted {
				ctx = rbac.WithCaller(ctx, rbac.Caller{UserID: 42, DataScope: "assigned", AllNodes: true})
			}
			store := &StoreService{mws: s}
			params := map[string]string{}
			err := store.applyExternalMiddleware(ctx, func(string, string, ...any) {}, StoreMiddlewareDecl{Type: "rabbitmq", Prefix: "MQ"}, inst.ID, StoreInstallInput{NodeID: tc.appNode, Network: tc.network}, params, "${MQ_HOST}", nil)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) || len(params) != 0 {
					t.Fatalf("应拒绝并且不注入参数: %v, %+v", err, params)
				}
			} else if err != nil || params["MQ_HOST"] != "queue" {
				t.Fatalf("复用失败: %v, %+v", err, params)
			}
		})
	}
}
