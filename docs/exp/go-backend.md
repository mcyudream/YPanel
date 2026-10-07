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

### langchaingo v0.1.15 流式回调三种 chunk 形态：content 是原始文本、tool_calls 是 JSON 数组、reasoning 只走专用回调

- **现象**：AI 对话 SSE 只有 scene/step/done 事件，正文 content 一条都没有；思考过程（reasoning）也永远是空。后端无任何报错。
- **根因**：langchaingo v0.1.15 openai 客户端（`internal/openaiclient/chat.go`）传给流式回调的 chunk **不是 SSE 原始 JSON**：① content delta 被剥成**原始文本字节**（`[]byte(choice.Delta.Content)`），拿它 `json.Unmarshal` 到 `{"choices":[...]}` 必然失败——回调里"解析失败静默 return nil"就把正文全丢了；② tool_calls delta 是**累积后 marshal 的 JSON 数组**（元素含 `function` 键）；③ `reasoning_content` **只**经 `llms.WithStreamingReasoningFunc(ctx, reasoningChunk, chunk)` 透出，普通 `WithStreamingFunc` 收到的 reasoning chunk 恒为空字节——普通回调永远拿不到思考过程。
- **规避/解决**：改用 `WithStreamingReasoningFunc` 一个回调通吃：`reasoning` 非空即透出思考事件；`chunk` 先按"JSON 数组且首元素含 function 键"过滤掉 tool_calls 参数片段，其余按**原始文本**直接透出正文。深度思考型模型（deepseek-flash/reasoner 等）必须走这条路，否则"思考过程展示"无从谈起。
- **来源**：2026-10-07，B18 AI 流式无正文根因（core/internal/service/aichat.go + tmp/airepro 最小复现）。

### deepseek-flash 是思考型模型：先吐 reasoning_content 再吐 content，模型名以实测为准

- **现象**：`curl api.deepseek.com /chat/completions model=deepseek-flash` 返回 `content` 为空、`reasoning_content` 有值（`finish_reason=length` 时全部 token 被思考吃掉）。
- **根因**：deepseek-flash 并非无效模型名，而是思考型（对齐 deepseek-reasoner 行为）；工具循环每轮都会先流出一串思考 delta。给思考型模型配工具时，轮次耗时大头在思考阶段，前端"没有输出"的观感多半是思考在进行。
- **规避/解决**：模型能力判断以直连 API 实测为准（各配一个小 curl），不凭名字或旧文档下结论；UI 侧思考过程块（可折叠+字数）正好消化这段等待。
- **来源**：2026-10-07，B18 切换 DeepSeek 供应商（用户指定 deepseek-flash）。
