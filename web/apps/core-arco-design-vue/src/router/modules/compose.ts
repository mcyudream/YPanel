import type { RouteRecordRaw } from 'vue-router'

// M23：Compose 编排并入 /container 模块（应用二级页），旧路由重定向兼容。
const routes: RouteRecordRaw = {
  path: '/compose',
  name: 'compose',
  redirect: { path: '/container/apps' },
  meta: {
    title: 'menu.compose',
    menu: false,
  },
  children: [],
}

export default routes
