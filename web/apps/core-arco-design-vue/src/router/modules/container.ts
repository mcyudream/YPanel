import type { RouteRecordRaw } from 'vue-router'

function Layout() {
  return import('@/layouts/index.vue')
}

// 容器域（M23）：五个功能区做成「单子页模块」（子页 menu:false → fa Menu 渲染为叶子项，
// 点击直达、无父标题行）；容器列表页挂在应用模块下作为隐藏子页（路由可达、菜单不出现）。
// 详情页走独立顶级模块（container-detail.ts / container-app.ts）。

function singleModule(path: string, name: string, title: string, icon: string, component: () => Promise<any>) {
  return {
    path,
    component: Layout,
    name,
    meta: {
      title,
      icon,
    },
    children: [
      {
        path: '',
        name: `${name}Index`,
        component,
        meta: {
          title,
          icon,
          menu: false,
        },
      },
    ],
  }
}

export const ContainerApps = singleModule('/container/apps', 'containerApps', 'menu.containerApps', 'i-lucide:layout-grid', () => import('@/views/container/tabs/AppsTab.vue'))
export const ContainerList = singleModule('/container/list', 'containerList', 'menu.containerList', 'i-lucide:boxes', () => import('@/views/container/tabs/ListTab.vue'))
export const ContainerImages = singleModule('/container/images', 'containerImages', 'menu.containerImages', 'i-lucide:disc', () => import('@/views/container/tabs/ImagesTab.vue'))
export const ContainerNetworks = singleModule('/container/networks', 'containerNetworks', 'menu.containerNetworks', 'i-lucide:network', () => import('@/views/container/tabs/NetworksTab.vue'))
export const ContainerVolumes = singleModule('/container/volumes', 'containerVolumes', 'menu.containerVolumes', 'i-lucide:hard-drive', () => import('@/views/container/tabs/VolumesTab.vue'))
export const ContainerEnvs = singleModule('/container/environments', 'containerEnvs', 'menu.containerEnvs', 'i-lucide:server-cog', () => import('@/views/container/tabs/EnvironmentsTab.vue'))
export const ContainerSettings = singleModule('/container/settings', 'containerSettings', 'menu.containerSettings', 'i-lucide:settings', () => import('@/views/container/tabs/SettingsTab.vue'))

// /container → /container/apps 兼容重定向
export const ContainerRoot: RouteRecordRaw = {
  path: '/container',
  name: 'containerRoot',
  redirect: { path: '/container/apps' },
  meta: {
    title: 'menu.container',
    menu: false,
  },
  children: [],
}
