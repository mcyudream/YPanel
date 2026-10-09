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

### fa 次侧栏的「父标题行」来自有可见子页的模块：单子页模块（子页 menu:false）才会渲染成纯叶子项

- **现象**：容器域拆分二级菜单时，无论怎么设菜单标题，次侧栏顶部总有一行「容器（图标+展开箭头）」父标题，子项挂在它下面；用户要的是直接平铺叶子项。
- **根因**：fa Menu 的 `initItems` 规则——节点 `children.some(c => c.meta?.menu !== false)` 为 true 时渲染为 **SubMenu**（父标题行 + 展开子项）；为 false（无 children 或全部 menu:false）时渲染为**叶子 item**（点击直达，title/icon 取**模块自身 meta**）。模块级 meta.menu 不参与该判断（写了也会被忽略）。
- **规避/解决**：想要「无父标题的平铺叶子」：每个功能做一个「单子页模块」——模块 `{path, component: Layout, meta: {title, icon}}` + 子页 `{path: '', component, meta: {title, icon, menu: false}}`，子页全隐藏后模块即叶子。需要「路由可达但菜单完全不出现」的页面（如列表页由分组头直达）：挂为某模块的隐藏子页或独立 menu:false 模块（模块级 menu:false 不影响 Menu 判断，仍会渲染叶子——需要彻底不出现就从分组 children 移除该模块并另行注册路由）。
- **来源**：2026-10-07，M23 容器域菜单平铺（应用/镜像/网络/卷/配置五叶子，容器列表由分组头直达）。

### 递归组件模板名 ≠ 文件名且未导入：构建零报错，运行时静默渲染为未知元素（树"有数据不显示"）

- **现象**：文件树（YdFileTree）点开目录显示「（空目录）」或毫无反应，网络面板请求全部 200 且数据完整；只有部分页面正常（文件管理页），终端工作台/容器详情全部异常。
- **根因**：`components/YdFileTree/TreeNode.vue` 模板递归写 `<YdFileTreeNode>`，但该文件既没 import 这个名字、名字也不等于文件名（TreeNode）——Vue SFC 只支持**文件名自引用**；unplugin-vue-components 的 components.d.ts 也没有该声明（只生成了目录 index.vue 的 YdFileTree）。编译产物里落成 `resolveComponent("YdFileTreeNode")`，运行时解析失败 → 渲染为**原生未知元素** `<ydfiletreenode>`（v-for 出 188 个隐形空标签），数据、expanded 全部正常只是看不见。对比 `FileTreeRow.vue` 用 `<FileTreeRow>`（=文件名）所以一直正常。
- **规避/解决**：递归自引用只有两种可靠写法——模板用**文件名**（`<TreeNode>`），或顶部**显式自导入** `import YdFileTreeNode from './TreeNode.vue'`（推荐，抗重命名）。排查特征：DOM 查未知标签 `document.querySelectorAll('ydfiletreenode').length`；产物 grep `resolveComponent("组件名")`（修复后字符串应消失、编译为直接绑定）。构建/vue-tsc 对此类问题零报错，不能依赖编译期拦截。
- **来源**：2026-10-07，终端文件树/容器文件树"无法获取文件夹子项"（YdFileTree/TreeNode.vue，组件抽取重命名时引入）。

### FaSwitch（reka-ui Switch）受控 prop 就是 modelValue：传 checked 会落非受控、全渲染 OFF

- **现象**：`<FaSwitch :checked="x" />` 页面上**所有开关初始都是关的状态**（与数据不符）；点击 UI 会翻但绑定的业务回调不执行。
- **根因**：FaSwitch 转发 reka-ui SwitchRoot，其受控 prop 是 **`modelValue`**（emits `update:modelValue`，trueValue/falseValue 默认 true/false）；`checked` 只是组件内部计算值（`modelValue === trueValue`），**不是 prop**——传 `:checked` 会作为无效 attr 落到根 button 上，SwitchRoot 因 modelValue===undefined 落非受控（passive 内部态、初始 false），UI 与数据彻底脱钩。排查时曾被「radix 旧版用 checked」的印象与 FaTabs 的 el.click() 坑带偏方向，绕了 v-model → :model-value → :checked 三轮才定位。
- **规避/解决**：正确写法就是标准 `<FaSwitch v-model="x" @update:model-value="v => 异步保存(x)" />`；验收开关视觉态用 `aria-checked` 属性与 API 状态对照。reka 系组件受控 prop 名不统一（Switch=modelValue、Dialog=open），接新组件先读它的 props 定义再绑定。自动化环境对 reka 组件 el.click()/合成 pointer 均不可靠，交互闭环以真实用户操作或 API 状态为准。
- **来源**：2026-10-07，AI 系统工具开关页（views/ai/tools.vue），用户反馈「开关与启用状态不对（全是关）」定位。

### Vue 模板中原生元素不要写自闭合标签：<div ref="x" /> 会被解析为未闭合开标签

- **现象**：echarts 图表容器写成 `<div ref="cpuChart" class="h-56 ..." />`，构建通过、页面渲染出边框框，但 ref 永远绑不上（组件内 ref 值为 null）、后续兄弟节点层级错乱，图表永不绘制且无任何报错。
- **根因**：Vue 模板遵循 HTML 解析规则，**原生 HTML 元素的自闭合斜杠被忽略**（`<div/>` ≡ `<div>` 开标签），后续内容全部成为其子节点直至"真正的"闭合标签；ref 绑定与 DOM 结构随之错乱。组件标签（`<FaXxx />`）自闭合才是合法的。
- **规避/解决**：原生元素一律显式闭合 `<div></div>`；排查「ref 恒为 null/兄弟节点消失」类问题时先 grep 模板自闭合的原生元素。同族坑：模板 ref 属性名必须与 `<script setup>` 中**同名**变量/`useTemplateRef('名')` 一致，名字对不上 ref 恒为 null 且无报错（rolldown-vite 构建下 useTemplateRef 亦有失效案例，最稳的是同名 `ref()` 变量）。
- **来源**：2026-10-08，M23 容器统计图表空白（init 5 次全部因 ref null 跳过，靠诊断行定位）。

### node_modules/.vite 缓存目录被外部清掉：预打包依赖全 504，应用卡「载入中」且极易误判为代码问题

- **现象**：dev 首页永远停在 fa 启动 loading；手动 `router.push` 报 `Failed to fetch dynamically imported module: .../overview/index.vue?t=xxx`；curl 该 URL 却 200，vue-tsc/build 全绿。
- **根因**：`node_modules/.vite/deps` 目录被并行流程（或重装依赖）删除，vite 进程还活着但预打包产物全丢，所有 `.vite/deps/*?v=hash` 返回 504（Outdated Optimize Dep）。overview 引 echarts → 动态 import 失败 → vue-router 初始导航崩溃 → `router.isReady()` 永不 resolve → 整个应用卡启动遮罩。curl 200 的是 `.vue` 源模块，504 的是它依赖图里的预打包 deps——只 curl 顶层模块查不出来。
- **规避/解决**：诊断链：页内 `new Function('return import(url)')()` 逐层 import 顶层依赖（sandbox 会改写裸 import 语法）→ 定位 `utils/echarts.ts` 失败 → curl 其 `from "/node_modules/.vite/deps/xxx.js?v=hash"` 依赖确认 504。修复：杀掉 vite 进程重启重新预打包（新 hash 变化后旧 `?v=` 仍 504，属正常）。
- **来源**：2026-10-08，M27 验收期间 dev 环境瘫痪排查。

### 生产构建下 useTemplateRef 对「v-else-if 分支内」的 ref 失效的排查路径，与三处易混坑

- **现象**：同一文件里三个 ref（netChartRef/sysChartRef/usageChartRef）生产环境全部正常，唯独 duChartRef（`<template v-else-if>` 分支内）死活不绑：图表实例 0 个、无任何报错；dev 环境一切正常。
- **排查**：产物解剖四条 ref 编译完全一致（`ref_key:` + `ref:X`，`X=r('duChartRef')`）→ 排除编译差异；`__vueParentComponent` 生产包不存在 → 改从 `app._instance.subTree` 遍历组件树（生产包遍历也难命中异步组件）；最终在 dev 环境（dev server 重启 + 真实登录）确认 ref 绑定与实例创建全部正常，再回到生产比对**部署产物**才发现：入口 chunk hash（`index-*.js`）与本地不一致——部署用的临时构建目录里嵌的是旧 dist，三轮前端修复根本没进二进制。
- **教训**：① 「dev 正常生产异常」先比对部署产物 hash（`curl /` 的 index-*.js vs 本地 dist），再怀疑编译差异；② `/health` 版本串只证明二进制是新编译的，不证明嵌的前端是新的——隔离目录构建时 embed 源必须同步拷贝；③ `<div ref/>` 自闭合与显式闭合编译产物相同，但按 exp 首条规范仍应显式闭合；④ 背景窗口验收用 DOM/canvas 像素断言（`getImageData` 采样 alpha）+ `_echarts_instance_` 属性探针，截图在遮挡窗口下有「陈旧瓦片拼接」假象不可作准。
- **来源**：2026-10-08，M27 磁盘占用分析 treemap 生产不渲染排查（两小时弯路：先疑 ref 编译、再疑 rAF 冻结、实为隔离构建 embed 未同步）。

### 兄弟仓库前端库源码直连（YudreamWebOS 接入 M29）：alias 指目录、tsconfig paths、UnoCSS 走模块图

- **背景**：桌面工作台改用 `../yudream-web-os` 全家桶，要求库内修复即时生效、不做构建同步。
- **做法**：① `resolve.alias` 对象形式把 `@yudream/yudream-webos-*` 五包指到兄弟仓库 `packages/*/src` **目录**（不是 index.ts——rolldown 解析器对 alias 目标按目录解析，指文件会静默失配、报「Failed to resolve import」且无任何线索）；不发布不装依赖，`package.json` 零改动。② `tsconfig.app.json` paths 指到源码 `index.ts`，vue-tsc 按源码全量类型检查（ypanel 的 noUnusedLocals 比 webos 严格，会抓出对方仓库的死参数——修在对方仓库而不是放宽自己）。③ UnoCSS 无需配置：content.pipeline 首条是全量文件名正则，webos 源码沿 vite 模块图自动进扫描（`i-lucide-*` 图标类正常提取）。④ webos 插件在 `main.ts` 全局 `app.use(createWebOS(...))`，实例构造纯 TS 无 DOM 副作用，经典模式常驻无碍。⑤ 内部跨包互引（vue→core）靠同一组 alias 解析，无需 node_modules。
- **验收**：dev 双窗开窗（真实后端数据）、刷新会话恢复、Ctrl+K、返回经典、`vue-tsc -b` 与生产构建全绿。
- **来源**：2026-10-08，M29 桌面工作台 webos 化。

### app 目录同时存在 vite.config.js 与 vite.config.ts：js 抢先加载，改 ts 等于白改

- **现象**：给 `vite.config.ts` 加 resolve.alias，重启 dev 后 `Failed to resolve import "@yudream/..."`；对象/数组、字符串/正则、目录/文件目标全试遍无效；新增一个明显该生效的探针别名同样失效，而既有的 `@`/`#`/`#monaco/*` 却正常。
- **根因**：app 目录下存在并行流程转译出的 `vite.config.js`（内容是旧版 vite.config.ts 的逐句转译）。**vite 配置发现顺序 .js 先于 .ts**，实际生效的永远是旧 js——对 ts 的一切修改都是死代码。
- **规避/解决**：改 vite 配置必须**两份同步**（或删 js，但并行流程可能再生成旧版回灌，双写更稳）；在 ts 顶部加警告注释。排查手法：怀疑配置没生效时，先 `ls vite.config.*` 看有没有同名多扩展；再用一个「必然生效」的探针别名做对照（探针也失效=配置文件没被读，而非别名写法问题）。
- **来源**：2026-10-08，M29 webos 接入 alias 半小时排查（五种写法试遍后靠探针定位）。

### 遮挡/后台 IAB 的交互自动化边界：合成指针可用、可信输入(CUA)不可用、setPointerCapture 类交互无法仿真

- **补充**（webos 侧 exp 有姊妹条）：遮挡窗口里 `tab.cua.click()` 返回成功但页面零事件（capture 监听证实）；合成 `dispatchEvent` 完整序列（pointerdown→mousedown→pointerup→mouseup→click）对自绘组件一律有效，Vue 的 `@dblclick` 用单发 `dblclick` 事件即可触发。但依赖 `setPointerCapture` 的真实拖拽无法仿真（合成 pointerId 报 InvalidPointerId），窗口拖拽类验收只能靠真实鼠标或库内单测覆盖。断言时机要留足余量：最大化等带过渡的状态变更，断言过早会误判「没生效」。
- **来源**：2026-10-08，M29 桌面工作台浏览器验收（CUA 零到达、拖拽仿真失败、最大化断言过早三次踩坑）。

### presetAttributify 的三个毒源：SVG 静态表现属性、注释里的属性字面量、include 误入嵌套 node_modules

- **现象**：dev 首页样式全裸奔，`/__uno.css` 500，报 `Unclosed bracket`/`Unclosed comment`，行号随内容漂移；改掉一处后错误页字节不变（多个毒源并存，postcss 只报第一个）。
- **根因**：presetAttributify 对源码原文做 `key="value"` 形态提取，三类内容会生成非法 CSS：① SVG 静态表现属性 `stroke="var(--x, rgba(...))"`（值含空格/括号 → 选择器值未闭合）；`transform="rotate(-90 22 22)"` 同罪；② **注释里**的属性字面量（写注释解释「不要写 stroke="var(...)"」本身就会中毒）；③ uno include 通配 `packages/components/**` 误入嵌套 node_modules——tailwind-merge dist 里类校验器的文档文本（row-end、`*/`、`/**`、scaleGridColRowStartAndEnd() 等）被分词成属性规则，值含注释符直接打开 CSS 注释。
- **规避/解决**：SVG 表现属性一律 scoped CSS 类（动态值 `:style` 是既有安全模式）；注释措辞避开 `key="value"` 字面量；include 收窄到 `packages/components/src/**`。**排查手法**：`npx unocss "<glob>" -o out.css` 按目录二分 + `postcss.parse(out)` 逐段定位（CLI 不走 postcss 不报错，要主动 parse）；dev 的 500 错误页 HTML 里内嵌 pluginCode 可抠出完整产物按行看。
- **来源**：2026-10-08，M29 第二轮小组件接入，__uno.css 500 三轮排查。

### uno.config.js 与 vite.config.js 同款双写坑：.js 转译产物抢占加载，改 .ts 白改

- **补充**：uno.config.ts 之外还有并行流程留下的 `web/uno.config.js`（Oct 7），UnoCSS 配置发现同样 .js 优先——uno include 收窄改 .ts 无效，必须双写。**规律**：改任何构建/样式配置前先 `ls *.config.*` 看有没有同名多扩展；.ts 顶部加「需与 .js 同步」警告注释。
- **来源**：2026-10-08，M29 第二轮（uno 500 修复在 .ts 上三轮无效后定位）。

### webos 桌面壳：应用/小组件注册必须在 setup 阶段（子组件先于父 onMounted 消费定义）

- **现象**：小组件定义在 onMounted 里经 registry.register 注册（含 app.widgets），Host 挂载更早，slot 里 `getDefinition()` 首渲染求值为 undefined → 渲染成 v-if 注释；之后无任何响应式依赖能触发补渲染（WidgetStore 是普通类，Map 变更对 Vue 不可见），开合无关面板强制父重渲染也无效。
- **根因**：Vue 子先父后的挂载顺序 + 非响应式 store 的消费时序。
- **规避/解决**：registry.register/dock.pin/setSystemMenus 全部移到 setup 同步段；slot 不读 store 的 getDefinition，改读 setup 期填充的本地组件表（`widgetComponents[id]`，静态后安全）。通用规则：**往非响应式容器里注册、且渲染层要消费的东西，注册动作必须早于任何子组件挂载**。
- **来源**：2026-10-08，M29 第二轮小组件空白卡排查。

### webos 桌面布局被持久化固化：清 LS 会被自己加的 pagehide flushNow 写回

- **现象**：重排桌面图标（列优先竖排）后验证，清掉 `ypanel.webos.*` 再 reload，图标仍按旧横排坐标渲染；新增应用则竖排出现在最右列——新旧两套坐标并存。
- **根因**：桌面布局 scope（desktop.layout）在首轮已持久化旧坐标；restore 后内存模型持有旧项，seed 新项触发 onChange → persist（dirty）→ reload 的 pagehide 触发 flushNow（M29 修复①加的兜底）→ **旧布局在 reload 完成前被写回 LS**，新页面 restore 又读到旧值。
- **规避/解决**：重置布局不要 removeItem，而是 `localStorage.setItem('<prefix>.desktop.layout', '[]')`——restore 读到空数组零恢复，内存模型清空后重播种新布局并覆盖持久化。开发/验收期清 webos 全部状态时同理（windows.session 置 '[]'）。
- **来源**：2026-10-08，M29 第三轮桌面竖排验证（被自己的 flushNow 兜底反噬一次）。

### persist.set 用 structuredClone 防别名：宿主传 Vue reactive 对象必炸 DataCloneError 且被 void 吞掉

- **现象**：webos 控制中心选壁纸后 UI 即时生效但从不持久化（reload 回默认），无任何报错；dock/会话等其它 scope 却一直正常。
- **根因**：core 的 MemoryPersistence.set 用 `structuredClone(value)` 防别名，而 `settings.wallpaper` 是 Vue reactive Proxy——structuredClone 不能克隆 Proxy，抛 DataCloneError；`void save()` 把 rejection 吞成静默。dock/session 等存的恰好是普通对象所以幸存。
- **规避/解决**：持久化值的最终形态就是 JSON（flush 走 JSON.stringify），克隆语义与之对齐——改 `JSON.parse(JSON.stringify(value))`；补 Proxy 入参回归用例。**通用教训**：`void asyncFn()` 会吞一切 rejection，关键写路径至少 console.error；排查「部分 scope 持久化失效」先挂 unhandledrejection 钩子再复现操作。
- **来源**：2026-10-08，M29 第四轮壁纸持久化排查（unhandledrejection 钩子一次定位）。

### fa 基座给 localStorage 加了全局键前缀（fa_dev_）：读 LS 必须经包装对象，Object.keys 结果才一致

- **现象**：持久化明明在工作，直接 `localStorage.getItem('ypanel.webos.xxx')` 却是 null，`Object.keys(localStorage)` 也「看不到」相关键，极易误判为「写路径坏了」。
- **根因**：fa 基座 `utils/storage` 把 window.localStorage 换成带 `fa_dev_` 键前缀的包装实现；绕过包装的裸 getItem 查不到，但代理的 get/set/keys 双向一致（同前缀）。
- **规避/解决**：调试持久化一律走页面内 `localStorage.xxx`（经包装），再不济 `Object.keys(localStorage)` 全量 dump 看真实键名（会带 fa_dev_ 前缀）；不要用「键不存在」断言写路径失败。
- **来源**：2026-10-08，M29 第四轮壁纸持久化排查。

### 给元素起 class="ring" 撞了 UnoCSS 原子类：svg 外凭空多一圈 box-shadow「白框」

- **现象**：小组件的环形仪表外有一层白色圆角方框，circle 全部隐藏后依然存在（不是画的），computed background/border 全透明——像「凭空画的」。
- **根因**：`class="ring"` 正是 UnoCSS presetWind 的 ring 工具类（box-shadow 一圈 ring 阴影，默认画在元素方框外）；scoped 样式与 uno 规则**同时生效**（scoped 只附加属性选择器，不互斥）。同理 `card`/`badge`/`border`/`invisible` 等常见英文词做类名都会撞。
- **规避/解决**：自定义类名避开 uno 原子类词表（统一业务前缀最稳，如 gauge-*）；排查「元素外多出莫名框」先怀疑类名撞原子类——`getComputedStyle` 看 box-shadow。
- **来源**：2026-10-08，M29 第五轮小组件白框二次排查（逐层 display:none 二分定位）。

### webos token 消费契约速查：--yw-label 不存在；通道型/实体型别混用

- **补充**（exp 既有条目的落地速查）：webos 语义色是 **oklch 通道三元组**（`--yw-foreground`/`--yw-muted-foreground`/`--yw-primary` 等，消费 `oklch(var(--x))` 或 `oklch(var(--x) / a)`）；实体值 token 只有 `--yw-label-2/3/4`、`--yw-separator`、`--yw-window-bg` 等（直接 `var()` 消费）。**没有 `--yw-label`**——引用未定义变量整条声明静默失效（computed 显示 none），文字丢色、轨道不渲染。实体值要调透明度用 `color-mix(in oklab, var(--x) 14%, transparent)`。宿主自定义组件配色前先 grep `packages/core/src/theme/tokens.ts` 的名单。
- **来源**：2026-10-08，M29 第五轮（小组件文字/轨道引用了不存在的 --yw-label，computed stroke=none 定位）。

### 合成事件派发 pointerenter 不冒泡：必须派发到绑定元素本身

- **补充**（IAB 交互自动化姊妹条）：`pointerenter/pointerleave` **不冒泡**——Vue `@pointerenter` 绑在父容器时，向子元素派发 pointerenter 不会触发父级 handler（mouseover/mouseout 才冒泡）。悬停类交互（如 Snap 布局面板的 400ms 悬停触发）自动化必须把事件派发到**绑定元素本身**。
- **另**：跨 MCP cell 验收时会撞上 dev 环境瞬态（HMR/预打包）导致「上一 cell 还在、下一 cell 元素消失」的假象——把完整验收链合并进**单个 evaluate cell**（内含 wait）是最稳的自动化形态。
- **来源**：2026-10-08，M29 第六轮 Snap 面板验收（面板不弹两次误判为功能缺陷）。

### 会话自毁竞态：store 创建时的初始 sync 会把空窗口列表固化为持久化会话

- **现象**：窗口会话「有时恢复有时不恢复」；某次 dev 热更后恢复必失效。
- **根因**：compat useWindowsStore 工厂末尾无条件 `sync()` → `persistSession()` 把**空窗口列表**写入持久化（dirty 防抖），与 provider/desktop 的会话恢复流程赛跑；恢复前任何一次页面生命周期扰动都会让空列表抢先落盘。
- **规避/解决**：初始同步跳过持久化（`sync(persist = true)`，创建时 `sync(false)`）；同时注意 `bus.on(ev, sync)` 直接传函数会把事件 payload 当成 sync 的首参（加参后类型炸/语义错），须包箭头函数。
- **来源**：2026-10-08，M29 第六轮 Snap 验收期会话恢复间歇失效排查。

### 合成 HTML5 DnD 验收：DataTransfer 可构造、drop 可直接派发（dragstart 不可）

- **可行**：`const dt = new DataTransfer(); dt.setData(type, val); target.dispatchEvent(new DragEvent('drop', { bubbles: true, dataTransfer: dt }))`——接收端 `getData` 拿到的就是 set 的值，拖放**接收逻辑**可完整自动化验收；`dragover` 一并派发（有 .prevent 的悬停态才会激活）。
- **不可行**：源端 `dragstart` 无法启动真实拖拽会话——拖拽**发起侧**（dragstart 组装数据）只能代码走查或手动验证。
- **另**：向 xterm/vwt 终端「执行命令」不必派发键盘——终端对外暴露的数据通道（sendRaw/写入 API）原样发 `命令\r` 即执行（键盘合成对 vwt 不生效）；`\u0003`（ETX）即 Ctrl+C 清缓冲。
- **来源**：2026-10-08，M29 第七轮跨窗拖文件验收。

### Vue 组件 ref 链：中间组件模板绑了 ref 才能透传 expose 方法（忘绑=静默 no-op）

- **现象**：父组件调 `childRef.value?.method()` 静默不执行（`?.` 吞掉 undefined），无任何报错。
- **根因**：孙组件 defineExpose 了方法、子组件模板也用了孙组件，但**子组件模板忘了给孙组件绑 `ref="xxxRef"`**——子组件 script 里的 ref 永远 undefined。
- **规避/解决**：expose 链的每一跳都要「script 声明 ref + 模板绑定」成对出现；`?.` 调用点加临时日志或断言非空排查。
- **来源**：2026-10-08，M29 第七轮终端 pasteText 链（Workspace→Panel 漏绑 ref）。

### computed 包普通类字段 = 永不更新的假响应式（webos editing 状态）

- **现象**：webos 小组件「编辑模式」永远进不去（抖动/按钮不出现），侧栏与桌面形态同时失效。
- **根因**：`editing: computed(() => widgets.editing)`——WidgetStore 是普通类、`editing` 是普通字段，computed **没有任何响应式依赖**，求值一次后永不重算。同族：`instances` 之所以工作，是因为 onChange 里手动同步 ref——editing 漏了同样的处理。
- **规避/解决**：普通类字段暴露给 Vue 的统一模式——ref 初始化 + 类 onChange 回调里手动同步（instances/editing 一致处理）。凡是用 `computed` 包非响应式对象字段都应视为红旗。
- **来源**：2026-10-08，M29 第八轮小组件桌面平铺（编辑模式三连失效定位）。

### 拖拽系统的 stopPropagation 吞掉父级语义：点击标题栏/把手永不触发窗口聚焦

- **现象**：点击底层窗口（标题栏、边缘把手、被内容组件拦截指针的区域）窗口不置顶不聚焦；直接派发 pointerdown 到标题栏复现——事件在 titlebar 层后消失，section 的监听收不到。
- **根因**：usePointerDrag.start() 里 `ev.preventDefault(); ev.stopPropagation()`——标题栏 beginDrag/把手 beginResize 启动拖拽时阻断冒泡，而窗口聚焦监听（@pointerdown="onFocus")挂在 section 冒泡阶段，永远收不到被阻断的事件。
- **规避/解决**：父级语义监听改 **capture 阶段**（`@pointerdown.capture`）——capture 自外向内先于 target/冒泡，拖拽阻断不再影响；勿全局去掉拖拽系统的 stopPropagation（其它消费方依赖其隔离）。同类排查：capture/bubble 各挂一个计数监听即可定位事件死于哪一层。
- **来源**：2026-10-08，M29 第九轮「点击底层窗口不聚焦」排查（逐层计数监听一次定位）。

### 控制中心去伪存真：Web 面板别抄 OS 的演示件

- **教训**：webos 控制中心初版照搬 macOS 放了 Wi-Fi/蓝牙/亮度/音量——全是改本地 ref 的假开关，对服务器面板是负资产（用户一眼识破「又不是真的 os」）。控制面板类产品的每个可见控件都应有真实后端/系统效果，纯演示件宁可不做。
- **处置**：删除四件伪控件；保留真实生效的深色模式/壁纸，并补了真实生效的强调色选择（setAccent 全主题跟随 + 持久化）替代空缺。
- **来源**：2026-10-08，M29 第十轮控制中心重写。

### multiInstance 窗口标题带 #N：自动化按 title 匹配要用前缀

- **现象**：`querySelector` 定位「标题 === 文件管理」的窗口找不到——实际标题是「文件管理 #2」（multiInstance 累计编号）。
- **规避/解决**：多实例窗口断言/定位一律 `startsWith(app 名)`；同一应用的多个窗内容可能不同（各自 cwd/会话），取「最后一个打开的」用数组末位而非第一个匹配。
- **来源**：2026-10-08，M29 第十一轮 Dock 拖放验收。

### 终端主题读 fa settings store：webos 切深色后终端白底

- **现象**：桌面工作台（webos）深色模式下，终端窗内容区白底浅字。
- **根因**：YdTerminal 的 theme 读 **fa settings store 的 colorScheme**——webos 深色切换改的是 html.dark（webos 设置驱动），fa store 并未跟随，两套主题状态脱钩 → xterm 一直用浅色主题。
- **规避/解决**：跨主题体系的组件主题口径统一为「实际生效的 html.dark」：MutationObserver 监听 documentElement class 变化（attributeFilter: ['class']）驱动 isDark ref。谁切暗色都触发，经典/桌面两模式通用。
- **来源**：2026-10-08，M29 第十三轮终端深色白底修复。

### DesktopModel 整理/排序挤成一列：reassignCells 未尊重 rowsPerColumn

- **现象**：桌面图标「整理图标/按名称排序」后全部挤成第 0 列一根竖条。
- **根因**：reassignCells 只递增 row、col 恒 0——firstFreeCell 有列满换列逻辑而 reassignCells 没有（两处语义漂移）。
- **规避/解决**：整理/排序的单元格重排同样按 rowsPerColumn 换列；补三例单测（满列换列/rowsPerColumn=0 向后兼容/sortBy 同规则）。
- **来源**：2026-10-08，M29 第十三轮桌面图标一列修复。

### 自绘右键菜单不做视口钳制：屏幕底部/右缘的菜单被裁剪

- **现象**：右键 Dock 底部图标，菜单向下展开溢出视口，「关闭窗口」等尾部项被遮住不可点。
- **根因**：菜单定位裸 `left=x, top=y`，无视口钳制/翻转逻辑。
- **规避/解决**：渲染后同步测量菜单 rect（`host.firstElementChild.getBoundingClientRect()`，同步 render 后即可测量），x/y 各自钳制到 `[8, innerWidth - w - 8]` / `[8, innerHeight - h - 8]`（贴底向上翻转）。初始 `visibility: hidden` 渲染避免闪跳。webos 已在 `adapter/builtin.ts menu()` 统一修复——自绘菜单/弹层都应有这道钳制。
- **来源**：2026-10-08，M29 第十四轮 Dock 右键菜单遮挡修复。

### 页头隐藏一刀切会把页头里的操作按钮一起藏掉

- **现象**：webos 窗口内隐藏 FaPageHeader 后，文件管理等页的批量操作（删除/上传/新建目录/搜索）随之消失——这些按钮在页头的 default slot 里。
- **根因**：「藏页头」按整个组件 display:none 一刀切；FaPageHeader 的标题/描述（main 区）与操作按钮（default 区）是两块兄弟 DOM，藏组件即全藏。
- **规避/解决**：FaPageHeader 两块加稳定类（yp-page-header-main / yp-page-header-actions）；嵌入态 CSS 只藏 main 块并把组件压成一条紧凑工具条（padding/margin 压缩、去底边框），操作按钮保留。同时给 FaPageMain 加 yp-page-main 类，嵌入态把 m-4 外边距压到 8px。通用原则：**选择性隐藏要按「功能块」加钩子类，不能整组件一刀切**。
- **来源**：2026-10-08，M29 第十六轮窗口内边距/批量操作回归修复。

### 原生 <select> 的展开层是浏览器 UI：深色窗口里出现亮白系统下拉且位置脱管

- **现象**：webos 深色窗口里点工具栏的节点下拉（原生 `<select>` 收起态被 utility 美化成深色），展开的选项列表却是**亮白底蓝高亮的系统样式**且位置由浏览器决定（Windows 下偏移脱管）——收起态美化骗人，展开态穿帮。
- **根因**：原生 select 的弹出层是浏览器/OS 渲染，无法 CSS 主题化也无法控制定位；Windows Chrome 的弹层样式跟随机制在深色页面上也不稳定。
- **规避/解决**：**工具栏常驻的下拉一律用 FaDropdown**（reka 系、主题跟随、贴合触发器）；表单弹窗内的原生 select 暂可保留（弹窗上下文中观感冲突小），后续统一封装 yd-select。项目内 FaDropdown items 形状是**分组数组** `[ [{label, handle, disabled}...] ]`，触发器走 default slot。
- **来源**：2026-10-08，M29 第十七轮节点下拉换装。

### flex 链断在「组件内部无主类的内层 div」：mainClass prop 是唯一注入点

- **现象**：AI 对话窗外层全给了 flex-1/min-h-0，发送框仍悬在窗口中部——逐层量高发现断点在 FaPageMain **模板内部**的包装 div（display: block、高度由内容撑），这层没有任何业务类可挂。
- **根因**：flex 高度链要求每一层都传递；包装组件的内层结构不在业务方控制范围。
- **规避/解决**：包装组件若暴露 `mainClass`/`contentClass` 之类的内层类透传 prop，嵌入场景用它注入 `min-h-0 flex-1`（tailwind-merge 会合并掉内层原 p-4/p-0 冲突）；没有透传 prop 的组件只能改组件或整体绕过。排查手法：从窗口 body 沿 firstElementChild 逐层量 offsetHeight + display/flex，断在哪层一目了然。
- **来源**：2026-10-08，M29 第十九轮智能窗发送框贴底修复（190→87→贴底两步）。

### 原生 <select> 弹层不可主题化：封装 YdSelect 分批替换

- **方案沉淀**（延续前条）：YdSelect（components/YdSelect）= FaDropdown 包一层——props `options`（string/number 或 {label,value,disabled}）、v-model 透传原始值类型、size 枚举对齐 FaButton 变体（'sm'|'default'|'icon-sm'，别造 'xs'/'md'——FaButton 不认会 TS2322）、buttonClass 透传宽度。unplugin 自动注册。
- **替换分批策略**：先换「桌面窗口直接暴露」的页面级工具栏（商店筛选×2/容器列表过滤/日志行数/级别过滤/AI 供应商+模型等 ≈10 处）；**表单弹窗内**的几十处原生 select 留待第二批（弹窗上下文观感冲突小、v-model.number/联动复杂）。
- **来源**：2026-10-08，M29 第二十轮原生 select 严查第一批。

### 最大化/全屏窗口时 Dock 自动避让（macOS 语义）+ 状态类判定要用 store 而非 DOM

- **交付**：webos Dock 在存在 maximized/fullscreen 窗口时自动滑出避让（按停靠方向 translateY/X 110% + opacity 0 + pointer-events none，transition 平滑），全部还原后回归；样式与拖拽数学互不影响。
- **排查坑**：还原后 Dock 仍未回归的假象——判据读的是 store.windows（含 minimized 等不渲染 DOM 的窗），而 DOM 查询只能看到可见窗，两边的「maximized 计数」会不一致；判定一律基于 store 状态，验证时先 dump store 再下结论。
- **来源**：2026-10-08，M29 第二十一轮 Dock 避让。

### webos 窗口内业务弹窗约束到所属窗口：FaModal portalTo + 宿主 provide 注入

- **方案沉淀**：FaModal 的 reka DialogPortal 默认挂 body——webos 窗口内业务弹窗浮在整个桌面且遮罩盖全部窗口（违反窗口模态语义）。修复三层：① DialogContent/DialogScrollContent props 加 `portalTo`，`<DialogPortal :to="portalTo">`；② FaModal `inject('fa:modal-container', undefined)` 解析挂载目标（字符串选择器或元素，缺省 body）；③ webos 桌面壳 `provide('fa:modal-container', computed(() => 聚焦窗 body 元素))`（YwWindow section 加 data-window-id 供定位）。全部业务弹窗自动进所属窗口（遮罩只盖本窗），经典模式无 provide 走 body 零影响。
- **注意**：DialogPortal `:to` 传 undefined 回落 body ✓；组件内 inject 的 key 用字符串常量即可（fa 包无需导出 key 对象）。
- **来源**：2026-10-08，M29 第二十三轮弹窗作用域改造。

### FaModal 的受控 prop 是 modelValue 不是 open：绑 open 静默失败（DOM 挂载、display:none、零报错）

- **现象**：`<FaModal v-model:open="x">` 的弹窗点击后"毫无反应"——无 JS 错误、弹窗 DOM 在 body 里已挂载（含内容文本），但内容容器 `display:none`，用户看不到。6 个弹窗（编辑/索引/审计等）全部中招且**从未真正显示过**，极易误判为"按钮没绑上事件"或环境问题。
- **根因**：fa 对 reka Dialog 的封装把受控 prop 归一为 **modelValue**（`v-model`）；`open` 不是它的 prop，绑定后落到根元素成无效 attr，DialogRoot 恒为关闭态 → content 挂载但隐藏。exp 既有条目「reka 系受控 prop 名不统一（Switch=modelValue、Dialog=open）」说的是 reka 原生，**fa 封装层已统一为 modelValue**，不能按 reka 原生记忆写。
- **规避/解决**：一律 `<FaModal v-model="x" :destroy-on-close="true">`（项目内正常弹窗全部此形态，database/index.vue 可证）；`destroy-on-close` 顺带解决「FaModal 开启动画期间挂载 monaco → 0 尺寸」的挂载时机问题。排查特征：`getBoundingClientRect` 全 0 / computed display=none 且无报错，先查受控 prop 名。
- **变体（受控回环）**：组件内 `<FaModal :model-value="visible" @update:model-value="close">` 且 close 无条件上抛 false——FaModal 打开时 watch(isOpen) 会**回发 update:modelValue=true**，无条件 close 形成「开→回发→关」死循环，表现为点击触发按钮**毫无反应**（弹窗瞬时开关）。修复：update handler 如实上抛新值（`emit('update:visible', v)`），由父层 v-model 收敛，不做无条件 close。YdDangerDelete 组件曾因此导致 6 个页面删除/解除弹窗全部打不开。
- **来源**：2026-10-08，M30 DB Admin 编辑/审计等弹窗全部静默不显示（对照 database 页正常弹窗定位）；同日 M31 用户反馈「解除点击无效果」确认为受控回环（YdDangerDelete）。

### YdLogViewer 时间解析正则需兼容时区偏移与斜杠日期，否则整行落入 continuation（时间丢失、级别继承）

- **现象**：EasyTier 容器日志 `2026-10-08T01:46:54.760893296+08:00 INFO CORE: …` 与 nginx `2026/10/07 17:01:15 [notice] 1#1: …` 的时间没有提取到最左栏、级别列空白（级别全灰继承）。
- **根因**：行解析正则的时间段只认 `Z` 结尾与连字符日期，不认 **时区偏移（+08:00）** 与 **斜杠日期（2026/10/07）**；不匹配则整行走 continuation 分支（level 继承上一行、time 为空）。
- **规避/解决**：时间正则统一为 `\d{4}[-/]\d{2}[-/]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?`；新增「仅时间戳」分支（时间 + 无级别 token 的行：time 提取、level 继承、content 为剩余）。新增日志源时先在 node 里对真实样本跑一遍等价正则再上线。
- **来源**：2026-10-08，M23 组网 EasyTier 日志解析修复（node 等价自测全部样本通过后部署）。

### rolldown-vite 构建下 useTemplateRef 可能失效：模板 ref 恒为 null，echarts 等图表静默空白

- **现象**：图表组件 `useTemplateRef('chart')` + 模板 `ref="chart"`，运行时 ref 值永远 null（诊断行证实），echarts 拿不到容器、canvas 数为 0；数据/采样一切正常，无任何报错。数据卡与图表在同一个组件内，采样正常、图表空白是识别特征。
- **根因**：rolldown-vite 构建产物中 useTemplateRef 宏的 ref 收集与模板关联失效（特定嵌套/条件渲染场景，机理未深究）。经典写法（`const el = ref(null)` + 模板 `ref="el"` 同名自动绑定）不受影响。
- **规避/解决**：图表类组件一律用**同名 ref() 变量**绑定模板 ref；且注意模板 ref 属性名必须与变量名完全一致（名字对不上同样恒 null 且无报错）。排查手段：组件内加临时诊断行输出 ref 状态，一次部署即可定位。
- **来源**：2026-10-08，M23 容器统计图表（诊断行 init=5/ok0 cpuRefAtInit=null 定位）。

### 内联事件调用工厂函数：返回的 handler 被丢弃，事件从未绑定

- **现象**：webos 窗口边缘把手拖拽调整大小**从未工作过**——拖把手直接触发浏览器原生拖选（全选文字），窗口尺寸不变；且因 start() 的 preventDefault 未执行，表现像「事件没绑」。
- **根因**：模板写成 `@pointerdown="beginResize('se')"`——beginResize 是**工厂函数**（返回真正 handler），内联语句执行工厂后**返回值被丢弃**，start(ev) 永远不执行。同时工厂内本应做的 preventDefault 也没跑 → 浏览器原生拖选接管。
- **规避/解决**：工厂改直执签名 `beginResize(direction, ev)`，模板传 `$event`。**通用红旗**：`@事件="fn('参数')"` 里 fn 若是「返回 handler 的工厂」就是此坑；正常内联要么调用无返回值函数（参数含 $event），要么 `@事件="handlerRef"` 直接引用。顺带：`functionResize` 类内部状态（如 min 尺寸钳制）在 handler 从未执行时也全部静默失效。
- **来源**：2026-10-08，M29 第二十二轮 resize 修复（此前多轮把 resize 失效误归因于把手遮挡/选区，真因是 handler 未绑定）。

### 全屏避让 Dock 的贴边呼出：mousemove 热区触发 + 离区余量防误缩

- **方案沉淀**（延续 Dock 避让）：避让（is-suppressed）期间监听 window mousemove——鼠标压到 Dock 所在侧边缘（8px 热区）→ 解除抑制滑入（peek）；鼠标移到 Dock 区域外（上方 24px 余量）→ 恢复抑制。**余量判断必须保留**：仅「不在热区就缩回」会让鼠标在 Dock 上短暂停留时中途缩回。
- **注意**：suppress 类挂载条件改为 `suppressed && !peek`；dockEl ref 供余量计算取 Dock 实际位置。
- **来源**：2026-10-08，M29 第二十六轮全屏 Dock 贴边呼出。

### 桌面级命名弹窗 z-index 低于窗口层：被窗口盖住导致「功能失效」假象

- **现象**：桌面右键「新建文件夹/文本文件」后命名框看不见（或部分被盖），确认后桌面无变化——误判为创建逻辑失效。
- **根因**：命名弹窗 `.yw-desktop-dialog` z-index 100，而窗口层 zIndex 125+（动态递增）——弹窗被窗口盖住，用户无法完成确认。
- **规避/解决**：桌面级浮层（命名框/确认框）z-index 提到与菜单同档（9400）；「右键功能失效」类反馈先检查**中间浮层**（确认框/对话框）是否被更高层盖住，别只盯功能本身。
- **来源**：2026-10-08，M29 第二十七轮桌面新建修复。

### 右键菜单引用未注册应用：openApp 静默失败

- **现象**：桌面右键「系统设置…」点击无任何反应（无窗、无提示）。
- **根因**：菜单项 `os.openApp('settings')`，而宿主未注册 settings 应用——openApp 对不存在应用仅 ui.message 提示（且消息位置不显眼时等于无反馈）。
- **规避/解决**：宿主注册对应应用（复用 webos arco 内置 YwSettingsApp——壁纸/强调色/深浅/关于四节原生设置，markRaw 注册即可）；或菜单项按 registry.get 存在性条件渲染。**右键菜单引用的 appId 必须在宿主注册清单里核对一遍**。
- **来源**：2026-10-08，M29 第二十七轮 settings 应用注册。

### webos 深色切换不生效：settings setter 不自应用，依赖外部 watch 兜底

- **现象**：webos 控制中心切深色，壁纸/桌面层变了但 ypanel 页面（fa token 层）保持浅色——html.dark 未切换。
- **根因**：useSystemSettings 的 setMode/setAccent/setWallpaper 只写状态+持久化+发事件，**不调 applyNow**（html.dark/token 切换）——依赖外部 themeStore（compat）创建 watch 兜底；宿主（ypanel 壳）没创建该 watch → 切换断链。
- **规避/解决**：webos setters 自身调 applyNow（设置变更即应用是 settings 系统的自身职责，不外包给消费者）；顺带把 fa 层组件的主题判定源统一抽 `useHtmlDark`（composables/useHtmlDark.ts，MutationObserver 监听 html class）——YdCodeEditor/YdTerminal/TerminalPanel 全改判定源，经典/桌面两模式通用。
- **来源**：2026-10-08，M29 第二十八轮深色即时生效修复。

### 控制中心深色开关重做：tile 点击切换改行式滑动开关（macOS 形态）

- **方案沉淀**：深色模式从「整个 tile 点击切换」（点击区域大但无开关形态、ON 态无视觉差异）改为**行式滑动开关**——圆底图标（ON 时主色底白图标）+ 名称/状态两行 + 右侧 40×22 滑槽开关（knob 18px，spring 位移，ON 态主色底）。
- **细节**：生效态用 `isDarkEff`（system 模式按系统暗色折算，而非直接比 settings.mode）；knob 位移用 transform + spring ease。
- **来源**：2026-10-08，M29 第三十轮控制中心开关重做（用户反馈「开关有点丑」）。

### 脚本批量替换模板的组合替换要逐处独立锚定（一次失配全批不落盘）

- **现象**：python 脚本对同一文件做多处模板替换，第一处替换成功但后续 assert 失败——**整个文件写回被跳过**，所有替换都没落盘；导致页面编译报 Invalid end tag（双闭合标签），排查绕远。
- **规避/解决**：多处替换逐处独立锚定（每处自己的 assert 与替换，最后一次性 write）；或分多次小脚本执行。替换后必须立刻 vue-tsc/构建验证落盘结果。
- **来源**：2026-10-08，M29 第三十轮 app-detail 双闭合标签（替换叠加）排查修复。

### 后台 IAB 里 Playwright 高层 locator（click/fill）会卡 actionability 超时：evaluate 取坐标 + CUA 点击是稳定兜底

- **补充**（遮挡 IAB 交互自动化边界条目的重要补充）：后台/遮挡窗口下不仅 rAF 冻结影响路由切换，**Playwright 高层 locator 的 click/waitFor 也可能卡在 actionability 检查上超时**——即使元素 `getBoundingClientRect` 可见且有尺寸（text/role/title 选择器全试遍均超时，count() 都没机会执行）。
- **规避/解决**：`playwright.evaluate` 里 `querySelector` + `getBoundingClientRect` 拿中心坐标（同步返回、不受 actionability 影响）→ `tab.cua.click({x,y})` 真实点击；表单填写用「原生 value setter + 派发 input 事件」驱动 v-model（不经过 locator fill）。该路径在后台窗口稳定可复现。
- **再补充**（2026-10-09，M53 SSH 页开关验收）：`tab.cua.click` 在遮挡窗口下**并非总有效**——本次对 reka Switch 坐标点击后事件状态无任何变化（API 侧确认请求未发出）。最稳兜底是 evaluate 内**派发完整指针序列** `pointerdown/mousedown/pointerup/mouseup/click`（PointerEvent+MouseEvent、bubbles、button:0），switch/reka 类组件一次通过；点击后用 aria-checked 等属性断言，API 侧对账确认请求真实发出。
- **来源**：2026-10-08，AI 工作空间页面浏览器验收（text/role/title 三种 locator 全超时，CUA 兜底一次通过）。

### IAB 上传验收：DataTransfer 构造 File 塞 input.files + 派发 change 走真实上传链路

- **可行**（file input 姊妹条，同 HTML5 DnD 条目）：IAB 明确不支持 filechooser（`waitForEvent("filechooser")` 报 capability_unsupported），但页内 `evaluate` 里 `const dt = new DataTransfer(); dt.items.add(new File([content], "name.txt")); input.files = dt.files; input.dispatchEvent(new Event("change", { bubbles: true }))` 会完整触发页面 `@change` 处理器 → 真实调用上传 API → toast + 列表刷新，端到端可断言（服务端 `ls` 对账文件与字节数）。
- **不可行**：绕过页面处理器的「直接调上传接口」只能证明 API 通，证明不了 UI 链路；本条方案两者兼得。
- **来源**：2026-10-08，AI 工作空间上传按钮验收（uploaded.txt 19 B 服务端对账一致）。

### FaInput 根元素硬编码 w-[200px]：窄容器里必须显式覆盖宽度

- **现象**：字段侧栏里的 FaInput 溢出容器约 12px（输入框右缘伸出侧栏边框）。
- **根因**：fa 的 FaInput 根 InputGroup 写死默认宽度 `w-[200px]`（cn 合并默认类），不放宽度类就是 200px；容器内容宽 < 200px 时必溢出。页面里其他 FaInput 因都显式给了 `w-40!`/`w-72!` 而从未显形。
- **规避/解决**：窄容器中使用必须带 `w-full!`（tailwind-merge 会压掉默认 w-[200px]，`!` 防内层优先级问题）；flex 场景再加 `min-w-0!`。新容器放 FaInput 先想宽度。
- **来源**：2026-10-08，M33 日志中心字段侧栏（几何断言 aside 右缘 vs 输入框右缘验证）。


### 全站 i18n 键化的四件套模式（B26-full 落地沉淀）

- **模式**：① 路由 `meta.title` 存 i18n key（`menu.*`），`generateTitle` 统一 `te()→t()`、无词条原样透传（中文标题/动态函数双兼容）；因菜单/面包屑/页签/document.title 全部经它渲染，语言切换**自动重渲染，无需任何 watch**（渲染读 locale ref 自带响应式）。② 设置类标题同法键化（`settings.ts` 的 `app.home.title: 'menu.overview'`——面包屑里藏的「主机概览」就是它）。③ 动态枚举文案用 `tr(\`域.key.${v}\`, v)`（不存在回落原值，后端新枚举不裸 key）；非组件模块（store/composables/ts 元数据表）统一 `import { i18n, tr } from '@/locales'` + **函数内求值**（模块顶层求值会固化语言）；元数据表的中文 label 改 getter 即时求值，消费方零改动。④ vue-i18n 消息里 `{ } @ |` 是语法字符，含这些字符的文案要转义（`@`→`{'@'}`）或改用命名参数传值；LogsQL/JSON 示例含裸 `{}` 的干脆留在代码里。
- **校验**：语言包校验脚本用 esbuild transformSync 加载 TS 词条做 zh/en 键集比对与引用完整性扫描——**注意词条对象顶层没有域名层**（域名在文件名），扫描器必须给键补 `域名.` 前缀，否则 4000+ 假 MISSING / 结构比对假阴性。
- **验收手法**：浏览器 `localStorage.setItem('ypanel.locale','en-US') + reload` 直接进 EN 态；单 cell 批量 `location.hash` 导航 + body.innerText 断言（IAB evaluate 上限 ~32s，一批 ≤14 页）；剩余中文区分「数据」（供应商名/站点名/探针名等后端内容）与「UI」，只有后者是缺陷。
- **来源**：2026-10-08，B26-full 全站双语（14 个迁移批次、34 域 3555 对键、vue-tsc/生产构建/双语言走查全绿）。

### 子代理限速中断的续传纪律：交接状态必须以文件实况为准

- **现象**：并行 13 个迁移子代理撞账号限速（1302）批量失败；失败代理自称「已改 N 个文件」，但续传代理实测**多数改动并未落盘**（git M 状态实为并行会话的功能改动，非代理的 i18n 半成品）；而个别「已写入」的语言包又真实存在。交接描述与磁盘实况错位率高。
- **根因**：代理被限速杀死时，编辑批次可能停在任意中间态；且同一工作区里其他并行会话的改动会让 git status 无法区分「谁改的」。
- **规避/解决**：续传前先跑机械盘点（grep 引用键数 vs 词条文件行数、抽查 `$t(` 落盘情况），把「补齐既有引用键」作为续传代理的第一优先级任务写进指令；并发控制在 5-6 个以内（13 并发必撞限速）；失败域的 vue 文件一律当「可能半迁移」处理。
- **来源**：2026-10-08，B26-full 迁移期间 6 个代理 1302 失败后的续传实践。

### getCurrentInstance 塞进 computed 定位宿主窗口：点击期求值恒为 null（webos 详情页返回逃逸）

- **现象**：桌面工作台里节点/容器详情窗点「返回」没关自己窗，而是整个桌面被顶掉（hash 从 `#/desktop` 变 `#/nodes`）；应用详情的「关闭（桌面承载）」点击后**静默无效**（窗还在）。该模式 2026-10-08 验收时是通过的，回归排查发现是稳定复现的确定性 bug。
- **根因**：`const selfWinId = computed(() => closestWindowId(getCurrentInstance()?.proxy?.$el))` —— Vue 的 `getCurrentInstance = () => currentInstance || currentRenderingInstance`，只在 setup 执行期与渲染执行期非空。computed 是**惰性**的：模板没引用它时，首次求值发生在用户点击回调里，此刻两个全局都为 null → `closestWindowId(undefined)` 返回 null → embed 分支被跳过，`router.push` 逃逸/关闭失效。此前验收通过纯靠偶然（当日 HMR 热替换后的重渲染恰好在渲染期求值过一次，缓存住了正确值）。
- **规避/解决**：定位「自己所在的 webos 窗口」一律用**根元素 template ref**（`<div ref="rootRef">` + 经典 `ref()`，勿用 useTemplateRef——生产环境另有失效坑，见本文件前文）；`computed(() => closestWindowId(rootRef.value))` 挂载后即可确定求值。容器详情/应用详情/站点详情/节点详情四处已统一改掉。
- **附带**：webos 嵌入态的「返回」处理器要显式分叉（embed=关自己窗），站点详情此前漏接、返回直接 `router.push('/sites')` 也会顶掉桌面——新详情页接入桌面时，返回/关闭是和下钻同级的必改点。
- **来源**：2026-10-09，桌面工作台全量纳管轮（节点详情下钻回归，IAB evaluate 探针三段定位）。

### 会话式反代的绝对路径资源逃逸：iframe 内目标应用的 /assets 落在代理前缀之外

- **现象**：内网浏览器（gw 网关 `/s/{sid}/*` 会话式反代 + iframe）里访问 vmui 等相对路径应用一切正常；访问面板自身（fa 基座 `/assets/*.js` 绝对路径）时 JS 全部被拦：「不允许的 MIME 类型（text/html）」——资产请求打到了 `8881/assets/...`（代理端口的根），落在会话前缀外，网关回了 HTML 兜底页。
- **根因**：iframe 文档在 `/s/{sid}/` 下，但**绝对路径**（`/assets`）按域名根解析，不随文档路径；HTML 注入 `<base>` 也救不了（只影响相对 URL）。
- **规避/解决（根路径回捞）**：网关对非 `/s/` 前缀请求，从 `Referer` 中解析 `/s/{sid}` 找回会话继续代理到原目标。三个配套缺一不可：① iframe `referrerpolicy="no-referrer-when-downgrade"`（默认 strict-origin-when-cross-origin 跨源只发 origin，Referer 里没有 sid；no-referrer 则完全没有）；② 网关 Cookie 不能用 `Path=/s`（根路径请求带不上），改随机不透明令牌 + `Path=/`（别把 JWT 放进去——同 IP 其它服务会收到该 Cookie）；③ 同名 Cookie 新旧共存（Path 不同）时 Go `r.Cookie()` 只取第一条（浏览器按路径长度排序），要用 `r.Cookies()` 遍历任一有效。无 Referer 的裸根访问仍走 404 兜底页。
- **另**：webos 快速启动/启动台搜索按应用名与 keywords 匹配——keywords 只写拼音/英文时，EN locale 下应用名变英文、中文搜索词不中；keywords 字面量同时给拼音+英文+中文（`'neiwang browser 内网 浏览器'`）任一 locale 全可达。
- **来源**：2026-10-09，M51 内网浏览器（用户实测面板自身场景暴露，1105-m42 修复）。

### 会话式反代浏览「面板自身」的两个专属缺口：安全入口 404 与动态 import 的 Referer 竞态

- **现象**：内网浏览器（gw 会话式反代）访问一般内网服务都正常，唯独访问面板自身时：① iframe 里登录必失败（login 404）；② 偶发启动卡「载入中」（部分 chunk 404，手动重发同 URL 却 200）。
- **根因**：① 面板安全入口（SecurityGate）要求 login 请求带 `?entry=`/X-Safe-Entry，经网关代理的 iframe 登录路径上没有；② 根逃逸回捞依赖 Referer 携带 /s/{sid}，但启动期部分动态 import 请求的 Referer 不在（竞态/策略差异），回捞失败落 404 兜底页。
- **规避/解决**：① 网关识别「目标是本机面板」（端口==面板端口且主机为 loopback/本地网卡地址）时自动注入 `X-Safe-Entry` 头（值取 SecuritySettingsService.SafeEntry()；不对其它目标注入，避免入口值泄漏给局域网服务）；② 网关记录「令牌→最近使用会话」（/s/{sid} 命中时 TouchSession），根逃逸无 Referer 时用最近会话兜底——令牌已过门禁，语义安全。
- **来源**：2026-10-09，M51 内网浏览器（用户要求实测「经网关登录面板及其功能」暴露，1238-m42 修复）。

### vue-i18n 词条含裸 `@`：弹窗渲染中断、遮罩残留挡住整页点击（错误被全局 errorHandler 吞掉）

- **现象**：SSH 管理页「生成密钥」弹窗点不开（DOM 从未挂载、无网络请求），且**点过一次后全页点击失效**（开关/按钮全部无响应）——用户感知是「开关点不动」；JS 控制台干净，无任何报错。
- **根因**：三层叠加。①词条 `keyCommentPlaceholder: '可选，如 user@host'` 里的**裸 `@` 是 vue-i18n linked-message 语法字符**（`@:key` / `@.modifier`），message compiler lexer 抛 `SyntaxError: 10`（UNEXPECTED_LEXICAL_ANALYSIS 家族）；②该编译发生在**组件渲染期**（运行时消息编译），异常被 fa 的 `app.config.errorHandler` 吞掉 → **整个弹窗子树渲染中断**，弹窗框架 DOM 永远不出现；③fa DialogContent 的遮罩（overlay）与内容**分别渲染**——遮罩已挂到 body（`fixed inset-0 pointer-events-auto`、z-2000），内容失败后无人关闭它 → **透明全屏层挡住全部点击**。用户侧表现为「A 功能坏了，B/C 也跟着全坏」，实际只有一个根因。
- **定位手法**：`window.addEventListener('unhandledrejection')` 抓不到（异常走 errorHandler）；**替换 `document.querySelector('#app').__vue_app__.config.errorHandler`** 记录 `(err, inst, info)` 后重现，堆栈直指 vue-i18n `nextToken/parse`。另注意 Vue 同步渲染错误也不会进 unhandledrejection。
- **规避/解决**：词条文本含 `@` `|` `{`/`}` 必须转义——`@` 用 literal 插值 `{'@'}`（TS 字符串用双引号包裹：`"user{'@'}host"`，单引号串会撞字符串边界）；`|` 只在复数语境用，文案中避免；`{}` 保留给合法 `{name}` 插值。**新增词条入包前扫一遍 `@|{}`**（插值以外）。连带修复：fa Modal 的 `setTransform`/`handleOpenAutoFocus` 对 `dialogContentRef.value?.el?.$el` 加真实元素防护（占位注释节点无 style/focus，此前抛错同样会打断挂载流程）。
- **来源**：2026-10-09，M53 SSH 页生成密钥弹窗（「弹窗不弹 + 开关点不动」双表象一个根因；替换 errorHandler 一次定位）。

### fa 路由守卫不做 meta.auth 硬拦截：权限硬边界必须在后端，meta.auth 只管菜单观感

- **现象**：直觉认为 fa 的 `meta.auth` 会拦住直连 URL 的越权访问；实测守卫（guards.ts）只做菜单过滤（menu.ts filterAsyncMenus 递归按 auth 过滤、空组自动隐藏）与「父级无 redirect 时跳第一个有权限子路由」，导航本身不校验 `to.meta.auth`——未授权用户手输 URL 仍能渲染页面，只是页面里的 API 全部 403。
- **规避/解决**：权限模型设计时明确「meta.auth = UI 过滤，后端中间件 = 安全边界」，二者缺一不可但不可互相当作；验收越权用 curl 直调 API 断言 403，不要用页面可达性断言。fa 的 `auth()` 是 permissions 数组 some 交集（composables/app/auth.ts），`v-auth` 指令是无权限时 display:none——按钮级藏按钮够用，但同样不是边界。**另：`hasPermission` 是精确 `includes` 匹配、不认通配——后端权限集含 `*`/`模块:*` 时必须在下发前展开成具体权限点列表（rbac.Expand），否则超管所有带 auth 的菜单整组消失（「admin 看不到系统设置」即此症）**。
- **来源**：2026-10-09，M54 RBAC P1 改造（admin/user 二值角色 → 角色权限点；fa 侧 permissions 从 ['admin']/['user'] 换成真实权限点列表）。
