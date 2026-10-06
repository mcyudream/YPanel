import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/docker',
  component: Layout,
  name: 'docker',
  meta: {
    title: 'Docker 管理',
    icon: 'yd:container',
  },
  children: [
    {
      path: '',
      name: 'dockerIndex',
      component: () => import('@/views/docker/index.vue'),
      meta: {
        title: 'Docker 管理',
      },
    },
  ],
}

export default routes
