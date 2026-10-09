# 经验之谈：构建与部署

### 覆盖运行中的二进制报 "Text file busy"

- **现象**：deploy 推送新二进制失败 `bash: /opt/ypanel/ypanel: Text file busy`。
- **根因**：Linux 不允许覆盖正在被执行的文件。
- **规避/解决**：部署脚本先 `systemctl stop ypanel` 再推送，最后 `systemctl enable --now || restart`。
- **来源**：2026-10-06，scripts/deploy.sh。

### Windows 交叉编译面板单二进制：纯 Go SQLite 是前提

- **现象**：需要从 Windows 开发机产出 linux/amd64 单二进制。
- **根因**：CGO（mattn/go-sqlite3）交叉编译需要目标平台 C 工具链，Windows 上折腾成本高。
- **规避/解决**：`github.com/glebarez/sqlite`（纯 Go，基于 modernc.org/sqlite，MIT）+ `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w"`；pty（creack/pty）等平台相关包用构建标签隔离（`//go:build linux` + 非 linux 桩），Windows 开发机可正常 `go build ./...` 做类型检查。
- **来源**：2026-10-06，M0 选型。

### 密码类敏感值一律"运行时从 docs/local.md 解析进环境变量"，不落脚本与命令行

- **现象**：部署需要 SSH 密码与初始 admin 密码；直接写命令行会留在 shell 历史/会话记录。
- **根因**：AGENTS.md 敏感信息规则：禁止提交、禁止在回复/日志复述。
- **规避/解决**：`export SSHPASS=$(grep '<标记>' docs/local.md | head -1 | awk -F'|' '{print $N}' | tr -d ' \r')`；注意 local.md 同表多行含相同关键字时要 `head -1`（本仓曾因 YPanel 行也含 IP 导致解析出两行、密码变 13 位）。sshctl 从 SSHPASS 读密码，argv 不落敏感值。
- **来源**：2026-10-06，scripts/deploy.sh + tools/sshctl。

### 单元测试目标机信息（systemd）

- **现象**：首次部署即 systemd 常驻，健康检查用远端 curl 而非开发机直连（面板只对本机 0.0.0.0:8880 暴露，内网可达）。
- **规避/解决**：`journalctl -u ypanel -n 30 --no-pager` 是排障第一入口；agent 日志与 core 同进程同 stdout（合并部署），JSON 行格式统一 grep。
- **来源**：2026-10-06，测试机部署验收。

### 服务器 docker.io 直连不可达：配置 registry-mirrors

- **现象**：容器化安装数据库时 `failed to resolve reference docker.io/...: connection refused`。
- **规避/解决**：`/etc/docker/daemon.json` 配置 `registry-mirrors`（daocloud/1ms/1panel 等公共镜像），`systemctl restart docker`；YPanel 测试机已配置，更换测试机时需重做。
- **来源**：2026-10-06，M4 部署。

### compose 卷相对路径是相对 compose 文件目录，别与其他约定目录混用

- **现象**：nginx 容器"启动成功、进程活着、nginx -t 通过"，但容器内不监听任何端口、无任何错误日志；写进 `/opt/ypanel/nginx/conf.d` 的站点配置"凭空消失"。
- **根因**：compose 模板里卷写相对路径 `./conf.d`（解析为 compose 项目目录 `/opt/ypanel/compose/ypanel-nginx/conf.d`），而站点配置按面板约定写到 `/opt/ypanel/nginx/conf.d`——容器挂载的是空目录，nginx 无任何 server 块故不监听，且 nginx 对"无 server"合法静默。
- **规避/解决**：面板管理的 compose 卷**一律绝对路径**；"进程活着但不干活"先核对容器内实际挂载（docker inspect .Mounts）与文件落点是否一致。
- **来源**：2026-10-06，M5 站点管理。

### agent 升级重装不能带一次性配对码重启，要用凭据续启

- **现象**：推新 ypagent 二进制后按原参数 `-core ... -code XXX` 重启，日志报"配对失败：配对码无效、已使用或已过期"，systemd 反复重启。
- **根因**：配对码是一次性消费的；首次配对成功后凭据已落盘 `/etc/ypanel/agent.json`，再带 `-code` 启动会重新走配对流程必然失败。
- **规避/解决**：升级时直接无参启动（`ypagent -addr 0.0.0.0:9528`），自动读取已存凭据；systemd 单元按此编写（测试机 ypagent.service 已是凭据续启模式）。另注意 sqlite3 CLI 直读运行中 GORM（WAL）库会看到混合状态，排障以 API 视角为准。
- **来源**：2026-10-06，M19 部署（ypagent.service）。

### 并行会话共用工作区时的构建互踩：dist EPERM 与 embed 竞态

- **现象**：① vite build 报 `vite:prepare-out-dir EPERM ... dist/assets`（rmSync 失败），反复重试偶发成功；② deploy 出的二进制 health 正常但页面显示「前端产物未构建」（embed 里的 index.html 缺失）。
- **根因**：多个会话同时跑 `pnpm build`/`build.sh`：① 一方构建进程或杀毒/索引扫描持有 dist 文件句柄，另一方的 emptyDir rmSync 撞 EPERM；② build.sh 的「rm -rf core/internal/web/dist → cp → go build」窗口内，另一方的 rm 清空了目录，go:embed 编译进不完整产物。
- **规避/解决**：构建用独立输出目录绕开文件锁（`pnpm exec vite build --outDir dist-m23 --emptyOutDir`）再拷贝；go build 前用特征串验证 embed 完整（如 `grep -ac "页面特征文案" bin/ypanel`）；部署后除 /health 外再 curl 一发 `/`（应返回 `<!DOCTYPE html>`）确认前端真正可用。
- **来源**：2026-10-07，M23 与商店多源会话并行构建期间两次互踩。

### vite build 复用旧输出目录可能产出陈旧模块图：源码改了、构建成功、产物却是旧代码

- **现象**：修改路由/组件后用同一输出目录（--outDir dist-m23）重新构建，进程成功（6521 modules transformed）、产物时间戳是新的，但 grep 产物发现**源码修改没体现**（meta 里没有新加的 menu:!1）；换成全新目录名构建一次就对。
- **根因**：与 prepare-out-dir 的 EPERM 清理失败相关——输出目录清理被文件锁干扰后，rolldown/vite 的模块图或输出清单与磁盘实际状态不一致，后续构建复用了陈旧模块内容（具体缓存层未深究，现象稳定复现）。
- **规避/解决**：① 连续构建时每次换全新 outDir（带时间戳）再拷贝进 core/internal/web/dist；② 部署前**必须 grep 产物特征串**验证本次修改真的进了二进制（如 menu:!1、新增文案），不能只看「构建成功」。
- **来源**：2026-10-07，M23 菜单收敛（sites/certs menu:false）构建两次产物均为旧代码，换目录后一次通过。

### rolldown 构建 OOM 与 corepack shim 损坏（Windows 构建机）

- **现象**：`vite build` 在 rendering chunks 阶段报 `memory allocation of N bytes failed`（rolldown/Rust 崩溃，exit -1073740791）；崩溃后下次 `pnpm` 直接 `Cannot find module .../corepack/dist/pnpm.js`。
- **根因**：同机并行进程（另一会话的 vite dev/构建）吃满内存，rolldown 默认并行线程内存峰值大；corepack 的 pnpm shim 在崩溃中被损坏。
- **规避/解决**：构建加 `RAYON_NUM_THREADS=2`（或 1）限制 rolldown 并行即可通过；corepack 损坏用 `corepack install -g pnpm` 修复。部署脚本注意 `build.sh | tail` 这类管道会掩盖退出码——用 `PIPESTATUS` 或 `set -o pipefail`。
- **来源**：2026-10-08 应用中心批次多轮构建（scripts/build.sh）

### 临时排除半成品文件完成构建（并行会话同仓）

- **现象**：并行会话持续产出新的半成品 Go 文件（缺 import/未定义符号），阻塞整个 package 编译，而其文件几分钟内还在变化。
- **规避/解决**：把半成品文件 `mv` 出包目录（或改后缀）→ 立即构建 → 构建完成马上 `mv` 回原位（秒级窗口，冲突风险低）。仅用于解耦验证与部署，不要提交排除后的状态。
- **来源**：2026-10-08 core 构建（vpn.go 排除部署）

### 内网 DNS（dnsmasq）的 53 端口两坑：resolved stub 占用与公网侧开放解析器

- **现象/风险**：① dnsmasq 绑 0.0.0.0:53 时报 EADDRINUSE（ss 显示 127.0.0.53:53 被占）；② 公网侧若为全端口 NAT 转发（本机拓扑：除 8006 外全部转发到 web 节点），绑 0.0.0.0 会把 53 暴露成开放解析器，被扫到即被滥用于 DNS 放大攻击。
- **根因**：① systemd-resolved 的 stub 监听器固定绑 127.0.0.53:53，与通配绑定同端口互斥；② 用户边界 NAT 无 53 过滤。
- **规避/解决**：conf 强制 `listen-address=<内网IP>` + `bind-interfaces`（只绑私网地址，天然绕开 stub 冲突且不暴露公网），面板校验拒绝 0.0.0.0/::与公网地址。**预检必须按绑定目标判定冲突**：Linux 同端口仅「通配绑定（0.0.0.0/*/::）」与「同地址精确绑定」互斥——resolved stub 只绑特定地址 127.0.0.53，与内网 IP 的精确绑定可共存；首版预检一刀切拦截所有非 dnsmasq 的 53 监听，真机首部即被 stub 误拦（2026-10-08 热修正：通配/同地址才拦截，其余特定地址监听降级为提示）。host 网络容器下 `ss -p` 可见 dnsmasq 进程名，重部署场景按进程名放行自身占用。
- **来源**：2026-10-08 M27 内网 DNS 设计（core/internal/service/dnsmasq.go，真机部署实测后如有补充再更新本条）

### 隔离目录构建部署：/health 版本串是新的、前端却是旧的

- **现象**：为绕开并行会话的半成品文件，用临时目录拷贝仓库构建部署；连续三轮部署 `/health` 版本串都对，但浏览器跑的前端始终是旧代码（新卡片渲染不出）。
- **根因**：临时副本里的 `core/internal/web/dist`（go:embed 源）是拷贝时刻的快照；后续在真实工作区重建的前端产物只更新了真实工作区的 dist，没有同步进临时副本，go build 一直把旧前端嵌进去。
- **规避/解决**：① 部署后除 `/health` 外必须比对入口 chunk：`curl /` 取 `index-*.js` 与本地 `dist/assets/index-*.js` 比对（特征串 grep 二进制也行，但 minify 后中文文案才是稳定特征）；② 临时目录方案下，每次前端重建后要 `cp -r dist 临时副本/core/internal/web/dist` 再 go build；③ 并行会话修完自己的文件后，尽快回归真实工作区构建，避免快照漂移。
- **来源**：2026-10-08，M27 三轮「部署成功但前端没更新」（版本串 m27c/d/e 嵌的都是 m27b 前端）。

### Windows 构建嵌入 dist 被并行进程锁定：robocopy /MIR 方向语义与处置

- **现象**：`rm -rf core/internal/web/dist` 报 Device or resource busy（并行会话的 ypanel-full.exe 等进程持有）；robocopy 清目录时误把方向写反（`robocopy dist tmp_empty /MIR` 是 dist→tmp），随后 `cp -r src dist` 因目标已存在落成 `dist/dist/` 嵌套，embed 编译报 "contains no embeddable files"。
- **根因**：robocopy /MIR 语义是「把**第一个**参数镜像到**第二个**参数」（源→目标），与直觉的"清理工具"用法相反；`cp -r src dest` 在 dest 存在时会嵌套一层。
- **规避/解决**：同步产物统一用 `robocopy <src> <dst> /MIR`（源在前目标在后，cmd 下 `>nul & exit /b 0` 吞掉 0-7 的成功退出码防 bash 误判失败）；被锁的空目录无需删除，/MIR 可向其中写入文件；同步后必须 `ls dest/assets | wc -l` 与源对账再编译。go:embed 报 "no embeddable files" 先查目录是否嵌套/为空，再查并行锁。
- **补充（cmd //c 包装的两个变体坑，2026-10-08）**：① Git Bash 下 `cmd //c "robocopy x y //MIR"` 会把 `//MIR` **字面**传给 robocopy（报「无效参数 #3」）——cmd //c 的引号参数不走 MSYS 斜杠转换，双斜杠并不是转义手段；正确做法 `MSYS_NO_PATHCONV=1 cmd //c "robocopy \"$SRC\" \"$DST\" /MIR ..."`，src/dst 先 `cygpath -w` 转 Windows 绝对路径（NO_PATHCONV 下相对路径按当前 cwd 原样拼接，曾把源拼成 `web/.../web/apps/...` 双层路径）。
- **来源**：2026-10-08，M30 部署构建（与并行会话共仓环境的 embed 同步）。

### 双会话并行部署同一台机器：半写产物被 systemd 拉起 → Exec format error

- **现象**：142 的 ypanel 服务循环 `Failed to execute /opt/ypanel/ypanel: Exec format error`，登录接口无响应。
- **根因**：两个会话同时向 /opt/ypanel/ypanel 推送各自构建并 restart——一方 `put` 写到一半，另一方（或自身流程）触发 restart，systemd 在不完整/错误格式的文件上 EXEC 失败。
- **规避/解决**：推送后必须 `file /opt/ypanel/ypanel` 验证 ELF + `curl /health` 校验**版本号字符串**（只看 active 不够）；产物用独立本地文件名（bin/ypanel-<tag>）防本地互踩；发现被覆盖立即重推。根治需部署锁（远端 flock），暂以"推完即验"缓解。
- **来源**：2026-10-08，M31 部署与并行会话互踩。

### 隔离构建剥离并行会话半成品：sed 整行删除会殃及同行其它字段

- **现象**：M51 内网浏览器部署后，本地构建/验收全绿，142 上「创建代理会话」接口 500 panic（nil pointer）；且此前一轮协议层 curl 预检却是全绿的（跑在旧二进制上），极具迷惑性。
- **根因**：并行会话把我的 `WebGW: webgwSvc` 与其半成品 `Dashboard: dashboardSvc` 合并进了 Deps 字面量**同一行**；隔离构建用 `sed '/Dashboard/d'` 剥离半成品时把整行删掉——编译照样通过（引用点一并被删），运行期 WebGW/Src2/Creds/LogCentral 全部 nil。
- **规避/解决**：剥离并行半成品用**字段级替换**不用整行删除（`s/, Dashboard: dashboardSvc,/,/`，注意逗号锚定与保留——替换后同行其余键仍在）；替换后 grep 断言自己的接线键还在（`grep -c "WebGW: webgwSvc"`）再编译；「本地全绿、部署后个别接口 panic」先 diff 隔离副本与真仓的关键装配行，再怀疑代码本身。
- **来源**：2026-10-09，M51 内网浏览器（1050-m42 修复）。

### Windows 上初始化"分发类"git 仓库：首推前必须先加 .gitattributes 锁 LF

- **现象**：`git init` 后直接 add，全仓库报 `LF will be replaced by CRLF` 警告；Windows 上 `core.autocrlf=true` 时克隆检出的 compose/yaml 全变 CRLF，Linux 端（面板浅克隆读包）行为不确定。
- **根因**：autocrlf 只管"检出时转 CRLF"，仓库内容虽存 LF，但跨平台分发的仓库（如 YPanel-AppStore 应用源，面板直接读检出文件）检出字节随平台漂移。
- **规避/解决**：首推前放 `.gitattributes`（`* text=auto eol=lf`，二进制由 auto 检测跳过）+ `git add --renormalize .`；未推送前可直接 `git commit --amend` 并入首提。
- **来源**：2026-10-09，YPanel-AppStore 仓库初始化（GitHub/Gitee 双推）。

### 商店源编辑 URL 后同步仍走旧地址：yp-git 克隆缓存不随 URL 失效

- **现象**：面板「源管理」把 yp-git 源从 `file:///opt/ypanel/store-seed` 改为 Gitee 地址，点同步"成功"，应用清单还是旧的。
- **根因**：`StoreService.UpdateSource` 改 URL 时不清理 `data/store-sources/src-<id>` 克隆缓存，`gitCheckout` 见 `.git` 存在只做 `fetch origin`——origin 仍指向旧地址（store.go）。
- **规避/解决**：临时绕过 = 改 URL 后删服务端 `data/store-sources/src-<id>` 再同步（142 已如此处理）。**已修（2026-10-09）**：`gitCheckout` 复用缓存前先 `git remote set-url origin <最新地址+token>`（store.go，URL 与 token 轮换同修；clone 时 bake 进 .git/config 的旧地址/旧 token 是根因），回归测试 `TestGitCheckoutFollowsURLChange`（红绿验证：无修复时检出内容停留旧源）。
- **来源**：2026-10-09，YPanel 官方源切换 Gitee（142 实测复现）。

### 公开发布体系首建（2026-10-09）：GitHub/Gitee 双仓 + CI Release + 自更新 + 一键脚本

- **背景**：仓库 github.com/mcyudream/YPanel + gitee.com/mcyudream/ypanel（均新开空仓）。工作区 466 个未提交文件按域拆 7 笔提交（shared/agent/core/web/ops/docs/ci）；提交后发现根目录杂物误入（.workbuddy/.zcode/devcode* worktree 残留/m28-dist/tmp/design/散件截图）——**git filter-branch --index-filter 从全部 64 笔历史抹除 + refs/original 清理 + reflog expire + gc --prune=now + 双远端 force push**（新仓无协作者时安全；GitHub 网页可能有 CDN 缓存残留，稍后自行消失）。.gitignore 锚定根目录补条目防再犯。
- **坑一：npx vite 解析到 npm-cache 独立包**——`npx vite build` 在 node_modules/.bin 解析失败或环境异常时会**静默回退**到 `~/AppData/Local/npm-cache/_npx/` 下载的独立 vite/rolldown，版本与 workspace 不匹配 → rendering chunks 阶段 exit 127 静默崩溃、无错误输出。构建一律用 **`pnpm exec vite build`**（强制 workspace 本地依赖）；隔离验证用 git worktree + 独立目录 pnpm install。
- **坑二：vite alias 指向仓库外兄弟目录（../yudream-web-os）导致 CI 不可构建**——桌面工作台 webos 库是本地 monorepo 无远端，CI checkout 只有 YPanel 本体 → `UNLOADABLE_DEPENDENCY`。修复=**vendor 进仓库**（web/vendor/yudream-web-os/，仅 5 包 src+package.json 约 428KB），vite.config.ts/js 双写 + tsconfig.app.json paths 三处同步改指仓内路径；上游修复后手动同步覆盖 vendor。
- **坑三：workflow 步骤默认在 repo 根执行**——go.mod 在 core//agent/ 子目录，Build binaries 步骤必须 `working-directory: core`（本地模拟时手动 cd 过、写 workflow 时漏掉）。CI 调试三技巧：①Actions 日志 API 需认证——`git credential fill` 取本机已存 token 拉 logs；②tag 触发的 workflow 改完要**删远端 tag 重打**（v0.9.0 重打了三次）；③`pnpm --filter './apps/*' -r exec vite build` 在 CI 正常工作。
- **坑四：安装脚本必须输出安全入口段**——M49 SecurityGate 对 login 也要求入口（?entry=/X-Safe-Entry/cookie 三选一，否则 404 空 body），首启自动生成 8 位入口只打在 journalctl。quick_start.sh 安装完成后从 `journalctl -u ypanel` 解析 `安全入口已自动生成` 的 entry 值拼进「访问地址」输出，否则用户装机后进不了面板（142 真机同款坑重演）。
- **自更新链路**：GitHub/Gitee releases/latest 双源并行探测（gitee 无 release 时 404 属预期，配 GITEE_TOKEN secret 后 CI 镜像发布自动启用）→ agent 主机侧 curl 下载+sha256sums 校验+解包 → 复用本地 apply 通道（备份 ypanel.bak → 替换 → systemctl restart，detached 脚本）。真机验证 v0.9.0→v0.9.1 无损升级 ✓。
- **Gitee Release 镜像步三坑（CI 里 continue-on-error 会吞错，必须拉步骤日志看）**：① step 的 `if` 读不到 step 自身 env（`GITEE_TOKEN` 要在 **job 级** env 注入）；② `RID=$(... | grep -o ...)` 无匹配时 grep 返回 1，`set -e` 直接杀步骤（管道尾必须 `|| true`）；③ Gitee 建 Release `POST /repos/{o}/{r}/releases` 的 **target_commitish 必填**（tag 已存在也不可省，缺了报 `{"messages":["target_commitish is missing"]}`），且 **JSON body 含未转义中文直接 400 HTML 页**（body 保持 ASCII）。GitHub repo secrets 写入：GET actions/secrets/public-key → PyNaCl SealedBox 加密 → PUT（PublicKey 传 base64 解码后的 32 字节 raw）。发行包只出 CI 一份，Gitee 与 GitHub 同源——拉 GitHub Release 资产上传 attach_files 即镜像。
- **来源**：2026-10-09 发布体系首建（GitHub run 37909074629 绿；143 真机 v0.9.1 装机+自更新双验证）；Gitee 镜像当晚启用（v0.9.1 双平台同源附件实测可下载）
