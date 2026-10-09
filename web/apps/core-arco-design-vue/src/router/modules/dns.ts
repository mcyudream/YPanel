import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/dns',
  component: Layout,
  name: 'dns',
  meta: {
    title: 'menu.dns',
    icon: 'yd:radar',
    auth: ['tool:dns'],
  },
  children: [
    {
      path: '',
      name: 'dnsIndex',
      component: () => import('@/views/dns/index.vue'),
      meta: {
        title: 'menu.dns',
        menu: false,
      },
    },
  ],
}

export default routes
