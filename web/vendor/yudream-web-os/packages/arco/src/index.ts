// UI 适配器（Arco 实现）
export { arcoAdapter } from './adapter'
export { default } from './adapter'
export { default as YwAppIcon } from './components/app-icon/index.vue'
export { default as YwBrowserApp } from './components/browser/index.vue'
export { default as YwCard } from './components/card/index.vue'
export { default as YwControlCenter } from './components/control-center/index.vue'
/**
 * Arco Design Vue UI 层 — YudreamWebOS 全部可视组件。
 * 消费 @yudream/yudream-webos-vue 的 composables，渲染 core 状态、翻译交互。
 */
export { default as YwDesktop } from './components/desktop/index.vue'
export { default as YwDock } from './components/dock/index.vue'
export { default as YwFinder } from './components/finder/index.vue'
export { default as YwIconTile } from './components/icon-tile/index.vue'
export { default as YwLaunchpad } from './components/launchpad/index.vue'
export { default as YwMenubar } from './components/menubar/index.vue'
export { default as YwNotificationCenter } from './components/notification-center/index.vue'
export { default as YwQuickLaunch } from './components/quick-launch/index.vue'
export { default as YwSettingsApp } from './components/settings/index.vue'
export { default as YwTaskbar } from './components/taskbar/index.vue'
export { default as YwWidgetGallery } from './components/widgets/gallery.vue'

export { default as YwWidgetHost } from './components/widgets/host.vue'
export { default as YwWindow } from './components/window/index.vue'

// 兼容 stores
export {
  registerBrowserComponent,
  resolveBrowserComponent,
  resolveInput,
  useAppsStore,
  useBrowserStore,
  useThemeStore,
  useWindowsStore,
} from './stores/compat'
export type { BrowserSession, YwBrowserOptions } from './stores/compat'
