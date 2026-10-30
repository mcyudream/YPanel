package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

func TestComposeUpdateAPITarget(t *testing.T) {
	for _, tc := range []struct {
		name, query, bodyNode, project, oldName, wantNode string
		code                                              int
		restricted                                        bool
	}{
		{name: "旧版默认本机", oldName: "demo", wantNode: "local"},
		{name: "旧版名称加查询节点", query: "node=7", oldName: "demo", wantNode: "7"},
		{name: "显式远程节点", query: "node=7", bodyNode: "7", project: "demo", wantNode: "7"},
		{name: "查询节点别名", query: "nodeId=7", project: "demo", wantNode: "7"},
		{name: "请求体节点", bodyNode: "7", project: "demo", wantNode: "7"},
		{name: "查询参数冲突", query: "node=7&nodeId=8", project: "demo", code: errs.CodeBadRequest},
		{name: "请求体与查询冲突", query: "node=7", bodyNode: "8", project: "demo", code: errs.CodeBadRequest},
		{name: "项目别名冲突", project: "demo", oldName: "other", code: errs.CodeBadRequest},
		{name: "缺少项目", code: errs.CodeBadRequest},
		{name: "实际节点越权", bodyNode: "8", project: "demo", restricted: true, code: errs.CodeForbidden},
		{name: "隐式本机越权", project: "demo", restricted: true, code: errs.CodeForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/compose/pull-update?"+tc.query, nil)
			if tc.restricted {
				callerCtx := rbac.WithCaller(context.Background(), rbac.Caller{NodeSet: map[string]struct{}{"7": {}}})
				c.Request = c.Request.WithContext(callerCtx)
			}
			req := dto.ComposePullUpdateReq{NodeID: tc.bodyNode, Project: tc.project, Name: tc.oldName}
			ok := composeUpdateTarget(c, &req)
			if tc.code == 0 {
				if !ok || req.NodeID != tc.wantNode || req.Project != "demo" {
					t.Fatalf("参数未归一化: %+v, ok=%v, body=%s", req, ok, w.Body.String())
				}
				return
			}
			var response struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || ok || response.Code != tc.code {
				t.Fatalf("未返回预期错误: %s, ok=%v, err=%v", w.Body.String(), ok, err)
			}
		})
	}
}

// 走真实 JSON 绑定和 handler，防止新旧字段再次因 required 标签在业务逻辑前被拒绝。
func TestComposePullUpdateAPIAcceptsLegacyPayload(t *testing.T) {
	api := &ComposeAPI{Src2: &service.Src2ComposeService{}}
	for _, body := range []string{
		`{"name":"demo","dir":"/srv/demo"}`,
		`{"project":"demo","nodeId":"local","dir":"/srv/demo"}`,
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/compose/pull-update", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		api.PullUpdate(c)
		if !strings.Contains(w.Body.String(), "更新任务服务不可用") || strings.Contains(w.Body.String(), "required") {
			t.Fatalf("参数绑定没有到达业务校验: %s", w.Body.String())
		}
	}
}
