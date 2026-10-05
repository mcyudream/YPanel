#!/usr/bin/env bash
# YPanel 部署脚本：构建 linux-amd64 → SSH 推送 → systemd 常驻 → 健康检查。
#
# 环境变量（敏感信息一律经环境传入，不落仓库）：
#   DEPLOY_HOST   目标主机（必填）
#   DEPLOY_PORT   SSH 端口（默认 22）
#   DEPLOY_USER   SSH 用户（默认 root）
#   SSHPASS       SSH 密码（必填）
#   YPANEL_ADMIN_PASSWORD  初始 admin 密码（仅首启建号时生效；写远端 /opt/ypanel/ypanel.env，权限 600）
#
# 用法: DEPLOY_HOST=x.x.x.x SSHPASS=xxx bash scripts/deploy.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

: "${DEPLOY_HOST:?缺少 DEPLOY_HOST}"
: "${SSHPASS:?缺少 SSHPASS}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_USER="${DEPLOY_USER:-root}"

SSHCTL="$ROOT/bin/sshctl"
if [[ ! -x "$SSHCTL" ]]; then
  echo "== 构建 sshctl =="
  cd "$ROOT/tools/sshctl" && CGO_ENABLED=0 go build -o "$SSHCTL" .
fi

# Git Bash/MSYS 下禁用 POSIX 路径自动转换（防止 /opt/... 被改写为 Windows 路径）；
# 仅作用于 sshctl（全局设置会破坏 pnpm/node 的路径解析）
remote() {
  MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*" "$SSHCTL" -host "$DEPLOY_HOST" -port "$DEPLOY_PORT" -user "$DEPLOY_USER" "$@"
}

echo "== [1/4] 构建 linux-amd64 产物 =="
bash "$ROOT/scripts/build.sh" linux-amd64

echo "== [2/4] 推送产物 =="
# 本地路径转 Windows 形式（Git Bash 下 sshctl 为原生程序）；Linux 上 cygpath 不存在则原样
LOCAL_BIN="$ROOT/bin/ypanel-linux-amd64"
LOCAL_UNIT="$ROOT/scripts/ypanel.service"
if command -v cygpath >/dev/null 2>&1; then
  LOCAL_BIN="$(cygpath -w "$LOCAL_BIN")"
  LOCAL_UNIT="$(cygpath -w "$LOCAL_UNIT")"
fi
# 先停服务再覆盖二进制（运行中的文件不可写：Text file busy）
remote exec -- "systemctl stop ypanel 2>/dev/null; mkdir -p /opt/ypanel/data"
remote put -local "$LOCAL_BIN" -remote /opt/ypanel/ypanel
remote put -local "$LOCAL_UNIT" -remote /etc/systemd/system/ypanel.service
if [[ -n "${YPANEL_ADMIN_PASSWORD:-}" ]]; then
  remote exec -- "umask 077 && echo 'YPANEL_ADMIN_PASSWORD=$YPANEL_ADMIN_PASSWORD' > /opt/ypanel/ypanel.env"
fi

echo "== [3/4] 安装 systemd 服务 =="
remote exec -- "chmod +x /opt/ypanel/ypanel && systemctl daemon-reload && systemctl enable --now ypanel >/dev/null 2>&1 || systemctl restart ypanel"

echo "== [4/4] 健康检查 =="
sleep 3
HEALTH=$(remote exec -- "curl -sf http://127.0.0.1:8880/health || echo FAIL" || echo FAIL)
if [[ "$HEALTH" == *FAIL* || -z "$HEALTH" ]]; then
  echo "健康检查失败，远端日志："
  remote exec -- "journalctl -u ypanel -n 30 --no-pager"
  exit 1
fi
echo "部署完成: http://$DEPLOY_HOST:8880  ($HEALTH)"
