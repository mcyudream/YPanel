# 经验之谈：构建与部署

### 覆盖运行中的二进制报 "Text file busy"

- **现象**：deploy 推送新二进制失败 `bash: /opt/ypanel/ypanel: Text file busy`。
- **根因**：Linux 不允许覆盖正在被执行的文件。
- **规避/解决**：部署脚本先 `systemctl stop ypanel` 再推送，最后 `systemctl enable --now || restart`。
- **来源**：2026-10-06，scripts/deploy.sh。

### Windows 交叉编译面板单二进制：纯 Go SQLite 是前提

- **现象**：需要从 Windows 开发机产出 linux/amd64 单二进制。
- **根因**：CGO（mattn/go-sqlite3）交叉编译需要目标平台 C 工具链，Windows 上折腾成本高。
- **规避/解决**：`github.com/glebarez/sqlite`（纯 Go，基于 modernc.org/sqlite，MIT）+ `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w"`；pty（creack/pty）等平台相关包用构建标签隔离（`//go:build linux` + 非 linux 桩），Windows 开发机可正常 `go build ./...` 做类型检查。
- **来源**：2026-10-06，M0 选型。

### 密码类敏感值一律"运行时从 docs/local.md 解析进环境变量"，不落脚本与命令行

- **现象**：部署需要 SSH 密码与初始 admin 密码；直接写命令行会留在 shell 历史/会话记录。
- **根因**：AGENTS.md 敏感信息规则：禁止提交、禁止在回复/日志复述。
- **规避/解决**：`export SSHPASS=$(grep '<标记>' docs/local.md | head -1 | awk -F'|' '{print $N}' | tr -d ' \r')`；注意 local.md 同表多行含相同关键字时要 `head -1`（本仓曾因 YPanel 行也含 IP 导致解析出两行、密码变 13 位）。sshctl 从 SSHPASS 读密码，argv 不落敏感值。
- **来源**：2026-10-06，scripts/deploy.sh + tools/sshctl。

### 单元测试目标机信息（systemd）

- **现象**：首次部署即 systemd 常驻，健康检查用远端 curl 而非开发机直连（面板只对本机 0.0.0.0:8880 暴露，内网可达）。
- **规避/解决**：`journalctl -u ypanel -n 30 --no-pager` 是排障第一入口；agent 日志与 core 同进程同 stdout（合并部署），JSON 行格式统一 grep。
- **来源**：2026-10-06，测试机部署验收。
