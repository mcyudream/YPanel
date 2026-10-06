# 经验之谈：Go 后端

### Docker 官方 SDK 已拆分为 moby/moby 多模块（2025），旧导入路径编译失败

- **现象**：`go build` 报 `github.com/docker/docker/api@vX: module declares its path as: github.com/moby/moby/api but was required as: github.com/docker/docker/api`。
- **根因**：Moby 2025 年把 API 类型与客户端拆成独立模块，`github.com/docker/docker/api` 变成重定向壳，拉到的新版本声明自己的模块路径是 `github.com/moby/moby/api`。
- **规避/解决**：改用新路径——类型 `github.com/moby/moby/api/types/container`、客户端 `github.com/moby/moby/client`（`client.ContainerListOptions` 等 Options 也迁到了 client 包）、`stdcopy` 在 `github.com/moby/moby/api/pkg/stdcopy`。注意新 API 差异：`ContainerList` 返回 `client.ContainerListResult{Items []container.Summary}`；`Summary.State` 是 `container.ContainerState` 类型需 `string()` 转换；`Ports` 元素是 `PortSummary`（`IP` 为 `netip.Addr`）；`Stop/Restart` 的 `Timeout` 是 `*int`（秒）而非 `*time.Duration`；`Ping` 需要 `client.PingOptions{}`。
- **来源**：2026-10-06，agent/internal/dockerx。

### 跨模块不能引用 internal 包（core 内嵌 agent 服务）

- **现象**：core 编译报 `use of internal package github.com/ypanel/agent/internal/server not allowed`。
- **根因**：Go 规定 `internal/` 只允许同一 module（更准确地说是 internal 祖先目录之下）导入；core 与 agent 是两个 module。
- **规避/解决**：需要被宿主嵌入的包放公共路径（`github.com/agent/server`），`internal/` 只放模块私有实现。设计时先想清楚哪些包是"对外 SDK 面"。
- **来源**：2026-10-06，agent/server。

### proxy.golang.org 不可达时用 GOPROXY 环境变量切镜像，不进仓库

- **现象**：`go mod tidy` 卡死/超时（dial tcp 2607:f8b0... 失败）。
- **根因**：网络环境无法访问官方模块代理。
- **规避/解决**：`export GOPROXY=https://goproxy.cn,direct`（会话级环境变量，符合 package-management.md "代理配置只写环境变量或 docs/local.md"）；前端 pnpm 无需额外配置。
- **来源**：2026-10-06，全仓首次依赖解析。

### MySQL DDL 不支持占位符：IDENTIFIED BY ? 报 1064

- **现象**：`CREATE USER ... IDENTIFIED BY ?` 参数化执行报 `Error 1064 syntax near '?'`。
- **根因**：MySQL 服务端预处理器不接受 DDL 密码位置的占位符（mysqldump 时代的限制仍在）。
- **规避/解决**：密码先过严格白名单正则（`^[A-Za-z0-9_-]{8,64}$`，天然排除引号/反斜杠/`$`），再以单引号字面量拼接进 DDL；数据值仍全部参数化。这套"白名单前置 + 字面量拼接"是标识符/密码类 DDL 的通用安全范式。
- **来源**：2026-10-06，M4 数据库用户管理。

### gin 通配符路由 `*file` 后不能再接子路径，且 Param 值带前导斜杠

- **现象**：注册 `POST /x/*file/restore` 启动 panic `catch-all routes are only allowed at the end`；`DELETE /x/*file` 匹配后 `Param("file")` 值为 `/name.sql`（带前导斜杠），`strings.Contains(file,"/")` 误判拒绝。
- **根因**：gin httprouter 限制 catch-all 必须在路径末尾；catch-all 捕获值包含前导分隔符。
- **规避/解决**：带通配的资源操作一律改 query 传参（`DELETE /x?file=`, `POST /x/restore?file=`）；或手动 TrimPrefix 后再校验。
- **来源**：2026-10-06，M4 备份接口。

### gopsutil Percent(0) 全局态并发调用会互相吞差值

- **现象**：告警服务每 30s 评估时 CPU 恒为 0，而面板 overview（3s 轮询）正常显示。
- **根因**：`cpu.Percent(0, false)` 非阻塞模式依赖包内全局"上次调用时刻"，采样器（2s 周期）与 API 并发调用会互相消费差值窗口——紧跟着调用的那一方拿到 ~0。
- **规避/解决**：单一采样器独占 Percent 调用，其余消费方读采样环形缓冲的最新值（agent sysinfo.Overview 的 CPU 字段改取 Latest()）。
- **来源**：2026-10-06，M7 告警服务。

### GORM Assign(struct)+FirstOrCreate 更新时零值字段被静默忽略

- **现象**：`SettingService.Set(key, "")` 返回成功，API 响应也是空值，但重启进程后旧值"回魂"——白名单关了又出现、安全入口关了又生效，行为像"设置丢失"。
- **根因**：`db.Where(...).Assign(model.Setting{Value: ""}).FirstOrCreate(...)` 在记录已存在时走 GORM Updates 语义，**struct 更新跳过零值字段**——空字符串、0、false 永远写不进库；内存缓存 `mem` 却同步更新了，形成"当次生效、重启回滚"的假象。
- **规避/解决**：upsert 手写三分支（查 → 不存在 Create / 存在且值变 `Model.Where.Update("列名", v)`）；或 Assign 传 map。凡是"改了没生效、重启又变回去"类问题，先怀疑零值更新被吞。
- **来源**：2026-10-06，M19 安全基线（IP 白名单/安全入口持久化）。

### time.Duration 转 int64 是纳秒：TOTP 步长除法恒为 0

- **现象**：手写 TOTP 全链路（生成/格式/时钟全对）但验证码永远"错误"，RFC 6238 标准向量测试失败。
- **根因**：`t.Unix() / int64(totpStep)` 中 `totpStep = 30 * time.Second`，转 int64 是 30_000_000_000（纳秒）而非 30——计数器恒为 0，所有时刻算出同一个码。
- **规避/解决**：除前先 `int64(totpStep/time.Second)`；凡 Duration 参与算术，先转目标单位再运算。另：手写密码学实现必须配标准测试向量（RFC 附录）作回归测试，"自测自通过"不算通过。
- **来源**：2026-10-06，M19 TOTP 2FA（core/internal/service/security.go + security_test.go）。

### Go 1.22 ServeMux 重复注册同 pattern 不报编译错，启动时 panic

- **现象**：agent 编译通过，启动即崩；日志无语法线索。
- **根因**：`mux.HandleFunc` 同一 pattern 注册两次（编辑时粘贴重复），ServeMux 在运行时注册阶段 panic。
- **规避/解决**：路由表集中定义（循环注册或常量表）可从根上避免重复；启动崩溃先 grep 路由注册段的重复行。
- **来源**：2026-10-06，M19 容器管理路由（agent/server/server.go）。

### apr1（Apache MD5 crypt）手写实现是坑：字节重排与清零语义错一个全盘错

- **现象**：手写 apr1 生成的哈希格式完全合法（$apr1$salt$22字符），但 nginx auth_basic 校验永远 401；与 openssl passwd -apr1 对照哈希不同，且两次"修复"（weird 循环字节、输出重排）均不对。
- **根因**：md5-crypt 有三处极易错的细节：① 输出编码前 16 字节要按 (12,6,0)(13,7,1)(14,8,2)(15,9,3)(5,10,4)(11) 重排（passlib _transpose_map）；② "weird 循环"里 final 变量指向的是**当时**的摘要（MD5(pw+salt+pw)），且部分实现 memset 时机不同导致首字节是 0 还是 db[0] 各版本歧义；③ 1000 轮 update 顺序按 i%2/3/7 组合。手写对照记忆写，三处全对才算对。
- **规避/解决**：**不要手写**。htpasswd 生成走容器内 `openssl passwd -apr1 -salt <salt> -stdin`（密码经 base64 中转防 shell 引号注入），与 nginx 天然兼容；openssl 已因自签证书存在于 nginx 容器，零新增依赖。注意 `openssl passwd` 不加 `-stdin` 时不读管道（打印 Password: 提示）。
- **来源**：2026-10-06，S20 Basic 认证（三轮错误实现后改为 openssl 通道，一次通过）。

### acme.sh 续签控制与证书探测的非显然语义（B23 证书库）

- **现象**：证书库要支持"按证书开关自动续签"，但 acme.sh 的续签模型是全局 cron 对**所有**已 install 的证书统一续期，没有 `--no-renew` 单证书开关；同时用 Go 解析 `openssl x509 -enddate` 输出按常规 `"Jan 2 15:04:05 2006 MST"` 布局解析失败。
- **根因**：① acme.sh 单证书退出续签的唯一途径是 `acme.sh --remove -d <domain> --ecc`——它只把证书从续签列表移除，**证书文件保留**，语义正好可复用为"关闭自动续签"；重新挂回靠 `--install-cert`（reloadcmd 钩子会一并恢复）。② openssl 的日期输出日/月之间是**两个空格**，Go 布局必须用 `Jan _2 15:04:05 2006 MST`（`_2` 带空格填充）。
- **规避/解决**：关闭续签 → `--remove`；开启续签 → 重放 `--install-cert ... --reloadcmd 'docker exec ypanel-nginx nginx -s reload'`。手动续签用 `--renew -d x --ecc --force`（不带 `--force` 60 天内会跳过）。证书过期时间探测统一 `openssl x509 -in crt -noout -enddate -issuer` 后按 `_2` 布局解析；上传证书的"证书-私钥配对校验"用 `openssl x509 -pubkey | openssl md5` 与 `openssl pkey -pubout | openssl md5` 比对（EC/RSA 通吃，x509 的 `-modulus` 只支持 RSA）。
- **来源**：2026-10-07 B23 证书库（core/internal/service/cert.go、cert_issuing.go）

### 1Panel v2 站点 conf 生成的可借鉴要点（2026-10 实测 v2.3.2）

- Web 服务器是 **OpenResty**（应用商店安装），站点 conf 由面板生成（`/www/sites/<域名>/{index,ssl,log}` 目录约定），面板"配置文件"页可直接编辑整段 server 块并"保存并重载"。
- 生成的 server 块固定带：敏感文件 `location ~ ^/(\.user.ini|\.htaccess|\.git|\.env|...)` return 404、`^~ /.well-known/acme-challenge` 放行、`ssl_protocols TLSv1.3 TLSv1.2`、`ssl_prefer_server_ciphers off`、`ssl_session_cache shared:SSL:10m`、`error_page 497 https://$host$request_uri`（HTTP 打到 443 时重定向）、启用 HSTS 时 `add_header Strict-Transport-Security "max-age=31536000" always`。YPanel B23 的 sslServer 段（site.go confTemplate）已对齐其中协议版本/套件/会话缓存/HSTS；497 与 acme-challenge 放行暂未加，后续补。
- **来源**：2026-10-07 对照用户在用 1Panel（yudream 实例）B23 网站增强批次

### GORM AutoMigrate 不迁移已有索引：唯一索引改复合唯一要手动先删

- **现象**：`appstore_apps.key` 原为单列唯一索引，模型改成 `(source_id,key)` 复合唯一后 AutoMigrate 直接跑，旧唯一索引仍在（新复合索引也不一定建出），多源同 key 应用插入必然冲突。
- **根因**：GORM AutoMigrate 只"增量加"不"改造"——已存在的同名/同列索引不会删除或重建，SQLite 也没有在线改索引的 DDL。
- **规避/解决**：AutoMigrate 之前 `gdb.Migrator().HasIndex(&Model{}, "旧索引名") → DropIndex`；索引名用 GORM 默认命名（`idx_<表名>_<列>`）才能被识别。历史行数据归属（source_id=0 → 内置源 ID）在 seed 步骤一并 UPDATE。
- **来源**：2026-10-07 应用商店多源改造（core/internal/db/db.go seedStore）

### gin 同一位置"静态段 + 参数段"跨方法可共存，勿用通配绕路

- **现象**：`POST /database/instances/external`（静态）与 `DELETE /database/instances/:id`（参数）担心 httprouter 冲突，曾考虑统一成 `/instances/takeover/:id` 之类的绕路路由。
- **根因**：gin 的路由树按 HTTP 方法各自建树；同方法内同位置静态优先于参数（POST 树下只有 `external`，DELETE 树下只有 `:id`），互不冲突。真正会炸的是**同方法**下同位置既有静态又有参数且路径前缀交错歧义的场景。
- **规避/解决**：直接用语义化静态段（`/external`），前端同步更新；不要为规避不存在的冲突牺牲 REST 语义。
- **来源**：2026-10-07 数据库/PHP 外部接管路由（core/internal/router/router.go）

### exec git 做"源同步"的三个安全点：ext:: 注入、token 内嵌、浅克隆

- **现象**：用 `git clone` 给"应用商店 git 源"做同步时，git 传输协议里 `ext::<command>` 形态会执行任意 shell 命令；私有仓库 token 若拼错位置会进日志；整克隆大仓库拖慢同步。
- **根因**：git URL 是"协议 DSL"不是纯地址；`ext::` 与 `-` 开头的参数会被 git 当作传输命令/选项。
- **规避/解决**：URL 白名单校验（http/https/ssh/git@/file:// 前缀 + 禁 `ext::` + 禁前导 `-`）；token 用 `https://ypanel:<token>@host/path` 内嵌（argv 传递不经 shell、不落日志）；`clone --depth 1` + 已有仓库 `fetch --depth 1 && reset --hard FETCH_HEAD`；`GIT_TERMINAL_PROMPT=0` 防交互挂死。
- **来源**：2026-10-07 应用商店 yp-git 源（core/internal/service/store.go gitCheckout）
