import { setSettings } from '@fantastic-admin/settings'

export default setSettings({
  app: {
    // 开启后登录时拉取角色权限（admin/user），用于菜单显隐
    account: {
      auth: true,
    },
    home: {
      enable: true,
      title: '主机概览',
    },
    copyright: {
      enable: true,
      dates: '2026',
      company: 'YuDream · YPanel',
    },
  },
  menu: {
    mainMenuClickMode: 'smart',
    subMenuCollapseButton: true,
  },
  topbar: {
    tabbar: true,
    toolbar: true,
  },
  toolbar: {
    breadcrumb: true,
    menuSearch: {
      enable: true,
      hotkeys: true,
    },
    fullscreen: true,
    pageReload: true,
    colorScheme: true,
  },
  tabbar: {
    icon: true,
  },
})
