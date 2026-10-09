# yp-main 官方源已迁至独立仓库

YPanel 官方应用源（yp 格式）的种子内容已迁至独立 git 仓库维护，本目录不再保留副本（避免两份漂移）：

- GitHub：<https://github.com/mcyudream/YPanel-AppStore>
- Gitee（国内推荐）：<https://gitee.com/mcyudream/YPanel-AppStore>

## 维护方式

- 独立仓库本地挂双 remote（origin=GitHub、gitee=Gitee），每次更新同时推送两边：
  `git push origin main && git push gitee main`
- yp 格式规范（目录结构 / index.json 字段 / compose 约定）以该仓库 README 为准
- 面板接入：应用商店 → 源管理 →「YPanel 官方源」（yp-git，内置默认 Gitee 地址）
