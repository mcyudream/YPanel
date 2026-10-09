#!/bin/sh
# YPanel Python 运行时启动脚本
set -e
cd /app
if [ "$RUN_INSTALL" = "1" ] && [ -f requirements.txt ]; then
  pip install --no-cache-dir -r requirements.txt
fi
exec sh -c "${START_CMD:-python main.py}"
