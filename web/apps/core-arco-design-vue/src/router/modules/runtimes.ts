import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/runtimes',
  component: Layout,
  name: 'runtimes',
  meta: {
    title: 'menu.runtimes',
    icon: 'yd:file-code',
  },
  children: [
    {
      path: '',
      name: 'runtimesIndex',
      component: () => import('@/views/runtimes/index.vue'),
      meta: {
        title: 'menu.runtimes',
        menu: false,
      },
    },
  ],
}

export default routes
