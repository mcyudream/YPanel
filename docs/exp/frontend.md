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
