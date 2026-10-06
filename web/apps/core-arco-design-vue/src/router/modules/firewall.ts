import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/firewall',
  component: Layout,
  name: 'firewall',
  meta: {
    title: '防火墙',
    icon: 'yd:shield',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'firewallIndex',
      component: () => import('@/views/firewall/index.vue'),
      meta: {
        title: 'firewall',
      },
    },
  ],
}

export default routes
