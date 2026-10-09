import type { FreePosition, GridPosition } from '@yudream/yudream-webos-shared'
import { shortId } from '@yudream/yudream-webos-shared'

export type DesktopItemType = 'app' | 'folder' | 'file' | 'widget'
export type ArrangeMode = 'grid' | 'free'

/** 桌面项 */
export interface DesktopItem {
  id: string
  type: DesktopItemType
  /** appId / vfs path / widgetInstanceId（folder 为空） */
  refId: string
  name: string
  icon?: string
  position: GridPosition | FreePosition
  /** 仅 folder：子项快照 */
  children?: DesktopItem[]
}

export interface DesktopGridConfig {
  cellWidth: number
  cellHeight: number
  gap: number
  /** rtl：从屏幕右侧起排（macOS 默认） */
  direction: 'ltr' | 'rtl'
}

export type DesktopChangeReason = 'add' | 'remove' | 'move' | 'folder' | 'sort' | 'arrange'

/** 换位策略 */
export type MoveCollisionStrategy = 'swap' | 'shift'

export interface MoveResult {
  ok: boolean
  /** swap 时被换位项的 id */
  swappedWith?: string
}

/**
 * 桌面布局模型：grid/free 两种排布、swap/shift 换位策略、文件夹、排序整理。
 * 多桌面（Spaces）预留：首版单 space。
 */
export class DesktopModel {
  private items = new Map<string, DesktopItem>()
  private listeners = new Set<(reason: DesktopChangeReason) => void>()

  arrangeMode: ArrangeMode = 'grid'
  grid: DesktopGridConfig = { cellWidth: 84, cellHeight: 92, gap: 4, direction: 'rtl' }
  collision: MoveCollisionStrategy = 'swap'
  /** 每列最大行数（由 ui 层按视口高度计算）；0 = 不限（一列到底） */
  rowsPerColumn = 0

  constructor(items?: DesktopItem[]) {
    for (const item of items ?? []) {
      this.items.set(item.id, item)
    }
  }

  list(): DesktopItem[] {
    return [...this.items.values()]
  }

  get(id: string): DesktopItem | undefined {
    return this.items.get(id)
  }

  onChange(fn: (reason: DesktopChangeReason) => void): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  add(item: Omit<DesktopItem, 'id'> & { id?: string }): DesktopItem {
    const withId: DesktopItem = { ...item, id: item.id ?? shortId('desk') }
    if (this.arrangeMode === 'grid') {
      const pos = withId.position as GridPosition | undefined
      // grid 模式：未给坐标或坐标被占用 → 自动找空格（列优先向下）
      const occupied = pos && this.list().some(
        x => x.id !== withId.id && x.type !== 'folder'
          && (x.position as GridPosition).col === pos.col && (x.position as GridPosition).row === pos.row,
      )
      if (!pos || occupied) {
        withId.position = this.firstFreeCell(pos ?? { col: 0, row: 0 })
      }
    }
    this.items.set(withId.id, withId)
    this.notify('add')
    return withId
  }

  remove(id: string): void {
    if (this.items.delete(id)) {
      this.notify('remove')
    }
  }

  moveTo(id: string, pos: GridPosition | FreePosition): MoveResult {
    const item = this.items.get(id)
    if (!item) {
      return { ok: false }
    }
    if (this.arrangeMode === 'free') {
      item.position = pos
      this.notify('move')
      return { ok: true }
    }
    const target = pos as GridPosition
    // 网格模式：占用检测 → swap / shift
    const occupant = this.list().find(
      x => x.id !== id && x.type !== 'folder' && (x.position as GridPosition).col === target.col && (x.position as GridPosition).row === target.row,
    )
    if (occupant) {
      if (this.collision === 'swap') {
        const origin = item.position as GridPosition
        occupant.position = { ...origin }
        item.position = { ...target }
        this.notify('move')
        return { ok: true, swappedWith: occupant.id }
      }
      // shift：目标起整体顺移
      this.shiftFrom(target, id)
      item.position = { ...target }
      this.notify('move')
      return { ok: true }
    }
    // 同位移动不通知（避免双击时 pointerup 触发无谓重建，吞掉 dblclick）
    if ((item.position as GridPosition).col === target.col && (item.position as GridPosition).row === target.row) {
      return { ok: true }
    }
    item.position = { ...target }
    this.notify('move')
    return { ok: true }
  }

  /** 拖 A 到 B 成组（文件夹）：子项以快照存入 folder.children */
  createFolder(name: string, childIds: string[]): DesktopItem {
    const children: DesktopItem[] = []
    let anchor: GridPosition | FreePosition = { col: 0, row: 0 }
    for (const id of childIds) {
      const child = this.items.get(id)
      if (!child) {
        continue
      }
      anchor = child.position
      children.push({ ...child })
      this.items.delete(id)
    }
    const folder = this.add({ type: 'folder', refId: '', name, position: anchor, children })
    this.notify('folder')
    return folder
  }

  dissolveFolder(id: string): void {
    const folder = this.items.get(id)
    if (folder?.type === 'folder') {
      // 子项快照放回桌面
      for (const child of folder.children ?? []) {
        this.items.set(child.id, { ...child })
      }
      this.items.delete(id)
      this.notify('folder')
    }
  }

  folderChildren(id: string): DesktopItem[] {
    const folder = this.items.get(id)
    if (folder?.type !== 'folder') {
      return []
    }
    return (folder.children ?? []).map(c => ({ ...c }))
  }

  /** 「整理」排序 */
  sortBy(criteria: 'name' | 'kind' | 'dateModified'): void {
    const items = this.list()
    items.sort((a, b) => {
      if (criteria === 'name') {
        return a.name.localeCompare(b.name, 'zh-CN')
      }
      if (criteria === 'kind') {
        return a.type.localeCompare(b.type) || a.name.localeCompare(b.name, 'zh-CN')
      }
      return a.id.localeCompare(b.id)
    })
    this.reassignCells(items)
    this.notify('sort')
  }

  /** 「按网格对齐」 */
  autoArrange(): void {
    this.reassignCells(this.list())
    this.notify('arrange')
  }

  /** 网格占用探测：从起点向下找第一个空格 */
  private firstFreeCell(start: GridPosition): GridPosition {
    const taken = new Set(this.list().filter(x => x.type !== 'folder').map(x => `${(x.position as GridPosition).col},${(x.position as GridPosition).row}`))
    const startCol = Math.max(0, start.col)
    let col = startCol
    let row = start.row
    for (let guard = 0; guard < 10000; guard++) {
      if (!taken.has(`${col},${row}`)) {
        return { col, row }
      }
      row++
      // 列满（超出每列行数）→ 换下一列回到顶部（macOS 列优先排布）
      if (this.rowsPerColumn > 0 && row >= this.rowsPerColumn) {
        row = 0
        col++
      }
    }
    return { col: startCol, row: start.row }
  }

  /** shift：目标格起，其余项按列优先顺移一格 */
  private shiftFrom(start: GridPosition, excludeId: string) {
    const movable = this.list().filter(x => x.id !== excludeId && x.type !== 'folder').map(x => ({ x, p: x.position as GridPosition })).sort((a, b) => (a.p.col - b.p.col) || (a.p.row - b.p.row))
    for (const { x, p } of movable) {
      // 在目标之后的项整体后移一行
      if (p.row >= start.row) {
        ;(x.position as GridPosition) = { col: p.col, row: p.row + 1 }
      }
    }
  }

  private reassignCells(items: DesktopItem[]) {
    const movable = items.filter(x => x.type !== 'folder')
    const next = new Map<string, DesktopItem>()
    // 保持 folder 等其它项
    for (const item of items) {
      if (item.type === 'folder') {
        next.set(item.id, item)
      }
    }
    let col = 0
    let row = 0
    for (const item of movable) {
      next.set(item.id, { ...item, position: { col, row } })
      row++
      // 列满换列（与 firstFreeCell 一致的列优先语义），整理/排序不再挤成一根竖条
      if (this.rowsPerColumn > 0 && row >= this.rowsPerColumn) {
        row = 0
        col++
      }
    }
    this.items = next
  }

  private notify(reason: DesktopChangeReason) {
    this.listeners.forEach(fn => fn(reason))
  }
}
