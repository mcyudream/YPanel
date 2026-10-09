# YPanel PHP 运行时（基于官方 php:*-fpm-alpine，supervisord 托管 php-fpm 与自定义进程）
ARG BASE_IMAGE=php:8.2-fpm-alpine
FROM ${BASE_IMAGE}

ARG PHP_EXTENSIONS=""
ARG APK_MIRROR=""

COPY install-ext.sh /usr/local/bin/install-ext

RUN set -eux; \
    chmod +x /usr/local/bin/install-ext; \
    apk add --no-cache supervisor; \
    if [ -n "$APK_MIRROR" ]; then \
        sed -i "s#https://dl-cdn.alpinelinux.org#$APK_MIRROR#g" /etc/apk/repositories; \
    fi; \
    if [ -n "$PHP_EXTENSIONS" ]; then \
        install-ext "$PHP_EXTENSIONS"; \
    fi
