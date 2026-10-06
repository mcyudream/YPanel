import type { RouteRecordMainRaw } from '@fantastic-admin/types'
import type { RouteRecordRaw } from 'vue-router'
import ComposeRoutes from './modules/compose'
import DatabaseRoutes from './modules/database'
import SitesRoutes from './modules/sites'
import ContainerRoutes from './modules/container'
import CronRoutes from './modules/cron'
import FileRoutes from './modules/file'
import IconsRoutes from './modules/icons'
import ManageRoutes from './modules/manage'
import NodesRoutes from './modules/nodes'
import MarketRoutes from './modules/market'
import FirewallRoutes from './modules/firewall'
import AlertRoutes from './modules/alert'
import TerminalRoutes from './modules/terminal'

// 固定路由（默认路由）
const constantRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login.vue'),
    meta: {
      title: '登录',
    },
  },
  {
    path: '/desktop',
    name: 'desktop',
    component: () => import('@/views/desktop/index.vue'),
    meta: {
      title: '桌面工作台',
      breadcrumb: false,
    },
  },
  {
    path: '/:all(.*)*',
    name: 'notFound',
    component: () => import('@/views/[...all].vue'),
    meta: {
      title: '找不到页面',
    },
  },
]

// 系统路由
const systemRoutes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/index.vue'),
    meta: {
      breadcrumb: false,
    },
    children: [
      {
        path: '',
        name: 'overview',
        component: () => import('@/views/overview/index.vue'),
        meta: {
          title: '主机概览',
          icon: 'yd:gauge',
          breadcrumb: false,
        },
      },
      {
        path: 'reload',
        name: 'reload',
        component: () => import('@/views/reload.vue'),
        meta: {
          title: '重新加载中...',
          breadcrumb: false,
        },
      },
    ],
  },
]

// 动态路由（异步路由、导航菜单路由）
const asyncRoutes: RouteRecordMainRaw[] = [
  {
    meta: {
      title: '资源管理',
      icon: 'yd:layers',
    },
    children: [
      SitesRoutes,
      DatabaseRoutes,
      ComposeRoutes,
      FileRoutes,
      ContainerRoutes,
      TerminalRoutes,
      CronRoutes,
    ],
  },
  {
    meta: {
      title: '外观与扩展',
      icon: 'yd:shapes',
    },
    children: [
      IconsRoutes,
    ],
  },
  {
    meta: {
      title: '系统管理',
      icon: 'yd:shield-check',
      auth: ['admin'],
    },
    children: [
      ManageRoutes,
      NodesRoutes,
      MarketRoutes,
      FirewallRoutes,
      AlertRoutes,
    ],
  },
]

export {
  asyncRoutes,
  constantRoutes,
  systemRoutes,
}
