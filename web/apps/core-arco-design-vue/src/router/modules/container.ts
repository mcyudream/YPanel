import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/container',
  component: Layout,
  name: 'container',
  meta: {
    title: '容器',
    icon: 'i-logos:docker-icon',
  },
  children: [
    {
      path: '',
      name: 'containerIndex',
      component: () => import('@/views/container/index.vue'),
      meta: {
        title: '容器',
      },
    },
    {
      path: 'detail/:id',
      name: 'containerDetail',
      component: () => import('@/views/container/detail.vue'),
      meta: {
        title: '容器详情',
        menu: false,
      },
    },
    {
      path: 'app/:project',
      name: 'containerAppDetail',
      component: () => import('@/views/container/app-detail.vue'),
      meta: {
        title: '应用详情',
        menu: false,
      },
    },
  ],
}

export default routes
