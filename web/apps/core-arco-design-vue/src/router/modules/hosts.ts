import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/hosts',
  component: Layout,
  name: 'hosts',
  meta: {
    title: 'menu.hosts',
    icon: 'yd:book-user',
    auth: ['tool:hosts'],
  },
  children: [
    {
      path: '',
      name: 'hostsIndex',
      component: () => import('@/views/hosts/index.vue'),
      meta: {
        title: 'menu.hosts',
        menu: false,
      },
    },
  ],
}

export default routes
