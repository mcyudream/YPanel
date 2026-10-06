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
