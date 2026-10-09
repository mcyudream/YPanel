#!/bin/bash
# ==============================================================================
# YPanel Release 镜像脚本：把 GitHub Release 产物镜像到 Gitee（国内下载加速）。
#
# 背景：GitHub Actions runner 出口对 gitee 附件上传（30MB×2）挂连不可行（实测零字节挂满
# 超时），而国内/本地网络上传很快——CI 只建 Release 骨架，附件镜像由本脚本在本地执行。
#
# 用法（需 GITEE_TOKEN，可用环境变量或 -t 传入）：
#   GITEE_TOKEN=xxx bash deploy/mirror-release.sh v0.9.2
#   bash deploy/mirror-release.sh v0.9.2 -t xxx
# ==============================================================================
set -eu

TAG="${1:?用法: mirror-release.sh <vX.Y.Z> [-t gitee_token]}"
shift || true
TOKEN="${GITEE_TOKEN:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    -t) TOKEN="$2"; shift 2 ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
done
[ -n "$TOKEN" ] || { echo "缺少 GITEE_TOKEN（环境变量或 -t）" >&2; exit 2; }

REPO_GH="mcyudream/YPanel"
REPO_GITEE="mcyudream/ypanel"
API="https://gitee.com/api/v5/repos/${REPO_GITEE}"
WORK=$(mktemp -d /tmp/yp-mirror.XXXXXX)
trap 'rm -rf "$WORK"' EXIT

echo "[1/3] 下载 GitHub Release ${TAG} 资产…"
for f in ypanel-linux-amd64.tar.gz ypanel-linux-arm64.tar.gz sha256sums.txt; do
  curl -fsSL --retry 2 --max-time 600 -o "${WORK}/${f}" \
    "https://github.com/${REPO_GH}/releases/download/${TAG}/${f}"
  echo "  ${f}: $(du -h "${WORK}/${f}" | cut -f1)"
done
(cd "$WORK" && sha256sum -c sha256sums.txt --ignore-missing) || { echo "sha256 校验失败" >&2; exit 1; }

echo "[2/3] 确保目标主机已推送 tag ${TAG} 到 Gitee（git push gitee ${TAG}）"
# 幂等：已有 Release 复用
RID=$(curl -sS --max-time 20 "${API}/releases/tags/${TAG}?access_token=${TOKEN}" \
  | grep -o '"id": *[0-9]*' | head -1 | grep -o '[0-9]*' || true)
if [ -z "$RID" ]; then
  # target_commitish 必填（tag 已存在也不可省）；body 保持 ASCII（未转义中文 400）
  RID=$(curl -sS --max-time 20 -X POST "${API}/releases" \
    -H "Content-Type: application/json" \
    -d "{\"access_token\":\"${TOKEN}\",\"tag_name\":\"${TAG}\",\"target_commitish\":\"main\",\"name\":\"YPanel ${TAG}\",\"body\":\"YPanel ${TAG} - same build artifacts as the GitHub release\",\"prerelease\":false}" \
    | grep -o '"id": *[0-9]*' | head -1 | grep -o '[0-9]*' || true)
fi
[ -n "$RID" ] || { echo "Gitee Release 创建失败" >&2; exit 1; }
echo "  release id: $RID"

echo "[3/3] 上传附件…"
for f in ypanel-linux-amd64.tar.gz ypanel-linux-arm64.tar.gz sha256sums.txt; do
  curl -sS --max-time 600 --retry 1 -X POST "${API}/releases/${RID}/attach_files" \
    -F "access_token=${TOKEN}" \
    -F "file=@${WORK}/${f}" | head -c 120
  echo "  <- ${f}"
done
echo "镜像完成: https://gitee.com/${REPO_GITEE}/releases/tag/${TAG}"
