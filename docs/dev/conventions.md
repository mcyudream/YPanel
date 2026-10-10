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
- RBAC 权限三层校验（M54 起强制，详见 core/internal/rbac）：
  - **新增路由必须声明权限点**：先在 `core/internal/rbac/perm.go` 目录登记（`模块:read|write`，高危动作单列），再在 router.go 标注 `pm("...")`；不带权限点的路由仅限面板级自助端点（auth 自助/通知/任务中心等，需注释说明）。新增 AI 工具模块必须在 `rbac.aiModulePerm` 登记模块→权限点映射，未登记的模块回落 ai:use/ai:admin。
  - **前端 meta.auth 只是 UI 过滤**，不是安全边界（fa 守卫不做硬拦截）；权限硬边界只有后端中间件。fa `hasPermission` 是精确 includes，后端下发权限列表必须经 `rbac.Expand` 展开通配。
  - **节点范围**：带 `?node=`/`?nodeId=` 的接口自动过 NodeScope 中间件；新写「隐式操作本机节点」的接口要评估是否加入 `middleware/localImplicitPrefixes` 前缀表。资源「分配属主」类接口必须 `requireAllScope` 把关。
  - **数据范围**：新资源域引入属主时沿用三件套——List 服务层 ctx 过滤（`rbac.CallerFrom`）、单资源 API 断言（api/owner.go 的 ownedXxx 助手，越权 404）、创建归属（`stampOwnerForCreate`：assigned→自己，all→公共）。

## 4. 前端编码约定

- 基于 fantastic-admin（**仅 Arco 变体** `apps/core-arco-design-vue`，其余 UI 变体一律删除不引入）的目录与命名惯例；业务模块放 `src/modules/`，桌面工作台放 `src/workbench/`
- **组件强约束**：
  - 只使用 fa 封装组件（`fa-` 前缀），**禁止直接使用裸 Arco Design Vue 组件**（禁止 `<a-button>` 等直写）
  - **禁止使用原生 `<select>`**（含弹窗/表单内）：原生下拉展开层是浏览器 UI，不可主题化且位置脱管——一律用 `YdSelect`（FaDropdown 封装，drop-in 支持 options/v-model）；存量替换与规则见 eslint `vue/no-restricted-html-elements`
  - fa 缺失的组件以 `yd-` 前缀封装（`YdXxx`），保持 fa 视觉风格与交互效果（主题变量、圆角、阴影、动效一致）；`yd-` 组件集中在 `src/components/`（或 `src/ui/`），全局注册
  - 封装 `yd-` 组件时允许在组件内部使用 Arco 原语，但必须包成完整语义组件，不向外暴露 Arco API
- 双模式共用同一套 API store（Pinia）与路由权限；模式切换不刷新页面、状态不丢失
- 组件内通信走 store/事件总线，禁止跨模块直接引用内部实现
- API 层统一封装（错误处理、token 刷新、WS 重连）；禁止组件内裸 fetch
- 类型完备：API 请求/响应均有 TS 类型，与后端 shared/ 的 dto 对齐（后续考虑代码生成）

## 5. 商店应用约定

应用打包规范以商店仓库 `YPanel-AppStore/README.md` 为准；扩展可用性实测矩阵见 `docs/dev/php-runtime-extensions.md`。

### 5.1 数据库需求声明与纳管接入

应用需要数据库时**必须按标准键集声明**，安装向导才会出现「数据库」下拉（应用自带 / 使用纳管实例自动建库建号注入）：

- **键集命名**（compose 应用在 `versions[].env`、php 应用由 app.json `database` 声明合成）：`<前缀>_HOST` 必备，前缀词干须含 `DB|DATABASE|SQL|MYSQL|MARIA|MONGO` 之一（如 `DATABASE_HOST`、`MYSQL_HOST`），其余按 `<前缀>_PORT/_NAME/_USER/_PASSWORD` 推导。**不遵守命名的声明不会触发下拉**（前端 dbHostKey 与后端 dbFieldSetOf 双端同规则检测）。
- **接管行为**：向导选中纳管实例后，同前缀的原始连接字段（地址/端口/库名/用户/密码）从表单**隐藏接管**，由 `applyExternalDB` 自动建库建号授权并注入连接参数。连接地址按三层策略生成（同节点应用容器内不搬宿主 IP）：① 实例容器与应用**同一容器网络** → 容器名:内部端口（docker DNS 直连；面板自建实例 compose 统一接入 `ypanel_default`）；② 同节点**不同网络** → `host.docker.internal:映射端口`（安装时自动注入 `extra_hosts: host.docker.internal:host-gateway`，配置零 IP；host 网络模式应用直接用回环）；③ **跨节点**或外部远端实例 → 宿主/远端 IP:端口。「应用自带」时原始字段照常显示。
- **PG 扩展一键创建**：应用需要 PG 扩展时在包内声明——compose 应用写 `index.json` 版本项 `"pgExtensions": ["pg_trgm"]`，php 应用写 `app.json` 的 `database.pgExtensions`；安装选纳管 PG 后建库自动 `CREATE EXTENSION IF NOT EXISTS`（包声明 ∪ 向导 `externalDB.extensions` 附加，去重），创建失败即中止安装（缺扩展的应用装了也是坏的）。php 应用向导传实例时会校验声明引擎与实例类型一致（mysql 应用选 PG 实例直接报错）。
- **默认值**：compose 应用默认「应用自带」；php 应用（app.json `database.create: true`）默认「纳管实例」。
- 应用自带数据库的 compose 应用（如 element-skin）：内置服务密码字段用 `random: true` 自动随机，不走纳管下拉。

### 5.2 php 站点 nginx 模板

php 站点 fastcgi 段**必须显式包含** `fastcgi_param CONTENT_TYPE $content_type;` 与 `fastcgi_param CONTENT_LENGTH $content_length;`——nginx 的 `fastcgi_params` 文件不含这两条（在 fastcgi.conf 里），缺失会导致 **PHP 收不到任何 POST body**（GET 正常，极难察觉）。修改 conf 后存量站点需重新生成或 sed 补行 + reload。

### 5.3 PHP 应用类型（kind: php）

重装语义=站点目录原位保留 + 源码覆盖（.env/storage 不丢）；app.json `upgrade.backup=true` 时重装前自动 tar 站点目录（备份失败即中止），`upgrade.commands` 在版本变更时替代 install 命令执行。

## 6. AI 工具同步规范（M33 固化）

面板功能与 AI 工具注册表**必须同步维护**——功能上线而工具缺失 = 功能不完整：

- **新增功能**（新路由域/新服务能力）：在 `core/internal/service/aitools_*.go` 同步提供工具（名称/描述/JSON Schema/风险级/节点参数），并在 `internal/rbac/perm.go` 的 `aiModulePerm` 确认权限点映射、`aiModuleRegistry` 确认目录条目；
- **修改功能**（参数/行为/返回变化）：同步更新对应工具的 Desc/Schema/实现，确保模型调用不失效；
- **强制校验**：`go test ./internal/service/ -run TestAITools`——`TestAIToolsCoverAllRouteDomains` 扫描 router.go 全部 authed 路由域，未映射到工具模块的域会点名失败（新域在测试的 `routeDomainMap` 补映射或加入 `routeWhitelist`）；`TestAIModuleRegistryConsistent` 校验模块注册表与权限表一致；
- **节点语义**：涉及节点资源的工具必须支持 `node` 可选参数（helper 层 `withAINode`/`acFromCtx` 统一解析；Docker 域走 `DockerExtService.WithNode`）；
- **风险分级**：每个工具必须声明 risk（read/write/danger），danger 工具描述中注明确认语义；未登记模块回落 ai:use/ai:admin；
- **节点化现状豁免域**：databases（实例 host 即位置）、sites_certs（nginx 单机）、panel_ops/monitor 的面板全局数据、srcbuild（构建固定面板机）。

## 7. Git 规范

- 分支：`main` 稳定；功能分支 `feat/xxx`，修复 `fix/xxx`
- 提交信息：中文，格式 `类型: 描述`（feat/fix/docs/refactor/chore），如 `feat: 容器列表接口`
- 禁止提交：`docs/local.md`、`reference/`、`.env*`、构建产物、真实凭证
- 提交前自查：无调试日志、无注释掉的死代码、无 TODO 无主的临时代码

## 8. 文档与计划

- 新模块开发前：docs/plan/ 下先有计划（目标/工作项/验收标准），经确认后动工
- 计划完成后在计划文档标注状态；长期结论沉淀回 docs/ 或 docs/dev/
- 接口变更同步更新 docs 中的 API 说明（后续引入 OpenAPI 后以其为准）
