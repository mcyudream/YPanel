# 经验之谈：中间件发现与纳管

### 登记表不是容器清单，扫描按钮也不能替代主列表发现

- **现象**：「容器 → 中间件」显示空表，本机已有 RustFS、Neo4j、Redis、MongoDB 容器。独立扫描仅识别 RabbitMQ/Memcached，主表只读 MiddlewareInstance；扫描的 `admittable` 与前端 `adoptable` 不一致，纳管按钮也不出现。
- **根因**：登记、运行容器、数据库域连接记录是不同数据来源。仅查询其中一份会遗漏条目；把数据库域 ID 当中间件 ID 会进一步调用错误的测试/删除接口。
- **规避/解决**：`GET /middleware/instances?node=` 返回 `{items,total,discoveryError?}`。`items[].source` 分为 `middleware/database/discovered`，`key` 包含来源；`managedBy` 决定操作归属，`adoptable` 只对本域未纳管容器为真。主列表只读合并运行和停止容器；数据库类型关联已有记录或进入数据库管理接入。扫描失败保留登记记录且返回 discoveryError，扫描接口本身返回错误，不能用空数组掩盖失败。
- **来源**：2026-10-31，中间件空列表修复；middleware_inventory.go、api/middleware.go、middleware/index.vue、TestMiddleware*。

### 镜像的第一个冒号可能是 registry 端口，路径前缀也不是类型边界

- **现象**：`docker.io/library/redis:7.2`、带代理路径的 Neo4j，以及 `registry:5000/bitnami/rabbitmq:4.2` 漏识别；宽松前缀又会把 rabbitmq-exporter 当 RabbitMQ。
- **根因**：用第一个冒号裁 tag，或只对镜像全文做 startsWith。digest 内的数字也可能被误当版本。
- **规避/解决**：先去 digest，仅在最后一个冒号位于最后一个斜杠之后时裁 tag；按完整仓库路径段后缀匹配，不匹配名字的任意字符前缀。版本仅从 tag 提取。S3 的 MinIO/RustFS 使用完整仓库路径，避免将同命名空间的 console 误识别为服务。
- **来源**：2026-10-31，middleware_registry.go；已用本机 Docker 元数据输出验证模板，TestMiddlewareImageRecognition 覆盖镜像仓库端口、代理、digest 和误命中。

### 节点和属主必须参与发现去重，凭据只在显式纳管时提取

- **现象**：跨节点同名容器被标为已纳管；远程版本刷新或端口查询落到 local；扫描 HTTP 响应含从容器环境提取的密码。
- **根因**：MiddlewareInstance 未存节点，仅按容器名去重，扫描与凭据读取共用一条路径。NodeScope 只看查询参数，不能代替服务层对请求体节点、实例归属的校验。
- **规避/解决**：MiddlewareInstance 新增 NodeID，旧记录空值视作 local；去重使用节点和容器身份，远程默认实例名加节点后缀。测试和版本探测使用已保存节点，属主与节点范围在服务层校验。自动发现仅查询 Docker 元数据；显式纳管才 inspect 凭据并加密入库，列表/扫描不返回密码。已登记或商店属主不可见的容器不重新暴露为可纳管条目。商店中间件候选只展示同节点容器；注入时检查同节点、同网络和运行状态，跨节点/跨网络使用显式登记的可访问外部地址。
- **来源**：2026-10-31，middleware.go、store_middleware.go、aitools_middleware.go；节点、属主、密码响应与商店复用回归测试。

### 数据库域记录没有节点/容器字段，关联需要保守判断

- **现象**：S3、Neo4j 已在数据库管理登记，却仍被发现为另一份容器；一个 Compose 项目有两个同类服务时，仅凭项目名会让两个服务覆盖到同一行。
- **根因**：DatabaseInstance 尚无 NodeID/ContainerName，且数据库实例名、容器名、项目名不是同一个标识。
- **规避/解决**：沿用数据库记录作为权威身份；节点仅从唯一的商店项目关联取得，否则存量容器记录按 local。关联要求同类型、同节点，结合项目、明确容器名或回环连接的宿主发布端口，而且记录与容器必须一一对应。外部远端实例不能仅因实例名相同而关联本机容器。有歧义时保留独立发现条目，不自动改库或重复纳管；本次未重构数据库域的位置模型。
- **来源**：2026-10-31，middleware_inventory.go；数据库跨节点、歧义项目与受限记录回归测试。
