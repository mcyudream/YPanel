# 经验之谈：商店升级与应用身份

### 商店版本、主应用标签与伴生组件标签各自发布

- **现象**：旧包版本为 1.0，主镜像使用 latest；沿用旧 TAG 会让新包仍跑浮动版本，所有服务统一用 1.12.0 又会拉不到 render-server。
- **根因**：商店包、主应用、伴生组件各自发布；制品平台 unknown/unknown 引用是构建证明，不能当 ARM64 镜像。
- **规避/解决**：逐制品验证标签和平台，目标版本用 versionParams 显式覆盖旧标签。YuDream Admin backend/frontend 固定 1.12.0，render-server 独立使用 31de144a，主应用仅声明 amd64。新清单能力与支持它的面板一起交付，旧面板不会因同步源获得新能力。
- **来源**：2026-10-30，Harbor artifact API、上游发布脚本、store.go 的 manifest 转换。

### 仅保留 data 会丢插件，旧容器漏挂的配置也要先导出

- **现象**：旧 YuDream Admin 插件位于项目根 plugins/、market-source/；引导配置只在容器内 /app/config。旧重装先 down 再清理除 data 外全部内容，会丢失这些文件。
- **根因**：持久化源没有统一固定在 data。逐文件复制时若每次判断「目标根目录存在则跳过」，首个文件写入又会阻止其余文件复制。
- **规避/解决**：先独立准备、校验、预拉新包，保护新旧 bind 源和其他用户文件；缺失挂载源整棵初始化。包可显式声明从旧容器导出配置，导出结果按用户数据保留，失败恢复也不能删除。仅用 rmdir 修复确认空的同名文件目录，非空目录或符号链接中止部署。失败仅恢复本次替换的部署文件并尝试启动旧编排，不能撤销数据库迁移。
- **来源**：2026-10-30，store_deployment.go；真实 shell 回归覆盖多文件恢复、插件、市场目录、失败恢复和符号链接。

### .env 单引号不能保留尾反斜杠，randomLen 也不能表达 Base64 密钥契约

- **现象**：单引号封装的参数末尾带反斜杠时，Compose 报 unterminated quoted value。旧 sanitize 还会删掉密码中的美元符、分号等字符。主密钥要求 Base64 解码后恰好 32 字节，旧随机长度不能保证。
- **根因**：Compose dotenv 与 shell quoting 不同；删字符会改变凭据，字符数也不等于解码字节数。
- **规避/解决**：新值用双引号转义反斜杠、双引号、美元符，拒绝 NUL/换行。旧 .env 保留原文，只追加显式修改、新字段和目标版本参数。主密钥先生成 32 随机字节再标准 Base64；选填且未声明随机的 Redis/邮件密码允许留空。检查 compose config --environment 和隔离容器实际 Config.Env，不把 config JSON 为再次序列化而转义的 $$ 误认成真实凭据。
- **来源**：2026-10-30，真实 Docker Compose 与隔离 Python 容器验证；TestStoreParametersAndEnvContract、TestStoreUpgradeRestoresCredentialsAndOptions。

### 共享 backend 别名可能串实例，可选 nginx upstream 又会阻止启动

- **现象**：多实例都在共享网络声明 backend，固定 backend:8080 可能连错实例；nginx 静态引用 kkfileview，未勾选它也可能无法启动。
- **根因**：Compose 服务名自动成为网络别名；nginx 加载静态 upstream 时解析域名。容器名、项目名与宿主发布端口也不能互相替代。
- **规避/解决**：稳定保留项目身份，容器名采用实例前缀加服务名，标签记录 app/instance/project/node/service。前端和伴生服务只接项目私有网络，可选 upstream 请求时解析 DNS。一键反代用目标节点的宿主发布端口，本机回环由站点服务按 nginx 部署方式改写，远程用节点地址。注入标签必须合并包内已有 override。
- **来源**：2026-10-30，上游 nginx 与新商店包检查；在本机缓存前端镜像中验证无 backend/伴生服务时 nginx 启动和 SPA 回退；完整 1.12.0 运行验收受下条限制。

### Harbor 元数据可读不代表 registry 能拉镜像

- **现象**：匿名 API 可列出 1.12.0 的 digest/tag/platform，实际 Docker pull 却返回 401。空 Docker 配置仍失败，HTTPS token 虽声明 pull 权限，manifest 请求仍返回 Basic 鉴权挑战。
- **根因**：元数据 API 与 registry 拉取是不同链路。本次尚未定位服务端鉴权原因，不能归结为标签不存在或本机缓存凭据。
- **规避/解决**：分别记录制品检查和真实拉取，明确报告拉取阻塞。缺少运行验证不能宣称完整安装通过；不修改共享 Docker 凭据或擅自修改 registry 服务。
- **来源**：2026-10-30，registry.yudream.online 实际 pull、匿名 token/manifest 请求；不记录 token 内容。
- **补充（2026-10-31）**：使用用户提供的 robot 凭据，HTTPS token API 返回包含 pull 权限的令牌，但 Docker login、Basic 与 Bearer 的 manifest 请求仍返回上述 401。应检查 Harbor 内部鉴权链路，尚未通过服务端配置确认根因；不能据此声称用户凭据无效。YPanel v0.9.44 与商店包可以交付，镜像拉取及完整应用安装验收仍须单独跟进。
