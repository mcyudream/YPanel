#!/bin/bash
# ==============================================================================
# YPanel 一键安装脚本（Linux / systemd）
#
# 用法：
#   curl -sSL https://gitee.com/mcyudream/ypanel/raw/main/deploy/quick_start.sh | bash -s -- [选项]
#   bash quick_start.sh [选项]
#
# 选项：
#   --source github|gitee      下载源（默认 auto：先 Gitee 后 GitHub，国内环境无需配置）
#   --version vX.Y.Z           指定版本（默认最新 Release）
#   --port N                   面板端口（默认 8880）
#   --password XXXX            初始 admin 密码（默认随机生成）
#   --with-docker              同时安装 Docker Engine + Compose（已装则跳过）
#   --docker-mirror SRC        Docker 安装源：official|aliyun|tuna|ustc（默认 tuna）
#   --with-agent               额外安装独立节点 agent（ypagent，多节点场景）
#   --uninstall                卸载（保留数据目录）
#   --upgrade                  升级到最新版（保留数据与配置）
#
# 仓库：https://github.com/mcyudream/YPanel | https://gitee.com/mcyudream/ypanel
# ==============================================================================
set -eu

REPO_GITHUB="mcyudream/YPanel"
REPO_GITEE="mcyudream/ypanel"
INSTALL_DIR="/opt/ypanel"
DATA_DIR="${INSTALL_DIR}/data"
UPDATE_DIR="${INSTALL_DIR}/updates"
SERVICE_NAME="ypanel"

SOURCE="auto"
VERSION=""
PORT="8880"
ADMIN_PASSWORD=""
WITH_DOCKER=0
DOCKER_MIRROR="tuna"
WITH_AGENT=0
UNINSTALL=0
UPGRADE=0

log()  { printf '\033[32m[YPanel]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[YPanel]\033[0m %s\n' "$*"; }
err()  { printf '\033[31m[YPanel]\033[0m %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --source)        SOURCE="$2"; shift 2 ;;
    --version)       VERSION="$2"; shift 2 ;;
    --port)          PORT="$2"; shift 2 ;;
    --password)      ADMIN_PASSWORD="$2"; shift 2 ;;
    --with-docker)   WITH_DOCKER=1; shift ;;
    --docker-mirror) DOCKER_MIRROR="$2"; shift 2 ;;
    --with-agent)    WITH_AGENT=1; shift ;;
    --uninstall)     UNINSTALL=1; shift ;;
    --upgrade)       UPGRADE=1; shift ;;
    *) err "未知参数: $1（见脚本头部用法）" ;;
  esac
done

# ---- 前置检查 ----
[ "$(id -u)" = "0" ] || err "请以 root 运行（sudo bash quick_start.sh）"
command -v systemctl >/dev/null 2>&1 || err "未检测到 systemd（仅支持 systemd 发行版）"
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  PKG_ARCH="amd64" ;;
  aarch64) PKG_ARCH="arm64" ;;
  *) err "暂不支持的架构: $ARCH（支持 x86_64 / aarch64）" ;;
esac

# ---- 卸载 ----
if [ "$UNINSTALL" = "1" ]; then
  systemctl disable --now "$SERVICE_NAME" 2>/dev/null || true
  systemctl disable --now ypagent 2>/dev/null || true
  rm -f /etc/systemd/system/ypanel.service /etc/systemd/system/ypagent.service
  systemctl daemon-reload
  rm -rf "$INSTALL_DIR"
  log "已卸载（数据目录一并移除；如需保留数据请先备份 $DATA_DIR）"
  exit 0
fi

# ---- 下载源解析 ----
fetch() { # fetch <gitee_url> <github_url> <out_file>
  _gitee="$1"; _github="$2"; _out="$3"
  case "$SOURCE" in
    gitee)  curl -fSL --retry 2 --connect-timeout 10 -o "$_out" "$_gitee" ;;
    github) curl -fSL --retry 2 --connect-timeout 10 -o "$_out" "$_github" ;;
    *)      curl -fSL --retry 2 --connect-timeout 8 -o "$_out" "$_gitee" 2>/dev/null \
       || curl -fSL --retry 2 --connect-timeout 10 -o "$_out" "$_github" ;;
  esac
}

# 解析下载地址：指定版本按 releases/download 规律拼；最新版走 Release API（不依赖 jq）
# ASSET_TAR / ASSET_SUM 始终落在同一域，避免 gitee API 配 github 附件的错配
resolve_asset() {
  _domain="gitee.com"; _repo="${REPO_GITEE}"
  [ "$SOURCE" = "github" ] && _domain="github.com" && _repo="${REPO_GITHUB}"
  if [ -n "$VERSION" ]; then
    ASSET_TAR="https://${_domain}/${_repo}/releases/download/${VERSION}/ypanel-linux-${PKG_ARCH}.tar.gz"
    ASSET_SUM="https://${_domain}/${_repo}/releases/download/${VERSION}/sha256sums.txt"
    return 0
  fi
  _api="https://gitee.com/api/v5/repos/${REPO_GITEE}/releases/latest"
  [ "$SOURCE" = "github" ] && _api="https://api.github.com/repos/${REPO_GITHUB}/releases/latest"
  resp=$(curl -fsSL --connect-timeout 8 "$_api" 2>/dev/null || true)
  if [ -z "$resp" ] && [ "$SOURCE" != "github" ]; then
    resp=$(curl -fsSL --connect-timeout 8 "https://api.github.com/repos/${REPO_GITHUB}/releases/latest" 2>/dev/null || true)
  fi
  [ -z "$resp" ] && err "无法从 Gitee/GitHub 获取最新版本信息（检查网络或用 --version 指定）"
  ASSET_TAR=$(printf '%s' "$resp" | grep -o '"browser_download_url":[[:space:]]*"[^"]*ypanel-linux-'"${PKG_ARCH}"'\.tar\.gz"' | head -1 | sed 's/.*"\(https[^"]*\)"/\1/')
  ASSET_SUM=$(printf '%s' "$resp" | grep -o '"browser_download_url":[[:space:]]*"[^"]*sha256sums\.txt"' | head -1 | sed 's/.*"\(https[^"]*\)"/\1/')
  [ -n "$ASSET_TAR" ] && [ -n "$ASSET_SUM" ] || err "Release 资产不完整（缺 ypanel-linux-${PKG_ARCH}.tar.gz 或 sha256sums.txt）"
}

resolve_asset
# auto 模式经 GitHub API 解析出的地址在 github 域，统一回写 gitee 域（gitee 附件支持 releases/download 直链）
if [ "$SOURCE" != "github" ]; then
  case "$ASSET_TAR" in
    https://gitee.com/*) ;;
    *)
      ASSET_TAR="${ASSET_TAR/https:\/\/github.com\/${REPO_GITHUB}\//https:\/\/gitee.com\/${REPO_GITEE}\/}"
      ASSET_SUM="${ASSET_SUM/https:\/\/github.com\/${REPO_GITHUB}\//https:\/\/gitee.com\/${REPO_GITEE}\/}"
      ;;
  esac
fi

# ---- 可选：安装 Docker（与面板「容器→一键安装」同一套源清单） ----
install_docker() {
  if command -v docker >/dev/null 2>&1; then
    log "Docker 已安装（$(docker --version 2>/dev/null | head -1)），跳过"
    return 0
  fi
  . /etc/os-release
  case "${ID:-}" in
    debian|ubuntu) FAMILY=debian ;;
    centos|rhel|rocky|almalinux|anolis|alinux|opencloudos|tencentos) FAMILY=rhel ;;
    *) warn "发行版 ${ID:-未知} 暂不支持自动安装 Docker，可装好面板后在「容器 → 一键安装」处理"; return 0 ;;
  esac
  case "$DOCKER_MIRROR" in
    official) APT_BASE="https://download.docker.com/linux";   YUM_BASE="https://download.docker.com/linux/centos" ;;
    aliyun)   APT_BASE="https://mirrors.aliyun.com/docker-ce/linux";  YUM_BASE="https://mirrors.aliyun.com/docker-ce/linux/centos" ;;
    ustc)     APT_BASE="https://mirrors.ustc.edu.cn/docker-ce/linux"; YUM_BASE="https://mirrors.ustc.edu.cn/docker-ce/linux/centos" ;;
    *)        APT_BASE="https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux"; YUM_BASE="https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/centos" ;;
  esac
  PKGS="docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"
  if [ "$FAMILY" = "debian" ]; then
    CODENAME="${VERSION_CODENAME:-$(lsb_release -cs 2>/dev/null || true)}"
    [ -n "$CODENAME" ] || { warn "无法确定发行版代号，跳过 Docker 安装"; return 0; }
    case "$ARCH" in x86_64) APT_ARCH=amd64 ;; aarch64) APT_ARCH=arm64 ;; esac
    log "安装 Docker（apt / ${DOCKER_MIRROR}）…"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq ca-certificates curl gnupg >/dev/null
    install -m 0755 -d /etc/apt/keyrings
    # keyring 必须存 .gpg：apt 2.4 对 signed-by 的 .asc 按 armored 解析，dearmor 二进制会 NO_PUBKEY
    curl -fsSL "${APT_BASE}/${ID}/gpg" | gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
    chmod 0644 /etc/apt/keyrings/docker.gpg
    printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] %s/%s %s stable\n' \
      "$APT_ARCH" "$APT_BASE" "$ID" "$CODENAME" > /etc/apt/sources.list.d/docker.list
    apt-get update -qq || { sleep 3; apt-get update -qq; }
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq $PKGS
  else
    MAJOR=$(awk -F= '/^VERSION_ID=/{gsub(/"/,"",$2); split($2,a,"."); print a[1]}' /etc/os-release)
    # Alibaba Cloud Linux 3 兼容 RHEL8、2 兼容 RHEL7；仓库路径用显式主版本（勿用 $releasever）
    if [ "${ID}" = "alinux" ]; then [ "$MAJOR" = "3" ] && MAJOR=8; [ "$MAJOR" = "2" ] && MAJOR=7; fi
    case "$MAJOR" in 7|8|9) ;; *) warn "docker-ce 仓库不支持 ${ID} ${MAJOR}，跳过 Docker 安装"; return 0 ;; esac
    log "安装 Docker（yum / ${DOCKER_MIRROR}）…"
    command -v curl >/dev/null 2>&1 || yum install -y curl
    curl -fsSL -o /etc/yum.repos.d/docker-ce.repo "${YUM_BASE}/docker-ce.repo"
    sed -i "s#https://download.docker.com/linux/centos#${YUM_BASE}#g; s#\\\$releasever#${MAJOR}#g" /etc/yum.repos.d/docker-ce.repo
    yum -y makecache || { sleep 3; yum -y makecache; }
    yum install -y $PKGS
  fi
  systemctl enable --now docker
  log "Docker 安装完成：$(docker --version)"
}

[ "$WITH_DOCKER" = "1" ] && install_docker

# ---- 下载与校验 ----
mkdir -p "$INSTALL_DIR" "$UPDATE_DIR" "$DATA_DIR"
TMP_TAR="${UPDATE_DIR}/ypanel-linux-${PKG_ARCH}.tar.gz"
log "下载 YPanel ${VERSION:-最新版}（linux-${PKG_ARCH}）…"
fetch "$ASSET_TAR" "$(printf '%s' "$ASSET_TAR" | sed 's#https://gitee.com/#https://github.com/#')" "$TMP_TAR" \
  || err "下载失败：$ASSET_TAR"
SUM_REMOTE=$(printf '%s' "$ASSET_SUM" | sed 's#https://gitee.com/#https://github.com/#')
if curl -fsSL --connect-timeout 8 -o "${UPDATE_DIR}/sha256sums.txt" "$SUM_REMOTE" 2>/dev/null; then
  (cd "$UPDATE_DIR" && sha256sum -c sha256sums.txt --ignore-missing >/dev/null 2>&1) \
    || err "sha256 校验失败（下载不完整或源被篡改），已中止"
  rm -f "${UPDATE_DIR}/sha256sums.txt"
  log "sha256 校验通过"
else
  warn "未获取到 sha256sums.txt，跳过校验"
fi

if [ "$UPGRADE" = "1" ]; then
  tar -xzf "$TMP_TAR" -C "$UPDATE_DIR"
  rm -f "$TMP_TAR"
  [ -f "${UPDATE_DIR}/ypanel" ] || err "包内缺少 ypanel 二进制"
  log "应用更新（约 3 秒后自动替换并重启）…"
  setsid nohup sh -c "sleep 2; cp ${INSTALL_DIR}/ypanel ${INSTALL_DIR}/ypanel.bak; mv ${UPDATE_DIR}/ypanel ${INSTALL_DIR}/ypanel; chmod +x ${INSTALL_DIR}/ypanel; systemctl restart ${SERVICE_NAME}" >/tmp/ypanel-upgrade.log 2>&1 < /dev/null &
  log "升级已启动，稍后用 ypanel version 或访问面板确认新版本"
  exit 0
fi

# ---- 全新安装 ----
if [ -f "${INSTALL_DIR}/ypanel" ] && [ "$UPGRADE" != "1" ]; then
  warn "检测到已有安装（$INSTALL_DIR/ypanel），本次仅覆盖二进制（数据保留）"
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
fi
tar -xzf "$TMP_TAR" -C "$INSTALL_DIR"
rm -f "$TMP_TAR"
chmod +x "${INSTALL_DIR}/ypanel"
[ -f "${INSTALL_DIR}/ypagent" ] && chmod +x "${INSTALL_DIR}/ypagent"

# 初始密码（未指定则生成 16 位随机）
if [ -z "$ADMIN_PASSWORD" ]; then
  ADMIN_PASSWORD=$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 16)
fi
# 敏感值走 env 文件（0600），不进 argv/日志
cat > "${INSTALL_DIR}/ypanel.env" <<EOF
YPANEL_ADMIN_PASSWORD=${ADMIN_PASSWORD}
YPANEL_PORT=${PORT}
YPANEL_DATA_DIR=${DATA_DIR}
EOF
chmod 600 "${INSTALL_DIR}/ypanel.env"

cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=YPanel Server
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/ypanel.env
ExecStart=${INSTALL_DIR}/ypanel -port \${YPANEL_PORT} -data \${YPANEL_DATA_DIR}
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME"

log "等待服务就绪…"
for i in $(seq 1 20); do
  sleep 1
  if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    VERSION_NOW=$(curl -sf "http://127.0.0.1:${PORT}/health" | sed 's/.*"version":"\([^"]*\)".*/\1/')
    break
  fi
done

# 安全入口（M49 强制开启，首启自动生成；不带入口段访问面板一律 404 伪装）
ENTRY=""
for i in $(seq 1 10); do
  ENTRY=$(journalctl -u "${SERVICE_NAME}" --no-pager 2>/dev/null | grep -oE '安全入口已自动生成[^/]*/[a-z0-9]{6,}' | tail -1 | grep -oE '/[a-z0-9]{6,}$' || true)
  [ -n "$ENTRY" ] && break
  sleep 2
done

# ---- 可选：独立 agent（多节点场景；面板节点管理生成配对码后手动接入） ----
if [ "$WITH_AGENT" = "1" ] && [ -f "${INSTALL_DIR}/ypagent" ]; then
  cat > /etc/systemd/system/ypagent.service <<EOF
[Unit]
Description=YPanel Node Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/ypagent
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  log "ypagent 已安装（未启动）：在主面板「节点管理」生成配对码后执行"
  log "  /opt/ypagent -core http://<主面板地址>:<端口> -code <配对码> -name <节点名>"
fi

PUBLIC_IP=$(curl -sf --connect-timeout 3 https://ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")
echo
log "=============================================="
log " YPanel 安装完成${VERSION_NOW:+（版本 ${VERSION_NOW}）}"
log " 访问地址:  http://${PUBLIC_IP}:${PORT}${ENTRY}"
log " 默认账号:  admin"
log " 初始密码:  ${ADMIN_PASSWORD}"
if [ -n "$ENTRY" ]; then
  log " 安全入口:  ${ENTRY}（已拼入访问地址；URL 含入口段请妥善保存，可在面板「安全设置」修改）"
  log " 注意:      不带入口段的访问一律 404（安全伪装），登录 API 同样要求入口"
else
  warn "未能从日志解析安全入口，请执行 journalctl -u ${SERVICE_NAME} | grep 安全入口 获取"
fi
log " 数据目录:  ${DATA_DIR}"
log " 服务管理:  systemctl status ${SERVICE_NAME}"
log " Docker:    装好面板后可在「容器」域一键安装/管理"
log "=============================================="
