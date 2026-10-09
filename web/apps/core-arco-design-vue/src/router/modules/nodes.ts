import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/nodes',
  component: Layout,
  name: 'nodes',
  meta: {
    title: 'menu.nodes',
    icon: 'yd:network',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'nodesIndex',
      component: () => import('@/views/nodes/index.vue'),
      meta: {
        title: 'menu.nodes',
        menu: false,
      },
    },
  ],
}

export default routes
