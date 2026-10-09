# YPanel 生成：前端构建静态站（按 lockfile 选择包管理器安装并构建，nginx 托管产物；npmmirror 加速）
FROM node:{{.NODE_VERSION}}-alpine AS build
WORKDIR /app
RUN corepack enable && npm config set registry https://registry.npmmirror.com
COPY . .
RUN if [ -f pnpm-lock.yaml ]; then pnpm install --frozen-lockfile; elif [ -f yarn.lock ]; then yarn --frozen-lockfile; else npm install; fi
RUN {{.BUILD_CMD}}

FROM nginx:alpine
COPY --from=build /app/{{.DIST_DIR}}/ /usr/share/nginx/html/
