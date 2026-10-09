
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
