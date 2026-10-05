# YPanel Agent Entry

- 项目名称：**YPanel**
- 开发团队：**YuDream**

YPanel：YuDream 团队自研的一体化服务器管理面板（Docker/站点管理、桌面/经典双工作台与 AI、多节点、数据库/备份/安全/防火墙）。

- 后端：Go（core + agent 架构，单机合并部署，多节点 agent 独立分发）
- 前端：Vue 3 + fantastic-admin 基础版（MIT），桌面工作台模式自研窗口层，默认经典面板
- 功能总清单：[docs/YPanel-feature-list.md](./docs/YPanel-feature-list.md)

## 开始工作前

在分析、设计、修改代码、补测试、写文档之前，先阅读：

1. [docs/dev/conventions.md](./docs/dev/conventions.md)（开发规范总纲）
2. 按任务范围选读：
   - [docs/dev/package-management.md](./docs/dev/package-management.md)（包管理规范）
   - [docs/dev/plugin-system.md](./docs/dev/plugin-system.md)（插件体系规范）
   - [docs/exp/](./docs/exp/)（经验之谈：坑与经验，遇问题先查这里）
3. 任务相关的 [docs/plan/](./docs/plan/) 计划与 [docs/](./docs/) 文档

每次开始新任务，都必须重新完整阅读以上文件及当前任务范围内适用的专用规范，并在首次工作进展中明确说明本次已读取的文件。不得以历史会话、已有摘要或此前的读取结果代替；未完成阅读前，不得分析、设计、给出修改方案或进行任何实质性代码写入。

## 硬规则

- 修改代码前，先给出可选方案并等待确认。
- 默认使用中文沟通、说明和评审。
- 未经确认，不进行实质性代码写入、删除、迁移、重构。
- 不得覆盖、回滚或污染用户已有未提交改动。
- 新发现的稳定约定，需要提示是否沉淀到 docs/dev/ 规范。
- **踩坑与非显然经验必须补充到 [docs/exp/](./docs/exp/)**（格式见该目录 README）；排查超过半小时的问题、环境/版本/平台差异坑、从源码或试错得出的结论都算。
- **前端组件强约束**：
  - 只准使用 fantastic-admin 封装组件（`fa-` 前缀，如 `FaButton`/`FaModal`）；**禁止直接使用任何裸 Arco Design Vue 组件**。
  - UI 变体只保留 **Arco**（fantastic-admin `apps/core-arco-design-vue`），其余变体（element-plus / ant-design-vue / naive-ui / tdesign / vexip-ui 等）一律删除，不得引入。
  - fa 没有的组件，以 **`yd-` 前缀**自行封装（如 `YdXxx`），且必须保持 fa 的视觉风格与交互效果（主题变量、圆角、阴影、动效一致），禁止引入第二套视觉体系。

## 敏感信息

本地敏感信息（测试账密、SSH、环境路径、代理配置、临时 token）统一放 [docs/local.md](./docs/local.md)（已被 gitignore）：

- 禁止提交、禁止外发；代码/配置/规范文档中一律用占位符。
- 调试任务可读取使用，但不得在回复或日志中完整复述密钥内容。

## reference/ 参考源码

[reference/](./reference/) 下是若干第三方参考项目的浅克隆源码，**只读对照、已被 gitignore**：

- 各项目协议各异（自定义 EULA / AGPL / GPL 均有）——复制代码存在授权或传染性风险。
- 原则：**参考架构与逻辑、重手写实现**；逐项目协议细节见本地文档 docs/YPanel-feature-list.md 第二节（不提交）。

## 文档组织（全部在 docs/ 下）

- `docs/` — 长期文档（功能清单、调研、local.md 敏感信息）
- `docs/dev/` — 开发规范（总纲、包管理、插件体系等）✅ 提交
- `docs/exp/` — 经验之谈（坑与经验库）✅ 提交
- `docs/plan/` — 阶段计划与任务拆解（不提交）
- 其余文档（功能清单、local.md 等）不提交；git 提交范围以 .gitignore 为准
