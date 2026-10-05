import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/icons',
  component: Layout,
  name: 'icons',
  meta: {
    title: '图标库',
    icon: 'yd:shapes',
  },
  children: [
    {
      path: '',
      name: 'iconsIndex',
      component: () => import('@/views/icons/index.vue'),
      meta: {
        title: '图标库',
      },
    },
  ],
}

export default routes
