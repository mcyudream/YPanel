/**
 * YudreamWebOS Core — 框架无关核心。
 * 事件总线 / 应用注册表 / 窗口管理器 / 桌面布局 / Dock / 小组件 / 菜单 / VFS / 持久化 / 配置 / 主题 token。
 */
export const YW_CORE_VERSION = '0.2.0'

// 配置
export { defaultConfig, resolveConfig } from './config/config'
export type { WebOSConfig } from './config/config'

// 桌面布局
export { DesktopModel } from './desktop/desktop-model'
export type {
  ArrangeMode,
  DesktopChangeReason,
  DesktopGridConfig,
  DesktopItem,
  DesktopItemType,
  MoveCollisionStrategy,
  MoveResult,
} from './desktop/desktop-model'

// Dock
export { DockModel } from './dock/dock-model'
// 事件总线
export { createEventBus } from './events/bus'

export type { EventBus, EventHandler, YwBusEvents, YwEventBus } from './events/bus'
// 菜单
export { mergeMenuItems, mergeMenus } from './menu/menu-model'

export type { MenuContext, MenuContribution, MenuItem } from './menu/menu-model'

// 持久化
export {
  LocalStoragePersistence,
  MemoryPersistence,
  migrate,
} from './persist/persistence'
export type { PersistenceAdapter, VersionedData } from './persist/persistence'

// 应用注册表
export { AppRegistry } from './registry/registry'
export type { AppDefinition, AppIcon, AppLaunchContext } from './registry/types'

export { darkTheme, lightTheme, mergeTheme, tokensToStyle } from './theme/tokens'
// 主题 token（ui 层注入用）
export type { ThemeMode, ThemeTokens, WallpaperMeta } from './types/theme'

export { GLASS_LEVELS } from './types/theme'
// UI 适配器接口（菜单/对话框/提示 —— core 类型定义，ui 层实现）
export type {
  YwAlertOptions,
  YwConfirmOptions,
  YwMenuHandle,
  YwMenuItem,
  YwMenuOptions,
  YwMessageType,
  YwUiAdapter,
} from './types/ui-adapter'

// 颜色工具
export { hexToOklchChannels } from './utils/color'
export type { VFSAdapter, VFSEvent, VFSEventType, VNode } from './vfs/types'

// VFS
export {
  IndexedDBAdapter,
  LocalStorageAdapter,
  MemoryAdapter,
  VFS,
} from './vfs/vfs'
// 小组件
export { WidgetStore } from './widgets/widget-model'
export type {
  WidgetConfigField,
  WidgetDefinition,
  WidgetInstance,
  WidgetPlacement,
  WidgetSize,
} from './widgets/widget-model'

// 窗口管理器
export {
  cascadeRect,
  detectSnapZone,
  snapBounds,
  WindowManager,
} from './window/window-manager'

export type {
  OpenAppSpec,
  OpenOptions,
  SnapZone,
  WindowConstraints,
  WindowInstance,
  WindowSnapshot,
  WindowStateType,
} from './window/window-manager'
