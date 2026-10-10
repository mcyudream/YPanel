# docs/exp/ — 浏览器自动化与无头渲染（Edge headless / CDP / 面板 UI 自动化）

### Edge headless `--screenshot` 相对路径写入被拒（拒绝访问 0x5）

- **现象**：`msedge.exe --headless=new --screenshot="cover.png" file:///...` 报 `Failed to write file cover-v1.png: 拒绝访问`，Exit code 2。
- **根因**：headless command handler 对相对路径的解析/写入目录与 shell cwd 不一致。
- **规避/解决**：`--screenshot` 一律用 Windows 绝对路径；输入用 `file:///D:/...` URL。
- **来源**：2026-10-11 推广短视频封面渲染（docs/plan/20261011_015147_YPanel推广短视频/03-Cover）

### CDP WebSocket 握手 403 Forbidden

- **现象**：python `websocket-client` 连 `ws://127.0.0.1:9222` 报 `Handshake status 403`，提示需要 `--remote-allow-origins`。
- **根因**：新版 Chromium/Edge 的 CDP 默认拒绝带 Origin 头的跨源 WebSocket 连接。
- **规避/解决**：启动参数加 `--remote-allow-origins=*`（本地渲染/自动化场景可接受）。
- **来源**：2026-10-11 同上（06-Render/render.py）

### headless 长连接连续截帧会在数百帧后挂死；"定期主动重启"是反模式

- **现象**：CDP 单连接逐帧 `Runtime.evaluate(seek)` + `Page.captureScreenshot` 渲染 3491 帧，首轮在 159 帧处永久阻塞（ws.recv 无响应）；加 30s 超时与"每 400 帧主动重启"后，每次重启点又连续 3-5 次 `CDP 端口不可达`。
- **根因**：①长连接合成器偶发挂死（无超时则整卡）；②`kill` 后 user-data-dir 的 Singleton 锁释放慢，立刻重启新实例会起不来，连杀连启形成风暴。
- **规避/解决**：ws 设 30s 超时，异常才重启；重启后必须轮询 `GET /json/version` 真正就绪再连（最长 60s）；断点续渲（按 frames/ 已有帧数继续，GSAP `seek(t)` 是绝对时间、帧间独立）；同帧连败 5 次复制前一帧补位跳过。修复后 9-10 帧/秒零重试跑完。
- **来源**：2026-10-11 同上（06-Render/render.py v2）

### ffmpeg 一条命令多 `-ss` 多输出：输出帧全来自第一段

- **现象**：`ffmpeg -ss 1 -i V out1.png -ss 33 -i V out2.png -ss 75 -i V out3.png` 三张输出内容相同（都是第一段的帧），误判为视频渲染错位，差点重渲。
- **根因**：多输出命令里输入选项/输出选项作用域解析与直觉不符，用 `md5` 对比中间帧文件后才定位到是抽帧命令问题。
- **规避/解决**：抽帧逐条命令跑；验证"视频是否错位"先对比源帧文件哈希再怀疑渲染。
- **来源**：2026-10-11 同上

### 142 测试机面板 admin 会话约 60-90s 被顶掉

- **现象**：浏览器自动化登录 YPanel 后逐页截图，约 1-2 分钟后被登出跳 `#/login?redirect=...`；AI 长请求期间尤其容易掉。
- **根因**：疑似与其他已登录会话互踢/短时效 token（未深究），表现为单会话存活极短。
- **规避/解决**：把「重登（fill 两框+坐标点登录）→ 点导航 → 截图」合并为一个动作链一气呵成；每页一条链；登出态先判断 URL 含 `/login` 再重登。
- **来源**：2026-10-11 面板截图（08-Assets/shots）

### YPanel SPA 上 Playwright locator `click()` 稳定超时（fill 正常）

- **现象**：`getByRole(...).click()` / `getByText(...).click()` 均 3s 超时（含 force:true），但 `fill()`、DOM 快照、`get_visible_dom()` 都正常；登录按钮两次超时后用坐标点击一次成功。
- **根因**：IAB 桥接的 locator click 可信度在该 SPA（reka-ui 组件库）上不可靠，actionability 检查过不去。
- **规避/解决**：优先 `dom_cua.click({node_id})`（`get_visible_dom()` 拿 ref，且必须在同一次连接批次内点击，跨调用 ref 会漂移）；无 ref 时 `cua.click({x,y})` 坐标点击（配合截图定位）。会话中途被顶掉后 tab 渲染面可能冻结（截图超时/`could not be restored`），关掉重开新 tab 最快。
- **来源**：2026-10-11 面板截图
