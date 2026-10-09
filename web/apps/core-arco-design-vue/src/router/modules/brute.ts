import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/bruteforce',
  component: Layout,
  name: 'bruteforce',
  meta: {
    title: 'menu.bruteForce',
    icon: 'i-lucide:shield-ban',
    auth: ['tool:bruteforce'],
  },
  children: [
    {
      path: '',
      name: 'bruteforceIndex',
      component: () => import('@/views/tools/bruteforce.vue'),
      meta: {
        title: 'menu.bruteForce',
        menu: false,
      },
    },
  ],
}

export default routes
