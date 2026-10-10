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
- **变体（Create 侧，2026-10-09 M54 真机验收揪出）**：`tx.Create(&struct)` 对**零值 bool + `default:x` 标签**的字段同样跳过插入、落成 DB 默认值——`Role{ScopeAllNodes:false}`（default:true）建出来恒为 true，前端"限定节点"形同虚设。规避：建后立刻 `tx.Model(&row).Update("列", 零值)` 显式回写（或 Create 用 map）。判别特征：API 提交 false/0 却入库 true/默认值，Update（map）路径正常、仅 Create 路径异常。
- **来源**：2026-10-06，M19 安全基线（IP 白名单/安全入口持久化）；2026-10-09 M54-RBAC CreateRole 变体。

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

### errs.Wrap 会把基础错误的通用文案拼到用户消息尾部

- **现象**：NAT 转发占用拦截返回 `映射端口...已被监听占用: xxx/tcp（sshd）: 请求参数错误`——业务消息后面拖着一句莫名其妙的"请求参数错误"。
- **根因**：`errs.Wrap(base, context)` 的语义是 `context + ": " + base.Message`，拿 `ErrBadRequest`（Message=请求参数错误）做基底时，用户可见消息永远多出通用后缀。此模式遍布既有代码（bind 失败等），单看一条不觉得，消息里再拼明细时就很扎眼。
- **规避/解决**：给用户看的参数类错误直接 `errs.New(errs.CodeBadRequest, "error.badRequest", 具体消息)` 构造（项目里已沉淀 `natBadReq` 范式）；`Wrap` 只用于"在通用错误上补上下文、通用文案本来就该出现"的场景。新功能评审时把"消息尾部是否拖后缀"列为检查点。
- **来源**：2026-10-07，M24 NAT 转发真机 E2E。

### 面板自建 iptables 链的"整链重建"必须覆盖零规则场景，渲染视图必须过滤停用态

- **现象**：NAT 转发两处真机才暴露的缺陷：① 停用规则在链里照常生效（禁用开关形同虚设）；② 把规则全部删光/停用后，内核链里残留旧规则不冲刷。
- **根因**：① 组装渲染视图时只按地址族过滤、漏了 enabled 过滤——重建脚本"flush 后只追加启用规则"的语义在组装层就破了；② apply 在规则数为 0 时提前 return——"删光最后一条"恰好落入该分支，flush 永远不会执行。
- **规避/解决**：渲染视图提取成纯函数（`natEnabledViews`，过滤族+enabled）并单测；零启用规则不走"建链+挂载"脚本，改走**容忍式冲刷**脚本（`command -v iptables || exit 0`、链存在才 `-F`，不建链不挂载不碰 sysctl）——同时天然覆盖"目标机没装 iptables 但也从来没有过规则"的场景。凡是"flush + 全量重放"型管理器，删空态必须显式设计，不能当优化提前返回。
- **来源**：2026-10-07，M24 NAT 转发（`core/internal/service/natforward.go`）。

### ss 判断端口占用：同端口 v4/v6 双套接字会重复计数

- **现象**：占用检测返回两条 22/tcp（进程同为 sshd），UI 重复展示。
- **根因**：`ss -H -lntp` 对 `0.0.0.0:22` 与 `[::]:22` 各输出一行（v4/v6 各一套接字），按行解析不去重就是两条。
- **规避/解决**：按"端口+协议"去重、保留首个非空进程信息即可（监听判定语义不受影响）；另 `Local Address:Port` 列取最后一个冒号之后可同时兼容 `0.0.0.0:22` 与 `[::]:22`。
- **来源**：2026-10-07，M24 NAT 转发占用检测（142 真机）。

### 1Panel 应用包 formFields 全量形态与 compose 错误提取（商店安装参数修复）

- **现象**：按 label/default/type 基础解析 1panel.json 后，安装向导所有字段都是文本框；密码用源里固定 default（全网一致弱密码）；compose up 失败只显示 "mysql Pulling" 这类首行进度，端口被占用等真实原因看不到。
- **根因**：① 1Panel formFields 字段远比基础形态丰富——`random:true`（密码/名称安装时随机生成，401 处）、`values`（select 选项，154 处）、`edit/disabled`（只读）、`type: service/apps`（关联已装服务的复合字段）、`rule: paramPort`（端口类，602 处，默认值直接是 3306 这类低位端口，极易撞宿主已有服务）；② `docker compose up` 失败输出首行是 Pulling 进度，真实错误（bind: address already in use 等）在尾部。
- **规避/解决**：字段解析补全 random/values/edit/description；前端按类型渲染（select→下拉、password+随机按钮、端口字段默认随机高位 32768-61000、service/apps 一期只读）；后端安装任务化三件套——重装先 `compose down` 同名项目释放端口 → `ss -tln` 端口占用预检 → 先 `compose pull`（1800s）再 `up -d`（600s），失败返回 `tailOutput(output, 1200)` 尾部而非首行。
- **来源**：2026-10-07 应用商店安装参数与任务中心改造（core/internal/service/store.go、task.go）

### agentclient 对 404 响应报 "cannot unmarshal number"：mux 404 body 以数字开头，误导成信封格式问题

- **现象**：前端报「agent 响应解析失败: json: cannot unmarshal number into Go value of type struct { Code int ... }」——看似 agent 返回了错误格式的信封，实际请求根本没到 handler。
- **根因**：两个因素叠加。① core 拼 agent URL 时把第一个 query 参数用 `&` 拼接（`q := "/agent/v1/processes"; q += "&sort=cpu"`），没有 `?`，`&sort=...` 成了**路径**的一部分，agent 的 Go 1.22 mux 匹配不到 → 404；② agentclient 的 `doResp` 不检查 HTTP 状态码直接把 body 当 `{code,message,data}` 信封解，而 Go mux 404 body 是 `404 page not found`——JSON 解析器读出第一个 token 是数字 `404`，与 struct 类型不符报 "cannot unmarshal **number**"，完全没提 404。
- **规避/解决**：① 拼 query 一律用 `url.Values` + `v.Encode()`（`?` 由 `Encode` 所在分支保证），禁止手写 `+= "&"`；② `doResp` 已加固：非 2xx 先拦截并报 `agent HTTP <状态码>: <body 前 256 字节>`（agent 业务错误是 200+信封，不受影响）。判别技巧：看到 "cannot unmarshal number into Go value of type struct{Code...}" 先 curl 目标 URL 看原始 body，多半是 404/502 文本而非 JSON。
- **来源**：2026-10-07，M25 进程代理 query 拼接失误 + agentclient 无状态码检查，线上进程页全挂定位。

### GORM `Select` 部分列 + 内存过滤：没查出的字段恒为零值，过滤条件静默吞掉全部行

- **现象**：应用商店分类聚合接口返回永远为空数组，但应用列表正常、卡片上分类标签也有值。
- **根因**：`Tags()` 为省流量写了 `db.Select("tags").Find(&rows)`，随后用 `enabled[r.SourceID]` 过滤启用源——`source_id` 不在 Select 里，查回的行该字段恒为 0，map 里没有 key 0，**所有行被 continue，且无任何报错**。
- **规避/解决**：`Select` 部分列时，必须把**后续内存过滤/分组用到的每一列**都列全（`Select("source_id", "tags")`）；或者改用 `Omit`（排除大字段）而不是 `Select`（白名单列），Omit 语义下漏列只会多查数据不会丢数据。此类 bug 无报错、列表页正常，只在某个聚合接口显形，排查时应先打印中间行数。
- **来源**：2026-10-07 部署实测分类栏为空（core/internal/service/store.go Tags()）

### GORM Save 空/nil 切片报 "empty slice found"：列表页空数据直接 500

- **现象**：证书库一条记录都没有时，打开证书列表必然返回「系统内部错误」；有证书时一切正常。日志 `api internal error err="empty slice found"`。
- **根因**：GORM v2 `db.Save(slice)` 在 slice 长度为 0（含 nil 切片）时直接返回 `ErrEmptySlice`，不执行 SQL。`Find` 空表不报错、返回空切片，紧随其后的 `Save` 回写探测结果就在空库上炸了。列表接口常见的「查出来 → 内存加工 → Save 回写」模式都会踩：功能在有数据的开发/验收期永远正常，数据被清空后才暴露。
- **规避/解决**：Save 切片前必须 `len(x) > 0` 判空。顺带范式：回写仅为刷新派生字段时，先浅拷贝原值、加工后逐字段比较（`*time.Time` 指针字段用 `.Equal` 按值比），只 Save 实际变化的行——既避开空切片，也把「每次开列表页全表写 SQLite」的写锁竞争降到仅变更行。
- **来源**：2026-10-07，B23 证书库空库 500（core/internal/service/cert.go List）。

### docker compose 日志"静默为空"：不传 --project-name 时项目名取编排文件目录名

- **现象**：B20 运行环境的容器日志接口恒返回空（HTTP 200、Content-Length: 0，无任何报错），但手工执行 `docker compose -f ... -p rt-java-b20j logs` 有输出。受影响面不止运行时：任何"编排目录名 ≠ compose 项目名"的项目（外部接入的 compose、B20 的 /opt/ypanel/runtime/<type>/<name>）在 agent compose Logs 上都是空。
- **根因**：agent compose 管理器的 Up/Down/ServiceAction 都显式传了 `--project-name`，唯独 **Logs 漏传**——compose CLI 在无 top-level `name:` 时以**编排文件所在目录的 basename** 作为项目名推断（/opt/ypanel/runtime/java/b20j → 项目名 "b20j"），按 label `com.docker.compose.project=b20j` 过滤容器自然一条都没有，且**不报错**。运行时目录命名（rt-<type>-<name> ≠ 目录名）正好踩中。
- **规避/解决**：Logs 参数组补 `--project-name`（agent/internal/compose/compose.go）；凡是"compose 命令拼参数"的封装，up/down/logs/service 动作的项目名传递必须成套对齐，新增动作时先核对 compose 的隐式项目名推断规则。排查手法：同一命令手工加 `-p` 与不加各跑一遍对比输出，即可二分定位到项目名错位。
- **来源**：2026-10-07，B20 Java/Go 运行时日志补验（agent/internal/compose/compose.go Logs）。

### langchaingo v0.1.15 流式回调三种 chunk 形态：content 是原始文本、tool_calls 是 JSON 数组、reasoning 只走专用回调

- **现象**：AI 对话 SSE 只有 scene/step/done 事件，正文 content 一条都没有；思考过程（reasoning）也永远是空。后端无任何报错。
- **根因**：langchaingo v0.1.15 openai 客户端（`internal/openaiclient/chat.go`）传给流式回调的 chunk **不是 SSE 原始 JSON**：① content delta 被剥成**原始文本字节**（`[]byte(choice.Delta.Content)`），拿它 `json.Unmarshal` 到 `{"choices":[...]}` 必然失败——回调里"解析失败静默 return nil"就把正文全丢了；② tool_calls delta 是**累积后 marshal 的 JSON 数组**（元素含 `function` 键）；③ `reasoning_content` **只**经 `llms.WithStreamingReasoningFunc(ctx, reasoningChunk, chunk)` 透出，普通 `WithStreamingFunc` 收到的 reasoning chunk 恒为空字节——普通回调永远拿不到思考过程。
- **规避/解决**：改用 `WithStreamingReasoningFunc` 一个回调通吃：`reasoning` 非空即透出思考事件；`chunk` 先按"JSON 数组且首元素含 function 键"过滤掉 tool_calls 参数片段，其余按**原始文本**直接透出正文。深度思考型模型（deepseek-flash/reasoner 等）必须走这条路，否则"思考过程展示"无从谈起。
- **来源**：2026-10-07，B18 AI 流式无正文根因（core/internal/service/aichat.go；langchaingo openaiclient/chat.go 源码 + go run 最小复现定位）。

### deepseek-flash 是思考型模型：先吐 reasoning_content 再吐 content，模型名以实测为准

- **现象**：`curl api.deepseek.com /chat/completions model=deepseek-flash` 返回 `content` 为空、`reasoning_content` 有值（`finish_reason=length` 时全部 token 被思考吃掉）。
- **根因**：deepseek-flash 并非无效模型名，而是思考型（对齐 deepseek-reasoner 行为）；工具循环每轮都会先流出一串思考 delta。给思考型模型配工具时，轮次耗时大头在思考阶段，前端"没有输出"的观感多半是思考在进行。
- **规避/解决**：模型能力判断以直连 API 实测为准（各配一个小 curl），不凭名字或旧文档下结论；UI 侧思考过程块（可折叠+字数）正好消化这段等待。
- **来源**：2026-10-07，B18 切换 DeepSeek 供应商（用户指定 deepseek-flash）。

### Alpine apk del virtual 包会把 -dev 引入的运行库一起删掉：扩展编译进镜像后加载报缺 .so

- **现象**：B20 PHP 运行时构建镜像时用 `apk add --virtual .build libpng-dev … → docker-php-ext-install gd → apk del .build`，构建成功，但运行时 PHP 加载 gd.so 报 `Error loading shared library libpng16.so.16: No such file or directory`（redis 这类无运行库依赖的 pecl 扩展不受影响）。
- **根因**：apk 的 del 会移除该 virtual 包**连同它作为依赖拉进来的独有包**——libpng-dev 依赖 libpng，del virtual 时 libpng（运行库本体）一并被删。这与 apt 的 autoremove 保守语义直觉相反，"dev 包装进 virtual、del 后只留运行库"的思路在 alpine 上不成立。
- **规避/解决**：**运行库在 virtual 之外显式安装**（`apk add --no-cache libpng libjpeg-turbo …` 持久保留），virtual 里只放 -dev/编译工具（PHPIZE_DEPS 等）再 del。同构坑：gettext 需要 `gettext` + virtual `gettext-dev`；pgsql 需要 `postgresql-libs` + virtual `postgresql-dev`；imagick 需要 `imagemagick` + `imagemagick-dev`。真机排查手法：容器内直接 `php -m` 看启动 Warning（错误出在 Startup 而非模块缺失清单）+ `ls /usr/local/etc/php/conf.d/` 对照 ini 存在与 .so 是否可加载。
- **来源**：2026-10-07，B20 运行环境 v2 真机首建（runtimedata/php/install-ext.sh）。

### 容器内 php-fpm slowlog 静默失效：ptrace 被 Docker 默认 seccomp 拦截，需 cap_add SYS_PTRACE

- **现象**：B20 三期验收慢日志时，`request_slowlog_timeout=5s` + `sleep(6)` 请求正常执行超过阈值，fpm-error.log 也打出 `WARNING: child ... executing too slow (5.6s), logging`，但 slow.log 始终 0 字节；紧接着一行 `ERROR: failed to ptrace(ATTACH) child 9: Operation not permitted (1)`。
- **根因**：fpm 慢日志抓堆栈依赖 `ptrace(ATTACH)`，Docker 默认 seccomp profile 与默认 capabilities 都不含 SYS_PTRACE，容器内 attach 子进程被内核拒绝——WARNING 照打、堆栈写不出来，表现为"慢日志静默为空"。
- **规避/解决**：运行 php-fpm 的 compose 服务加 `cap_add: [SYS_PTRACE]`（1Panel 同款做法）。排查手法：看到 "executing too slow ... logging" 却无堆栈时直接看紧邻的 ERROR 行，ptrace 权限问题一眼定位；凡面板模板依赖 ptrace 类系统能力（slowlog/strace/调试器 xdebug 断点附加）都要补该 capability。
- **来源**：2026-10-07，B20 三期慢日志真机验收（runtimedata/php/docker-compose.yml）。

### moby client v0.6 的 PortMap 键是 struct：PortBindings 遍历不能当 "80/tcp" 字符串切

- **现象**：从 `HostConfig.PortBindings`（`network.PortMap`）反提端口映射时，`string(portKey)` 编译报 `cannot convert portKey (variable of struct type network.Port) to type string`。
- **根因**：moby 拆分模块（api v1.56）里 `network.Port` 是 struct（`num uint16 + proto unique.Handle[IPProtocol]`），不再有 `"80/tcp"` 字符串形态；取值走方法：`Num() uint16`、`Port() string`（端口数字串）、`Proto() network.IPProtocol`（"tcp"/"udp"/"sctp"）。
- **规避/解决**：遍历写法 `for portKey, binds := range hc.PortBindings { proto := string(portKey.Proto()); cPort := portKey.Port() }`；正向构造仍用 `network.ParsePort("80/tcp")`（normalize 大小写、缺省 tcp）。`HostConfig.RestartPolicy.Name` 的空/`no` 判断用 `container.RestartPolicyDisabled` 常量，别裸比较字符串。
- **来源**：2026-10-07，容器编辑重建（dockerx/crecreate.go inspectToReq，编译期发现）。

### gin 路由「静态段 + 参数段混合」编译不报错：冲突在注册期 panic，必须启动验证

- **现象**：给 core 加 `POST /docker/containers/:id/recreate`、`:id/update` 时，同路由组已有 `POST /docker/containers/:id/:action` 与静态 `POST /docker/containers/prune`——`go build` 全绿，但 gin 的 httprouter 树若不兼容该混合形态会在**启动注册路由时直接 panic**，编译/静态检查完全无感。
- **根因**：gin 路由冲突是运行期（engine 注册函数）行为；本项目 gin 版本已支持同层「静态段优先于参数段」共存（`prune` 与 `:id` 先例可证），但静态与参数是否可混在同一子层级（`:id/:action` 与 `:id/recreate`）依赖版本实现，不能靠编译确认。
- **规避/解决**：改完 core 路由必须本地起一次 core（`timeout 8 go run ./cmd/ypanel`，看到「YPanel 启动完成」监听日志即无冲突），顺带确认没有污染 data/（起来会建库写随机密码，测完 `rm -rf core/data`）。新增静态子路由前先 grep 同前缀有没有 `:param` 段路由。
- **来源**：2026-10-07，容器 recreate/update 路由接入（core/internal/router/router.go）。

### core 透传 agent 的 POST 动作路由不能用 GET 透传：Go 1.22 mux 对方法不匹配返回 405

- **现象**：面板「清理悬空镜像/卷」报「agent HTTP 405：节点 agent 不可达」，而容器清理正常；直接 curl 复现 405，agent 进程正常。
- **根因**：`ImagesPrune/VolumesPrune` 的 core 透传走 `Passthrough`（固定 GET），agent 路由是 `POST .../prune`；Go 1.22 ServeMux 对「路径匹配、方法不匹配」返回 405（不是 404），报错文案里的「不可达」是 core 包装误导。containers/prune 用 `DoJSON(..., "POST", ...)` 所以正常。
- **规避/解决**：core 透传 agent 动作路由时**方法必须与 agent 注册一致**；新增 `PassthroughPost` 透传 POST 并解包 agent 信封（业务码非 0 转 core 业务错误，成功统一返回 `{output}` 与同类接口形状一致）。审计要点：grep 所有 `Passthrough(` 调用，核对 agent 侧对应路由的注册方法。
- **来源**：2026-10-08，M23 镜像/卷清理 405（containers 正常、images/volumes 405 的差异定位）。

### net.ParseIP 拒绝 IPv4 前导零：192.168.100.002 直接解析失败

- **现象**：用 "192.168.100.002" 写 IP 归一化测试，net.ParseIP 返回 nil，"合法"地址被拒。
- **根因**：Go 1.17 起 net.ParseIP 拒绝 IPv4 前导零（消除八进制解释歧义的安全加固，CVE-2021-29923 家族），不再宽容解析。
- **规避/解决**：测试与文档示例用规范写法；用户输入 IP 校验失败时错误信息给规范示例。验证归一化逻辑用 IPv6 大小写（FD00::2 → fd00::2）或 v6 压缩形式，不要用前导零 v4。
- **来源**：2026-10-08 M27 DNS 记录校验单测（core/internal/service/dnsmasq_test.go）

### docker compose build 在面板进程环境下进度输出丢尾部：任务日志只见前几行进度

- **现象**：经 agent（Go `exec.CommandContext` 或 execx 通道）执行 `docker compose build`，失败时输出只有前几行 plain 进度（恒停在 `#3/#5 load metadata` 一带、甚至半行），看不到真实编译错误；但 `BUILD_EXIT=$?` 拿到的是真实退出码 1，dockerd 日志里 solve 完整跑完并正确报错（`process ... exit code: 1`）。同一命令在 ssh shell、`env -i` 最小环境、`sh -c` + 文件重定向三种方式下输出全部完整。
- **根因**（未完全定论）：该 compose 版本的 plain 进度 writer 在面板进程这一父环境下的缓冲于进程退出时被丢弃/提前停写（daemon 侧完整，客户端丢尾）。管道回传与文件重定向在手动 shell 下均完整，唯独经 agent 进程链路必现。
- **规避/解决**：**构建输出一律先落文件再回读**：`docker compose ... build > build.log 2>&1; ec=$?; echo BUILD_EXIT=$ec; tail -c 1600 build.log`。三个要点：① 真实错误在 stderr，必须 `2>&1`；② 管道后 `$?` 是 tail 的，退出码要先用 `echo BUILD_EXIT=$ec` 标记再由调用方解析；③ 失败时**保留 build.log** 并在错误信息里给出路径（完整错误在文件里），成功才清理。排查此类"日志戛然而止"先对照 dockerd 日志确认 solve 是否完整，再决定是链路问题还是构建问题。
- **来源**：2026-10-08，M26 P2 源码构建真机验收（core/internal/service/src2compose.go，四轮真机复现 + 三组对照实验）。

### mongo-driver UnmarshalExtJSON 到 any：$oid/$date 落成 primitive.D，不还原原生类型

- **现象**：Mongo 文档编辑保存后 `_id` 变成嵌套文档 `{oid: "..."}`（原 ObjectId 丢失）；按 `{"$oid":...}` 定位文档永远"文档不存在"。
- **根因**：`bson.UnmarshalExtJSON(data, false, &anyVar)` 对 `{"$oid":"..."}` 解出的是 `primitive.D{{"$oid","..."}}`（实测本地最小复现），**不会**转成 `primitive.ObjectID`；拿它做 `_id` filter 或直接写库都语义变形。文档里的日期字段同理（`$date` → primitive.D）。
- **规避/解决**：写递归归一化器——`bson.D` 单键 `$oid` → `primitive.ObjectIDFromHex`、单键 `$date` → `time.Time`（relaxed RFC3339 / canonical `$numberLong` 毫秒两形态），`bson.M`/`bson.A` 逐层递归；应用到 `_id` 解析、filter、文档插入/替换所有 ExtJSON 入口（core/internal/dbdriver/mongo_browser.go normalizeExt）。凡"ExtJSON 进、BSON 出"的边界都要过一遍。
- **来源**：2026-10-08，M30 DB Admin v2（Mongo 文档 CRUD 真机验收）。

### 用字符串 Replace 构造连接 DSN：用户名段被误伤，报 user=X database=基座库

- **现象**：PG 切库连接报 `failed to connect to user=pgdemo database=postgres`（用户名变成了目标库名、库名还是基座库），该库全部浏览接口 500。
- **根因**：`strings.Replace(dsn, "/postgres", "/"+db, 1)` 替换的是**第一处**子串——DSN `postgres://postgres:pwd@host:port/postgres` 中第一个 `/postgres` 出现在 **`//postgres:`（用户名）**，不在路径尾。URL 各段都可能包含目标词，Replace 不可控。
- **规避/解决**：连接参数拆字段存（host/port/user/pwd），按库**重新 Sprintf** 构造 DSN；绝不对整串 DSN 做子串替换。判别特征：错误信息里 user/database 与预期"错位互换"，八成是替换错段。
- **来源**：2026-10-08，M30 PG 多库连接池（pg_driver poolFor）。

### MySQL SHOW INDEX 列数随版本漂移：Scan 固定列数必炸，用 information_schema.statistics

- **现象**：索引列表接口在 MySQL 8 上报列数不匹配（驱动 Scan 要求 vars 数 = 列数）。
- **根因**：`SHOW INDEX FROM t` 的列集随版本增长（8.0 比旧版多 Visible、Expression 等 15 列），按记忆写死 Scan 列表跨版本必碎。
- **规避/解决**：索引元数据改查 `information_schema.statistics`（index_name/non_unique/seq_in_index/column_name 四列版本稳定），按 index_name 聚合、seq_in_index 保序。凡 SHOW 语句的输出列都要警惕版本漂移，优先 information_schema/performance_schema 等稳定视图。
- **来源**：2026-10-08，M30 MySQL 索引管理。

### Windows shell curl 发中文 JSON：GBK 字节被 encoding/json 逐字节替换成 U+FFFD，首个「GBK 对恰好合法 UTF-8」幸存成怪符号

- **现象**：计划任务列表任务名显示 `ʱ�������`（形似"h/y"的怪符号 + 一串替换符），疑云先指向前端字体/截断/`gorm size:64`。库里 `hex(name)` = `CAB1EFBFBD…`——首两字节是原始 GBK，其后全是 `EFBFBD`（U+FFFD 的 UTF-8 编码）。
- **根因**：Windows（CP936）shell 里 `curl -d` 带中文，JSON body 按 GBK 编码发出；Go `encoding/json` 解析字符串时对非法 UTF-8 **逐字节静默替换为 U+FFFD**（不报错不拒收），坏字节入库前已不可逆。首字符两字节若恰好构成合法 UTF-8 二字节序列则幸存成 IPA 怪符号——GBK"时"=`CA B1`→U+02B1 `ʱ`（像 h）、"失"=`CA A7`→U+02A7 `ʧ`（像 y）。SQLite 的 `varchar(N)` 不限长，与 GORM `size:64` 无关。
- **规避/解决**：从 Windows shell curl 带**非 ASCII** JSON，一律先写 UTF-8 文件再 `curl -d @file`（或 `--data-binary`）；排查看库用 `hex(col)`，见到 `EFBFBD` 连串即「入库前已被 JSON 解码污染」，显示链路无罪、原名不可恢复。GUI 路径浏览器恒发 UTF-8 不受影响；防御性校验只能在 bind 后拒绝含 U+FFFD 的输入。
- **来源**：2026-10-08，计划任务列表乱码排查（M3 种子任务，142 真机库 hex 定位）。

### systemd 启动的 agent 无 HOME，bash 不会从 passwd 补齐：Web 终端用户 shell 配置整体失效

- **现象**：Web 终端无颜色（ls 目录不高亮、提示符素色）、无用户别名/补全，看似「前端丢 ANSI」；WS 链路与 xterm 渲染均无损直传，排查极易跑偏到前端。
- **根因**：agent 由 systemd 启动，进程环境无 HOME；pty 里 spawn 的 bash **不会**自行从 passwd 补齐 HOME（实测，仅登录场景处理），rcfile 里 `[ -f "$HOME/.bashrc" ]` 判空失败，用户 shell 配置（别名/dircolors/彩色提示符/命令历史/~ 展开）整体没加载。代码注释「bash 会自行从 passwd 补齐」是错误假设。
- **规避/解决**：spawn shell 前显式补 HOME（`os/user.Current().HomeDir`，已存在且非空则不动）；排查「终端表现与 SSH 登录不一致」类问题，先在目标机上用与 agent 完全相同的 spawn 方式（无 HOME + pty + --rcfile）做对照复现，一个 `echo HOME=[$HOME]` 即可定位。另：Debian 老式 root bashrc 的彩色提示符需 `force_color_prompt=yes`（source 之前置位）才开启，发行版默认 root 配置的颜色块可能整段注释，需要在注入 rcfile 里做 alias/dircolors 兜底（已有则不覆盖）。
- **来源**：2026-10-08，终端颜色排查（term_linux.go HOME 修复 + rcfile 颜色兜底，已部署 142 验证）。

### Go 方法不允许类型参数：泛型助手只能定义成包级函数

- **现象**：`func (s *AIService) agentGetJSON[T any](...)` 写成方法后编译报 `syntax error: method must have no type parameters`，报错行即方法签名行，易误判为泛型语法写错。
- **根因**：Go 规范禁止方法（有 receiver）声明类型参数；泛型只能挂在包级函数或类型上。
- **规避/解决**：泛型助手写成包级函数、receiver 作首参（`func agentGetJSON[T any](s *AIService, ctx, path)`），调用点 `agentGetJSON[T](s, ctx, ...)`。批量改造用正则 `s\.agentGetJSON\[...\]\(` → `agentGetJSON[...](s, `——类型参数含嵌套 `[]`（如 `[[]dto.Item, map[string]string]`）时 `[^\]]+` 会截断匹配，须用非贪婪 `\[.*?\]`。
- **来源**：2026-10-08，M31 AI 工具注册表泛型助手（aibase.go，编译期发现后机械改造）。

### 类型 switch 的 case 定义类型不匹配底层类型：json 产物静默落 default

- **现象**：Compass 导出的 JSON（`_id: {"$oid": ...}`）导入 Mongo 报 `_id fields may not contain '$'-prefixed fields`——normalizeExt 的修复（单键 $oid 还原）"明明写了"却不生效。
- **根因**：`bson.M` 是**定义类型**（`type M map[string]any`），类型 switch 的 `case bson.M:` 只匹配动态类型**恰为 bson.M** 的值；`json.Unmarshal` 产出的动态类型是 `map[string]interface{}`（底层类型相同但类型身份不同），不命中该 case，静默落 `default` 原样返回。修复代码写在了 bson.M 分支里所以从未执行。
- **规避/解决**：同一处理逻辑必须同时挂 `case bson.M:` 与 `case map[string]any:`（`case bson.A:`/`case []any:` 同理），抽公共函数；重构时优先用**显式函数 + 底层类型**而非定义类型 case。排查特征：加了 case 却"完全不生效"且无报错，先怀疑类型身份不匹配而非逻辑错。
- **来源**：2026-10-08，M31 Compass JSON 导入（本地最小复现 + 两轮真机才定位）。

### MSYS_NO_PATHCONV=1 下 POSIX 目标路径原样传给原生程序：robocopy 写去了 D:	mp

- **现象**：worktree 镜像后编译报旧代码错误——robocopy"成功"但目标目录内容没变。
- **根因**：`MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*"` 阻止了 `/MIR` 被转成盘符路径（目的），但也让目标路径 `/tmp/ypanel-wt/core` **原样**传给 robocopy（Windows 程序）→ 按"当前盘符根"解析成 `D:\tmp\ypanel-wt\core`，镜像写去了错误位置，真目标纹丝不动。
- **规避/解决**：禁路径转换的环境只该作用于**选项类短参数**；目标/源真实路径一律用 `cygpath -w` 转换后的 Windows 形式，或 worktree 直接放同盘真实路径（如 `D:\dev\code\ypanel-wt-m31`）。镜像后用"源目标各数一遍文件数/特征文件"对账，别信 robocopy 静默退出。
- **来源**：2026-10-08，M31 worktree 隔离构建（两轮编译旧代码的假象）。

### 恢复/重启自身服务的脚本会被 cgroup 连带杀死：用 systemd-run 瞬态单元脱离执行

- **现象**：面板「快照恢复」功能经 agent exec 执行恢复脚本（stop ypanel → 换 db → start），真机实测脚本总是死在 `systemctl stop ypanel` 之后——cp/start 未执行，面板停在停止态；nohup + `&` 后台化同样无效。
- **根因**：agent 嵌在 ypanel 服务内，脚本是其子进程；`systemctl stop ypanel` 触发 systemd 杀掉整个服务 cgroup，所有子进程（含 nohup/后台化的）一并收到终止信号。与客户端断连无关——是 cgroup 级死亡。
- **规避/解决**：用 `systemd-run --unit=<唯一名> --collect bash <script>` 把脚本放进独立瞬态单元执行，脱离 ypanel cgroup；脚本结束后单元自动回收。适用于一切「脚本会停掉自己所在服务」的场景（自升级、自恢复、自重装）。审计要点：systemd-run 返回后仅代表单元已排队，完成与否靠日志文件/健康轮询确认。
- **来源**：2026-10-09，M48 快照恢复真机三轮返工（阻塞式→nohup→systemd-run），marker 法（基线快照→制造标记→恢复→断言标记消失）验证回滚语义。

### sshd_config「首值生效」语义：Include 目录里字典序靠前的文件会静默压制后写的配置（开关永远不切换）

- **现象**：SSH 管理页认证开关点击后永远弹回旧状态；API 写入成功（sshd -t 过、reload 过）但 `sshd -T` 回读值不变。M49 当时只验收了读取与密钥列表，SetConfig 漏验，问题潜伏到真机反馈才暴露。
- **根因**：OpenSSH sshd_config 语义是 **first-value-wins**（首个读到的指令值生效，与直觉相反）。`Include /etc/ssh/sshd_config.d/*.conf` 按字典序展开，Ubuntu 云镜像预置 `50-cloud-init.conf`（`PasswordAuthentication yes`），YPanel 写的 `99-ypanel.conf` 排在后面，其 `passwordauthentication` 被**静默忽略**——写入、校验、reload 全部"成功"，生效值根本不是自己写的。
- **规避/解决**：程序化写 sshd 配置片段一律用 **`00-` 前缀**（如 `00-ypanel.conf`，umask 077 收权限），保证字典序最先；写入成功后幂等清理旧前缀文件。改主配置 `/etc/ssh/sshd_config` 无效（Include 在文件最前，永远先读）。同类模型可推广：任何「多来源合并配置」先查合并语义（首值/末值/显式优先级）再决定写入位置。验收此类功能必须「写入→sshd -T 回读」闭环，不能只看写入返回。
- **来源**：2026-10-09，M53 开关 bug 真机诊断（142 上 `50-cloud-init.conf` 压制 `99-ypanel.conf`，对比 sshd_config.d 目录即定位）。

### 布尔语义的配置写入必须双向实测：取反逻辑只在「开」方向验收永远发现不了

- **现象**：SSH「密钥认证」开关用户报「无法关闭」——点关闭写入后回弹开启；但「密码认证」开关一切正常。API 层 curl 实证：PUT `pubkeyAuth:false` 后 00-ypanel.conf 写入的是 `pubkeyauthentication yes`（false→yes、true→no，完全取反）。
- **根因**：M49 的 SetConfig 里 pubkeyAuth 分支 `v := "yes"; if *pubkeyAuth { v = "no" }` 写反（与 passwordAuth 分支方向相反）；当时 SetConfig 未做真机验证，且若只验「开」方向或只看 HTTP 200，取反 bug 永远不暴露——回读值与写入值总是「成功地相反」。
- **规避/解决**：布尔开关类写配置逻辑，验收必须**双向各一次**（true→回读 true、false→回读 false），并直接核对落盘文件内容而非只看 API 状态码；同类字段（passwordAuth/pubkeyAuth）实现自同一模板时逐字段对照方向。另：经 shell curl 传 JSON body 时内容含单引号会截断参数（表现为空响应），测试用远端临时文件 `-d @file`。
- **来源**：2026-10-09，M53 返修三「密钥认证无法关闭」（142 现场 00-ypanel.conf 内容为证，一次 curl 双向定位）。

### agent 信封错误码透传会冒充面板会话语义：鉴权类码（2001/2002）必须在 core 重映射

- **现象**：用户切到「离线」节点后全页报「无权执行该操作」并被踢回登录页，重新登录无效（节点选择持久化，回来还打同一节点）。切换前一切正常，权限也未变。
- **根因**：两层叠加。① agent 的 Bearer PSK 校验失败回 `writeErr(errs.ErrForbidden)` → HTTP 200 + `{code:2002}` 信封（agent/server.go auth 中间件）；core `agentclient.doResp` 对 agent 信封非零码**原样透传**（`&errs.Error{Code: env.Code}`），于是「agent 认证失败」冒充成面板权限拒绝；② 前端 M0 时代契约 `isSessionError = 2001 || 2002` 全局登出（当时 2002 只可能来自 admin 闸门），RBAC 后 2002 是常规业务拒绝。节点"离线"（online:false）只代表心跳过期，agent 进程可能还活着且在拒绝 token——这类节点最易踩。
- **规避/解决**：① core `doResp` 对 agent 信封中的鉴权码（2001/2002）重映射为 `CodeAgentUnreach` + 明确文案「节点 agent 认证失败（token 不匹配），请重新配对该节点」——agent 的会话语义永远不能穿透到面板用户面；② 前端 `isSessionError` 收窄为仅 2001，2002 只拦截提示不登出（exp/frontend.md「meta.auth 硬拦截」条的姊妹约定：会话语义只认 2001）。排查特征：所有走某节点的请求统一回 2002 且 `data:{}`（respErr 形态）→ 先怀疑 agent 信封透传，curl 对照 admin 同请求即可证实。
- **来源**：2026-10-09，M54 部署后用户切到 node-143（agent token 失配，心跳停摆但进程存活）触发登出风暴；两处修复 + 前端产物解剖验证（编译后 `te(e)=e===ee.Unauthorized`）。

### Go httputil.ReverseProxy 流式注入 HTML 的三个坑：压缩、Flush 冲刷、doctype 怪异模式

- **现象**：给 webgw 反代加「向 text/html 响应流式注入 <script>」时三处翻车：① 目标返回 gzip 压缩体，注入器拿到的是密文；② 注入锚点（<head>）经常找不到——第一块 body 可能只有 `<!do` 几个字节；③ 若把注入内容放在 `<!doctype html>` 之前，页面整页进入浏览器怪异模式（quirks mode），布局全面失真。
- **根因**：① 浏览器子请求带 `Accept-Encoding: gzip` 被反代原样透传，目标返回压缩流，注入必须在明文上做——Go transport 只对「请求未带该头」的请求自动加 gzip 并透明解压；② `FlushInterval: -1`（SSE 即时冲刷）让 ReverseProxy **每个 chunk 都调一次 ResponseWriter.Flush()**，包装 writer 若照单透传，第一块就把缓冲里未决的锚点冲走了；③ doctype 之前出现任何非空白内容都会使浏览器放弃 standards mode。
- **规避/解决**：① Director 里 `r.Header.Del("Accept-Encoding")`，transport 自动补 gzip 并透明解压，响应头里的 Content-Encoding 一并消失（代价：出口方向无压缩）；② html 未完成锚点判定前**抑制 Flush 透传**（注入完成/非 html 才放行），配 16KB 缓冲上限 + drain 兜底防小文档悬挂；③ 锚点查找降级链：<head…>（大小写不敏感、属性值引号状态机防 `>` 误判）→ doctype `>` 之后 → 文档最前；`Content-Length` 必须在注入模式下删除（body 变长，交给 chunked）。单测重点：跨 chunk 截断的 `<head`、`<header` 干扰项、恰好等于缓冲上限。
- **来源**：2026-10-10，M56 webgw proxy-lib 注入器（core/internal/gwserver/inject.go，两轮真机验收定位）。

### 反代内嵌端点（/s/{sid}/__yp_proxy.js）的拦截点必须放在路径变形解析之后

- **现象**：跨 host 页面（/s/{sid}/~h/<b64>/...）注入的脚本 src 带 host 段，请求落到网关后被**代理到目标服务**（拿到 404/unsupported path）而不是返回注入库；同 target 页面（裸 /s/{sid}/__yp_proxy.js）却正常。
- **根因**：脚本端点拦截写在 splitCrossHost 之前，只匹配裸 remainder；跨 host 形态解析后 remainder 已剥掉 ~h 段，但代码顺序上已被放过。
- **规避/解决**：内嵌端点拦截统一放在「前缀/路径段解析全部完成之后、代理发起之前」，用最终 remainder 匹配；注入的 src 与解析规则同步设计（注入器拿到的 scriptURL 就是最终解析形态）。新增路径形态（~h 段这类）时，先过一遍所有内嵌端点的前缀匹配顺序。
- **来源**：2026-10-10，M56 跨 host 注入验收（142 实测 src 带 ~h 段 404，拦截点后移一处修复）。
