import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/panelbackup',
  component: Layout,
  name: 'panelbackup',
  meta: {
    title: '面板备份',
    icon: 'yd:save',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'panelbackupIndex',
      component: () => import('@/views/notifications/index.vue'),
      meta: {
        title: '面板备份',
      },
    },
  ],
}

export default routes
