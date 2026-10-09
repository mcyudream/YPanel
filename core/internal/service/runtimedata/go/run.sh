#!/bin/sh
# YPanel Go 运行时启动脚本（go mod 缓存挂载在 /go/pkg/mod）
set -e
cd /app
if [ "$RUN_INSTALL" = "1" ]; then
  go mod tidy || go mod download
fi
exec sh -c "${START_CMD:-go run .}"
