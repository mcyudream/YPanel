import type { DeepPartial } from '@yudream/yudream-webos-shared'
import type { MenuContribution } from '../menu/menu-model'
import type { AppDefinition } from './types'
import { deepMerge } from '@yudream/yudream-webos-shared'
import { mergeMenuItems } from '../menu/menu-model'

/**
 * 应用注册表：register / registerAll / unregister / override（深合并） / get / list / queryByFileType。
 * override 是「可重载一切」的根基：宿主可只替换 Finder 的菜单而不重写整个应用。
 */
export class AppRegistry {
  private apps = new Map<string, AppDefinition>()
  private listeners = new Set<(appId: string, action: 'register' | 'unregister' | 'override') => void>()

  register(app: AppDefinition): void {
    this.apps.set(app.id, app)
    this.notify(app.id, 'register')
  }

  registerAll(apps: AppDefinition[]): void {
    for (const app of apps) {
      this.register(app)
    }
  }

  unregister(id: string): void {
    if (this.apps.delete(id)) {
      this.notify(id, 'unregister')
    }
  }

  /**
   * 深合并覆盖已注册应用的任意字段（内置应用也能被宿主重载）。
   * menus 特殊处理：数组按条目 id 深合并（而非整体替换），覆盖单条菜单不丢其它项。
   */
  override(id: string, patch: DeepPartial<AppDefinition>): void {
    const existing = this.apps.get(id)
    if (!existing) {
      throw new Error(`[webos] override: app "${id}" not registered`)
    }
    const merged = deepMerge(existing, patch)
    const pm = (patch as { menus?: MenuContribution }).menus
    if (existing.menus && pm) {
      merged.menus = {
        appMenus: mergeMenuItems(existing.menus.appMenus ?? [], pm.appMenus),
        overrides: { ...existing.menus.overrides, ...pm.overrides },
      }
    }
    this.apps.set(id, merged)
    this.notify(id, 'override')
  }

  get(id: string): AppDefinition | undefined {
    return this.apps.get(id)
  }

  list(): AppDefinition[] {
    return [...this.apps.values()]
  }

  /** 按扩展名查可打开该类型文件的应用（fileHandlers 路由） */
  queryByFileType(ext: string): AppDefinition[] {
    const e = ext.toLowerCase().replace(/^\./, '')
    return this.list().filter(app => app.fileHandlers?.includes(e))
  }

  /** 桌面生态派生：Dock 显示项 */
  dockApps(): AppDefinition[] {
    return this.list()
      .filter(app => app.dock?.showInDock !== false)
      .sort((a, b) => (a.launchpad?.order ?? 100) - (b.launchpad?.order ?? 100))
  }

  /** 启动台派生 */
  launchpadApps(): AppDefinition[] {
    return this.list()
      .filter(app => app.launchpad?.show !== false)
      .sort((a, b) => (a.launchpad?.order ?? 100) - (b.launchpad?.order ?? 100))
  }

  onChange(fn: (appId: string, action: 'register' | 'unregister' | 'override') => void): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  private notify(appId: string, action: 'register' | 'unregister' | 'override') {
    this.listeners.forEach(fn => fn(appId, action))
  }
}
