/**
 * 插槽命名协议（四级覆盖体系第③级）：Provider 上可注入的插槽 key。
 */
export const WEBOS_SLOTS = [
  'menubar-right',
  'menubar-tray',
  'desktop-item',
  'desktop-context-menu',
  'dock-item',
  'dock-tray',
  'finder-sidebar',
  'finder-sidebar-footer',
  'finder-toolbar',
  'launchpad-item',
  'widget-host-footer',
  'window-titlebar',
] as const

export type WebOSSlotKey = typeof WEBOS_SLOTS[number]
