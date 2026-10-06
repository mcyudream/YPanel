import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/certs',
  component: Layout,
  name: 'certs',
  meta: {
    title: '证书',
    icon: 'yd:shield',
  },
  children: [
    {
      path: '',
      name: 'certsIndex',
      component: () => import('@/views/certs/index.vue'),
      meta: {
        title: '证书',
      },
    },
  ],
}

export default routes
