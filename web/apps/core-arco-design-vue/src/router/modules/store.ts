import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/store',
  component: Layout,
  name: 'store',
  meta: {
    title: '应用商店',
    icon: 'yd:package',
  },
  children: [
    {
      path: '',
      name: 'storeIndex',
      component: () => import('@/views/store/index.vue'),
      meta: {
        title: '应用商店',
      },
    },
  ],
}

export default routes
