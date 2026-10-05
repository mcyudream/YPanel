# YPanel 开发规范（总纲）

> 专项规范：[package-management.md](./package-management.md)（包管理）、[plugin-system.md](./plugin-system.md)（插件体系）。
> 功能范围以 [../YPanel-feature-list.md](../YPanel-feature-list.md) 为准。

## 1. 技术栈

| 层 | 选型 | 说明 |
|---|---|---|
| 后端 | Go（core + agent 双进程） | 单机合并部署，多节点 agent 独立分发 |
| 前端 | Vue 3 + Vite + TS + Pinia + Vue Router + UnoCSS | 基于 fantastic-admin 基础版（MIT）二次开发 |
| 桌面工作台 | 自研窗口层（组件内窗口） | 与经典模式共享账号/路由/状态，默认经典面板 |
| 数据库 | SQLite（面板自身数据） | 单文件零依赖；被管理的数据库（MySQL/PG/Redis/MongoDB）走各自 Go 驱动直连 |
| 通信 | HTTP API + WebSocket（进度/终端/通知推送） | core↔agent 通信方案待定（见功能清单 §五.2） |
| 交付 | 单二进制（go:embed 前端产物） | |

## 2. 仓库结构（目标态）

```
ypanel/
├── AGENTS.md            # 本仓库 Agent 入口与硬规则
├── core/                # Go 管理面（账号/界面API/调度/市场）
│   └── go.mod
├── agent/               # Go 节点端（资源采集/Docker/Nginx/数据库/防火墙操作）
│   └── go.mod           # 独立 module，core 经 replace 引用共享包
├── shared/              # core/agent 共享的协议与类型（proto/dto/常量）
│   └── go.mod
├── web/                 # 前端（fantastic-admin 基座）
│   └── src/
│       ├── workbench/   # 桌面工作台模式（自研窗口层）
│       └── modules/     # 经典面板各业务模块
├── plugins/             # 自研插件（首个：db-admin），规范见 plugin-system.md
├── docs/                # 全部文档（local.md 敏感信息，不提交）
│   ├── dev/             # 开发规范（本目录）
│   └── plan/            # 阶段计划与工作项
└── reference/           # 参考项目源码（只读，不提交）
```

规则：

- 代码写入前必须先有对应 docs/plan/ 计划或经用户确认的方案。
- `reference/` 只读；严禁把其中代码复制进 core/agent/web/plugins（协议传染性，见 AGENTS.md）。
- 文档归属：长期事实与规范进 `docs/`（规范进 `docs/dev/`），阶段计划与拆解进 `docs/plan/`。

## 3. 后端编码约定

- 包分层：`cmd`（入口）→ `router`/`middleware` → `api`（控制器，薄）→ `service`（业务）→ `repo`（数据访问）→ `model`（实体），层间单向依赖、禁止跨层直达
- 无 `init()` 隐式装配、无全局可变单例；启动失败 panic，运行期禁止 panic
- 错误：业务错误带 i18n key 的统一错误体，系统错误向上抛；禁止静默吞错误
- 日志：标准库 `log/slog`，结构化 key-value；敏感信息（密码/密钥/token）不进日志
- 空列表返回 `make([]T, 0)`；资源关闭 `defer func() { _ = x.Close() }()`
- 安全红线：
  - 出站 HTTP 必须过 SSRF 校验封装
  - 路径拼接必须防穿越（锚定 root 的 SafeJoin）
  - shell 参数必须转义，禁止拼接用户输入
  - SQL 全部参数化；对 MySQL/PG 的管理操作（建库/授权）同样需要标识符白名单校验
  - 节点间通信必须认证+加密；agent 操作需能力白名单
- WebSocket：连接必须有鉴权与退出条件；协程不得持有请求上下文

## 4. 前端编码约定

- 基于 fantastic-admin（**仅 Arco 变体** `apps/core-arco-design-vue`，其余 UI 变体一律删除不引入）的目录与命名惯例；业务模块放 `src/modules/`，桌面工作台放 `src/workbench/`
- **组件强约束**：
  - 只使用 fa 封装组件（`fa-` 前缀），**禁止直接使用裸 Arco Design Vue 组件**（禁止 `<a-button>` 等直写）
  - fa 缺失的组件以 `yd-` 前缀封装（`YdXxx`），保持 fa 视觉风格与交互效果（主题变量、圆角、阴影、动效一致）；`yd-` 组件集中在 `src/components/`（或 `src/ui/`），全局注册
  - 封装 `yd-` 组件时允许在组件内部使用 Arco 原语，但必须包成完整语义组件，不向外暴露 Arco API
- 双模式共用同一套 API store（Pinia）与路由权限；模式切换不刷新页面、状态不丢失
- 组件内通信走 store/事件总线，禁止跨模块直接引用内部实现
- API 层统一封装（错误处理、token 刷新、WS 重连）；禁止组件内裸 fetch
- 类型完备：API 请求/响应均有 TS 类型，与后端 shared/ 的 dto 对齐（后续考虑代码生成）

## 5. Git 规范

- 分支：`main` 稳定；功能分支 `feat/xxx`，修复 `fix/xxx`
- 提交信息：中文，格式 `类型: 描述`（feat/fix/docs/refactor/chore），如 `feat: 容器列表接口`
- 禁止提交：`docs/local.md`、`reference/`、`.env*`、构建产物、真实凭证
- 提交前自查：无调试日志、无注释掉的死代码、无 TODO 无主的临时代码

## 6. 文档与计划

- 新模块开发前：docs/plan/ 下先有计划（目标/工作项/验收标准），经确认后动工
- 计划完成后在计划文档标注状态；长期结论沉淀回 docs/ 或 docs/dev/
- 接口变更同步更新 docs 中的 API 说明（后续引入 OpenAPI 后以其为准）
