import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/compose',
  component: Layout,
  name: 'compose',
  meta: {
    title: 'Compose 编排',
    icon: 'yd:layers',
  },
  children: [
    {
      path: '',
      name: 'composeIndex',
      component: () => import('@/views/compose/index.vue'),
      meta: {
        title: 'Compose 编排',
      },
    },
  ],
}

export default routes
