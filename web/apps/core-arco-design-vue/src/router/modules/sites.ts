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
    {
      path: ':id',
      name: 'sitesDetail',
      component: () => import('@/views/sites/detail.vue'),
      meta: {
        title: '站点配置',
        menu: false,
        activeMenu: '/sites',
      },
    },
  ],
}

export default routes
