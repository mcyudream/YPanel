// 流式构建骨架行过滤单测（M59 runtime 构建进度）。
package service

import "testing"

func TestBuildStepLine(t *testing.T) {
	keep := []string{
		"#5 [php 1/7] FROM docker.io/library/php:8.2-fpm-alpine",
		"#5 DONE 0.5s",
		"#12 [php 2/7] RUN apk add --no-cache gcc",
		"#12 CACHED",
		"#20 [php 7/7] RUN install-ext.sh gd imagick",
		"#20 ERROR: process \"/bin/sh\" did not complete successfully",
		"#24 writing image sha256:abc123",
		"#25 naming to docker.io/library/php-main:8.2",
	}
	for _, l := range keep {
		if !buildStepLine(l) {
			t.Fatalf("应保留: %q", l)
		}
	}
	skip := []string{
		"#0 building with \"default\" instance using docker driver", // 步骤内文本（非骨架）
		"#5 resolve docker.io/library/php:8.2-fpm-alpine",           // 步骤子行
		"#12 1.234 fetch https://dl-cdn.alpinelinux.org/alpine/v3.18/main",
		"#20 45.67 In file included from gd.c:30:",
		"NOTE: something",
		"#abc DONE",
		"#",
		"",
	}
	for _, l := range skip {
		if buildStepLine(l) {
			t.Fatalf("应过滤: %q", l)
		}
	}
}
