# docs/exp/ — 经验之谈（坑与经验库）

这里沉淀开发与调试中踩过的坑、非显然的解决方案、值得复用的经验。**本目录随 git 提交**，与 docs/dev/ 规范互补：规范管"必须怎么做"，这里管"为什么、别再踩"。

## 何时必须写

- 排查超过半小时才定位的问题
- 因环境/版本/平台差异导致的坑（Windows/Linux、依赖版本、Docker 行为差异）
- 文档里没有、从源码或试错中得出的结论
- 踩过 reference/ 参考项目的坑并找到规避方式

## 组织方式

- 按主题分文件：`docker.md`、`frontend.md`、`go-backend.md`、`database.md`、`deploy.md` 等，按需新增
- 每条经验格式：

```markdown
### 标题（一句话说清坑）

- **现象**：怎么表现出来的
- **根因**：为什么会这样
- **规避/解决**：正确做法
- **来源**：日期 + 任务/文件位置
```

- 经验稳定后，若属于"必须/禁止"级规则，提示沉淀到 docs/dev/ 对应规范（规范引用本条，本条不删除）

## 条目

- [go-backend.md](./go-backend.md) — moby 模块拆分、internal 跨模块限制、GOPROXY 镜像、ExtJSON 归一化、DSN Replace 误伤、SHOW INDEX 版本漂移
- [frontend.md](./frontend.md) — fa 基座对接真实后端、vue-tsc 假性 TS6133、MSYS 路径转换、xterm 排障、lucide 图标名漂移
- [deploy.md](./deploy.md) — Text file busy、纯 Go SQLite 交叉编译、敏感信息解析、systemd 排障、robocopy /MIR 方向与 dist 锁定处置
- [logging.md](./logging.md) — LogsQL v2 语法（\|=/\~ 已废）、stream_field_values 必带 query、vector 自排日志、VL/Vector 镜像 tag 速记
