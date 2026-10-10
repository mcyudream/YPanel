<div align="center">

# YPanel

**一体化服务器管理面板 —— 容器、网站、数据库、多节点、AI，一个面板全搞定**

YPanel 是 [YuDream](https://github.com/mcyudream) 团队自研的开源服务器管理面板：
经典面板与桌面工作台双模式，单机开箱即用，多节点统一纳管。

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/mcyudream/YPanel)](https://github.com/mcyudream/YPanel/releases)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev/)
[![Vue 3](https://img.shields.io/badge/Vue-3.x-4FC08D.svg)](https://vuejs.org/)

[Gitee 镜像](https://gitee.com/mcyudream/ypanel) · [问题反馈](https://github.com/mcyudream/YPanel/issues)

</div>

---

## ✨ 特性

### 🖥️ 双工作台，像操作系统一样管服务器
- **桌面工作台**：多窗口 + 任务栏 + 启动台 + 小组件 + 文件夹卡片，窗口吸附分屏、全局快捷键——基于自研开源框架 [yudream-web-os](https://github.com/mcyudream/yudream-web-os) 打造（npm 可独立使用）
- **经典面板**：侧栏 + 多标签的效率形态，与桌面共享账号与状态，一键互切
- **桌面小组件**：时钟 / 磁盘 / Docker 概览 / 网络流量直接铺在桌面，自由拖拽、三档大小
- 图标按功能分类配色、可自定义图标底色；文件夹卡片把同类应用归组一处

### 🌐 内网浏览器
- 面板内置浏览器，**直接打开路由器 / NAS / 打印机 / 监控等任意内网 Web 管理页**——不用记 IP、不用改 hosts、不用给管理页开公网
- 会话式反向代理：登录态 / WebSocket / 上传下载全透传，私网地址防重绑定攻击

### 🤖 AI 管家，会动手的那种
- 内置 **118+ 个工具**：装应用、建网站、配数据库、查日志、排故障——一句话，直接干
- 写操作默认人工审批，操作全程审计；支持 OpenAI 兼容 / Anthropic / Gemini 与 **MCP 协议**接入
- 知识库 / 长期记忆 / 技能包，越用越懂你的服务器

### 🧩 容器与应用
- 容器 / 镜像 / 网络 / 卷 / 配置全覆盖，Compose 编排，卡片式应用总览
- **应用商店兼容 1Panel 应用格式**，一键安装为 Compose 项目
- Docker / Compose 一键安装（官方源 + 国内镜像源 + 加速器预设）

### 🌍 网站与运行环境
- 静态站 / 反向代理 / 负载均衡，Nginx 配置校验与重载
- ACME 证书自动申请与续期、批量部署、到期告警
- PHP / Node 等容器化运行环境，扩展在线装卸

### 🗄️ 数据库
- MySQL / PostgreSQL / Redis / MongoDB 实例一站式管理
- **DB Admin 数据工作台**：库表浏览、数据编辑、多标签 SQL 查询器
- 备份到本地或 S3 / OSS / COS / WebDAV / SFTP，恢复一键完成

### 🛰️ 多节点
- core + agent 架构，配对码接入，agent 面板内一键升级
- 容器 / 网站 / 数据库 / 文件 / 终端，节点能力全量纳管
- **日志中心聚合多节点日志**（VictoriaLogs），统一查询与告警

### 🛡️ 安全
- RBAC 三层权限：权限点 × 节点范围 × 数据属主
- 安全入口（404 伪装）、2FA、IP 白名单、登录审计、危险操作锁定
- 防火墙 / NAT / 暴力破解防护 / **磁盘保护**（写满前自动停容器，一键恢复）

### 🧰 运维工具箱
- 文件管理、多标签终端、计划任务、脚本库
- 内网 DNS、异地组网、SSH 密钥统一分发、FTP
- 面板备份 / 系统级快照 / 远程备份存储

### 📦 面板自身
- **单二进制交付**：go:embed 内嵌前端，SQLite 单文件零依赖
- 自更新双通道（GitHub / Gitee），中英双语

---

## 🚀 快速开始

### 环境要求

- Linux（x86_64 / aarch64），systemd 发行版（Debian/Ubuntu/CentOS/Rocky/AlmaLinux 等）
- root 权限；Docker 可选（没装可在安装时或面板内一键安装）

### 一键安装主面板

```bash
curl -sSL https://gitee.com/mcyudream/ypanel/raw/main/deploy/quick_start.sh | bash -s -- --mode panel
```

交互引导中可设置端口（默认 `8880`）、初始 admin 密码（留空随机生成）、是否同时安装 Docker。

安装完成后脚本会输出**访问地址**（含自动生成的**安全入口**路径）、默认账号与初始密码——不带安全入口的访问一律 404，请妥善保存完整地址。

### 纳管其他节点

在已有面板的服务器上执行（配对码在主面板「节点管理 → 添加节点」生成，一次性、10 分钟有效）：

```bash
curl -sSL https://gitee.com/mcyudream/ypanel/raw/main/deploy/quick_start.sh | bash -s -- \
  --mode node --core http://面板地址:8880 --code 配对码 --node-name node-1
```

### 升级

**方式一（推荐）：面板内升级** —— 登录面板 → 系统 →「面板设置」，在线检查 GitHub/Gitee 双源后一键升级（下载 → sha256 校验 → 备份旧版 → 替换重启）。

**方式二：脚本升级**（面板不可用时的救急通道）：

```bash
bash quick_start.sh --upgrade    # 保留数据与配置
```

**方式三：SSH 命令行**：

```bash
ypanel update                        # 在线升级（默认 Gitee 源，失败自动回落 GitHub）
ypanel update --source github        # 指定更新源
```

### 常用操作（CLI）

安装/升级后 `ypanel` 已软链到 `/usr/local/bin`，任意路径可用：

```bash
ypanel user-info        # 访问信息：地址（含安全入口）/端口/服务状态
ypanel entry            # 查看安全入口
ypanel status           # 服务状态（ypanel + ypagent）
ypanel start|stop|restart
ypanel version          # 查看版本
ypanel backup           # 面板备份
ypanel clean-cache      # 清理 30 天前历史监控
ypanel reset mfa        # 关闭面板 2FA（验证器丢失救急）
ypanel -reset-admin <user>   # 重置用户密码（交互输入，可选同时关闭 2FA）
ypanel uninstall [--yes]     # 卸载（删除 /opt/ypanel，含数据）
```

> 忘记安全入口时，`ypanel user-info` 可直接找回完整访问地址；升级/重启后用 `ypanel version` 确认。

### 卸载

```bash
ypanel uninstall            # 交互确认后卸载（删除 /opt/ypanel 含数据）
bash quick_start.sh --uninstall   # 等价脚本方式
```

> 国内网络下载缓慢时，脚本默认已优先走 Gitee 源；也可用 `--source gitee|github` 显式指定。

### 节点 CLI

被纳管节点（`--mode node` 安装）同样有全局命令 `ypagent`：

```bash
ypagent info            # 节点信息：版本/节点名/core 地址/凭据/服务状态/core 可达性
ypagent status|start|stop|restart
ypagent version
ypagent update [--source gitee|github]   # 命令行自升级（面板不可达时的兜底通道）
ypagent uninstall [--yes]                # 卸载（删除 /opt/ypagent 与配对凭据）
```

---

## 🏗️ 架构与技术栈

| 层 | 选型 |
|---|---|
| 后端 | Go（`core` 管理面 + `agent` 节点端双进程，单机合并部署，多节点 agent 独立分发） |
| 前端 | Vue 3 + Vite + TypeScript + Pinia + UnoCSS，基于 [fantastic-admin](https://github.com/fantastic-admin/basic)（MIT）二次开发 |
| 桌面工作台 | 自研窗口层，基于 [yudream-web-os](https://github.com/mcyudream/yudream-web-os) |
| 存储 | SQLite（面板自身数据，单文件零依赖） |
| 交付 | 单二进制（go:embed 前端产物） |

```
ypanel/
├── core/      # Go 管理面（账号/权限/界面 API/调度/应用市场）
├── agent/     # Go 节点端（资源采集/Docker/Nginx/数据库/防火墙操作）
├── shared/    # core/agent 共享协议与类型
├── web/       # 前端（fantastic-admin 基座，Arco 变体）
├── plugins/   # 自研插件（如 DB Admin）
├── deploy/    # 安装与部署脚本
└── docs/      # 开发规范（docs/dev）、经验库（docs/exp）
```

---

## 🤝 参与贡献

欢迎提交 Issue 与 Pull Request！

### 开发环境

- Go ≥ 1.26、Node.js ≥ 22、pnpm 12

```bash
# 前端构建（产物嵌入 core）
cd web/apps/core-arco-design-vue
pnpm install
pnpm build

# 整体构建（前端 → 嵌入 → 交叉编译 core/agent）
bash scripts/build.sh [linux-amd64|windows-amd64|all]
# 产出 bin/ 目录
```

### 贡献约定

1. 动手前先阅读 [docs/dev/conventions.md](docs/dev/conventions.md)（开发规范总纲）与 [docs/exp/](docs/exp/)（踩坑经验库）
2. 分支：功能 `feat/xxx`、修复 `fix/xxx`，从 `main` 拉出
3. 提交信息：中文，`类型: 描述`（feat / fix / docs / refactor / chore）
4. 提交前自查：无调试日志、无死代码；禁止提交敏感信息与构建产物
5. 前端组件规范：优先使用 fantastic-admin 封装组件（`fa-` 前缀）；缺失组件以 `yd-` 前缀自行封装，不直接使用裸 UI 库组件

---

## 🙏 鸣谢

YPanel 的产品形态站在巨人的肩膀上。以下优秀开源项目为本项目的架构与功能设计提供了重要参考（YPanel 参考其架构与交互思路，代码为基于自身需求的重写实现），在此致谢：

| 项目 | 说明 |
|---|---|
| [1Panel](https://github.com/1Panel-dev/1Panel) | 数据库、备份、安全、防火墙等运维能力的设计参考，core + agent 多节点架构范式 |
| [KPanel](https://github.com/kejilion/KPanel) | 桌面/经典双工作台、多主机管理与 AI 助手的设计思路参考 |
| [DPanel](https://github.com/donknap/dpanel) | Docker 可视化与站点管理能力的设计参考 |
| [Portainer](https://github.com/portainer/portainer) | 容器管理交互的参考 |
| [yudream-web-os](https://github.com/mcyudream/yudream-web-os) | YuDream 同团队开源项目，YPanel 桌面工作台模式基于其窗口层开发 |
| [fantastic-admin](https://github.com/fantastic-admin/basic) | 前端基座（MIT），经典面板的框架与工程体系 |
| [Arco Design Vue](https://github.com/arco-design/arco-design-vue) | UI 组件库 |

同时感谢 Go 与 Vue 生态的众多开源库（Docker SDK、gopsutil、xterm.js 等），它们是 YPanel 得以高效构建的基石。

---

## 📄 开源协议

本项目基于 **[GNU AGPL-3.0](LICENSE)** 协议开源。

- 你可以自由使用、修改、分发本项目，修改与衍生作品须以同一协议开源
- 通过网络向用户提供服务（含修改后的版本）时，同样须向用户提供对应源代码
- 前端基座 fantastic-admin 为 MIT 协议，其版权声明保留于 `web/` 目录

Copyright © 2026 YuDream
