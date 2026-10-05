# YPanel 包管理规范

> 适用范围：Go 后端（core/agent/shared）、前端（web）、插件（plugins）。
> 目标：依赖可审计、可复现、不被协议传染。

## 1. Go 依赖

### 1.1 Module 结构

- `core/`、`agent/`、`shared/` 各自独立 `go.mod`（分 module 便于 agent 独立分发与依赖白名单控制）
- core/agent 引用 shared 用 `replace github.com/ypanel/shared => ../shared`
- `GOTOOLCHAIN=local`，Go 版本统一锁定（写入各 go.mod 与 CI），不自动下载工具链
- agent 端依赖白名单制：只引入采集/操作必需依赖，保持 agent 二进制精简可独立分发

### 1.2 依赖准入（新增第三方库前必须过）

1. **许可证白名单**：MIT / BSD / Apache-2.0 / ISC ✅；**GPL/AGPL/SSPL ❌**（传染性，除非 YPanel 自身协议明确允许）；自定义/未知许可证逐案评审
2. 维护状态：近一年有提交、无未修复的高危 CVE
3. 能用标准库解决的不引库；能在现有依赖内解决的不加新库
4. 新增依赖需在 PR/任务说明中写明：用途、许可证、替代方案为何不行

### 1.3 关键领域选型基线

| 领域 | 基线 | 备注 |
|---|---|---|
| Docker | `docker/docker` 官方 SDK + `compose-spec/compose-go/v2` | |
| Web 框架 | 待定（gin / echo / chi，立项时拍板） | 只选一个，全仓统一 |
| ORM | GORM + gen（面板自身 SQLite） | gen 代码标记 DO NOT EDIT |
| 数据库驱动 | `go-sql-driver/mysql`、`jackc/pgx`、`redis/go-redis`、`mongo-driver` | 管理面直连被管实例 |
| ACME | `go-acme/lego` | |
| SSH | `golang.org/x/crypto/ssh` | |
| 终端 pty | `creack/pty` | |
| 采集 | `shirou/gopsutil` | |

### 1.4 版本与审计

- `go.mod`/`go.sum` 必须提交；禁止 `replace` 指向公网 fork 而不注明原因
- 升级依赖：功能升级与安全升级分开提交，注明变更范围
- 定期 `govulncheck`（后续进 CI）

## 2. 前端依赖

- **pnpm**（fantastic-admin 基座原生命令），`pnpm-lock.yaml` 必须提交；禁止混用 npm/yarn 产生多 lockfile
- Node 版本：最新 LTS，写入 `package.json` `engines` 与 `.nvmrc`
- 依赖分层：`dependencies`（运行时）与 `devDependencies`（构建/工具）严格分开；运行时代理新增需说明体积影响
- UI 组件：以 fantastic-admin 内置组件与 UnoCSS 优先，确需新增组件库需评审（避免多 UI 体系混用）
- 许可证准入同 Go（前端 dist 会随二进制分发，同样受传染约束）

## 3. 插件依赖

- 插件依赖与宿主隔离，规则详见 [plugin-system.md](./plugin-system.md) §5：
  - 后端插件包：独立 `go.mod`，仅允许依赖 shared 定义的插件 SDK 与准入白名单库
  - 前端插件模块：独立构建产物，依赖不外泄到宿主 bundle
- 插件不得引入宿主未准入的新许可证类型

## 4. 镜像与制品

- 第三方容器镜像（数据库、nginx 等）固定 digest 或 major 版本 tag，禁止裸 `latest`
- 镜像源/代理配置只写环境变量或 docs/local.md，不进仓库
