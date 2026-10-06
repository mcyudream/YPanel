# 经验之谈：前端（fantastic-admin 基座）

### fa 基座对接真实后端的最小侵入点只有三处

- **现象**：fa 基座默认跑 fake mock（`{status:1,error,data}` 契约 + `Token` 请求头 + 后端路由模式），直接连真实后端全是 401/解析失败。
- **根因**：基座契约固定在 `src/api/index.ts` 拦截器与 `src/store/modules/app/account.ts` 的字段期望（`res.data.{token,account,avatar}`、`permissions`）。
- **规避/解决**：只改三个文件即可整体切换——① `src/api/index.ts` 重写拦截器为 YPanel 契约（`Authorization: Bearer` + `{code,message,data}`，code 2001/2002 全局登出）；② `src/api/modules/app.ts` 做字段映射（login 返回时把 `user.username` 映射回 `account`；permission 由 `/auth/me` 的 role 推导）；③ `src/settings.ts` 设 `app.routeBaseOn: 'frontend'`（路由走 `src/router/modules/*.ts` 静态聚合，绕开后端路由接口）。mock 用 `vite-plugin-fake-server`，删除 `src/api/fake_modules` 目录后必须同时从 `vite/plugins.ts` 移除该插件，否则 build 报 "fake_modules folder does not exist"。
- **来源**：2026-10-06，web 基座对接 YPanel core。

### vue-tsc 对整个模板报成片 TS6133"未使用"，先查模板结构是否未闭合

- **现象**：某 .vue 文件几十个 setup 绑定全被报"声明未使用"，但这些绑定明明在模板里用了；没有其他错误。
- **根因**：模板里有一个 `<div>` 忘了闭合（被父元素隐式吞掉），vue-tsc 的模板分析在该点中断，其后所有模板引用都不参与"已使用"判定，于是 script 侧全部误报 TS6133。
- **规避/解决**：看到单个文件成片 TS6133 且模板确实用到时，先检查模板标签闭合/嵌套，不要去删"未使用"的代码。修复闭合后误报全消。
- **来源**：2026-10-06，views/file_management/index.vue。

### MSYS2_ARG_CONV_EXCL 设为 "*" 会破坏 Git Bash 下的 pnpm/node

- **现象**：脚本里 `export MSYS2_ARG_CONV_EXCL="*"` 防止 POSIX 路径被改写，结果 `pnpm build` 报 `Cannot find module D:\d\dev\...\corepack\dist\pnpm.js`。
- **根因**：全局禁用路径转换后，pnpm/corepack shim 自身需要的 Windows 路径解析也被波及。
- **规避/解决**：环境变量只以"命令前缀"形式作用于需要的单个命令（如 sshctl 调用），不要 export 到整个脚本；被调起的原生 exe 需要的本地路径用 `cygpath -w` 手动转换。前者管"远端路径不被改写"，后者管"本地路径 Windows 可读"，两者都要。
- **来源**：2026-10-06，scripts/deploy.sh。

### xterm.js 排障：先看 DOM 的 .xterm-rows 再怀疑链路

- **现象**：Web 终端"白屏"，截图看不到任何输出，但命令实际已执行。
- **根因**：截图时机/渲染层滞后导致视觉误判；xterm 的真实内容始终在 `.xterm-rows` 的 DOM 文本里。
- **规避/解决**：用 `document.querySelectorAll('.xterm-rows')[i].textContent` 直接读终端内容判断数据链路是否通（本例一次定位为"链路通、纯渲染/时机问题"）；键盘输入要聚焦 `.xterm textarea`。
- **来源**：2026-10-06，views/terminal/index.vue 联调。

### lucide 图标名随版本演进，yd: 图标名未命中注册表会静默占位

- **现象**：`YdMorphIcon name="trash-2"` 渲染成"未知图标"占位圈。
- **根因**：新版本 lucide-static 已无 `trash-2`（只有 `trash`、`trash-off`）；图标数据是构建期快照（scripts/gen-icons.mjs 产物）。
- **规避/解决**：选择器数据与页面引用同源（都来自 data.json），页面写图标名前可在 `src/ui/icons/data.json` 里确认存在；未命中时 YdMorphIcon 显示占位并带 title，开发期肉眼可查。
- **来源**：2026-10-06，views/icons/index.vue。

### fa 的 simple-git-hooks 会装到仓库根导致 pre-commit 必然失败

- **现象**：`git commit` 报 `No package.json found in <仓库根>` 且自动触发 `pnpm install`。
- **根因**：fa 基座 `pnpm install` 时 postinstall 运行 simple-git-hooks，钩子装到 git 仓库根的 `.git/hooks/pre-commit`（根目录不是 node 工程，无 package.json，钩子必挂）。
- **规避/解决**：删除根 `.git/hooks/pre-commit`（及同类钩子）；后续在 web/ 内做提交前检查，或等 monorepo 顶层具备 node 工程后再统一配钩子。
- **来源**：2026-10-06，首次 git 提交。

### nginx reload 是异步的：保存配置后立即请求打在旧 worker 上

- **现象**：WAF/站点配置保存（writeConf→nginx -t→nginx -s reload）返回后立即 curl，行为是"上一步的配置"——disable 后仍 200、enable 后 404，看似代码 bug。
- **根因**：`nginx -s reload` 发 SIGHUP 后 master 异步拉起新 worker、旧 worker 处理完存量连接才退出；保存接口返回 ≠ 新配置已生效。
- **规避/解决**：功能代码无需改（最终一致）；自动化验收需在 reload 后 sleep 1-2s 再断言，或轮询直到行为变化。
- **来源**：2026-10-06，M11 WAF 与 M5 站点禁用/启用。

### fantastic-admin 菜单"拍平"重构会撞上双栏导航与异步组件的耦合，白屏且难定位

- **现象**：直接改 `store/modules/app/menu.ts` 的 `convertRouteToMenu/Recursive` 实现"单页模块上提为一级菜单"，构建通过、菜单数据（allMenus）完全正确，但页面白屏——RouterView 渲染出空注释，错误为 Vue 内部 `locateNonHydratedAsyncRoot: Cannot read properties of null (reading 'component')`。
- **根因**：三层耦合。① fa 是"主导航（分组图标）+ 次侧栏（子项）"双栏模式，`MainSidebar` 只渲染 `item.children.length !== 0` 的分组，直达叶子节点（children 被删）在主导航没有可渲染分支；② `filterAsyncMenus` 会 `delete` 空 children，而 `isPathInMenus`/`getExpandPaths` 对 undefined 未设防（此为确定的崩溃点，已单独修复）；③ 白屏主因在 Layout 异步组件（`() => import`）与 fa 守卫/keepAlive 组合的渲染期，菜单节点形态变化会传导到 RouterView 重渲染路径，具体触发链 dev sourcemap 只能定位到 Vue 内部。
- **规避/解决**：涉及 fa 菜单/布局层的结构性改动，必须：先在 dev 模式（连真实后端）复现与验证，production 无 console 线索时用 `app.config.errorHandler` + `window.onerror` 注入抓栈；改动前确认 `MainSidebar`/`filterAsyncMenus`/`isPathInMenus` 对新节点形态的兼容性。本次已回滚，重做方案需连 MainSidebar 渲染分支一起改。
- **来源**：2026-10-06，菜单重构回滚。

### rolldown-vite（Vite 8）下 monaco worker 无法用 `?worker` 深导入，且静态分析缺一分支就丢 chunk

- **现象**：`import w from 'monaco-editor/esm/vs/.../xx.worker.js?worker'`（带不带 `.js` 都一样）构建报 "Rolldown failed to resolve import"；改用 `new Worker(new URL('monaco-editor/...', import.meta.url))` 字符串变量三目分支后构建通过，但 dist 里**缺 editor.worker chunk**（明文/Go 等非语言文件运行时 worker 404）。
- **根因**：rolldown 的包 exports 解析对 `?worker` 查询后缀不剥离，子路径匹配失败；`new URL` 的字面量分支才被 vite 静态分析，变量分支走 glob 且与字面量混用时部分分支静默丢失。另外 worker 子构建连普通 `monaco-editor/esm/...` 深导入也解析失败（主构建能解析）。
- **规避/解决**：本地建 5 个 worker 包装文件（`src/utils/monaco-workers/*.js`，内容仅一行 `import '#monaco/worker-xxx'`；用 `.js` 避开 vue-tsc 对无类型 worker 入口的 TS2882）；vite.config 用 `createRequire` 定位 monaco 物理目录建 `#monaco/worker-*` alias 指向 worker 物理文件；loader 里 `new Worker(new URL('./monaco-workers/xx.js', import.meta.url), {type:'module'})` 每分支字面量；`worker: { format: 'es' }`。验收必须 `ls dist/assets | grep worker` 数够 5 个。
- **来源**：2026-10-07，M20 文件编辑器 monaco 接入。

### splitpanes v4（Vue3）API 速记与 allotment 不可用

- **现象**：做 IDE 式分栏时选型踩坑：`allotment` 是 **React** 库，Vue 项目不可用；splitpanes v4 的 payload 类型与旧版文章不一致。
- **根因**：splitpanes v4 `resized` 事件载荷是 `SplitpanesResizedPayload`（`{panes: PaneData[], ...}`），不是旧版的裸数组；Pane 组件 props 为 `size/minSize/maxSize`，Splitpanes 方向 prop `horizontal`（默认垂直排布=左右分栏）。
- **规避/解决**：`import { Splitpanes, Pane } from 'splitpanes'` + `import 'splitpanes/dist/splitpanes.css'`；类型从包主入口 `import type { SplitpanesResizedPayload }`；尺寸回写在 `@resized` 里读 `e.panes[i].size`，size prop 只作初始值避免拖拽时与响应式绑定打架。
- **来源**：2026-10-07，M20 文件编辑器分栏。

### pnpm install（web/）每次都会把 pre-commit 钩子重装回仓库根

- **补充**：此前已记录 simple-git-hooks 装到仓库根导致提交必挂；实测 web/ 下每次 `pnpm install` 都会重新写入 `.git/hooks/pre-commit`，提交前若报 "No package.json found" 先删根钩子。根治需 monorepo 顶层具备 node 工程或在 web/package.json 关闭 simple-git-hooks。
- **来源**：2026-10-07，M20 依赖安装后钩子复现。

### splitpanes 的 `<Pane>` 忘写 import：vue-tsc 不报错、构建无告警、运行期静默空白

- **现象**：M20 文件编辑器里 FileTreePanel/TerminalPanel 模板用了 `<Pane>` 但没 import（EditorGroup 有 import 正常），整片侧栏/终端消失；生产构建与 `vue-tsc -b` 全绿，无任何线索。
- **根因**：vue-tsc 默认非 strictTemplates，**不校验未解析的组件标签**；生产 Vue 对 "Failed to resolve component" 只发 console.warn，且组件渲染为空不抛错。
- **规避/解决**：第三方库的模板组件（splitpanes 的 Pane/Splitpanes 等）一律显式 import；排查这类"DOM 缺失但零报错"问题时，先 `grep 模板标签对应 import`，再用 console.warn 钩子（只钩 console.error 看不到）。
- **来源**：2026-10-07，M20 文件编辑器侧栏/终端消失排查。

### dev server 监视了构建产物 dist/：外部触碰即整页 reload，且反复冲掉自动化验证

- **现象**：dev 下页面每隔几十秒自发整页刷新（vite 日志 `page reload dist/index.html`），长流程浏览器自动化反复被打断。
- **根因**：vite 默认 watch 项目根，`dist/`（此前构建产物）被外部进程触碰（杀毒/索引/其它构建）即触发 reload。
- **规避/解决**：`server.watch.ignored: ['**/dist/**', '**/dist-*/**']`；跑长链路浏览器验证时把整个流程压进单次 evaluate（工具/协议有 ~30s 上限），或避开有人在用的共享环境。
- **来源**：2026-10-07，M20 验收期间 dev 环境反复 reload。

### monaco 在 v-show 隐藏容器里挂载后视图 0 行：watch 必须用 flush: 'post' 再调 layout()

- **现象**：生产环境首次打开文件编辑器，monaco 外壳/状态栏正常、模型已挂载且 `getValue()` 内容完整（`getLayoutInfo` 也能取到），但 `.view-lines` 0×0、0 行，`execCommand` 插入无显示；dev 模式不复现（时序不同）。`automaticLayout: true` 也救不了。
- **根因**：编辑器在 `v-show="activeTab"` 为 false（display:none）的容器里创建，随后 model 由异步 open 流程填充。model 的 watch 默认 `flush: 'pre'`，在 **Vue DOM 补丁之前**执行 `editor.layout()`——此刻容器仍 display:none，monaco 量到 0 高并固定；之后容器尺寸 0→516 虽有变化，monaco 内部 automaticLayout 的 RO 触发时序在此场景下不再纠正（实测 4s 不恢复）。
- **规避/解决**：model watch 加 `{ flush: 'post' }`（DOM 补丁后 v-show 已可见，layout() 量到真实高度即渲染）；诊断手法：把 editor 实例挂到 `window.__ydEditor`，生产包里直接 `getValue()/getLayoutInfo()` + 手动 `layout()` 看行数变化（0→1 即为此症）。
- **来源**：2026-10-07，M20 文件编辑器 prod 首开空白排查。

### vite dev 代理转发 WebSocket 必须显式 `ws: true`，否则终端类页面握手挂起/失败

- **现象**：dev（`VITE_ENABLE_PROXY`）下终端页 WS 停在"连接中"，页面直探 `ws://localhost:9000/proxy/api/v1/terminal` 无 `ws:true` 时 8s 无响应（挂起），加 `ws:true` 后变成 `error + close 1006`（未带合法 token 被拒）——两种表象都是代理层问题。
- **根因**：vite `server.proxy` 底层 http-proxy 默认**不转发 upgrade 事件**；不配 `ws:true` 时 WS 握手既不成功也不立刻失败。后端本身（192.168.100.142:8880）直连 WS 正常。
- **规避/解决**：`vite.config.ts` 的 `'/proxy'` 代理加 `ws: true`（终端/容器 exec/日志流全靠它）；排查 WS 问题时用真实 token 分别探"直连后端"与"经代理"两条路，快速二分定位。
- **来源**：2026-10-07，M21 终端工作台 dev 联调。

### vue-web-terminal 对接真实 PTY 的四个坑（ANSI 过滤、测量标尺、输入行、内联对象重连）

- **现象**：① 终端输出出现 `]0;root@host: /opt/xx`、`□K` 等乱码；② aria 快照/textContent 里出现 `aaaaaaaaaa你你你你你你你你你你`；③ 直接给输入 textarea set value + dispatch input 事件不生效；④ 父组件模板内联 `:endpoint="{...}"` 对象 + 子组件 `watch(..., {deep:true})` 导致 WS 在 open→close 死循环重连。
- **根因**：① vwt 的 ANSI 解析**只翻译 SGR 着色码**，OSC（窗口标题 `\x1b]0;…\x07`）和其余 CSI（`\e[K` 擦行、`\e[?2004h` 括号粘贴）原样漏成可见文本；② 那是 vwt 内部的中英字符宽度测量标尺（`terminalEnFlagRef/terminalCnFlagRef`，span.t-cmd-line-content），恒在 DOM 里；③ vwt 输入行走自维护 cursorConf 的逐键 keydown 渲染，不走 v-model；④ 内联对象字面量每次渲染都是新引用，deep watch 按引用比较必触发。
- **规避/解决**：① 写入前清洗：`去 OSC → CSI 仅保留 …m（SGR）→ 去除 \x00-\x08\x0b-\x1f\x7f`（保留 \t\n）；② 测量标尺是正常现象别误判；③ 自动化驱动 vwt 输入要用逐键真实 keydown（Enter 需 keydown+keyup 都发）；④ 组件对外部对象 prop 的重连监听一律用**序列化 key**（如 `host:${node}`、`exec:${id}:${cmd}`）而非 deep watch 对象引用。
- **来源**：2026-10-07，M21 YdTerminal（components/YdTerminal/VwtEngine.vue）。

### xterm.js 默认前景色是白色：浅色主题只改 background 会"白字白底"假性白屏

- **现象**：xterm 实例已挂载（`.xterm` 存在、`fit` 报出正常 cols/rows、`.xterm-rows` 里提示符文本完好），但屏幕看上去全白"没输出"。
- **根因**：xterm 默认 theme `{ foreground: '#ffffff', background: '#000000' }`；只覆写 `background: '#ffffff'` 不给 `foreground` → 白字白底。旧版 `views/terminal/index.vue`、容器 exec 弹窗、编辑器 TerminalPanel 都有此潜在问题（仅浅色模式触发）。
- **规避/解决**：自定义主题时 background/foreground/cursor/cursorAccent 成对显式给出；排查"白屏"先用 `.xterm-rows` textContent 确认数据链路（有文本=纯配色问题），再查渲染层。
- **来源**：2026-10-07，M21 XtermEngine（components/YdTerminal/XtermEngine.vue），并顺带修掉旧页面的同款问题。

### fantastic-admin 多标签的"恢复上次激活页"会改写 hash，自动化验证路由时别信初始 hash

- **现象**：脚本 `location.hash = '#/xxx'` 后 reload，页面落在了另一个路由；或 reload 后 DOM 探针按预期路由查不到元素，误判为"视图切换卡死/组件没渲染"。
- **根因**：fa 多标签持久化了"最后激活页"，应用启动时按存储恢复路由，覆盖脚本预设的 hash；另外引擎切换类状态存 localStorage（如 `ypanel.terminal.engine`），探针选择器要跟引擎状态对齐（vwt 引擎下查 `.xterm` 恒为 false，反之亦然）。
- **规避/解决**：自动化验证路由类功能：reload 后先断言 `location.hash` 再探测 DOM；用截图做最终判据，DOM 探针选择器必须与持久化状态一致。
- **来源**：2026-10-07，M21 终端工作台浏览器验证。

### xterm.css 给 `.xterm-viewport` 硬编码黑底：滚动条槽露出"黑框"

- **现象**：xterm 终端（容器终端弹窗最明显）右侧有一条 ~14px 纵向黑边，看起来像终端外面套了层黑框；亮色主题下尤其扎眼。
- **根因**：`@xterm/xterm/css/xterm.css` 写死 `.xterm .xterm-viewport { background-color: #000000 }`（官方注释：macOS 滚动条需要不透明背景）。viewport 层比文字层宽出一个滚动条槽（14px），主题背景在 v6 DOM 渲染器下**不再以内联样式写回 viewport**，于是槽位永远露出这条 CSS 黑底。
- **规避/解决**：宿主样式按命名空间覆盖为透明（`.yd-xterm .xterm .xterm-viewport { background-color: transparent }`，选择器三级压过库样式且随明暗主题自适应）；排查思路是对比 viewport 与 screen 的 `getBoundingClientRect` 宽度差 = 滚动条槽宽，再看 viewport 计算背景色是否被库 CSS 写死。
- **来源**：2026-10-07，M21 容器终端黑框排查（components/YdTerminal/XtermEngine.vue）。
