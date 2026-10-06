import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/notifications',
  component: Layout,
  name: 'notifications',
  meta: {
    title: '通知中心',
    icon: 'yd:bell',
  },
  children: [
    {
      path: '',
      name: 'notificationsIndex',
      component: () => import('@/views/notifications/index.vue'),
      meta: {
        title: '通知中心',
      },
    },
  ],
}

export default routes
