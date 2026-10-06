import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/alert',
  component: Layout,
  name: 'alert',
  meta: {
    title: '告警通知',
    icon: 'yd:bell',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'alertIndex',
      component: () => import('@/views/alert/index.vue'),
      meta: {
        title: '告警通知',
        menu: false,
      },
    },
  ],
}

export default routes
