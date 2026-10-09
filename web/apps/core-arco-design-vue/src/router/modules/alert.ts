import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/alert',
  component: Layout,
  name: 'alert',
  meta: {
    title: 'menu.alert',
    icon: 'yd:bell',
    auth: ['alert:read'],
  },
  children: [
    {
      path: '',
      name: 'alertIndex',
      component: () => import('@/views/alert/index.vue'),
      meta: {
        title: 'menu.alert',
        menu: false,
      },
    },
  ],
}

export default routes
