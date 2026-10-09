import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/monitor',
  component: Layout,
  name: 'monitor',
  meta: {
    title: 'menu.monitors',
    icon: 'yd:activity',
  },
  children: [
    {
      path: '',
      name: 'monitorIndex',
      component: () => import('@/views/manage/monitor.vue'),
      meta: {
        title: 'menu.monitors',
        menu: false,
      },
    },
  ],
}

export default routes
