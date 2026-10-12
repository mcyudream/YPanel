import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

// M68 可复用中间件页：容器域二级叶子（单子页模块形态，与容器域其他 Tab 同款）
export const ContainerMiddleware: RouteRecordRaw = {
  path: '/container/middleware',
  component: Layout,
  name: 'containerMiddleware',
  meta: {
    title: 'menu.containerMiddleware',
    icon: 'i-lucide:blocks',
  },
  children: [
    {
      path: '',
      name: 'containerMiddlewareIndex',
      component: () => import('@/views/middleware/index.vue'),
      meta: {
        title: 'menu.containerMiddleware',
        icon: 'i-lucide:blocks',
        menu: false,
      },
    },
  ],
}

export default ContainerMiddleware
