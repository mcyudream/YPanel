import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/plugin/db-admin',
  component: Layout,
  name: 'dbAdmin',
  meta: {
    title: '数据库管理台',
    icon: 'yd:database',
  },
  children: [
    {
      path: '',
      name: 'dbAdminIndex',
      component: () => import('@/views/dbadmin/index.vue'),
      meta: {
        title: '数据库管理台',
        menu: false,
      },
    },
  ],
}

export default routes
