# yp-main 种子源（YPanel 应用源骨架）

YPanel 自有源格式的参考实现，可直接推到任意 git 托管（github / gitee / gitlab / 自建 git），
然后在面板「应用商店 → 源管理」把内置的「YPanel 官方源」编辑为该仓库地址并同步。

## 目录结构

```
index.json                     # 源清单（唯一入口）
apps/<id>/logo.svg             # 图标（svg/png/jpg/webp）
apps/<id>/README.md            # 应用介绍（详情抽屉渲染 Markdown）
apps/<id>/<version>/package/   # 版本包目录（compose.yml + 附属文件，纯文本）
```

## index.json 字段

| 字段 | 说明 |
|---|---|
| `apps[].id` | 源内唯一 key（小写字母/数字/中划线） |
| `apps[].kind` | `app` 普通应用 / `service` 环境服务（面板功能可引用）/ `middleware` 可共享复用中间件 |
| `apps[].category` | 分类（面板按源聚合下发到分类筛选） |
| `apps[].logo` / `apps[].readme` | 仓库相对路径（同步时读入并入库） |
| `apps[].reverseProxy` | 一键反代端口 env key（如 `PANEL_APP_PORT`）；安装向导可填域名，安装后自动创建反代站点 |
| `apps[].versions[]` | 新版本放前面（第一条 = 最新，升级判定依据） |
| `versions[].package` | 仓库内包目录（git 源）或 tar.gz URL（远程源） |
| `versions[].env[]` | 参数定义 `{key,label,type(text/number/password/select),default,required,rule}` |
| `versions[].ports[]` | 端口定义 `{envKey,default}`（自动并入表单字段） |

## compose.yml 约定

- 变量渲染：`${VAR}`，来源 = 参数定义 default ∪ 用户输入；`CONTAINER_NAME` 自动注入（= `app-<应用名>`）
- 网络统一接入 external 网络 `1panel-network`（与面板 nginx/站点互通，反代可用容器名直连）
- 版本包内文件统一纯文本（compose/配置模板），单文件 ≤2MB、总数 ≤200
- compose 文件名支持 `compose.yml` / `docker-compose.yml` 等，部署时统一落为 `docker-compose.yml`

## 本地联调

```bash
cd deploy/store-seed && git init && git add -A && git commit -m "yp-main seed"
# 面板源管理 → 编辑「YPanel 官方源」→ URL 填 file:///abs/path/deploy/store-seed → 同步
```
