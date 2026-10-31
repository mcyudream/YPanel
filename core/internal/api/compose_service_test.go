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
	"github.com/ypanel/shared/errs"
)

// 走真实 JSON 绑定，旧 name 和新 project 都须通过参数校验并进入服务层。
func TestComposeServiceActionAPIPayload(t *testing.T) {
	for _, tc := range []struct {
		name, query, body, message string
		code                       int
		restricted                 bool
	}{
		{name: "旧页面名称", body: `{"name":"demo","service":"web","action":"restart"}`, code: errs.CodeInternal, message: "节点服务不可用"},
		{name: "标准项目和外部目录", query: "node=7", body: `{"project":"demo","service":"web","action":"restart","dir":"/srv/external app"}`, code: errs.CodeInternal, message: "节点服务不可用"},
		{name: "节点查询别名", query: "nodeId=7", body: `{"project":"demo","service":"web","action":"restart"}`, code: errs.CodeInternal, message: "节点服务不可用"},
		{name: "冲突项目", body: `{"project":"demo","name":"other","service":"web","action":"restart"}`, code: errs.CodeBadRequest, message: "不一致"},
		{name: "缺少项目", body: `{"service":"web","action":"restart"}`, code: errs.CodeBadRequest, message: "编排项目"},
		{name: "缺少服务", body: `{"project":"demo","action":"restart"}`, code: errs.CodeBadRequest, message: "编排服务"},
		{name: "非法动作", body: `{"project":"demo","service":"web","action":"delete"}`, code: errs.CodeBadRequest, message: "服务操作只支持"},
		{name: "相对目录", body: `{"project":"demo","service":"web","action":"restart","dir":"../other"}`, code: errs.CodeBadRequest, message: "绝对路径"},
		{name: "冲突节点", query: "node=7&nodeId=8", body: `{"project":"demo","service":"web","action":"restart"}`, code: errs.CodeBadRequest, message: "不一致"},
		{name: "实际节点越权", query: "node=8", restricted: true, body: `{"project":"demo","service":"web","action":"restart"}`, code: errs.CodeForbidden, message: "节点"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/compose/service-action?"+tc.query, strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if tc.restricted {
				c.Request = c.Request.WithContext(rbac.WithCaller(context.Background(), rbac.Caller{NodeSet: map[string]struct{}{"7": {}}}))
			}
			(&ComposeAPI{}).ServiceAction(c)
			var out struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Code != tc.code || !strings.Contains(out.Message, tc.message) || strings.Contains(out.Message, "required") {
				t.Fatalf("参数绑定或业务错误不正确: %s, err=%v", w.Body.String(), err)
			}
		})
	}
}
