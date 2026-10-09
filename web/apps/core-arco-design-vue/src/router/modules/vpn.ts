import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/vpn',
  component: Layout,
  name: 'vpn',
  meta: {
    title: 'menu.vpn',
    icon: 'i-lucide:network',
    auth: ['tool:vpn'],
  },
  children: [
    {
      path: '',
      name: 'vpnIndex',
      component: () => import('@/views/vpn/index.vue'),
      meta: {
        title: 'menu.vpn',
        menu: false,
      },
    },
  ],
}

export default routes
