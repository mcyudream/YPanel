#!/usr/bin/env bash
# YPanel 构建脚本：前端产物 → core 嵌入 → 交叉编译。
# 用法: bash scripts/build.sh [target]
#   target: linux-amd64（默认）| windows-amd64 | all
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET="${1:-linux-amd64}"
VERSION="$(date +%Y%m%d.%H%M)-dev"

echo "== [1/3] 前端构建（web/apps/core-arco-design-vue）=="
cd "$ROOT/web/apps/core-arco-design-vue"
pnpm build

echo "== [2/3] 嵌入前端产物到 core =="
rm -rf "$ROOT/core/internal/web/dist"
cp -r dist "$ROOT/core/internal/web/dist"

echo "== [3/3] 编译（version=$VERSION）=="
LDFLAGS="-s -w -X main.version=$VERSION"
build_one() {
  local goos=$1 goarch=$2
  cd "$ROOT/core"
  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags "$LDFLAGS" -o "$ROOT/bin/ypanel-$goos-$goarch" ./cmd/ypanel
  echo "   产出 bin/ypanel-$goos-$goarch"
  # 多节点 agent 独立分发二进制
  cd "$ROOT/agent"
  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags "$LDFLAGS" -o "$ROOT/bin/ypagent-$goos-$goarch" ./cmd/agent
  echo "   产出 bin/ypagent-$goos-$goarch"
}
case "$TARGET" in
  linux-amd64) build_one linux amd64 ;;
  windows-amd64) build_one windows amd64 ;;
  all) build_one linux amd64; build_one windows amd64 ;;
  *) echo "未知 target: $TARGET"; exit 2 ;;
esac

echo "== 构建完成 =="
