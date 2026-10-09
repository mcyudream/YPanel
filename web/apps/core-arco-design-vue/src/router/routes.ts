import type { RouteRecordMainRaw } from '@fantastic-admin/types'
import type { RouteRecordRaw } from 'vue-router'
import ComposeRoutes from './modules/compose'
import DatabaseRoutes from './modules/database'
import SitesRoutes from './modules/sites'
import CertsRoutes from './modules/certs'
import {
  ContainerApps,
  ContainerList,
  ContainerImages,
  ContainerNetworks,
  ContainerVolumes,
  ContainerEnvs,
  ContainerSettings,
  ContainerRoot,
} from './modules/container'
import ContainerDetailRoutes from './modules/container-detail'
import ContainerAppRoutes from './modules/container-app'
import CronRoutes from './modules/cron'
import DnsRoutes from './modules/dns'
import VpnRoutes from './modules/vpn'
import FileRoutes from './modules/file'
import MonitorsRoutes from './modules/monitors'
import {
  AiChat,
  AiProviders,
  AiKnowledge,
  AiWorkspace,
  AiMemory,
  AiTools,
  AiSkills,
  AiMcp,
  AiRoot,
} from './modules/ai'
import ManageRoutes from './modules/manage'
import NodesRoutes from './modules/nodes'
import NodesDetailRoutes from './modules/nodes-detail'
import FirewallRoutes from './modules/firewall'
import NatRoutes from './modules/nat'
import HostsRoutes from './modules/hosts'
import AlertRoutes from './modules/alert'
import ProbeRoutes from './modules/probe'
import LogCenterRoutes from './modules/logcenter'
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
      title: 'menu.login',
    },
  },
  {
    path: '/desktop',
    name: 'desktop',
    component: () => import('@/views/desktop/index.vue'),
    meta: {
      title: 'menu.desktop',
      breadcrumb: false,
    },
  },
  {
    path: '/:all(.*)*',
    name: 'notFound',
    component: () => import('@/views/[...all].vue'),
    meta: {
      title: 'menu.notFound',
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
          title: 'menu.overview',
          icon: 'yd:gauge',
          breadcrumb: false,
        },
      },
      {
        path: 'reload',
        name: 'reload',
        component: () => import('@/views/reload.vue'),
        meta: {
          title: 'menu.reload',
          breadcrumb: false,
        },
      },
    ],
  },
]

// 动态路由（异步路由、导航菜单路由）
// 一级大菜单统一 2 字命名；单页功能用"同名分组 + 子页 menu:false"配合
// mainMenuClickMode=smart 实现主导航点击直达；强关联功能按域分组。
const asyncRoutes: RouteRecordMainRaw[] = [
  {
    meta: {
      title: 'menu.sites',
      icon: 'yd:globe',
    },
    children: [
      SitesRoutes,
      CertsRoutes,
      RuntimesRoutes,
    ],
  },
  // 智能域：叶子平铺（对话/供应商/知识库/工作空间/记忆/系统工具/技能/MCP），无分组头行
  {
    meta: {
      title: 'menu.ai',
      icon: 'yd:sparkles',
    },
    children: [
      AiChat,
      AiProviders,
      AiKnowledge,
      AiWorkspace,
      AiMemory,
      AiTools,
      AiSkills,
      AiMcp,
      AiRoot,
    ],
  },
  {
    meta: {
      title: 'menu.database',
      icon: 'yd:database',
    },
    children: [
      DatabaseRoutes,
      DbAdminRoutes,
    ],
  },
  // 容器域：次侧栏平铺叶子项（应用/镜像/网络/卷/配置，无父标题行）；
  // 容器列表路由保留（分组头直达）；分组内收纳详情页模块与旧路由重定向（均 menu:false）。
  {
    meta: {
      title: 'menu.container',
      icon: 'i-tabler:brand-docker',
    },
    children: [
      ContainerApps,
      ContainerList,
      ContainerImages,
      ContainerNetworks,
      ContainerVolumes,
      ContainerEnvs,
      ContainerSettings,
      ContainerRoot,
      ContainerDetailRoutes,
      ContainerAppRoutes,
      ComposeRoutes,
      DockerRoutes,
    ],
  },
  {
    meta: {
      title: 'menu.app',
      icon: 'yd:package',
    },
    children: [
      StoreRoutes,
    ],
  },
  {
    meta: {
      title: 'menu.tools',
      icon: 'i-lucide:wrench',
    },
    children: [
      CronRoutes,
      DnsRoutes,
      VpnRoutes,
    ],
  },
  {
    meta: {
      title: 'menu.monitor',
      icon: 'yd:bell',
    },
    children: [
      AlertRoutes,
      MonitorsRoutes,
      ProbeRoutes,
      LogCenterRoutes,
    ],
  },
  {
    meta: {
      title: 'menu.system',
      icon: 'yd:shield-check',
      auth: ['admin'],
    },
    children: [
      FileRoutes,
      TerminalRoutes,
      FirewallRoutes,
      NatRoutes,
      HostsRoutes,
      ProcessesRoutes,
      NodesRoutes,
      NodesDetailRoutes,
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
