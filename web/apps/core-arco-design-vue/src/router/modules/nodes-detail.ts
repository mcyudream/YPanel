import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

// 节点详情（M25）：独立顶级路由模块。
// 不能作为 /nodes 的 children——fa 多标签 keepAlive 下同父兄弟子路由切换时 RouterView 不换组件
// （见 container-detail.ts 注释）；独立父记录则切换正常，URL 保持 /nodes/detail/:id。
const routes: RouteRecordRaw = {
  path: '/nodes/detail/:id',
  component: Layout,
  name: 'nodesDetailModule',
  meta: {
    title: '节点详情',
    menu: false,
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'nodesDetail',
      component: () => import('@/views/nodes/detail.vue'),
      meta: {
        title: '节点详情',
        menu: false,
      },
    },
  ],
}

export default routes
