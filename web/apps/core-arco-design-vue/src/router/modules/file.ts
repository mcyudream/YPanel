import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/file_management',
  component: Layout,
  name: 'fileManagement',
  meta: {
    title: 'menu.file',
    icon: 'yd:folder-open',
  },
  children: [
    {
      path: '',
      name: 'fileManagementIndex',
      component: () => import('@/views/file_management/index.vue'),
      meta: {
        title: 'menu.file',
        menu: false,
      },
    },
  ],
}

export default routes
