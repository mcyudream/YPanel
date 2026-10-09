#!/bin/sh
# YPanel Node 运行时启动脚本
set -e
cd /app
if [ "$RUN_INSTALL" = "1" ]; then
  case "${PKG_MGR:-auto}" in
    npm)
      npm install
      ;;
    yarn)
      yarn install
      ;;
    pnpm)
      corepack enable >/dev/null 2>&1 || true
      COREPACK_ENABLE_DOWNLOAD_PROMPT=0 pnpm install
      ;;
    *)
      if [ -f pnpm-lock.yaml ]; then
        corepack enable >/dev/null 2>&1 || true
        COREPACK_ENABLE_DOWNLOAD_PROMPT=0 pnpm install
      elif [ -f yarn.lock ]; then
        yarn install
      else
        npm install
      fi
      ;;
  esac
fi
exec sh -c "${START_CMD:-npm start}"
