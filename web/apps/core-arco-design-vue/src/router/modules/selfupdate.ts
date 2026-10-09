import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/selfupdate',
  component: Layout,
  name: 'selfupdate',
  meta: {
    title: 'menu.selfupdate',
    icon: 'yd:refresh-cw',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'selfupdateIndex',
      component: () => import('@/views/selfupdate/index.vue'),
      meta: {
        title: 'menu.selfupdate',
        menu: false,
      },
    },
  ],
}

export default routes
