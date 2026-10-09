import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/logs',
  component: Layout,
  name: 'logcenter',
  meta: {
    title: 'menu.logcenter',
    icon: 'i-lucide:scroll-text',
    auth: ['log:read'],
  },
  children: [
    {
      path: '',
      name: 'logcenterIndex',
      component: () => import('@/views/logcenter/index.vue'),
      meta: {
        title: 'menu.logcenter',
        menu: false,
      },
    },
  ],
}

export default routes
