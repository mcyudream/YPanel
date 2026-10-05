import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/container',
  component: Layout,
  name: 'container',
  meta: {
    title: '容器管理',
    icon: 'yd:container',
  },
  children: [
    {
      path: '',
      name: 'containerIndex',
      component: () => import('@/views/container/index.vue'),
      meta: {
        title: '容器管理',
      },
    },
  ],
}

export default routes
