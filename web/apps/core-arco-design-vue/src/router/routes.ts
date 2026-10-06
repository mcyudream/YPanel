import type { RouteRecordMainRaw } from '@fantastic-admin/types'
import type { RouteRecordRaw } from 'vue-router'
import ComposeRoutes from './modules/compose'
import DatabaseRoutes from './modules/database'
import SitesRoutes from './modules/sites'
import ContainerRoutes from './modules/container'
import CronRoutes from './modules/cron'
import FileRoutes from './modules/file'
import MonitorsRoutes from './modules/monitors'
import AIRoutes from './modules/ai'
import ManageRoutes from './modules/manage'
import NodesRoutes from './modules/nodes'
import MarketRoutes from './modules/market'
import FirewallRoutes from './modules/firewall'
import AlertRoutes from './modules/alert'
import NotificationRoutes from './modules/notifications'
import ProcessesRoutes from './modules/processes'
import DbAdminRoutes from './modules/dbadmin'
import StoreRoutes from './modules/store'
import RuntimesRoutes from './modules/runtimes'
import DockerRoutes from './modules/docker'
import SelfUpdateRoutes from './modules/selfupdate'
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
// 一级大菜单统一 2 字命名；单页功能用"同名分组 + 子页 menu:false"配合
// mainMenuClickMode=smart 实现主导航点击直达；强关联功能按域分组。
const single = (title: string, icon: string, mod: RouteRecordRaw) => ({
  meta: { title, icon },
  children: [mod],
})

const asyncRoutes: RouteRecordMainRaw[] = [
  single('网站', 'yd:globe', SitesRoutes),
  single('智能', 'yd:sparkles', AIRoutes),
  {
    meta: {
      title: '数据库',
      icon: 'yd:database',
    },
    children: [
      DatabaseRoutes,
      DbAdminRoutes,
    ],
  },
  {
    meta: {
      title: '容器',
      icon: 'yd:container',
    },
    children: [
      ContainerRoutes,
      ComposeRoutes,
      DockerRoutes,
    ],
  },
  {
    meta: {
      title: '应用',
      icon: 'yd:package',
    },
    children: [
      StoreRoutes,
      MarketRoutes,
    ],
  },
  {
    meta: {
      title: '工具',
      icon: 'i-lucide:wrench',
    },
    children: [
      CronRoutes,
      RuntimesRoutes,
    ],
  },
  {
    meta: {
      title: '监控',
      icon: 'yd:bell',
    },
    children: [
      AlertRoutes,
      MonitorsRoutes,
      NotificationRoutes,
    ],
  },
  {
    meta: {
      title: '系统',
      icon: 'yd:shield-check',
      auth: ['admin'],
    },
    children: [
      FileRoutes,
      TerminalRoutes,
      FirewallRoutes,
      ProcessesRoutes,
      NodesRoutes,
      ManageRoutes,
      SelfUpdateRoutes,
    ],
  },
]

export {
  asyncRoutes,
  constantRoutes,
  systemRoutes,
}
