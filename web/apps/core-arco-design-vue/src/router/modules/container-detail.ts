import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

// 容器详情（M23）：独立顶级路由模块。
// 不能作为 /container 的 children——fa 多标签 keepAlive 下同父兄弟子路由切换时 RouterView 不换组件
// （这也是 M20 文件工作台选弹窗形态的原因）；独立父记录则切换正常，URL 保持 /container/detail/:id。
const routes: RouteRecordRaw = {
  path: '/container/detail/:id',
  component: Layout,
  name: 'containerDetailModule',
  meta: {
    title: 'menu.containerDetail',
    menu: false,
  },
  children: [
    {
      path: '',
      name: 'containerDetail',
      component: () => import('@/views/container/detail.vue'),
      meta: {
        title: 'menu.containerDetail',
        menu: false,
      },
    },
  ],
}

export default routes
