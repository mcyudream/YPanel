# 经验之谈：Go 后端

### Docker 官方 SDK 已拆分为 moby/moby 多模块（2025），旧导入路径编译失败

- **现象**：`go build` 报 `github.com/docker/docker/api@vX: module declares its path as: github.com/moby/moby/api but was required as: github.com/docker/docker/api`。
- **根因**：Moby 2025 年把 API 类型与客户端拆成独立模块，`github.com/docker/docker/api` 变成重定向壳，拉到的新版本声明自己的模块路径是 `github.com/moby/moby/api`。
- **规避/解决**：改用新路径——类型 `github.com/moby/moby/api/types/container`、客户端 `github.com/moby/moby/client`（`client.ContainerListOptions` 等 Options 也迁到了 client 包）、`stdcopy` 在 `github.com/moby/moby/api/pkg/stdcopy`。注意新 API 差异：`ContainerList` 返回 `client.ContainerListResult{Items []container.Summary}`；`Summary.State` 是 `container.ContainerState` 类型需 `string()` 转换；`Ports` 元素是 `PortSummary`（`IP` 为 `netip.Addr`）；`Stop/Restart` 的 `Timeout` 是 `*int`（秒）而非 `*time.Duration`；`Ping` 需要 `client.PingOptions{}`。
- **来源**：2026-10-06，agent/internal/dockerx。

### 跨模块不能引用 internal 包（core 内嵌 agent 服务）

- **现象**：core 编译报 `use of internal package github.com/ypanel/agent/internal/server not allowed`。
- **根因**：Go 规定 `internal/` 只允许同一 module（更准确地说是 internal 祖先目录之下）导入；core 与 agent 是两个 module。
- **规避/解决**：需要被宿主嵌入的包放公共路径（`github.com/agent/server`），`internal/` 只放模块私有实现。设计时先想清楚哪些包是"对外 SDK 面"。
- **来源**：2026-10-06，agent/server。

### proxy.golang.org 不可达时用 GOPROXY 环境变量切镜像，不进仓库

- **现象**：`go mod tidy` 卡死/超时（dial tcp 2607:f8b0... 失败）。
- **根因**：网络环境无法访问官方模块代理。
- **规避/解决**：`export GOPROXY=https://goproxy.cn,direct`（会话级环境变量，符合 package-management.md "代理配置只写环境变量或 docs/local.md"）；前端 pnpm 无需额外配置。
- **来源**：2026-10-06，全仓首次依赖解析。
