# YPanel 插件体系规范

> 首个插件：DB Admin（数据库管理台，自研，对标主流桌面数据库客户端的轻量体验，**非第三方容器套壳**）。
> 插件体系服务两类目标：① 可选装的功能扩展（DB Admin、Redis 工具、日志分析等）；② 不膨胀宿主二进制与攻击面。

## 1. 插件形态

### 1.1 分期路线

| 期 | 形态 | 说明 |
|---|---|---|
| 一期（V1，DB Admin 落地） | **进程内插件包**：Go 包编译进宿主，构建标签/配置决定装不装 | 不做 `go plugin` 动态加载（跨平台差、版本耦合坑多）；前端模块按插件分包，未安装的插件不进入菜单/路由 |
| 二期（V2+） | 独立插件进程 + RPC（或 WASM 沙箱，立项时再评估） | 支持第三方插件、崩溃隔离、独立升级 |

一期的"可选装"语义：**构建时装配 + 运行时启用/禁用**两级——打包出 full / slim 两种发行（slim 不含插件代码），full 内用户可在界面启停单个插件。

### 1.2 为什么不用第三方容器方案

第三方 Web 数据库管理工具（容器套壳类方案）的问题：体验差（另一套 UI/登录）、跳转断裂、无法复用面板连接管理与权限体系、容器编排增加用户负担。YPanel 的插件是**工作台原生页面**，共享宿主的账号、权限、数据库连接池、双工作台窗口体系。

## 2. 插件目录结构

```
plugins/
└── db-admin/                  # 插件 ID = 目录名（kebab-case）
    ├── plugin.yaml            # 清单（见 §3）
    ├── server/                # 后端插件包（独立 go.mod）
    │   ├── cmd/register.go    # 装配入口：注册路由/服务/菜单
    │   ├── api/  service/  driver/   # driver/ 为各库适配层
    │   └── go.mod             # 仅依赖 shared 插件 SDK + 白名单库
    └── web/                   # 前端插件模块
        ├── src/               # 页面/组件/store
        ├── routes.ts          # 路由声明（宿主动态挂载）
        └── package.json       # 独立构建，产物进宿主 embed 目录
```

## 3. plugin.yaml 清单

```yaml
id: db-admin
name: 数据库管理台
version: 0.1.0
minCoreVersion: 0.1.0      # 宿主最低版本
description: 库表浏览、数据编辑、SQL 查询器（MySQL/PG/Redis/MongoDB）
permissions:               # 权限声明（宿主按此生成授权 UI 与运行时校验）
  - database.connection.read   # 读连接配置
  - database.connection.exec   # 经宿主连接层执行 SQL
  - ws.stream                  # 流式结果推送
menus:
  - { path: /plugin/db-admin, title: 数据库, icon: database, modes: [classic, desktop] }
defaultEnabled: true
```

规则：

- **权限最小化**：插件不得直接创建数据库连接，必须经宿主 `database.connection` 服务（统一加密存储凭据、连接池、超时与审计）
- 权限未声明的能力，宿主在运行时拒绝（路由/服务两层校验）
- 菜单声明 `modes`，同一页面在经典模式进路由、在桌面模式开窗口（窗口壳由宿主提供）

## 4. 生命周期

| 动作 | 行为 |
|---|---|
| install（构建期） | 代码编入 full 发行；注册表登记 |
| enable | 注册路由/菜单/WS 通道；懒初始化连接池等重资源 |
| disable | 摘除路由与菜单，释放连接池；**数据保留** |
| uninstall（仅 slim/二期） | 移除代码与数据（二次确认，数据先备份） |

宿主提供插件注册接口（shared/sdk）：`RegisterRoutes`、`RegisterMenus`、`RegisterServices`、`OnEnable/OnDisable` 钩子。插件**不得**持有宿主内部包，只面向 SDK 编程——这是二期进程化的前置保证。

## 5. 依赖与隔离

- 后端插件包：独立 `go.mod`；只允许依赖 `shared`（插件 SDK + dto）与 package-management.md §1.3 的白名单驱动库；许可证准入同宿主
- 前端插件模块：独立构建（库模式），运行时经宿主约定的动态挂载点注入；不污染宿主全局样式（UnoCSS 前缀隔离）
- 插件崩溃不得拖垮宿主：一期 recover 隔离 + 请求超时；二期进程隔离

## 6. DB Admin 插件落地要点（首插件样板）

- **适配层 driver/**：`mysql` / `pgsql` / `redis` / `mongo` 四个子包，统一接口：
  `Meta（库/表/结构）`、`Query（SQL/命令执行，分页游标）`、`Edit（结果集写回）`、`Explain（执行计划）`、`ImportExport`
- SQL 执行经 WS 流式返回（大结果集背压），单次执行有超时与行数上限（可配置）
- 危险操作（DROP/TRUNCATE/无 WHERE 的 UPDATE/DELETE）前端二次确认 + 后端审计日志
- Redis/MongoDB 不套 SQL 模型：Redis 为 key 浏览器 + 命令控制台（命令白名单），Mongo 为集合/文档视图 + 查询构造器
- 审计：所有经插件执行的写语句落审计表（谁、何时、对哪个实例、语句摘要）
