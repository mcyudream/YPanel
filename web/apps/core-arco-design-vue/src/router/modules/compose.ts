import type { RouteRecordRaw } from 'vue-router'

// M23：Compose 编排并入 /container 模块（应用 tab），旧路由重定向兼容。
const routes: RouteRecordRaw = {
  path: '/compose',
  name: 'compose',
  redirect: { path: '/container', query: { tab: 'apps' } },
  meta: {
    title: 'Compose 编排',
    menu: false,
  },
  children: [],
}

export default routes
