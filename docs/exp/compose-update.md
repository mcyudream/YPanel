# 经验之谈：编排镜像更新

### 单服务操作也有 name/project 契约，外部目录和多文件不能在代理时丢失

- **现象**：应用详情点击重启服务时报 `Project required`，同时出现页面和全局两条错误提示；即使修正字段，外部项目仍会在托管目录下寻找配置。
- **根因**：前端 serviceAction 发送 name，Core 匿名请求只接收 project；Core 未声明和透传 dir，agent 单服务入口也只在 baseDir/project 查文件。该端点实际返回 output，却用 exec 请求/响应类型代理。页面已有 catch 提示，API 仍弹全局提示。
- **规避/解决**：单服务请求使用独立 DTO，统一 project 并兼容旧 name、拒绝别名冲突；HTTP/AI 共用服务入口，按实际节点检查权限。保留 agent 查询参数协议并透传 dir，使用 output 响应；agent 依据容器标签保留全部 config_files 的顺序与工作目录，缺失任一文件都中止，未记录文件时按目录解析。该页面负责服务操作提示，API 设置 silent，避免重复。
- **来源**：2026-10-31，用户反馈应用详情服务重启失败；`shared/dto/compose_service.go`、`core/internal/service/compose_service.go`、`agent/internal/compose/compose.go`，API/agent/命令替身回归。

### 前端 name 和后端 project 不一致，且节点不能从商店安装记录推断

- **现象**：点击拉取更新返回 `NodeID`、`Project` 的 required 校验错误；外部编排项目在远程节点操作时可能回落本机。
- **根因**：原前端发 `{name, dir}`，后端绑定 `{nodeId, project}`；节点查找只依赖商店安装记录，外部项目没有对应记录。NodeScope 中间件检查查询参数，单独信任请求体的 nodeId 还会造成校验节点与实际执行节点不一致。
- **规避/解决**：前后端共享明确的 DTO，兼容旧 name 别名，统一当前 Docker 节点；查询参数和请求体冲突时拒绝；服务层对归一化后的实际节点再检查调用者范围，HTTP 与 AI 共用该入口。
- **来源**：2026-10-30，编排更新方案 A；`shared/dto/compose_update.go`、`core/internal/api/compose.go`、`core/internal/service/compose_update.go`、前端 `api/modules/compose.ts`。

### agent 执行信封成功不等于命令成功，镜像比较必须使用运行容器的实际 ID

- **现象**：Docker 拉取失败仍执行重建，任务显示成功；拉取前后比较 `compose images` 又可能一直判断无需更新。
- **根因**：`/agent/v1/exec` 信封成功表示命令已执行，命令失败通过 `ExitCode`、`TimedOut` 单独返回；`compose images` 显示已有容器的镜像，pull 不会改变这些容器。原拉取参数 `--ignore-buildable-images` 也不受 Compose 支持。
- **规避/解决**：显式检查命令退出码和超时，pull 失败立即中止。正确参数为 `pull --ignore-buildable --policy always`；先读取运行容器 `.Image`，再与拉取后镜像引用的实际 ID 比较，仅对变化的运行服务执行 `up -d --no-deps --no-build --pull never`。跳过 one-off 容器和停止的服务，多文件项目保留 config_files 标签中的文件顺序，并使用项目工作目录。任务结束后写回实际摘要，手动和自动更新按同节点、同项目互斥。
- **来源**：2026-10-30，`core/internal/service/compose_update.go`；Docker 官方 [pull](https://docs.docker.com/reference/cli/docker/compose/pull/) / [up](https://docs.docker.com/reference/cli/docker/compose/up/) 文档与执行回归测试。

### FaModal 不能照搬 Arco 的 @ok，异步保存和状态刷新要保留表单

- **现象**：自动更新设置弹窗点击确定不调用保存；刷新最近执行记录时，未保存的脚本和开关被覆盖。
- **根因**：FaModal 使用 `confirm` / `confirmButtonLoading`，原弹窗却使用 `ok` / `confirm-loading`；默认确定动作会关闭弹窗，不能承载保存失败后继续编辑的交互。状态刷新复用完整加载函数会重新初始化表单。
- **规避/解决**：异步保存用 FaModal 的 footer 插槽自行管理 loading 与关闭，读取失败禁用保存并允许重试，保存失败保留输入；执行状态刷新只更新状态数据。目标切换和关闭时使旧请求失效，避免迟到响应覆盖新目标；运行中轮询在任务完成或出错后停止。
- **来源**：2026-10-30，前端 `views/container/components/ComposeAutoUpdateModal.vue`；FaModal 源码 `web/packages/components/src/basic/modal/`。

### FaInput/FaTextarea 的透传属性可能同时落到外层和输入元素

- **现象**：为输入组件设置 id 后，页面出现重复 id，label 或浏览器自动化定位可能匹配到外层容器；按 placeholder 定位也可能匹配多个元素。
- **根因**：当前封装保留 Vue 默认属性继承，同时向内部输入元素透传 attrs，导致部分属性出现在两层 DOM 上。
- **规避/解决**：业务表单使用可见字段标题和 aria-label，为自动化定位组合输入角色与可访问名称；不要把透传 id 当作唯一输入节点。若要从根本修复，需单独确认后修改 fa 基础封装。
- **来源**：2026-10-30，自动更新弹窗桌面/移动端浏览器验收；FaInput/FaTextarea 源码与 DOM 实测。

### 自动更新首次保存 false 会被 GORM default:true 改成开启

- **现象**：用户第一次保存关闭设置，数据库中却是 enabled=true。
- **根因**：GORM Create 对带非零 default 的零值字段使用默认值；这与已有 GORM 零值经验中的角色范围问题相同。
- **规避/解决**：自动更新模型默认关闭，新请求显式保存 enabled；更新配置使用 map 保留 false 和空脚本，仅更新配置列，不覆盖任务并发写回的检查时间和结果。新增失败策略列的迁移默认值保留旧配置的 continue 行为，新配置由服务入口明确默认 stop。
- **来源**：2026-10-30，`model.ComposeAutoUpdate`、`SaveAutoUpdate` 及持久化回归测试；参见 [GORM 零值经验](./go-backend.md)。

### 镜像更新还要核对运行容器与本地编排的配置指纹

- **现象**：只比较镜像 ID 再执行 up，仍会顺带应用磁盘里尚未部署的端口、挂载、环境变量和镜像标签变更。
- **根因**：Compose up 同时应用配置与镜像；「只更新当前运行镜像」必须限制配置漂移。商店包升级与镜像更新也可能同时操作一个项目。
- **规避/解决**：比较 config --hash 与运行容器的 com.docker.compose.config-hash，在 pull 前和前脚本执行后各检查一次；缺失指纹或不一致都中止并指引部署配置/商店升级。安装、卸载、参数部署、手动及自动镜像更新共用同节点同项目任务互斥；读取旧参数在任务创建后完成，避免查到刚结束部署前的旧值。纯镜像 pull 仅下载，不承担运行更新或商店升版。
- **来源**：2026-10-30，compose_update.go、TaskService.StartProjectTask；真实隔离容器确认 CLI hash 与运行标签相等；配置漂移和任务互斥回归。
