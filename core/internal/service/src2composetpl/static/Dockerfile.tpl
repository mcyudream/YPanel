# YPanel 生成：静态站点（nginx，容器固定 80 端口）
FROM nginx:alpine
COPY . /usr/share/nginx/html
