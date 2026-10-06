import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/ai',
  component: Layout,
  name: 'ai',
  meta: {
    title: '智能',
    icon: 'yd:sparkles',
  },
  children: [
    {
      path: '',
      name: 'aiIndex',
      component: () => import('@/views/ai/index.vue'),
      meta: {
        title: 'AI 助手',
        menu: false,
      },
    },
  ],
}

export default routes
