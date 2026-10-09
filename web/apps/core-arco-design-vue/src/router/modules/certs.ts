import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/certs',
  component: Layout,
  name: 'certs',
  meta: {
    title: 'menu.certs',
    icon: 'yd:shield',
  },
  children: [
    {
      path: '',
      name: 'certsIndex',
      component: () => import('@/views/certs/index.vue'),
      meta: {
        title: 'menu.certs',
        // fa 单页约定：主导航平铺直达，无二级展开
        menu: false,
      },
    },
  ],
}

export default routes
