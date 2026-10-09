import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/terminal',
  component: Layout,
  name: 'terminal',
  meta: {
    title: 'menu.terminal',
    icon: 'yd:square-terminal',
    auth: ['terminal:access'],
  },
  children: [
    {
      path: '',
      name: 'terminalIndex',
      component: () => import('@/views/terminal/index.vue'),
      meta: {
        title: 'menu.terminal',
        menu: false,
      },
    },
  ],
}

export default routes
