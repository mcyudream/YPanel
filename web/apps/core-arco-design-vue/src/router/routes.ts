import type { RouteRecordMainRaw } from '@fantastic-admin/types'
import type { RouteRecordRaw } from 'vue-router'
import ComposeRoutes from './modules/compose'
import DatabaseRoutes from './modules/database'
import SitesRoutes from './modules/sites'
import ContainerRoutes from './modules/container'
import CronRoutes from './modules/cron'
import FileRoutes from './modules/file'
import MonitorsRoutes from './modules/monitors'
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
// 服务器运维面板分类：高频功能用"同名分组 + 子页 menu:false"，
// 配合 mainMenuClickMode=smart 实现主导航点击直达；强关联功能按域分组。
const single = (title: string, icon: string, mod: RouteRecordRaw) => ({
  meta: { title, icon },
  children: [mod],
})

const asyncRoutes: RouteRecordMainRaw[] = [
  // ---- 一级直达（同名分组 + smart 跳转） ----
  single('网站', 'yd:globe', SitesRoutes),
  single('终端', 'yd:square-terminal', TerminalRoutes),
  single('文件管理', 'yd:folder-open', FileRoutes),
  single('容器管理', 'yd:container', ContainerRoutes),
  single('Compose 编排', 'yd:layers', ComposeRoutes),
  single('计划任务', 'yd:calendar-clock', CronRoutes),
  single('防火墙', 'yd:shield', FirewallRoutes),
  single('节点管理', 'yd:network', NodesRoutes),
  single('进程与服务', 'yd:cpu', ProcessesRoutes),
  // ---- 分组 ----
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
      title: '容器进阶',
      icon: 'i-lucide:docker',
    },
    children: [
      DockerRoutes,
      RuntimesRoutes,
    ],
  },
  {
    meta: {
      title: '应用',
      icon: 'yd:package',
    },
    children: [
      MarketRoutes,
      StoreRoutes,
    ],
  },
  {
    meta: {
      title: '监控告警',
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
