/**
 * 组件覆盖注册表（uiRegistry）：四级覆盖体系第②级，整体替换某个 UI 区块。
 * key 清单即公共契约（semver 保护）。
 */
import type { Component, InjectionKey } from 'vue'
import { inject, provide } from 'vue'

/** 可覆盖组件 key 全量清单（首版） */
export const UI_COMPONENT_KEYS = [
  'WindowFrame',
  'WindowTitleBar',
  'DesktopItem',
  'DesktopFolder',
  'Dock',
  'DockItem',
  'Launchpad',
  'LaunchpadItem',
  'MenuBar',
  'MenuBarExtra',
  'ControlCenter',
  'NotificationCenter',
  'FinderSidebar',
  'FinderToolbar',
  'FinderView',
  'WidgetHost',
  'WidgetGallery',
  'ContextMenu',
] as const

export type UIComponentKey = typeof UI_COMPONENT_KEYS[number]

export type UIRegistryMap = Partial<Record<UIComponentKey, Component>>

export const UI_REGISTRY_KEY: InjectionKey<{ overrides: UIRegistryMap }> = Symbol('yw-ui-registry')

/** 全局注册表（供 ui-arco 渲染时查询；app 级 provide 合并） */
const globalRegistry: UIRegistryMap = {}

export const uiRegistry = {
  /** 覆盖组件（连窗口边框都能换） */
  override(key: UIComponentKey, comp: Component): void {
    globalRegistry[key] = comp
  },
  /** 取生效组件：app 级覆盖 > 全局覆盖 > 默认实现 */
  resolve(key: UIComponentKey, appOverrides?: UIRegistryMap, fallback?: Component): Component | undefined {
    return appOverrides?.[key] ?? globalRegistry[key] ?? fallback
  },
  clear(): void {
    for (const key of Object.keys(globalRegistry) as UIComponentKey[]) {
      delete globalRegistry[key]
    }
  },
}

/** Provider 内提供 app 级覆盖（与全局 uiRegistry 合并生效） */
export function provideUIRegistry(overrides: UIRegistryMap) {
  provide(UI_REGISTRY_KEY, { overrides })
}

export function useUIRegistry(): { overrides: UIRegistryMap } {
  return inject(UI_REGISTRY_KEY, { overrides: {} })
}
