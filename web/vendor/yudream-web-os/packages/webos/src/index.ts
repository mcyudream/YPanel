export {
  appStoreApp,
  browserApp,
  builtinApps,
  builtinWidgets,
  calculatorApp,
  calendarWidget,
  clockWidget,
  finderApp,
  imageViewerApp,
  launchpadApp,
  notesApp,
  settingsApp,
  systemMonitorWidget,
  terminalApp,
  textEditorApp,
} from '@yudream/yudream-webos-apps'
export {
  arcoAdapter,
  useAppsStore,
  useBrowserStore,
  useThemeStore,
  useWindowsStore,
  YwAppIcon,
  YwBrowserApp,
  YwCard,
  YwControlCenter,
  YwDesktop,
  YwDock,
  YwFinder,
  YwIconTile,
  YwLaunchpad,
  YwMenubar,
  YwNotificationCenter,
  YwQuickLaunch,
  YwSettingsApp,
  YwTaskbar,
  YwWidgetGallery,
  YwWidgetHost,
  YwWindow,
} from '@yudream/yudream-webos-arco'
/**
 * @yudream/yudream-webos 元包：聚合全部子包，一次安装全量引入。
 * 注意：core 经 vue 层单一链条 re-export（多个 export * 同名会触发 ESM ambiguous 静默剔除）。
 */
export * from '@yudream/yudream-webos-vue'
