/**
 * YudreamWebOS Vue 绑定层。
 * WebOSProvider / createWebOS / composables / uiRegistry / slots / 快捷键。
 * 可视组件在 @yudream/yudream-webos-arco（ui 层），内置应用在 @yudream/yudream-webos-apps。
 */
import type { YwUiAdapter } from '@yudream/yudream-webos-core'
import { useWebOS } from './provider'

// 内置 UI 适配器（缺省实现）
export { builtinAdapter } from './adapter/builtin'
// Composables（文档 §4.2 全表）
export {
  useAppRegistry,
  useDesktop,
  useDock,
  useFinder,
  useMenuBar,
  useSystemSettings,
  useVFS,
  useWebOSEvents,
  useWidgets,
  useWindowManager,
} from './composables'
// 四级覆盖：② 组件替换
export { provideUIRegistry, UI_COMPONENT_KEYS, uiRegistry, useUIRegistry } from './override/registry'

export type { UIComponentKey, UIRegistryMap } from './override/registry'

// Provider 与插件
export {
  createWebOS,
  createWebOSInstance,
  useWebOS,
  WEBOS_KEY,
  YW_UI_ADAPTER_KEY,
} from './provider'
export type { CreateWebOSOptions, WebOSInstance } from './provider'

export { default as WebOSProvider } from './provider-component.vue'
// 快捷键
export { shortcuts } from './shortcuts'

// 四级覆盖：③ 插槽协议
export { WEBOS_SLOTS } from './slots/constants'

export type { WebOSSlotKey } from './slots/constants'

// 样式注入
export { injectBuiltinStyles } from './styles'

// 系统设置工具（ui 层复用）
export { applySettings, injectThemeTokens } from './system/settings'
export type { YwSettingsState } from './system/settings'

// core 全量 re-export（宿主无需单独装 core）
export * from '@yudream/yudream-webos-core'

/** 便捷：从容器取 UI 适配器 */
export function useUiAdapter(): YwUiAdapter {
  return useWebOS().ui
}
