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
    icon: 'i-tabler:brand-docker',
  },
  children: [
    {
      path: '',
      name: 'containerIndex',
      component: () => import('@/views/container/index.vue'),
      meta: {
        title: '容器',
        // fa 单页约定（同 ai.ts）：子页 menu:false，主导航点击直达，无二级菜单
        menu: false,
      },
    },
    // 详情页走独立顶级模块（container-detail.ts / container-app.ts），
    // 避免 fa 多标签下同父兄弟子路由切换不渲染的问题
  ],
}

export default routes
