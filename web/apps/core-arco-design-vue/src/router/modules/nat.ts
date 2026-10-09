import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/nat',
  component: Layout,
  name: 'nat',
  meta: {
    title: 'menu.nat',
    icon: 'yd:network',
    auth: ['tool:nat'],
  },
  children: [
    {
      path: '',
      name: 'natIndex',
      component: () => import('@/views/nat/index.vue'),
      meta: {
        title: 'menu.nat',
        menu: false,
      },
    },
  ],
}

export default routes
