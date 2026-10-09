/**
 * Dock 模型：固定项 / 最近使用 / 角标 / 位置与尺寸配置。
 * 渲染数据由 ui 层组合：[固定区] | [运行中未固定+指示点] | [最近使用] | [最小化缩略] | [废纸篓]
 */
export class DockModel {
  /** 固定的 appId（有序） */
  pinned: string[] = []
  /** 「在 Dock 中显示最近使用的应用」 */
  showRecent = true
  position: 'bottom' | 'left' | 'right' = 'bottom'
  autohide = false
  /** 放大镜（ui 层实现） */
  magnification = true
  /** 基准尺寸 px */
  iconSize = 48

  private badges = new Map<string, string | number>()
  private listeners = new Set<(reason: string) => void>()

  constructor(init?: Partial<Pick<DockModel, 'pinned' | 'showRecent' | 'position' | 'autohide' | 'magnification' | 'iconSize'>>) {
    Object.assign(this, init ?? {})
  }

  onChange(fn: (reason: string) => void): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  pin(appId: string, index?: number): void {
    if (this.pinned.includes(appId)) {
      return
    }
    if (index === undefined) {
      this.pinned.push(appId)
    }
    else {
      this.pinned.splice(Math.max(0, Math.min(index, this.pinned.length)), 0, appId)
    }
    this.notify('pin')
  }

  unpin(appId: string): void {
    const idx = this.pinned.indexOf(appId)
    if (idx > -1) {
      this.pinned.splice(idx, 1)
      this.notify('unpin')
    }
  }

  reorder(from: number, to: number): void {
    if (from === to || from < 0 || from >= this.pinned.length) {
      return
    }
    const [moved] = this.pinned.splice(from, 1)
    this.pinned.splice(Math.max(0, Math.min(to, this.pinned.length)), 0, moved!)
    this.notify('reorder')
  }

  setBadge(appId: string, badge: string | number | null): void {
    if (badge === null) {
      this.badges.delete(appId)
    }
    else {
      this.badges.set(appId, badge)
    }
    this.notify('badge')
  }

  getBadge(appId: string): string | number | null {
    return this.badges.get(appId) ?? null
  }

  private notify(reason: string) {
    this.listeners.forEach(fn => fn(reason))
  }
}
