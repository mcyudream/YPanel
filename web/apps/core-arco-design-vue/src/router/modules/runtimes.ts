import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/runtimes',
  component: Layout,
  name: 'runtimes',
  meta: {
    title: '运行环境',
    icon: 'yd:file-code',
  },
  children: [
    {
      path: '',
      name: 'runtimesIndex',
      component: () => import('@/views/runtimes/index.vue'),
      meta: {
        title: '运行环境',
        menu: false,
      },
    },
  ],
}

export default routes
