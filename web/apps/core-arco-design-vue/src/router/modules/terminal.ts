import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/terminal',
  component: Layout,
  name: 'terminal',
  meta: {
    title: '终端',
    icon: 'yd:square-terminal',
  },
  children: [
    {
      path: '',
      name: 'terminalIndex',
      component: () => import('@/views/terminal/index.vue'),
      meta: {
        title: '终端',
      },
    },
  ],
}

export default routes
