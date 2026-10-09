#!/bin/sh
# YPanel Java 运行时启动脚本
set -e
cd /app
exec sh -c "${START_CMD:-java -jar app.jar}"
