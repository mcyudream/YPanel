import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/ssh',
  component: Layout,
  name: 'ssh',
  meta: {
    title: 'menu.sshManage',
    icon: 'i-lucide:terminal',
    auth: ['admin'],
  },
  children: [
    {
      path: '',
      name: 'sshIndex',
      component: () => import('@/views/tools/ssh.vue'),
      meta: {
        title: 'menu.sshManage',
        menu: false,
      },
    },
  ],
}

export default routes
