import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/store',
  component: Layout,
  name: 'store',
  meta: {
    title: 'menu.store',
    icon: 'yd:package',
  },
  children: [
    {
      path: '',
      name: 'storeIndex',
      component: () => import('@/views/store/index.vue'),
      meta: {
        title: 'menu.store',
        menu: false,
      },
    },
  ],
}

export default routes
