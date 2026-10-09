import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/database',
  component: Layout,
  name: 'database',
  meta: {
    title: 'menu.database',
    icon: 'yd:database',
  },
  children: [
    {
      path: '',
      name: 'databaseIndex',
      component: () => import('@/views/database/index.vue'),
      meta: {
        title: 'menu.database',
        menu: false,
      },
    },
  ],
}

export default routes
