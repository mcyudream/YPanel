import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/cron',
  component: Layout,
  name: 'cron',
  meta: {
    title: 'menu.cron',
    icon: 'yd:calendar-clock',
  },
  children: [
    {
      path: '',
      name: 'cronIndex',
      component: () => import('@/views/cron/index.vue'),
      meta: {
        title: 'menu.cron',
        menu: false,
      },
    },
  ],
}

export default routes
