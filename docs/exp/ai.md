# 经验之谈：AI 会话、流恢复与上下文

### 普通 HTTP 访问时 crypto.randomUUID 不可用，聊天会在请求发出前失败

- **现象**：AI 聊天发送、上传附件或带附件编辑重发时报 `crypto.randomUUID is not a function`；发送链路尚未调用后端启动接口。
- **根因**：`crypto.randomUUID()` 受安全上下文限制，普通 HTTP IP/域名访问可能不提供该方法；`crypto.getRandomValues()` 仍可用。聊天请求幂等键、草稿附件和分支附件原先三处直接调用原生 UUID API；既有 Node 回归注入了完整 crypto，未覆盖浏览器缺少 randomUUID 的环境。
- **规避/解决**：三处统一使用 `utils/uuid.ts` 的 `createUuid()`：先检测原生 randomUUID，缺失时用 getRandomValues 生成 UUID v4，保持随机源、版本和变体位正确，不使用 Math.random。回归用仅提供 getRandomValues 的 crypto 运行真实 Pinia store，覆盖附件上传/发送及网络失败后复用同一请求幂等键；原生分支同时验证方法接收者。
- **来源**：2026-11-22，用户反馈 AI 聊天报错；`web/apps/core-arco-design-vue/src/store/modules/aiChat.ts`、`src/components/YdAiChat/Panel.vue`、`web/tests/ai-chat.test.mjs`。修复前受控复现发送/上传失败，发送后端调用次数为 0。API 行为依据 [W3C Web Cryptography](https://www.w3.org/TR/webcrypto/#Crypto-interface)。

### 用 SSE 探测状态后再次订阅，会留下未消费连接并阻塞接入函数

- **现象**：进入运行中的会话时存在两次 stream 请求；接入 Promise 一直等到生成结束，返回状态提示延后；旧连接还占用订阅资源。
- **根因**：`useAiChat.attachLive` 先 fetch SSE 读取 X-Ai-Run，随后 consumeStream 另发一次 fetch，没有消费/取消第一条响应；同时把整个流消费过程作为 attachLive 的等待条件。后端 ChatStream 没有在退出时调用运行体的 unsubscribe。
- **规避/解决**：状态探测使用有限响应快照，或复用同一流响应；attach 建立后即返回状态，后台消费由独立生命周期管理。服务端每个订阅必须在退出时移除，处理写出错误与请求取消。已改为快照 + 单一增量订阅，并补充订阅清理/隔离测试。
- **来源**：2026-11-15，AI 体系优化调研；`web/apps/core-arco-design-vue/src/composables/useAiChat.ts:365`、`core/internal/api/ai.go:153`。隔离转译当前 composable 并注入可控 fetch：2 次请求，第一条 body 未读取，第二条读取中，attach 在结束前不返回。实际用户卡顿中各因素占比仍需浏览器性能测量。

### 完成态运行体仍可回放时，完整历史后再追加回放会复制答案

- **现象**：任务结束后短时间内重开会话，出现两条相同助手回答。
- **根因**：后端结束运行体保留 10 分钟供迟订阅；前端只区分 X-Ai-Run 为 none 与其他值，对 done/stopped/error 也在已保存历史后新建 assistant 并从 offset=0 回放。
- **规避/解决**：运行体状态和消息身份一起校验；已结束时展示服务端完整历史，运行中使用带 runID/messageID/eventID 的快照合并，不能无条件追加。已修复并覆盖 Go / Pinia 回归。
- **来源**：2026-11-15，AI 体系优化调研；`useAiChat.ts:365`、`core/internal/service/aichatrun.go`。隔离输入已有 user+assistant 历史、模拟 done 运行体，结果为 user+assistant+重复 assistant。

### 模型历史窗口直接复用为落库历史，会把上下文裁剪变成数据丢失

- **现象**：页面还显示较长会话，发送下一轮并重新打开后早期消息消失；工具块或思考过程也不能完整恢复。
- **根因**：前端 send 先 filter/map 为纯正文并 slice(-20)，后端 newAiMsgBuilder 使用这份请求历史，结束时 SaveConversation 整体覆盖 messages；前端手动保存也只映射 role/content。模型输入、展示记录和持久化真源混在同一数组。
- **规避/解决**：服务端持有完整原始消息；发送提交新消息和会话版本。模型输入另行构建预算视图与摘要，不能写回覆盖原始记录；重命名只更新元数据；工具调用与结果使用稳定 ID 和完整结构保存。已修复并覆盖 Go / Pinia 回归。
- **来源**：2026-11-15，AI 体系优化调研；`useAiChat.ts:320`、`aichatrun.go:303`/`:330`、`aiext.go:109`。隔离验证 40 条旧消息 + 新提问只提交 20 条，而页面为 42 条，结合实际服务端覆盖逻辑确认丢失路径。

### 事件缓冲截断和慢消费者丢帧后，客户端计数不再是可靠续传游标

- **现象**：长任务重连可能漏掉开头内容、重复某段或丢工具步骤，连接本身仍正常。
- **根因**：运行体超过 8000 帧后截半并累加 base；客户端从自身收到的数据帧数计算 offset。服务端订阅通道满时静默丢帧，SSE 又不提供服务端事件序号和缺口通知；从旧 offset 返回截断后的内容也不告诉客户端实际起点。
- **规避/解决**：服务端分配 runID/eventID；快照与 lastEventID 同步取得，续传只接受连续游标，缺口或过期显式要求重同步；慢消费者关闭并恢复，不能静默丢持久事件。已修复并覆盖 Go / Pinia 回归。
- **来源**：2026-11-15，AI 体系优化调研；`core/internal/service/aichatrun.go:47`/`:84` 与 `useAiChat.ts:114` 源码对照结论，尚未在真实长任务中量化复现。

### marked 是 Markdown 解析器，输出不能直接充当可信 HTML

- **现象**：助手回复中的原始 HTML 会进入 v-html；解析结果仍包含 img 的 onerror 等事件属性。
- **根因**：`YdAiBubble.renderMd` 直接返回 marked.parse 结果，没有安全 HTML 清理；解析异常时又返回原始文本给相同 HTML 渲染路径。
- **规避/解决**：统一安全渲染助手，清理 HTML、事件属性与危险 URL，解析失败按纯文本转义；覆盖流分段和历史两条展示路径。已统一使用 DOMPurify 清理，并复用有界 Markdown/高亮缓存。
- **来源**：2026-11-15，AI 体系优化调研；`web/apps/core-arco-design-vue/src/components/YdAiBubble/index.vue:59`。使用项目实际 marked 解析无外部资源的测试 HTML，确认事件属性原样保留；未执行浏览器载荷。

### Anthropic BaseURL 与 OpenAI-compatible 参数不能照搬同一请求

- **现象**：设置了 Anthropic 类型却请求 Chat Completions，或原生 SDK 请求遗漏 `/v1`；指定输出预算后部分兼容端点不识别 `max_completion_tokens`。SDK 也可能在用户未设温度时自动发送 0。
- **根因**：旧 buildLLM 没有按协议分流；SDK BaseURL 约定不同，且“选项未设置”不一定等于“序列化时省略”。
- **规避/解决**：真实协议适配 + 本地 HTTP 契约测试。Anthropic 使用 `/v1/messages` 与 x-api-key；Responses 保留 opaque output items；出站统一省略未设置采样值，并按端点/显式能力选择一个输出预算字段。API 字段细节不能只看页面的供应商类型选项。
- **来源**：2026-11-15，AI 改造；本地 SDK 源码与三协议 httptest、多模态/工具回放契约验证。

### HTTP SDK 接收原始 Client 时，Do 包装里的校验会被绕过

- **现象**：模型 HTTP 请求经过校验，但 MCP SDK 接收同一个包装器内部的 `*http.Client` 后绕开 Do 的初始请求校验。
- **根因**：安全检查只放在自定义 Doer 上，而 SDK 要求的是底层 Client；代理场景更容易掩盖这个差异。
- **规避/解决**：在 RoundTripper 层进行同源、地址校验和响应大小/空闲超时处理，重定向也复核。增加直接调用内部 Client 访问未授权私网的测试，不能只测试 Doer.Do。
- **来源**：2026-11-15，MCP 实际接入与出站回归。

### 附件在分叉后必须以引用关系管理

- **现象**：按“最初所属会话”拒绝复用会阻断编辑重发；删除原会话时直接删除文件会破坏分叉会话。
- **根因**：一个文件可以被多个分支引用，单一 conversation_id 无法表达生命周期。
- **规避/解决**：使用附件引用表；启动、分叉和删除与引用变更同一会话锁串行，回收仅处理过期且无引用的文件。结果文件也走同一引用链。
- **来源**：2026-11-15，附件分叉/删除与重启测试。

### PDF 提取依赖必须显式说明，文本需要保留页码

- **现象**：上传按钮允许 PDF，但服务端缺少 pdftotext；去掉分页符后无法标明引用页码。
- **根因**：运行时工具不是 Go 构建产物的一部分；纯文本抽取没有自动的文档来源模型。
- **规避/解决**：界面说明部署依赖，缺工具时返回明确错误；保留分页符并转为页码标记。用实际最小 PDF 验证提取正文和页码，扫描文档明确不提供 OCR。
- **来源**：2026-11-15，DOCX/图片/UTF-8/PDF 本地格式链路测试。

### 关键词检索的前三条必须先在候选全集排序

- **现象**：早期文档在列表前方但相关性低，真正匹配的文档始终进不了前三；文档增长后每次检索又读取全部正文。
- **根因**：旧逻辑只对最前几条局部交换，而不是对全部候选评分排序；检索没有候选数据上界。
- **规避/解决**：数据库对受权候选逐词评分，各来源只取前 12 个，再合并排序取前 3；Go 不装载全部原文，返回片段截长，按需深读保留原文。新增 30 条低分候选后高分条目仍排第一的回归。
- **来源**：2026-11-15，知识检索与中文/属主隔离回归。
