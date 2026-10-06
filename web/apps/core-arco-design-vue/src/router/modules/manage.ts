import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/manage',
  component: Layout,
  name: 'manage',
  meta: {
    title: '系统管理',
    menu: false,
  },
  children: [
    {
      path: 'user',
      name: 'manageUser',
      component: () => import('@/views/manage/user.vue'),
      meta: {
        title: '用户管理',
        icon: 'yd:users-round',
        auth: ['admin'],
      },
    },
    {
      path: 'audit',
      name: 'manageAudit',
      component: () => import('@/views/manage/audit.vue'),
      meta: {
        title: '登录审计',
        icon: 'yd:scroll-text',
        auth: ['admin'],
      },
    },
    {
      path: 'security',
      name: 'manageSecurity',
      component: () => import('@/views/manage/security.vue'),
      meta: {
        title: '安全设置',
        icon: 'yd:shield',
        auth: ['admin'],
      },
    },
    {
      path: 'backups',
      name: 'manageBackups',
      component: () => import('@/views/manage/backups.vue'),
      meta: {
        title: '面板备份',
        icon: 'yd:save',
        auth: ['admin'],
      },
    },
  ],
}

export default routes
