#!/bin/bash
# ==============================================================================
# YPanel 一键安装脚本（Linux / systemd）——支持交互引导与纯节点 agent 安装
#
# 用法：
#   curl -sSL https://gitee.com/mcyudream/ypanel/raw/main/deploy/quick_start.sh | bash -s -- [选项]
#   bash quick_start.sh [选项]
#
# 不带参数且在终端执行时进入交互引导；CI/自动化传参即全自动。
#
# 通用选项：
#   --mode panel|node          安装模式：主面板 / 纯节点 agent（不装面板）
#   --source github|gitee      下载源（默认 auto：先 Gitee 后 GitHub）
#   --version vX.Y.Z           指定版本（默认最新 Release）
#   --uninstall                卸载（面板模式卸面板；节点模式卸 agent）
#   --upgrade                  面板升级到最新版（保留数据与配置）
#
# 主面板（panel）选项：
#   --port N                   面板端口（默认 8880）
#   --password XXXX            初始 admin 密码（默认随机生成）
#   --with-docker              同时安装 Docker Engine + Compose（已装则跳过）
#   --docker-mirror SRC        Docker 安装源：official|aliyun|tuna|ustc（默认 tuna）
#
# 节点（node）选项：
#   --core URL                 主面板地址（如 http://192.168.1.10:8880）
#   --code CODE                一次性配对码（主面板「节点管理」生成）
#   --node-name NAME           节点名（默认主机名小写）
#
# 仓库：https://github.com/mcyudream/YPanel | https://gitee.com/mcyudream/ypanel
# ==============================================================================
set -eu

REPO_GITHUB="mcyudream/YPanel"
REPO_GITEE="mcyudream/ypanel"
INSTALL_DIR="/opt/ypanel"
DATA_DIR="${INSTALL_DIR}/data"
UPDATE_DIR="${INSTALL_DIR}/updates"
NODE_DIR="/opt/ypagent"
SERVICE_NAME="ypanel"
AGENT_SERVICE="ypagent"

MODE=""                 # panel | node
SOURCE="auto"
VERSION=""
PORT="8880"
ADMIN_PASSWORD=""
WITH_DOCKER=0
DOCKER_MIRROR="tuna"
CORE_URL=""
PAIR_CODE=""
NODE_NAME=""
UNINSTALL=0
UPGRADE=0

log()  { printf '\033[32m[YPanel]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[YPanel]\033[0m %s\n' "$*"; }
err()  { printf '\033[31m[YPanel]\033[0m %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --mode)        MODE="$2"; shift 2 ;;
    --source)      SOURCE="$2"; shift 2 ;;
    --version)     VERSION="$2"; shift 2 ;;
    --port)        PORT="$2"; shift 2 ;;
    --password)    ADMIN_PASSWORD="$2"; shift 2 ;;
    --with-docker) WITH_DOCKER=1; shift ;;
    --docker-mirror) DOCKER_MIRROR="$2"; shift 2 ;;
    --core)        CORE_URL="$2"; shift 2 ;;
    --code)        PAIR_CODE="$2"; shift 2 ;;
    --node-name)   NODE_NAME="$2"; shift 2 ;;
    --uninstall)   UNINSTALL=1; shift ;;
    --upgrade)     UPGRADE=1; shift ;;
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

has_tty() {
  if [ -t 0 ]; then return 0; fi
  # curl|bash 场景：stdin 是管道，但调用方有 controlling terminal 时 /dev/tty 可用；
  # paramiko/CI exec 无 controlling terminal，/dev/tty 设备存在却打不开（ENXIO）——必须实测打开
  { printf '' >/dev/tty; } 2>/dev/null
}
# ask <变量名> <提示> <默认值>：从 /dev/tty 读取（curl|bash 时 stdin 被管道占用，必须走 tty）
ask() {
  local __n="$1" __p="$2" __d="$3" __v=""
  if has_tty; then
    printf '\033[32m[YPanel]\033[0m %s [%s]: ' "$__p" "$__d" > /dev/tty
    IFS= read -r __v < /dev/tty || __v=""
  fi
  if [ -n "$__v" ]; then printf -v "$__n" '%s' "$__v"; else printf -v "$__n" '%s' "$__d"; fi
}
ask_yn() {
  local __n="$1" __p="$2" __d="$3" __v=""
  if has_tty; then
    printf '\033[32m[YPanel]\033[0m %s [y/n，默认 %s]: ' "$__p" "$__d" > /dev/tty
    IFS= read -r __v < /dev/tty || __v=""
  fi
  case "${__v:-$__d}" in y|Y|yes|YES|1) printf -v "$__n" '%s' 1 ;; *) printf -v "$__n" '%s' 0 ;; esac
}

# ---- 卸载 ----
if [ "$UNINSTALL" = "1" ]; then
  if [ "$MODE" = "node" ]; then
    systemctl disable --now "$AGENT_SERVICE" 2>/dev/null || true
    rm -f /etc/systemd/system/${AGENT_SERVICE}.service
    systemctl daemon-reload
    rm -rf "$NODE_DIR" /etc/ypanel
    log "节点 agent 已卸载"
  else
    systemctl disable --now "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable --now "$AGENT_SERVICE" 2>/dev/null || true
    rm -f /etc/systemd/system/${SERVICE_NAME}.service /etc/systemd/system/${AGENT_SERVICE}.service
    systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    log "已卸载（数据目录一并移除；如需保留数据请先备份 $DATA_DIR）"
  fi
  exit 0
fi

# ---- 交互引导（未指定 --mode 且终端可用时；传参的项跳过提问） ----
if [ -z "$MODE" ] && has_tty; then
  echo
  log "YPanel 安装引导（回车采用默认值）"
  local_mode=""
  ask local_mode "安装模式：1) 主面板  2) 节点 agent（被已有面板纳管，不装面板）" "1"
  [ "$local_mode" = "2" ] && MODE=node || MODE=panel
fi
[ -n "$MODE" ] || MODE=panel   # 无 tty 且未指定：保持兼容默认装面板

if [ "$MODE" = "panel" ] && has_tty && [ "$UPGRADE" != "1" ]; then
  ask PORT "面板端口" "$PORT"
  if [ -z "$ADMIN_PASSWORD" ]; then
    ask ADMIN_PASSWORD "初始 admin 密码（留空则随机生成）" ""
    if [ -z "$ADMIN_PASSWORD" ]; then
      ADMIN_PASSWORD=$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 16)
      log "已生成随机密码：$ADMIN_PASSWORD"
    fi
  fi
  if [ "$WITH_DOCKER" = "0" ]; then
    ask_yn WITH_DOCKER "同时安装 Docker Engine + Compose（已装自动跳过）？" "n"
  fi
  if [ "$WITH_DOCKER" = "1" ] && [ "$DOCKER_MIRROR" = "tuna" ]; then
    ask DOCKER_MIRROR "Docker 安装源：tuna/aliyun/ustc/official" "tuna"
  fi
  ask SOURCE "软件下载源：auto/gitee/github" "$SOURCE"
elif [ "$MODE" = "node" ] && has_tty; then
  if [ -z "$CORE_URL" ]; then
    ask CORE_URL "主面板地址（如 http://192.168.1.10:8880）" ""
    [ -z "$CORE_URL" ] && err "必须提供主面板地址（--core）"
  fi
  if [ -z "$PAIR_CODE" ]; then
    log "请先在主面板「节点管理 → 添加节点」生成配对码"
    ask PAIR_CODE "配对码" ""
    [ -z "$PAIR_CODE" ] && err "必须提供配对码（--code）"
  fi
  if [ -z "$NODE_NAME" ]; then
    __hn=$(hostname | tr 'A-Z' 'a-z' | tr -c 'a-z0-9-' '-' | sed 's/-*$//')
    ask NODE_NAME "节点名" "${__hn:-node-$(date +%s)}"
  fi
  ask SOURCE "软件下载源：auto/gitee/github" "$SOURCE"
fi

# ---- 引导后的参数兜底 ----
if [ "$MODE" = "node" ]; then
  [ -n "$CORE_URL" ] || err "节点模式必须提供 --core（主面板地址）"
  [ -n "$PAIR_CODE" ] || err "节点模式必须提供 --code（配对码）"
  if [ -z "$NODE_NAME" ]; then
    NODE_NAME=$(hostname | tr 'A-Z' 'a-z' | tr -c 'a-z0-9-' '-' | sed 's/-*$//')
  fi
  [ -n "$NODE_NAME" ] || NODE_NAME="node-$(date +%s)"
  echo "$NODE_NAME" | grep -qE '^[a-z][a-z0-9-]{1,30}[a-z0-9]$' || err "节点名不合法（小写字母开头，小写字母/数字/连字符，3~32 位）: $NODE_NAME"
  [ "$UPGRADE" = "1" ] && err "节点模式不支持 --upgrade（升级请用面板「节点管理」的一键更新）"
fi

# ---- 下载源解析（panel 与 node 共用同一 Release 包，内含 ypanel + ypagent） ----
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

[ "$MODE" = "panel" ] && [ "$WITH_DOCKER" = "1" ] && install_docker

# ---- 下载与校验 ----
TMP_DIR=$(mktemp -d /tmp/ypanel-install.XXXXXX)
TMP_TAR="${TMP_DIR}/ypanel-linux-${PKG_ARCH}.tar.gz"
log "下载 YPanel ${VERSION:-最新版}（linux-${PKG_ARCH}）…"
_GH_TAR=$(printf '%s' "$ASSET_TAR" | sed "s#https://gitee.com/${REPO_GITEE}/#https://github.com/${REPO_GITHUB}/#")
_GH_SUM=$(printf '%s' "$ASSET_SUM" | sed "s#https://gitee.com/${REPO_GITEE}/#https://github.com/${REPO_GITHUB}/#")
fetch() { # fetch <gitee_url> <github_url> <out_file>
  case "$SOURCE" in
    gitee)  curl -fSL --retry 2 --connect-timeout 10 -o "$3" "$1" ;;
    github) curl -fSL --retry 2 --connect-timeout 10 -o "$3" "$2" ;;
    *)      curl -fSL --retry 2 --connect-timeout 8 -o "$3" "$1" 2>/dev/null \
       || curl -fSL --retry 2 --connect-timeout 10 -o "$3" "$2" ;;
  esac
}
fetch "$ASSET_TAR" "$_GH_TAR" "$TMP_TAR" || err "下载失败：$ASSET_TAR"
if curl -fsSL --connect-timeout 8 -o "${TMP_DIR}/sha256sums.txt" "$_GH_SUM" 2>/dev/null \
   || curl -fsSL --connect-timeout 8 -o "${TMP_DIR}/sha256sums.txt" "$ASSET_SUM" 2>/dev/null; then
  (cd "$TMP_DIR" && sha256sum -c sha256sums.txt --ignore-missing >/dev/null 2>&1) \
    || err "sha256 校验失败（下载不完整或源被篡改），已中止"
  log "sha256 校验通过"
else
  warn "未获取到 sha256sums.txt，跳过校验"
fi

# ---- 面板升级通道 ----
if [ "$MODE" = "panel" ] && [ "$UPGRADE" = "1" ]; then
  mkdir -p "$UPDATE_DIR"
  tar -xzf "$TMP_TAR" -C "$UPDATE_DIR" ypanel
  rm -rf "$TMP_DIR"
  log "应用更新（约 3 秒后自动替换并重启）…"
  setsid nohup sh -c "sleep 2; cp ${INSTALL_DIR}/ypanel ${INSTALL_DIR}/ypanel.bak; mv ${UPDATE_DIR}/ypanel ${INSTALL_DIR}/ypanel; chmod +x ${INSTALL_DIR}/ypanel; systemctl restart ${SERVICE_NAME}" >/tmp/ypanel-upgrade.log 2>&1 < /dev/null &
  log "升级已启动，稍后用 ypanel version 或访问面板确认新版本"
  exit 0
fi

# ---- 节点模式：纯 agent 安装（不装面板） ----
if [ "$MODE" = "node" ]; then
  log "安装节点 agent（${NODE_DIR}，不装面板）…"
  # 旧布局兼容：早期手工部署把二进制直接放在 $NODE_DIR（文件），迁移为目录布局
  if [ -f "$NODE_DIR" ]; then
    warn "检测到旧布局（$NODE_DIR 为文件），迁移为 ${NODE_DIR}.old-bin"
    mv -f "$NODE_DIR" "${NODE_DIR}.old-bin"
  fi
  mkdir -p "$NODE_DIR"
  tar -xzf "$TMP_TAR" -C "$TMP_DIR" ypagent
  install -m 0755 "${TMP_DIR}/ypagent" "${NODE_DIR}/ypagent.new"
  rm -rf "$TMP_DIR"
  if [ -f "${NODE_DIR}/ypagent" ]; then
    warn "已存在旧版 agent，更新中…"
    systemctl stop "$AGENT_SERVICE" 2>/dev/null || true
  fi
  mv -f "${NODE_DIR}/ypagent.new" "${NODE_DIR}/ypagent"

  log "配对到主面板 ${CORE_URL}（节点名：${NODE_NAME}）…"
  # -pair-only：配对成功落凭据（/etc/ypanel/agent.json）即退出；正式服务由 systemd 无参启动
  if ! "${NODE_DIR}/ypagent" -core "$CORE_URL" -code "$PAIR_CODE" -name "$NODE_NAME" -pair-only; then
    err "配对失败：请检查主面板地址/配对码（配对码一次性且 10 分钟有效，可重新生成）"
  fi

  cat > /etc/systemd/system/${AGENT_SERVICE}.service <<EOF
[Unit]
Description=YPanel Node Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${NODE_DIR}/ypagent
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now "$AGENT_SERVICE"
  sleep 2
  if systemctl is-active --quiet "$AGENT_SERVICE"; then
    log "节点 agent 已启动并接入主面板"
  else
    warn "agent 未正常运行，请检查：journalctl -u ${AGENT_SERVICE} -n 20"
  fi
  log "=============================================="
  log " 节点 agent 安装完成（${NODE_NAME}）"
  log " 回到主面板「节点管理」即可看到本节点在线，"
  log " 后续 agent 升级可直接在面板上一键更新"
  log "=============================================="
  exit 0
fi

# ---- 主面板全新安装 ----
if [ -f "${INSTALL_DIR}/ypanel" ]; then
  warn "检测到已有安装（$INSTALL_DIR/ypanel），本次仅覆盖二进制（数据保留）"
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
fi
mkdir -p "$INSTALL_DIR" "$UPDATE_DIR" "$DATA_DIR"
tar -xzf "$TMP_TAR" -C "$INSTALL_DIR"
rm -rf "$TMP_DIR"
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
