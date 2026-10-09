/**
 * 菜单模型与合并器。
 * 一套 MenuItem 模型复用于：顶部菜单栏、Dock 右键、桌面右键、Finder 右键/工具栏下拉。
 * 合并规则：系统默认菜单 → 应用 appMenus → 宿主全局 overrides → 应用 overrides（按 id 深合并）。
 */

export interface MenuContext {
  appId?: string
  windowId?: string
  /** 触发上下文（如右键目标路径） */
  target?: unknown
}

export interface MenuItem {
  id: string
  label: string
  /** 'Cmd+N'（展示 + 全局快捷键绑定） */
  shortcut?: string
  icon?: string
  disabled?: boolean
  checked?: boolean
  separator?: boolean
  submenu?: MenuItem[]
  action?: (ctx: MenuContext) => void
}

export interface MenuContribution {
  /** 应用级菜单：文件 / 编辑 / 视图… */
  appMenus?: MenuItem[]
  /** 覆盖菜单中的条目（按 id 深合并） */
  overrides?: Record<string, Partial<MenuItem>>
}

/** 按 id 深合并菜单项数组（dst 为基，src 覆盖；src 新条目追加） */
export function mergeMenuItems(dst: MenuItem[], src?: Partial<MenuItem>[]): MenuItem[] {
  if (!src?.length) {
    return dst
  }
  const out = dst.map(item => ({ ...item }))
  for (const patch of src) {
    const idx = out.findIndex(x => x.id === patch.id)
    if (idx === -1) {
      out.push(patch as MenuItem)
      continue
    }
    const current = out[idx]!
    const merged = { ...current, ...patch } as MenuItem
    if (patch.submenu || current.submenu) {
      merged.submenu = mergeMenuItems(current.submenu ?? [], patch.submenu)
    }
    out[idx] = merged
  }
  return out
}

/**
 * 菜单合并器：系统默认 → 应用 → 宿主全局 → 应用级 overrides。
 */
export function mergeMenus(options: {
  systemMenus: MenuItem[]
  app?: MenuContribution
  hostOverrides?: Record<string, Partial<MenuItem>>
}): MenuItem[] {
  let menus = mergeMenuItems(options.systemMenus, options.app?.appMenus)
  // 宿主全局 overrides（按顶层菜单 id 深合并）
  menus = mergeMenuItems(menus, Object.entries(options.hostOverrides ?? {}).map(([id, patch]) => ({ id, ...patch })))
  // 应用级 overrides
  menus = mergeMenuItems(menus, Object.entries(options.app?.overrides ?? {}).map(([id, patch]) => ({ id, ...patch })))
  return menus
}
