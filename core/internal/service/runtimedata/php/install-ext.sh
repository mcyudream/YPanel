#!/bin/sh
# YPanel PHP 扩展安装器（官方 php:*-fpm-alpine 镜像）。
# 用法: install-ext "redis,gd,imagick"（逗号分隔；构建期与容器内热装共用）。
# 已收录清单见面板 php_extensions.json；未收录名称直接报错。
# 卸载 = 删除 /usr/local/etc/php/conf.d/docker-php-ext-<name>.ini 后重启。
set -eu

LIST="$(printf '%s' "$1" | tr ',' '\n' | tr -d ' \t\r' | grep -v '^$' || true)"
[ -n "$LIST" ] || { echo "install-ext: nothing to install"; exit 0; }

PHPVER="$(php -r 'echo PHP_VERSION_ID;')"

pecl_install() {
	_ext="$1"
	_in="${2:-}"
	if [ -n "$_in" ]; then
		printf '%b' "$_in" | pecl install "$_ext" >/tmp/yp-pecl.log 2>&1 || {
			tail -n 30 /tmp/yp-pecl.log >&2
			exit 1
		}
	else
		pecl install "$_ext" >/tmp/yp-pecl.log 2>&1 || {
			tail -n 30 /tmp/yp-pecl.log >&2
			exit 1
		}
	fi
	docker-php-ext-enable "$_ext"
}

for ext in $LIST; do
	echo "==> install-ext: $ext"
	case "$ext" in
	bcmath | calendar | exif | pcntl | shmop | sysvmsg | sysvsem | sysvshm | sockets | opcache | mysqli | pdo_mysql | ffi)
		docker-php-ext-install -j"$(nproc)" "$ext"
		;;
	gd)
		# 注意：运行库必须装在 virtual 之外 —— apk del <virtual> 会连同其独有依赖一起删除
		apk add --no-cache libpng libjpeg-turbo freetype libwebp
		apk add --no-cache --virtual .yp-build libpng-dev libjpeg-turbo-dev freetype-dev libwebp-dev
		docker-php-ext-configure gd --with-freetype --with-jpeg --with-webp
		docker-php-ext-install -j"$(nproc)" gd
		apk del .yp-build
		;;
	intl)
		apk add --no-cache icu-libs
		apk add --no-cache --virtual .yp-build icu-dev
		docker-php-ext-install -j"$(nproc)" intl
		apk del .yp-build
		;;
	zip)
		apk add --no-cache libzip
		apk add --no-cache --virtual .yp-build libzip-dev
		docker-php-ext-install -j"$(nproc)" zip
		apk del .yp-build
		;;
	soap | xsl)
		apk add --no-cache libxml2
		apk add --no-cache --virtual .yp-build libxml2-dev
		docker-php-ext-install -j"$(nproc)" "$ext"
		apk del .yp-build
		;;
	gettext)
		apk add --no-cache gettext
		apk add --no-cache --virtual .yp-build gettext-dev
		docker-php-ext-install -j"$(nproc)" gettext
		apk del .yp-build
		;;
	pgsql | pdo_pgsql)
		apk add --no-cache postgresql-libs
		apk add --no-cache --virtual .yp-build postgresql-dev
		docker-php-ext-install -j"$(nproc)" pgsql pdo_pgsql
		apk del .yp-build
		;;
	redis)
		apk add --no-cache --virtual .yp-build $PHPIZE_DEPS
		pecl_install redis
		apk del .yp-build
		;;
	imagick)
		apk add --no-cache imagemagick
		apk add --no-cache --virtual .yp-build imagemagick-dev $PHPIZE_DEPS
		pecl_install imagick
		apk del .yp-build
		;;
	memcached)
		apk add --no-cache libmemcached-libs zlib
		apk add --no-cache --virtual .yp-build libmemcached-dev zlib-dev $PHPIZE_DEPS
		pecl_install memcached '\n\n\n\n\n'
		apk del .yp-build
		;;
	xdebug)
		apk add --no-cache --virtual .yp-build $PHPIZE_DEPS
		if [ "$PHPVER" -lt 80000 ]; then
			pecl_install xdebug-3.1.6
		else
			pecl_install xdebug
		fi
		apk del .yp-build
		;;
	*)
		echo "install-ext: unsupported extension: $ext" >&2
		exit 1
		;;
	esac
	echo "install-ext: $ext done"
done
echo "install-ext: all done"
