import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/nodes',
  component: Layout,
  name: 'nodes',
  meta: {
    title: '节点管理',
    icon: 'yd:network',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'nodesIndex',
      component: () => import('@/views/nodes/index.vue'),
      meta: {
        title: '节点管理',
      },
    },
  ],
}

export default routes
