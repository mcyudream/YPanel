import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/probe',
  component: Layout,
  name: 'probe',
  meta: {
    title: 'menu.probe',
    icon: 'i-lucide:radar',
  },
  children: [
    {
      path: '',
      name: 'probeIndex',
      component: () => import('@/views/probe/index.vue'),
      meta: {
        title: 'menu.probe',
        menu: false,
      },
    },
  ],
}

export default routes
