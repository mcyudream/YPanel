import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/market',
  component: Layout,
  name: 'market',
  meta: {
    title: '应用市场',
    icon: 'yd:package',
  },
  children: [
    {
      path: '',
      name: 'marketIndex',
      component: () => import('@/views/market/index.vue'),
      meta: {
        title: '应用市场',
        menu: false,
      },
    },
  ],
}

export default routes
