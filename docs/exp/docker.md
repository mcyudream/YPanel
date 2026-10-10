
### docker bind-mount 源缺失时自动创建「同名目录」：失败重装必撞类型冲突

- **现象**：1p 应用（mysql）首次安装失败（compose 里的 `./conf/my.cnf` 未随包上移）；修复整包拷贝后重装仍失败 `cp: cannot overwrite directory '././conf/my.cnf' with non-directory`。
- **根因**：compose up 时 bind-mount 源 `./conf/my.cnf` 不存在，dockerd 自动创建了一个**同名目录**（经典行为）残留在项目目录；重装时整包拷贝要落的 `conf/my.cnf` 是文件，撞上残留目录报类型冲突。
- **规避/解决**：重装部署前先清理项目目录旧内容（`find . -mindepth 1 -maxdepth 1 ! -name 'data' -exec rm -rf {} +`，1Panel/YP 应用规范数据卷固定在 `./data`，保留即不丢数据）。另外判别"部署没生效"看任务日志里的**步骤名**：旧二进制显示「定位 compose」，新版是「整理应用包」。
- **补充（2026-10-08，M33 P2）**：YP 源（yp-git）包内容物约定在 `package/` 子目录一层，`writeLocalPackage` 只对 compose 文件按 basename 重命名落项目根、附属文件曾按相对路径写进 `package/` 子层——compose 挂载 `./附属文件` 时源缺失，同样触发 dockerd 自动建同名目录（vector.toml 变空目录、容器循环 "Configuration error. error=Is a directory"）。已修：非 compose 附属文件剥离首层 `package/` 前缀落项目根。排障特征：项目目录出现「空的同名目录 + 文件在 package/ 里」即此症。
- **来源**：2026-10-07 商店安装 mysql 两次失败排查（core/internal/service/store.go deployCompose）

### EasyTier（1p 源）安装成功 ≠ 组网可用：模板默认 config.toml 不创建虚拟网卡

- **现象**：商店安装 easytier 2.6.4（1p 源）一次成功、容器稳定运行、监听器全开，但宿主机/容器内均无 TUN 网卡，`easytier-cli peer` 的 Local 行 ipv4 为空——组网实际没建立。`dhcp = true` 后单节点仍不建（DHCP 需组网内有对端参与分配）。
- **根因**：1p 模板默认 `data/config.toml` 为占位配置（`dhcp = false` 且无 `ipv4`、网络名 default、密钥空）。EasyTier 核心在没有"确定的虚拟 IP"时只跑 P2P 监听、不创建 TUN 设备，且**不报错**。真正组网须显式给 `ipv4 = "10.144.144.x"`（首个节点）或 `dhcp = true` 且已有对端。
- **规避/解决**：装完后编辑 `/opt/ypanel/compose/app-<名>/data/config.toml`（网络名/密钥/ipv4/peer），重启容器生效。验证手段：容器内 `ip link | grep tun`、`easytier-cli peer` 看 Local ipv4。另：`easytier/easytier` 镜像内含 `easytier-web`（`/usr/local/bin/`），要 Web 控制台可改 command 或加服务。注意配置文件经 bind-mount 只读挂进 `/root/config.toml`，宿主侧改才有效；sed 远程改 TOML 时引号易丢（值必须带引号，`ipv4 = 10.144.144.1` 是非法 TOML 会让容器重启循环）。
- **来源**：2026-10-08 EasyTier 集成可行性实测（测试机 app-easytier；YPanel 侧顺带发现 1p formFields `disabled` 标志解析丢失、.env 合并出现重复 CONTAINER_NAME/包内遗留行）

### docker apt 仓库按发行版代号组织（无 stable 套件路径）；官方源在无国际网络环境直接 SSL reset

- **现象**：M52 源连通性测试首版用 `<base>/<distro>/dists/stable/Release`，阿里云/清华/中科大全 404；官方源 `download.docker.com` 在 142 直接 `curl (35) SSL_connect reset`（无国际直连环境）。
- **根因**：docker 的 apt 仓库（官方与全部国内镜像站一致）`dists/` 下是**发行版代号**（jammy/bookworm/…），repo 行 `<base> <codename> stable` 里 codename 是 suite、stable 是组件名——不存在 `dists/stable/` 路径。连通性测试必须按目标机 `VERSION_CODENAME` 拼 URL；yum 系测 `<yumBase>/<主版本>/repodata/repomd.xml`（主版本显式取 VERSION_ID 首段，**勿用 $releasever**——Alma/Rocky 的 releasever 是 9.x 形式，仓库无该路径）。
- **经验**：① 官方源在国内/内网环境不可达是常态而非异常——源选择 UI 必须自带连通性测试，让用户装前发现；curl 层错误（非三位数字输出）应归一为 000 并把原始错误放 detail。② 测试写 daemon.json 类「整文件覆写」接口时先 GET 留底，冒烟会真实覆盖用户配置（本次 ustc 单值覆盖了 142 原 4 个加速器，当场恢复）。③ 142 宿主机 docker 是 **Ubuntu 打包版（docker.io 29.1.3）非 docker-ce**——`docker --version` 带 `-0ubuntu` 后缀即可辨别；面板功能不受影响，但说明「已装 docker（非 ce 渠道）」场景真实存在，compose-only/none 判定要覆盖它。
- **来源**：2026-10-09 M52 Docker 一键安装开发与 142 冒烟（core/internal/service/dockerinstall.go TestSource）

### 143 真机安装实测揪出的三个 apt 坑（.asc 扩展名分流 / apt 架构名 / 步骤顺序）

- **坑一：apt 2.4 对 signed-by 按扩展名分流**——`gpg --dearmor` 输出是**二进制** keyring，存成 `.asc` 扩展后 apt 2.4 按 armored 路径解析 → `NO_PUBKEY`（binary 内容 armored 解析不出 key）。同一文件改扩展名 `.gpg` 立刻通过。Docker 官方文档的 `.asc` 流程在部分 apt 版本上真实翻车。keyring 一律存 `.gpg`。
- **坑二：uname -m ≠ apt 架构名**——repo 行 `arch=x86_64` 让 apt 找不到任何 Packages 索引（报 `Unable to locate package docker-ce`，极具误导性），apt 要的是 `amd64`/`arm64`。写 deb 源行必须做映射（x86_64→amd64、aarch64→arm64），不能直接透传 uname -m。
- **坑三：换源重装时「写源」必须排在所有 apt-get update 之前**——旧源残留的脏索引（如镜像站同步瞬态留下的 mismatch 状态）会毒死任何先跑的 update/install（`File has unexpected size ... Mirror sync in progress?`），任务在依赖安装步骤就 fail-fast，永远走不到后面的写源步骤，形成「换源重试永远失败」死循环。修法：写源最前 + 依赖「缺失才装」（`command -v curl && command -v gpg || apt-get update && apt-get install ...`，避免无谓 update）+ update 步骤自动重试一次（镜像站同步瞬态是常态，阿里云 jammy/stable 曾持续数分钟 InRelease 与 Packages 大小不匹配）。
- **附带**：143（Ubuntu 22.04 干净机）实测官方源完整安装 32s、清华源 compose-only 补装 12s；Docker Hub registry 被墙但 apt 下载站可通的网络分区很常见——加速器配置与安装源是两件独立的事。
- **来源**：2026-10-09 M52 143 真机验收（core/internal/service/dockerinstall.go buildSteps 三轮返修）

### 复杂应用（需 prepare/初始化脚本）收录 yp 商店的「初始化容器」模式：包内静态文件吃不到安装参数

- **现象/约束**：yp 安装管线是「整包拷到项目根 + 写 .env + compose up」——`${VAR}` 渲染**只发生在 compose.yml**（经 .env），包内其他文件（如 Harbor 的 `harbor.yml`）是静态的，安装参数进不去；管线也没有 pre-install 脚本钩子。
- **规避/解决（Harbor v2.15 实战验证）**：加一个一次性 `prepare` 服务做全部初始化——`entrypoint: ["/bin/sh","-c"]` + `command` 里用 heredoc 把安装参数 echo 成 harbor.yml，再调官方 `goharbor/prepare` 镜像生成全部组件配置（`./common/config`）；其余服务全部 `depends_on: prepare: condition: service_completed_successfully`。要点：① compose 命令串里只准出现 `${安装参数}`，其余 `$` 会被 compose 插值——heredoc 里别写 shell 变量；② 官方 prepare 会**无条件往 `/compose_location/docker-compose.yml` 写它自己生成的编排**——挂个丢弃目录（`./prepare-out`）盖掉，别让它覆写面板这份；③ 数据卷相对路径要「prepare 容器内挂载路径 == 其他服务 bind 源」（Harbor：全部服务挂 `./data/harbor:/data`，harbor.yml `data_volume: /data`，secretkey 等文件由 prepare 先于依赖服务创建，避开 dockerd 缺源自动建目录坑）；④ 密码类参数在 heredoc 里是文本插值，字段 description 必须劝退 `$` 引号反引号；⑤ 干跑验收顺序：`compose config --services`（不依赖生成物）→ 真跑 prepare → 再 `compose config` 全量（env_file 生成后才过）→ `diff` 官方生成的编排查缺挂载。验证平台磁盘紧就只跑到 prepare + config，别真装 2GB 镜像。
- **来源**：2026-10-10 商店收录 Harbor v2.15.4（YPanel-AppStore apps/harbor；读 goharbor/harbor make/prepare 源码得出容器契约：/input 配置、/config 产物、/data 数据卷、/compose_location 编排产物）

### 商店收录排障三连：YAML 尾冒号 / ghcr semver tag 无 v / yp 源随机字段从未生效

- **坑一：`.env` 风格值转 YAML 映射值，尾冒号即语法爆炸**——上游 `.env.example` 的 `REDIS_KEY_PREFIX=elementskin:` 搬进 compose 的 `KEY: elementskin:` 直接 `mapping values are not allowed in this context`（值尾冒号被当成嵌套映射指示符）。加引号。面板安装管线的 override 生成会用 Go yaml 解析 compose，等于多了一道比 docker 更早的语法关（帮我们提前拦了）。
- **坑二：ghcr 镜像 tag 规则看上游 CI 的 metadata-action 配置，别按 git tag 想当然**——element-skin 发的是 `v4.0.1` tag 但镜像 tag 是 `4.0.1`（`type=semver,pattern={{version}}` 剥 v 前缀）；另有 `dev`/`main`/`latest`/short-sha。收录前先读 `.github/workflows` 的 build-push tags 段，装不上 `not found` 十有八九是这个。
- **坑三（存量 bug）：yp 格式 `env` 的 `random/randomLen/description` 从未进过 FormFields**——`ypEnvItem` 结构体只有 key/label/type/default/required/rule，转换时全丢：后端随机兜底从未生效（空参数装出来是空密码/nil），向导骰子按钮靠前端 `random !== false` 默认真约定掩盖。已修：结构体补字段透传 + `randomLen` 新能力（后端 `randomHexBytes` 折算，前端 `randomPassword(f.randomLen ?? 16)`）。同场加映：`fmt.Sprint(nil)` 产生 `"<nil>"`，sanitize 后成 `"nil"` 写进 .env——空 default 必须加 nil 守卫。
- **排障手法**：商店安装失败先看任务日志（`GET /api/v1/tasks/:id` 的 logText，步骤名定位阶段：override 生成挂=YAML 语法、compose up 挂=镜像/网络、容器重启循环=`docker logs`）；重装场景注意 `data/` 保留导致旧密码与新随机参数不匹配（清理 data 或沿用参数）。
- **来源**：2026-10-10 收录 element-skin v4.0.1 五轮安装排障（YPanel-AppStore apps/element-skin + core/internal/service/store.go）

### 商店 PHP 应用类型验收期：磁盘保护压制新容器 + nginx 静态 fastcgi_pass 域名缓存失效 IP

- **现象一**：商店 php 应用装出的 runtime 容器启动 ~15 秒被 SIGQUIT 停掉（DB 里 runtime 状态却还是 running，站点 502/创建失败）。**根因**：142 磁盘 4.2G 低于保护阈值 5G，磁盘保护触发态的「压制」会停掉任何新拉起的容器——php:8.x 镜像 STOPSIGNAL=SIGQUIT，所以日志表现为收到 SIGQUIT。**解决**：清掉已死测试镜像（php-b20 遗留/已卸载应用的 uptime-kuma、rustfs 等 ~1.7G）→ 空间回线 → 调 `POST /api/v1/diskguard/restore`（空 JSON 体）解除锁存（顺带把被压制的容器全部拉起）。判别口诀：容器「起来几秒就被停」+ runtime/site 记录状态正常 = 先看磁盘保护再查代码。
- **现象二**：runtime 容器被停后重启换了 IP，nginx 对 php 站点 502——静态 `fastcgi_pass <容器名>:9000` 的域名**只在配置加载时解析一次**，容器死亡期间 reload 会把失效 IP 缓存住。`nginx -s reload` 刷新解析即恢复（新 IP 立即生效）。长期解（P3）：variables + `resolver 127.0.0.11` 方案。
- **附带**：遗留测试 conf 引用已删容器（如 `b20site.conf` → `php-b20`）会让 nginx 容器崩溃循环，阻塞面板一切建站操作且报错详情为空（agent exec 只捕 stdout，nginx 错误在 stderr）——「nginx 配置校验失败: 」后面空串 = 先手工 `docker exec ypanel-nginx nginx -t` 看真错。
- **附带两条**：① 站点创建 API（api/sites.go Create）用的是**内联白名单匿名结构体**，新增 SiteCreateInput 字段时必须同步补进 handler 透传，否则外部调用永远丢字段（runDir/rewriteName 静默为空，DB 无值）；② 站点 php 模板的 `location / { try_files $uri $uri/ /index.php?$query_string; }` 是**默认输出**（与 laravel 模板同文）——Laravel 系应用真正缺的只是 root 子目录（RunDir=/public）；web.rewrite 可用模板名以 rewrite.go 为准（spa/laravel/wordpress/thinkphp/typecho/discuz），app.json 写未知名安装即报错。
- **来源**：2026-10-10 商店 PHP 应用类型 P1 验收（core/internal/service/store_php.go；142 磁盘保护触发期）

### 商店 PHP 站点 POST 全空：nginx fastcgi_params 不含 CONTENT_TYPE/CONTENT_LENGTH

- **现象**：php 站点（Blessing Skin 安装向导）表单提交后应用读到空参数（连接配置用回 .env 默认值报 socket 错误），GET 一切正常；fastcgi 探针证实 `$_SERVER['CONTENT_LENGTH']` 缺失、`$_POST` 为空。
- **根因**：站点模板 `include fastcgi_params`——nginx 的 `fastcgi_params` 文件**不含** `CONTENT_TYPE`/`CONTENT_LENGTH`（这两条在 `fastcgi.conf` 里，与 fastcgi_params 的历史差异），PHP 拿不到请求体长度就不解析 POST。
- **规避/解决**：php 站点模板显式补 `fastcgi_param CONTENT_TYPE $content_type;` + `fastcgi_param CONTENT_LENGTH $content_length;`（site.go confTemplate php 分支，已修）；存量站点 conf 需 sed 补行 + reload。判别手法：写一个 `var_export([$_SERVER['CONTENT_LENGTH'] ?? null, array_keys($_POST)])` 探针进站点根（注意 RunDir 子目录的站点要放子目录里）。
- **附带**：Laravel 错误页/翻译加载（spatie translation-loader 查 language_lines 表）会在**未配置数据库时**抛嵌套异常，把真实错误（如 TokenMismatch 419）伪装成 500 空响应——排查时先看 `storage/logs/laravel-*.log` 而非响应体。
- **来源**：2026-10-10 Blessing Skin 6.0.2 商店安装 Web 向导驱动（BS 安装向导表单为 Laravel 标准 POST，无 JS 依赖，Playwright 原生 fill+click 即可驱动；密码类字段每次填表必须显式赋值，页面残留值不可信）

### 外接数据库「同节点不搬 IP」三层策略落地：存量容器接网的三个非显然坑

- **背景**：`applyExternalDB` 连接地址策略升级为三层（同网络容器名直连 → 同节点不同网络 `host.docker.internal:host-gateway` 代指 → 跨节点才搬 IP）。面板自建数据库实例的 compose 原本**没有任何 networks 配置**，只在自己项目网络（`db-<名>_default`）里，导致「同一容器网络」前提对自建实例永远不成立，同节点外接也只能走宿主 IP——实例接入统一网络 `ypanel_default`（composeYAML 服务级 networks + 顶层 `external: true` 引用，参照 php 运行时模板写法）后自然命中直连分支。
- **坑一：`docker network inspect <net>` 只列运行中容器**——对 Exited 容器 `network connect` 是生效的（启动后自动挂上），但 inspect 的 `Containers` 列表里看不到，别误判失败；验证用 `docker inspect <容器> --format '{{json .NetworkSettings.Networks}}'` 看容器侧。
- **坑二：存量容器手动 connect 后必须同步补写实例的编排文件**——实例编排文件不声明 networks 时，下次 compose up 重建容器会按文件创建，手动接入的网络直接丢掉。实例是容器 + 编排文件成对改（模板渲染的标准结构，`restart:` 行后插服务级 networks、文件尾加顶层 external 段，改完 `docker compose config --quiet` 校验）。
- **坑三：agent 写的实例编排文件名是 `compose.yaml` 不是 `docker-compose.yml`**——`/opt/ypanel/compose/<项目>/compose.yaml`，按后者的路径找文件会扑空（商店应用的则是 `docker-compose.yml`，两套命名并存）。
- **附带**：安装向导「自定义 hosts」校验正则 `extraHostPattern` 只认 `域名:IP`，`host.docker.internal:host-gateway` 过不了——host-gateway 条目只能代码内注入（applyExternalDB 返回 extraHosts，合并进 override hosts 列表，去重）；php 应用跑共享运行时容器，host-gateway 注入对象是**运行时容器**而非应用（新建运行时模板已自带，复用的存量缺时 `EnsureHostGateway` 补写 compose 并 up -d 重建，站点闪断数秒）。
- **来源**：2026-10-10 外接 DB 连接地址三层策略（core/internal/service/store.go / database.go / runtime.go；142 真机存量 5 实例补接）

### 商店应用 compose 的 env_file 变量未注入容器（Config.Env 只有 PATH）：force-recreate 重建即愈

- **现象**：Harbor 重装后 core 反复 FATAL `failed to initialize cache: cache type  is not supported`（注意两个空格——type 为空串），jobservice/nginx 跟着崩溃循环；`docker inspect <core> .Config.Env` 只有 PATH 一项——compose `env_file`（prepare 生成的 ./common/config/core/env，含 _REDIS_URL_CORE）完全没注入。Harbor 2.15 core 的 cache type 取自 `_REDIS_URL_HARBOR`（缺省回退 `_REDIS_URL_CORE`）URL 的 scheme，env 缺失 → scheme 空串 → FATAL。库表 0 张（migrate 也没跑）。
- **根因**：首轮 `docker compose up` 创建容器时 env_file 内容未进入容器定义（prepare 产物与 compose 解析时点的竞态，具体机制未深究）；容器创建后 compose 不会因 env_file 变化自动重建。
- **规避/解决**：装完发现服务因"配置为空"类错误崩溃循环时，先 `docker inspect <c> --format '{{len .Config.Env}}'` 对比 env_file 期望项数；不一致直接 `cd <compose目录> && docker compose up -d --force-recreate` 按当前文件重建，一步恢复（142 harbor 重装实测：recreate 前 core env=1 项 PATH + 49 表 0 张，recreate 后 47 项 + 全家 healthy + 49 表齐）。
- **来源**：2026-10-10，142 扩盘后恢复容器排障（app-harbor-test 重装 2.15.4）。

### syslog logging driver 用服务名做 syslog-address 是自举死锁：该服务自己永远起不来

- **现象**：容器 `docker start` 报 `failed to initialize logging driver: dial tcp: lookup log on 127.0.0.53:53: server misbehaving`，且被它依赖的一串服务连锁 Created/崩溃循环。
- **根因**：老版 Harbor 包给全部服务（**包括 log 服务自己**）配了 `logging: driver: syslog, options: syslog-address: tcp://log:10514`——syslog driver 在容器启动时初始化并解析地址，log 服务自己还没运行、compose DNS 里没有 `log`，解析失败直接挡死 start（连不上的 connect refused 只告警不挡，DNS 解析失败才挡）；其他服务在 log 停机期重启也会撞上。1Panel 官方用 `127.0.0.1:1514`+端口映射（IP 字面量可解析、连接失败异步重试）就是绕这个。
- **规避/解决**：日志服务端容器自己**不要**配 syslog driver（用默认 json-file），只有采集客户端配；新版包已整体移除 syslog。历史残留容器 LogConfig 固化不可改，只能按新 compose `--force-recreate` 重建，或直接换新版包重装。
- **来源**：2026-10-10，142 磁盘保护停机→恢复时 app-harbor-test 老包 db/log 容器永远起不来（同日已用新版包重装消除）。
