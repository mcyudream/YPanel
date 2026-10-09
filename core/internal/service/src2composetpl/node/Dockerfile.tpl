# YPanel 生成：Node（依赖安装失败时回退 npm install；npmmirror 加速）
FROM node:{{.NODE_VERSION}}-alpine
WORKDIR /app
RUN npm config set registry https://registry.npmmirror.com
COPY . .
RUN if [ -f package-lock.json ]; then npm ci --omit=dev || npm install --omit=dev; else npm install --omit=dev; fi
ENV PORT={{.PORT}}
EXPOSE {{.PORT}}
CMD ["sh", "-c", "{{.START_CMD}}"]
