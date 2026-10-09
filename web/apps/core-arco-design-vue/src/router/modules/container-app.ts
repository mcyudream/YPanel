import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

// 应用详情（M23）：独立顶级路由模块（原因同 container-detail.ts）。
const routes: RouteRecordRaw = {
  path: '/container/app/:project',
  component: Layout,
  name: 'containerAppDetailModule',
  meta: {
    title: 'menu.containerAppDetail',
    menu: false,
  },
  children: [
    {
      path: '',
      name: 'containerAppDetail',
      component: () => import('@/views/container/app-detail.vue'),
      meta: {
        title: 'menu.containerAppDetail',
        menu: false,
      },
    },
  ],
}

export default routes
