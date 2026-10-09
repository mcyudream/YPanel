# PHP 运行时扩展矩阵（实测）

> 2026-10-10 实测于 142（Ubuntu 22.04，官方 `php:<ver>-fpm-alpine` 镜像）。runtime 构建时扩展 = 镜像内置 ∪ 应用 app.json 声明（经 install-ext.sh 安装，catalog 见 `core/internal/service/runtimedata/catalog/php_extensions.json`，共 25 项）。

## 一、镜像内置扩展（php -m 实测）

7.4/8.0–8.4 各版本内置集一致（8.1+ 多 Zend OPcache）：

Core、date、libxml、openssl、pcre、sqlite3、zlib、ctype、curl、dom、fileinfo、filter、ftp、hash、iconv、json、mbstring、SPL、PDO、pdo_sqlite、session、posix、Reflection、standard、SimpleXML、mysqlnd、Phar、xml、xmlreader、xmlwriter、tokenizer、sodium、random（8.1+ 另有 Zend OPcache；7.4/8.0 有 readline）

**注意**：`pdo` 内置但 **pdo_mysql / pdo_pgsql 驱动不内置**；gd、zip 也**不内置**——用 MySQL/Laravel 系应用必须声明安装。

## 二、catalog 可安装扩展（25 项）

opcache、bcmath、gd、imagick、intl、zip、mysqli、pdo_mysql、pdo_pgsql、redis、apcu、memcached、xdebug、swoole 等（全量见 catalog json）。安装方式：install-ext.sh（docker-php-ext-configure/install；gd 自动带 freetype/jpeg/webp）；后装已构建的运行时走 `InstallPHPExtension`（docker exec + commit）。

## 三、已知应用需求对照

| 应用 | PHP 版本 | 必需扩展（非内置部分加粗） |
|---|---|---|
| Blessing Skin 6.0.2 | ≥8.0.2 | ctype/json/mbstring/openssl/pdo/tokenizer/xml 内置；**gd、zip、pdo_mysql** 需装（app.json 已声明） |
| WordPress | 7.4+ | **zip**（含内置场景通常已装） |
| Adminer | 7.4+ | 无（仅内置） |

## 四、约束与注意

- 扩展编译分钟级：构建缓存按「版本+扩展集」落在镜像层（同版本不同扩展集会各自构建一个 runtime 镜像）。
- 复用判定：同节点同版本且现有运行时扩展 ⊇ 应用必需（`phpExtSuperset`）；不满足新建，不就地改共享运行时。
- 版本约束表达（`gd>=3` 之类）暂未实现，当前按名字匹配。
