import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/diskguard',
  component: Layout,
  name: 'diskguard',
  meta: {
    title: 'menu.diskguard',
    icon: 'yd:hard-drive',
    auth: ['host:manage'],
  },
  children: [
    {
      path: '',
      name: 'diskguardIndex',
      component: () => import('@/views/diskguard/index.vue'),
      meta: {
        title: 'menu.diskguard',
        menu: false,
      },
    },
  ],
}

export default routes
