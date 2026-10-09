# yudream-web-os（vendored）

YuDream 自研桌面工作台 webos 库（YPanel 的 `/desktop` 承载层）的**源码快照**，vendor 进本仓库使 CI/第三方可独立构建。

- 开发源头：本地兄弟仓库 `../yudream-web-os`（独立 monorepo）
- 同步方式：源头修复后复制对应包 `src/` 覆盖此处（`packages/{vue,arco,core,shared,webos}`）
- 引用方式：vite alias / tsconfig paths 指向 `packages/*/src` 目录（rolldown alias 目标按目录解析）
