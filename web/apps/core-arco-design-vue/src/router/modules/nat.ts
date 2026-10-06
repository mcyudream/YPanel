import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/nat',
  component: Layout,
  name: 'nat',
  meta: {
    title: 'NAT 转发',
    icon: 'yd:network',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'natIndex',
      component: () => import('@/views/nat/index.vue'),
      meta: {
        title: 'NAT 转发',
        menu: false,
      },
    },
  ],
}

export default routes
