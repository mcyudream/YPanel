// M68 单测：键集扩展（MONGO_URI/S3/NEO4J）、语义化版本比较、声明规范化、可选服务 env。
package service

import (
	"testing"
)

func TestDBFieldSetsOfM68(t *testing.T) {
	fields := []StoreFormField{
		{EnvKey: "MONGO_URI"},
		{EnvKey: "S3_ENDPOINT"},
		{EnvKey: "NEO4J_URI"},
	}
	sets := dbFieldSetsOf(fields, "environment:\n  - S3_BUCKET=${S3_BUCKET}\n  - REDIS_HOST=${REDIS_HOST}")
	kinds := map[string]string{}
	for _, fs := range sets {
		if fs.Uri != "" {
			kinds[fs.Uri] = fs.Kind
			continue
		}
		if fs.Host != "" {
			kinds[fs.Host] = fs.Kind
		}
	}
	if kinds["MONGO_URI"] != "mongo" {
		t.Fatalf("MONGO_URI 应识别为 mongo 键集，got %v", kinds)
	}
	if kinds["S3_ENDPOINT"] != "s3" {
		t.Fatalf("S3_ENDPOINT 应识别为 s3 键集，got %v", kinds)
	}
	if kinds["NEO4J_URI"] != "neo4j" {
		t.Fatalf("NEO4J_URI 应识别为 neo4j 键集，got %v", kinds)
	}
	if kinds["REDIS_HOST"] != "redis" {
		t.Fatalf("REDIS_HOST 应回退 redis，got %v", kinds)
	}
	// 普通 SQL 键集不受影响
	sets2 := dbFieldSetsOf([]StoreFormField{{EnvKey: "DATABASE_HOST"}}, "")
	if len(sets2) != 1 || sets2[0].Kind != "sql" {
		t.Fatalf("DATABASE_HOST 应为 sql 键集，got %+v", sets2)
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"4.3.2", "3.10", 1},
		{"3.9", "3.10", -1},
		{"5.26", "5.26.7", -1},
		{"5.26.7", "5.26", 1},
		{"1.6", "1.6", 0},
		{"", "3.10", 0}, // 不可解析 → 0（跳过校验口径）
	}
	for _, c := range cases {
		if got := CompareSemver(c.a, c.b); got != c.want {
			t.Fatalf("CompareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestMiddlewareVersionOK(t *testing.T) {
	if !middlewareVersionOK("", "3.10") {
		t.Fatal("实例版本未知应放行")
	}
	if !middlewareVersionOK("4.3.2", "") {
		t.Fatal("基线为空应放行")
	}
	if middlewareVersionOK("3.9", "3.10") {
		t.Fatal("3.9 < 3.10 应禁用")
	}
	if !middlewareVersionOK("4.3.2", "3.10") {
		t.Fatal("4.3.2 ≥ 3.10 应放行")
	}
}

func TestMiddlewareDeclsWithDefaults(t *testing.T) {
	out := middlewareDeclsWithDefaults([]StoreMiddlewareDecl{
		{Type: "rabbitmq", Prefix: "rabbitmq"}, // label/minVersion 回退描述符，prefix 大写化
		{Type: "", Prefix: "X"},                // 无类型丢弃
	})
	if len(out) != 1 {
		t.Fatalf("应保留 1 条有效声明，got %d", len(out))
	}
	if out[0].Prefix != "RABBITMQ" {
		t.Fatalf("prefix 应大写化，got %s", out[0].Prefix)
	}
	if out[0].Label != "RabbitMQ" || out[0].MinVersion != "3.10" {
		t.Fatalf("label/minVersion 应回退描述符，got %+v", out[0])
	}
}

func TestOptionalServiceEnv(t *testing.T) {
	ver := StoreVersion{OptionalServices: []StoreOptionalSvc{
		{ID: "render-server", Env: map[string]string{"PLATFORM_MESSAGE_RENDER_ENABLED": "true", "MESSAGE_RENDER_BASE_URL": "http://render-server:3000"}},
		{ID: "kkfileview", Env: map[string]string{"PLATFORM_FILE_PREVIEW_ENABLED": "true"}},
	}}
	params := map[string]string{}
	applied := optionalServiceEnv(ver, []string{"kkfileview"}, params)
	if len(applied) != 1 || applied[0] != "kkfileview" {
		t.Fatalf("应只启用 kkfileview，got %v", applied)
	}
	if params["PLATFORM_FILE_PREVIEW_ENABLED"] != "true" {
		t.Fatalf("开关 env 未注入: %v", params)
	}
	if _, exists := params["PLATFORM_MESSAGE_RENDER_ENABLED"]; exists {
		t.Fatal("未勾选组件的 env 不应注入")
	}
}

func TestVersionFromImage(t *testing.T) {
	cases := map[string]string{
		"rabbitmq:4.3.2-management-alpine": "4.3.2",
		"neo4j:5.26-community":             "5.26",
		"redis":                            "",
		"rustfs/rustfs:latest":             "",
		"mongo:8.0.13":                     "8.0.13",
	}
	for img, want := range cases {
		if got := versionFromImage(img); got != want {
			t.Fatalf("versionFromImage(%q)=%q want %q", img, got, want)
		}
	}
}
