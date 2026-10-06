import type { RouteRecordRaw } from 'vue-router'

// M23：Docker 管理并入 /container 模块（镜像/网络/卷/配置 tab），旧路由重定向兼容。
const routes: RouteRecordRaw = {
  path: '/docker',
  name: 'docker',
  redirect: { path: '/container', query: { tab: 'images' } },
  meta: {
    title: 'Docker 管理',
    menu: false,
  },
  children: [],
}

export default routes
