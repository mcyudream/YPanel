import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/ftp',
  component: Layout,
  name: 'ftp',
  meta: {
    title: 'menu.ftp',
    icon: 'i-lucide:folder-sync',
    auth: ['tool:ftp'],
  },
  children: [
    {
      path: '',
      name: 'ftpIndex',
      component: () => import('@/views/tools/ftp.vue'),
      meta: {
        title: 'menu.ftp',
        menu: false,
      },
    },
  ],
}

export default routes
