import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/sites',
  component: Layout,
  name: 'sites',
  meta: {
    title: '网站',
    icon: 'yd:globe',
  },
  children: [
    {
      path: '',
      name: 'sitesIndex',
      component: () => import('@/views/sites/index.vue'),
      meta: {
        title: '网站',
      },
    },
  ],
}

export default routes
