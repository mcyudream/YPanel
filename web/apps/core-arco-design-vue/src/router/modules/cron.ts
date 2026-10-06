import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/cron',
  component: Layout,
  name: 'cron',
  meta: {
    title: '计划任务',
    icon: 'yd:calendar-clock',
  },
  children: [
    {
      path: '',
      name: 'cronIndex',
      component: () => import('@/views/cron/index.vue'),
      meta: {
        title: '计划任务',
      },
    },
  ],
}

export default routes
