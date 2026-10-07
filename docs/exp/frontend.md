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

### monaco 在 v-show 隐藏容器里挂载后视图 0 行：挂载时机 + flush: 'post' + settle 脉冲三件套

- **现象**：生产环境首次打开文件编辑器，monaco 外壳/状态栏正常、模型已挂载且 `getValue()` 内容完整（`getLayoutInfo` 也能取到），但 `.view-lines` 0×0、0 行，`execCommand` 插入无显示；dev 模式不复现（时序不同）。`automaticLayout: true` 也救不了（实测 >15s 不自愈）；手动 `layout()` 一次立即恢复（0→1 行）。
- **根因**：编辑器在 FaModal 开启动画期间（容器有效尺寸 0/变化中）挂载并完成首次测量；model 由异步 open 流程稍后填充，model watch 补 `layout()` 时动画仍未结束，量到的还是过渡尺寸，之后 monaco 内部布局状态停滞不再纠正。仅加 `flush: 'post'` 仍不够——动画时长不固定，post 时机也可能落在动画内。
- **规避/解决**（三件套，缺一可能复发）：① **挂载时机**：编辑器组件等弹窗动画结束再挂载（`v-if="editorOpened"`；注意 FaModal 定制尺寸下 `@opened` 事件可能不触发，需 `visible=true` 后 350ms 兜底置位）；② model 的 watch 用 `{ flush: 'post', immediate: true }`，setModel 后调 `layout()`；③ **settle 脉冲 + 交互兜底**：创建/换模型后 50/300/1000/2500/5000/12000ms 各补一次 layout，编辑区 pointerdown 与 window resize 也补——任何时序下用户一交互即渲染。诊断手法：把 editor 实例临时挂 `window.__ydEditor`，生产包直接 `getValue()/getLayoutInfo()` + 手动 `layout()` 看行数变化（0→1 即为此症）。
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

### langchaingo v0.1.15 三处 API 坑（tools 是接口/无 SystemMessageContent/chains 流式选项）

- **现象**：按网上常见示例写 langchaingo 工具调用，编译报 `invalid composite literal element type tools.Tool`、`undefined: llms.SystemMessageContent`、`chains.WithLLMOptions undefined`。
- **根因**：v0.1.15 中 `tools.Tool` 是**接口**（Name/Description/Call 三方法）而非 struct；system 消息用 `llms.TextParts(llms.ChatMessageTypeSystem, text)`（无 SystemMessageContent 辅助）；chains 流式用 `chains.WithStreamingFunc`（非 WithLLMOptions）。
- **规避/解决**：自定义工具写成实现接口的小 struct；以 GOMODCACHE 内实际源码为准逐 API 核对，勿凭记忆/旧示例。另：Bash heredoc 会吃一层反斜杠——写含 `\n` 的 Go 源码用 Edit 工具或 python chr() 构造，别在 heredoc python 里硬写。
- **来源**：2026-10-07，AI v2 工具循环（core/internal/service/aichat.go）。

### reka-ui Slot 转发（FaContextMenu 包裹）会丢掉子元素上的 @click

- **现象**：文件树行按钮（`<FaContextMenu><button @click=…>`）渲染正常但左键点击毫无反应，右键菜单正常；`btn.__vueParentComponent.vnode.props` 里只有 reka 的 `onContextmenu/onPointerdown/data-state…`，**onClick 消失**；无论合成派发还是 CDP 真实点击都无效。
- **根因**：FaContextMenu → reka ContextMenuTrigger（Primitive/Slot）用 cloneVNode 把 trigger 自身 props 合并进插槽子元素，本例中合并结果把子元素 @click 覆盖丢失（reka 2.10.5）。编辑器文件树（M20）与容器/终端文件树同构同病。
- **规避/解决**：不要把可交互处理绑在被 FaContextMenu 包裹的元素上——在外层再包一个普通元素绑 @click（`<span @click=…><FaContextMenu>…</FaContextMenu></span>`，点击冒泡触发）。排查此类"渲染正常但点击死"的问题时，直接检查 `__vueParentComponent.vnode.props` 里有没有自己的 onClick。
- **来源**：2026-10-07，M22 文件树行点击失效排查（YdFileTreeNode + editor/FileTreeRow 同步修复）。

### store 驱动的弹窗组件只在挂载它的页面上渲染：跨页调用 open() 不出现弹窗

- **现象**：容器页「终端」按钮调用 `fileEditorStore.openWorkspace(...)`，pinia 里 `visible=true`、`currentContainer` 正确，但界面毫无变化、无任何报错。
- **根因**：文件编辑工作台弹窗组件（FileEditorWorkspace）只写在 file_management 页面模板里；store 是全局的，组件实例不是——不在挂载页上调用自然没有渲染载体。
- **规避/解决**：需要跨页触发的弹窗，要么把组件挂到布局层（全局单实例），要么在每个触发页都挂一份组件（本项目容器页选择后者：`<FileEditorWorkspace />`）。排查时先看 pinia 状态是否正确，再查"组件实例在哪"。
- **来源**：2026-10-07，M22 容器终端接入文件管理弹窗。

### dev 长会话 + 边改边验：旧 chunk 动态 import 失效会造成"整页白屏/视图错乱"的假故障

- **现象**：验证中反复出现切换标签后内容区空白、路由视图错乱、异步组件加载失败，表现得像路由/keep-alive 层面的 bug；重启 dev 或硬刷新后消失。
- **根因**：长时间运行的 Vite dev 会话在源码多次变更后，已加载页面的旧 chunk 再去懒加载新 hash 的异步组件时会失败（或模块图失效），叠加 fa 多标签 keep-alive 后表现为整页级错乱。
- **规避/解决**：验证类操作前先硬刷新一次；出现"不可能的空白"先怀疑 dev 会话陈旧而非代码缺陷，用生产构建（部署产物）做最终判定。
- **来源**：2026-10-07，M21/M22 浏览器验证过程中的多次误判。

### UnoCSS presetIcons 的 i- 前缀图标是构建期按需内联：动态拼接的 class 不会被扫描生成

- **现象**：镜像名 → 品牌 logo 的映射如果写成 `` `i-logos:${kw}` `` 拼接，生产环境图标全部空白（dev 也一样）；而源码里静态写死的 `i-logos:docker-icon` 正常显示。
- **根因**：fa 基座的 UnoCSS `presetIcons`（collectionsNodeResolvePath 指向 @iconify/json）在**构建期**扫描源码 token、按需把图标 SVG 内联成 CSS。运行时拼出来的 class 没有对应 CSS 规则。另注意 FaIcon 的双通道：`i-xxx:yyy` 走 unocss（构建期内联，零 CDN），`xxx:yyy`（不带 i-）走 @iconify/vue 运行时（需 addCollection 预载或在线 API，本项目 isOfflineUse=false 时后者不可用）。
- **规避/解决**：映射表里存**完整 class 字面量**（`['redis', 'i-logos:redis']`），字面量出现在源码中即被扫描提取；品牌图标用 `i-logos:*`（@iconify/json 的 logos 集合，彩色品牌 logo），验证某图标存在先查 `web/node_modules/.pnpm/@iconify+json@*/node_modules/@iconify/json/json/logos.json` 的 icons 键。
- **来源**：2026-10-07，M23 YdAppIcon 品牌图标组件。

### 并行会话共用工作区：构建被对方活跃文件的半成品阻塞时的处置

- **现象**：本会话构建（vue-tsc/vite build）报 `src/views/ai/index.vue` 模板未闭合——该文件并非本会话改动，而是用户另一个并行会话正在编辑的半成品；同时 store.ts 的 API 签名也被对方改掉（新增 sourceId/分页），本会话调用点类型报错。
- **根因**：多个会话在同一 git 工作区并行开发，构建是全仓级的，任何人保留未完成的编辑都会挡住所有人的产物构建。
- **规避/解决**：① 类型检查用 `vue-tsc -b | grep -v <对方文件>` 隔离自己的范围，先保证**自己改动零错误**；② 不代改对方活跃文件（保存即冲突），等对方合流提交（git log 出现合流 commit）后重试构建；③ 自己的调用点主动适配对方已落地的新签名（如 store 分页返回 `.items`），比要求对方兼容旧签名更稳。
- **来源**：2026-10-07，M23 容器套件与商店多源/AI v2 会话并行期间。

### FaModal 插槽里的组件用 defineModel 接 visible 会断链：关闭态已挂载，打开时收不到更新

- **现象**：`<FaModal v-model="open"><MyComp v-model:visible="open" /></FaModal>`（MyComp 靠 watch(visible) 加载数据）：弹窗能正常开关，但 MyComp 的 visible 永远是初始 false——打开时数据不加载，界面永远显示空态/旧态。
- **根因**：FaModal（reka Dialog）默认插槽内容在**关闭态就已挂载**（未 destroy 或首次挂载先于打开），此时 visible=false；用户点击打开后外层 ref 变 true，但插槽内容经过 reka Primitive/Slot 的 cloneVNode 合并链，**defineModel 的更新没有传导到深层组件**（同 exp「reka Slot 转发丢 @click」的家族问题，这次丢的是 prop 更新）。
- **规避/解决**：把弹窗做进组件内部（组件自身持 FaModal + defineModel 控制开关），内容面板用子组件承载且随 `destroy-on-close` 的打开时机**重建**——onMounted 必然触发加载；需要内嵌在别处时给组件加 `bare` prop 只渲染面板。忌讳「外部弹窗 + 传 visible 驱动加载」的组合。
- **来源**：2026-10-07，M23 YdRevisionHistory 版本历史弹窗空白排查（弹窗开、请求零发）。

### 后台/遮挡状态的 Chromium 里验收 fa 页面：rAF 冻结导致 RouterView Transition out-in 永久卡在旧组件

- **现象**：浏览器自动化（IAB webview 被遮挡）里 `router.push` 后 hash/面包屑都变了，内容区却停留在旧页面；`#app-content` 子元素常年挂着 `fade-leave-from fade-leave-active`。截图 capture 也超时。硬导航（整页 goto）一切正常。
- **根因**：fa Layout 用 `<Transition mode="out-in">` 包 RouterView。out-in 要等旧组件 leave 完成——Vue 的 nextFrame 依赖 requestAnimationFrame，Chromium 对 occluded/后台渲染进程**暂停 rAF**（`document.visibilityState==='visible'` 也会发生，按窗口遮挡判定），leave 永不完成，新组件永不挂载。同理 xterm/echarts 的画布渲染（内部 rAF）在后台也不出图，但数据链路正常。
- **规避/解决**：后台自动化验收一律用**硬导航 + 组件内事件派发**（tab 类 reka 组件要派发完整指针序列 pointerdown/mousedown/pointerup/mouseup/click，仅 `el.click()` 不触发）；弹窗类组件不受影响（teleport 到 body，不走 RouterView Transition）。真实用户窗口可见时无此问题，非代码缺陷。
- **来源**：2026-10-07，M23 验收（hash 变内容不变的两小时弯路）。

### 前端 API 模块的 node 查询串拼接：nodeQ 产生 `&node=` 前缀，GET 无既有 query 时拼出坏 URL

- **现象**：给原本无 query 的 GET 接口加节点参数时，若直接 `api.get(\`api/v1/services${nodeQ(node)}\`)`，node 非 local 时会拼出 `api/v1/services&node=2`——`&` 开头的 query 被忽略，参数悄悄丢失（请求本身不报错，数据"看似正常"实为本机数据）。
- **根因**：`file.ts`/`system.ts` 的 `nodeQ()` 助手返回的是 `&node=xxx`（为追加在既有 query 后设计），只有调用方自己保证前面已有 `?`。
- **规避/解决**：无既有 query 的 GET 用 `?1=1` 占位（system.ts overview 的写法：`api/v1/system/overview?1=1${nodeQ(node)}`），或统一改用 nodeexec.ts 新增的 `nodeQS()`（内部以 `1=1` 起始并合并额外参数）；POST 类用 `nodeQ2()`（返回 `?node=xxx` 或空串）。新增带 node 参数的接口一律走这三个助手，不再手拼。
- **来源**：2026-10-07，M25 进程/服务与文件管理多节点化。

### 模板 ref 与 setup 变量同名：SFC 编译器劫持为变量引用，useTemplateRef 生产环境失效（dev 正常）

- **现象**：历史监控页生产环境 summary 卡片有值、图表容器 div 尺寸正常，但 echarts 永远不 init、canvas 数 0、**控制台零报错**；dev 环境一切正常。切时间范围、等 DOM、加 nextTick 都无效。
- **根因**：`script setup` 里有 `let cpuChart = null`（echarts 实例容器），模板又写了 `ref="cpuChart"`。生产构建的 SFC 编译产物把该 ref 编译成**对同名 setup 变量的引用**（`ref_key:'cpuChart', ref: <cpuChart变量>`），而非字符串 ref——运行时该变量是 `null`（非 RefImpl/函数），元素永远写不回，`useTemplateRef('cpuChart')` 的字符串桥接也彻底断掉。dev 产物却是字符串 `ref: "cpuChart"`（静态提升），故 dev 复现不了。
- **规避/解决**：模板 ref 的名字**不得与任何 setup 顶级绑定同名**——echarts 实例容器等变量一律加后缀（如 `cpuChartInstance`），`ref="cpuChart"` 只留给 `useTemplateRef`。排查此类"生产空白、dev 正常、零报错"问题：直接解剖生产 chunk（grep `ref:` 与 `useTemplateRef`），dev 编译产物用 `curl http://localhost:端口/src/xx.vue` 直接拿。
- **来源**：2026-10-07，历史监控图表空白（views/manage/monitor.vue）。

### 并行构建/检查流程把 .js 产物同步进 src：全仓 TS2307 成片误报 + vite 模块解析劫持

- **现象**：`src/**` 下出现与 .ts 成对的未跟踪 `.js`（同秒批量写入，删后几分钟内再生）；`vue-tsc` 全仓成片报 `TS2307: Cannot find module '@/utils/xxx'`（连基础模块都"找不到"）；vite 构建的模块解析**优先命中 .js**（resolve extensions 里 .js 在 .ts 前），可能吃进过期转译物。
- **根因**：另一并行流程（Temp 下 ypanel-master-check 副本环境）周期性把副本里的编译产物同步回真仓库 src。**同名 .js 的存在本身就让 TS 的 paths 别名解析报 TS2307**（受控实验：仅改名 `dayjs.js` → App.vue 全部 TS2307 消失）。本项目 `vue-tsc -b` 本身不 emit（受控实验前后 .js 数量/时间戳不变），不是它的锅。
- **规避/解决**：`find src -name "*.js" ! -path "*monaco-workers*" -delete` 清掉后**立即竞速发起** vue-tsc/vite build（模块解析发生在进程启动期，中途 .js 再生不影响已启动的解析）；monaco-workers 下 5 个 .js 是源码要排除。构建用独立 outDir 绕开 dist 竞争；Go 侧被对方半成品挡住编译时用 `git worktree add <tmp> HEAD` 隔离构建（HEAD 后端 + 本地新前端产物）。部署后 `curl /health` 看版本串 + 浏览器 `performance.getEntriesByType('resource')` 看 chunk hash，确认服务器跑的真是自己的产物（本次就被并行会话 0830 的部署覆盖过一次）。
- **来源**：2026-10-07，历史监控修复期间的并行构建互踩。

### useAiChat.send() 把刚 push 的 user 消息当 assistant 累积器：回答/思考/步骤全部灌进用户气泡

- **现象**：AI 回答以**用户气泡样式**（右对齐、纯文本、user 图标）显示，且用户问题和 AI 回答拼在同一条里；深度思考块、步骤时间线、Markdown 渲染全部不出现（YdAiReasoning/YdAiProcess 的 `role==='assistant'` 分支永远不成立）。一次对话数组里只有 1 条消息而不是 2 条。
- **根因**：已提交代码里 send() 写的是 `const assistant = messages.value[messages.value.length - 1]`——此时数组最后一条是**刚 push 的 user 消息**（"响应式断裂修复"提交时把 addAssistant() 删了却没补占位消息），SSE 的 content/reasoning/step 全部累积到 user 消息上。注释写着"流式更新最后一条 assistant"，代码语义却是"复用最后一条消息"。
- **规避/解决**：send() 内 push 完 user 消息后**必须补一条 assistant 占位消息**（`{role:'assistant', content:'', pending:true, steps:[]}`）再经数组索引取 proxy 引用（保留原"索引取 proxy 保响应式"的意图）；回归验证时数一下消息条数：一次 send 应产生 user+assistant 两条。
- **来源**：2026-10-07，AI 浮层对话气泡全变用户样式（composables/useAiChat.ts，d454e9b 引入）。

### 同源 hash 导航不重载文档：验证新前端时浏览器可能一直跑旧 bundle

- **现象**：部署新前端后，浏览器里修掉的 bug（如步骤显示原始 JSON）依然复现；`tab.goto('同源/#/其它页')` 前后页面表现毫无变化，一度误判"修复没生效/部署没成功"。
- **根因**：`http://host/#/a` → `http://host/` → `http://host/#/b` 全程是**同文档 hash 导航**，不会重新拉 index.html 和新 hash 的 JS bundle；`reload()` 也可能命中 index.html 的 HTTP 缓存继续用旧产物。之前"修复后仍有问题"的两次假象都是旧 bundle 在跑。
- **规避/解决**：验证新前端前先断言 bundle 版本——`performance.getEntriesByType('resource').map(e=>e.name).filter(n=>n.includes('index-'))` 与服务器 `curl / | grep -o "assets/index-[^\"]*\.js"` 的 hash 比对；不一致就 `reload()` 后复查。DOM 统计消息数时注意别用会命中嵌套组件根节点的宽泛选择器（如 `.space-y-1` 会把 YdAiProcess 步骤时间线也数成一条消息）。
- **来源**：2026-10-07，AI 浮层修复验证期两次假阴性（浏览器自动化 + 部署验证流程）。
