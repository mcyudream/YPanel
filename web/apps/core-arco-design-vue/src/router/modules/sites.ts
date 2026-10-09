import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/sites',
  component: Layout,
  name: 'sites',
  meta: {
    title: 'menu.sites',
    icon: 'yd:globe',
  },
  children: [
    {
      path: '',
      name: 'sitesIndex',
      component: () => import('@/views/sites/index.vue'),
      meta: {
        title: 'menu.sites',
        // fa 单页约定：主导航平铺直达，无二级展开
        menu: false,
      },
    },
    {
      path: ':id',
      name: 'sitesDetail',
      component: () => import('@/views/sites/detail.vue'),
      meta: {
        title: 'menu.siteConfig',
        menu: false,
        activeMenu: '/sites',
      },
    },
  ],
}

export default routes
