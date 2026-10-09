import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

const routes: RouteRecordRaw = {
  path: '/manage',
  component: Layout,
  name: 'manage',
  meta: {
    title: 'menu.manage',
    icon: 'yd:settings',
  },
  children: [
    {
      path: 'user',
      name: 'manageUser',
      component: () => import('@/views/manage/user.vue'),
      meta: {
        title: 'menu.manageUsers',
        icon: 'yd:users-round',
        auth: ['user:manage'],
      },
    },
    {
      path: 'role',
      name: 'manageRole',
      component: () => import('@/views/manage/role.vue'),
      meta: {
        title: 'menu.manageRole',
        icon: 'yd:key-round',
        auth: ['user:manage'],
      },
    },
    {
      path: 'audit',
      name: 'manageAudit',
      component: () => import('@/views/manage/audit.vue'),
      meta: {
        title: 'menu.manageAudit',
        icon: 'yd:scroll-text',
        auth: ['audit:read'],
      },
    },
    {
      path: 'security',
      name: 'manageSecurity',
      component: () => import('@/views/manage/security.vue'),
      meta: {
        title: 'menu.manageSecurity',
        icon: 'yd:shield',
        auth: ['setting:write'],
      },
    },
    {
      path: 'backups',
      name: 'manageBackups',
      component: () => import('@/views/manage/backups.vue'),
      meta: {
        title: 'menu.manageBackup',
        icon: 'yd:save',
        auth: ['backup:manage'],
      },
    },
    {
      path: 'system',
      name: 'manageSystem',
      component: () => import('@/views/manage/system.vue'),
      meta: {
        title: 'menu.manage',
        icon: 'i-lucide:wrench',
        auth: ['setting:write'],
      },
    },
    {
      path: 'storage',
      name: 'manageStorage',
      component: () => import('@/views/manage/storage.vue'),
      meta: {
        title: 'menu.manageStorage',
        icon: 'yd:cloud-backup',
        auth: ['backup:manage'],
      },
    },
  ],
}

export default routes
