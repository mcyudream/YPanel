import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/processes',
  component: Layout,
  name: 'processes',
  meta: {
    title: '进程与服务',
    icon: 'yd:cpu',
  },
  children: [
    {
      path: '',
      name: 'processesIndex',
      component: () => import('@/views/processes/index.vue'),
      meta: {
        title: '进程与服务',
        menu: false,
      },
    },
  ],
}

export default routes
