# 经验之谈：日志中心（VictoriaLogs / Vector 生态）

### LogsQL 已演进到 v2 语法：Loki 风格的 |= 与 |~ 完全无效

- **现象**：按旧资料写 `* |= "error"`、`{} |= "error"`、`|= "error"` 全部 HTTP 400，报错是误导性的 `cannot parse 'filter': missing ':' in front of "="`（看似提示缺冒号，实为整类语法不存在）；单独的 `*` 与裸词 `"error"` 却正常。
- **根因**：VictoriaLogs（实测 v1.53.0）的 LogsQL 用**查询语言 v2**：过滤器直接写在查询里，不再有 Loki 式管道操作符。`|=` → 裸短语 `"kw"`、`|~ "re"` → `~"re"`、排除 → `-kw`/`NOT kw`、子串 → `*kw*`、AND → 空格分隔。流选择器 `{field=~"a|b"}` 语法不变，可与裸过滤器空格连用（`{container_name="x"} WiredTiger`）；无任何过滤条件时查全部用 `*`。网上教程/旧文档/Grafana 插件示例大量还是 v1 语法，照抄必炸。
- **规避/解决**：构建查询器时按 v2 生成：`{container_name=~"a|b"} "kw"` / `~"re"`；无过滤条件回退 `*`。排查此类"语法对不对"先直连 VL `/select/logsql/query`（`--data-urlencode`）做最小对照，别隔着面板层猜。
- **来源**：2026-10-08，M33 P2 集中检索（142 真机 v1.53.0 实测 E/F/G 组对照）。

### stream_field_values 必须带 query 参数；stream 选择器是容器定位的正解

- **现象**：`/select/logsql/stream_field_values?field=container_name` 报 `query arg cannot be empty`。
- **根因**：该接口的 query（流过滤）是必填参数，不是可选；没有"列全部流"的免 query 形态。
- **规避/解决**：固定传 `query=*`（全部流）；取容器清单用 `field=container_name` + start/end。另注意 docker_logs 采集**不回填历史**——vector 启动前容器已产生的日志收不到，只有启动后新写入的日志；"集中检索能看到哪些容器"由 vector 启动后活跃的容器决定。
- **来源**：2026-10-08，M33 P2（142 实测）。

### Vector docker_logs 自排日志：hostname 自动排除 + 显式 exclude_containers 双保险

- **要点**：docker_logs 源会用「容器 ID == 自身 hostname」识别并自动排除自己（docker 默认 hostname=短 ID 时生效）；但显式 `exclude_containers = ["${VECTOR_SELF_NAME:-vector}"]`（前缀匹配容器名/ID，compose 环境变量注入容器名）更可依赖。自定义/改写 hostname 的容器自动排除会失效。
- **vector.toml 里的 `${VAR:-default}` 插值**：值来自 compose environment 注入（compose 的 .env 变量不会自动进容器环境），两处都要写。
- **来源**：2026-10-08，M33 P2 模板（vector 0.49.0）。

### VictoriaLogs/Vector 镜像选型速记（2026-10）

- VL 现行 tag 无 `-victorialogs` 后缀（旧资料是 `v1.23.3-victorialogs` 形态），现行如 `victoriametrics/victoria-logs:v1.53.0`；HTTP 端口 9428，`/health` 返回 OK；retention 默认 7d（`-retentionPeriod=30d` 可调）；数据目录 `-storageDataPath=/victoria-logs-data`。
- 验证镜像 tag 可拉性：`docker manifest inspect` 在部分环境（registry mirror 场景）恒失败不可用，直接 `docker pull -q` 实测最可靠。
- ES 协议接入：sink endpoint 写 `http://<vl>:9428/insert/elasticsearch/`（vector 会追加 `_bulk`），`api_version="v8"` + query 参数 `_msg_field=message&_time_field=timestamp&_stream_fields=host,container_name,image`。
- **来源**：2026-10-08，M33 P2 商店模板（142 registry mirror 实测拉取）。

### /select/logsql/hits 的 step 是必填参数：省略报 "cannot parse duration from the arg 'step='"

- **现象**：`/hits?query=...&start=5m`（不带 step）返回 400 纯文本 `cannot parse duration from the arg 'step='`——与直觉相反（多数接口省略即默认值）；VL 自身日志 warn 落在 logsql.go:224。
- **根因**：v1.53 的 hits 把 step 解析放在参数缺省判断之前，空串直接进 duration 解析。stream_field_values 的 query 必填是同族行为（见上条）。
- **规避/解决**：调 hits 恒传 step（直方图按范围选 30s~2h；纯计数场景传 1m 即可，total 与桶无关）。面板侧已在 Hits 代理层统一默认，调用方无需关心。
- **来源**：2026-10-08，M33 P3 日志量告警（告警计数走 hits 恒 0 → 静默跳过，journalctl 对照 VL warn 定位）。
