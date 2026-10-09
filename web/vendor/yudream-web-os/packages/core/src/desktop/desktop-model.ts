import type { FreePosition, GridPosition } from '@yudream/yudream-webos-shared'
import { shortId } from '@yudream/yudream-webos-shared'

export type DesktopItemType = 'app' | 'folder' | 'file' | 'widget'
export type ArrangeMode = 'grid' | 'free'

/** 多格跨度（文件夹卡片 / 大图标）：默认 1x1 */
export interface DesktopSpan {
  w: number
  h: number
}

/** 桌面项 */
export interface DesktopItem {
  id: string
  type: DesktopItemType
  /** appId / vfs path / widgetInstanceId（folder 为空） */
  refId: string
  name: string
  icon?: string
  position: GridPosition | FreePosition
  /** 多格跨度：文件夹卡片与大图标；缺省 1x1 */
  span?: DesktopSpan
  /** 仅 file：文本内容（随布局持久化） */
  content?: string
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

export type DesktopChangeReason = 'add' | 'remove' | 'move' | 'folder' | 'sort' | 'arrange' | 'file'

/** 换位策略 */
export type MoveCollisionStrategy = 'swap' | 'shift'

export interface MoveResult {
  ok: boolean
  /** swap 时被换位项的 id */
  swappedWith?: string
  /** 目标格被文件夹占据（未移动）——UI 层可据此解释为「拖入文件夹」 */
  folderId?: string
}

interface Rect {
  col: number
  row: number
  w: number
  h: number
}

const spanOf = (item: DesktopItem): DesktopSpan => item.span ?? { w: 1, h: 1 }

/**
 * 桌面布局模型：grid/free 两种排布、swap/shift 换位策略、文件夹（含多格卡片）、排序整理。
 * 多桌面（Spaces）预留：首版单 space。
 *
 * 多格与遮挡：blockedCells 是「被小组件等其它层占据的格子」（UI 层换算后注入），
 * 落点/换位/整理一律视其为已占用——图标永远不会落在小组件底下。
 */
export class DesktopModel {
  private items = new Map<string, DesktopItem>()
  private listeners = new Set<(reason: DesktopChangeReason) => void>()

  arrangeMode: ArrangeMode = 'grid'
  grid: DesktopGridConfig = { cellWidth: 84, cellHeight: 92, gap: 4, direction: 'rtl' }
  collision: MoveCollisionStrategy = 'swap'
  /** 每列最大行数（由 ui 层按视口高度计算）；0 = 不限（一列到底） */
  rowsPerColumn = 0
  /** 被其它层（小组件）占据的格子，键 "col,row"（模型坐标） */
  blockedCells = new Set<string>()

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

  /** 项占据的矩形（模型坐标） */
  rectOf(item: DesktopItem): Rect {
    const s = spanOf(item)
    return { col: (item.position as GridPosition).col, row: (item.position as GridPosition).row, w: s.w, h: s.h }
  }

  private rectsOverlap(a: Rect, b: Rect): boolean {
    return a.col < b.col + b.w && b.col < a.col + a.w && a.row < b.row + b.h && b.row < a.row + a.h
  }

  /** 目标矩形是否撞上任何其它项（含多格）或被遮挡格 */
  private collides(rect: Rect, excludeId?: string): DesktopItem | 'blocked' | null {
    for (let c = rect.col; c < rect.col + rect.w; c++) {
      for (let r = rect.row; r < rect.row + rect.h; r++) {
        if (this.blockedCells.has(`${c},${r}`)) {
          return 'blocked'
        }
      }
    }
    for (const x of this.items.values()) {
      if (x.id === excludeId) {
        continue
      }
      if (this.rectsOverlap(rect, this.rectOf(x))) {
        return x
      }
    }
    return null
  }

  add(item: Omit<DesktopItem, 'id'> & { id?: string }): DesktopItem {
    const withId: DesktopItem = { ...item, id: item.id ?? shortId('desk') }
    if (this.arrangeMode === 'grid') {
      const s = spanOf(withId)
      const pos = withId.position as GridPosition | undefined
      const rect: Rect = pos
        ? { col: pos.col, row: pos.row, w: s.w, h: s.h }
        : { col: 0, row: 0, w: s.w, h: s.h }
      const hit = this.collides(rect, withId.id)
      if (!pos || hit) {
        withId.position = this.firstFreeCell(pos ?? { col: 0, row: 0 }, s)
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
    // 目标矩形被文件夹占据 → 不移动，交由 UI 解释为「拖入文件夹」
    const hit = this.collides(this.rectOf({ ...item, position: target }), id)
    if (hit && hit !== 'blocked' && hit.type === 'folder') {
      return { ok: false, folderId: hit.id }
    }
    if (hit) {
      const s = spanOf(item)
      const occupant = hit as DesktopItem
      if (this.collision === 'shift') {
        // shift：目标起整体顺移（仅 1x1 项参与顺移）
        this.shiftFrom(target, id)
        item.position = { ...target }
        this.notify('move')
        return { ok: true }
      }
      // swap：1x1 对 1x1 才换位；跨格项撞其它项 → 找最近空位
      if (s.w === 1 && s.h === 1 && spanOf(occupant).w === 1 && spanOf(occupant).h === 1) {
        const origin = item.position as GridPosition
        occupant.position = { ...origin }
        item.position = { ...target }
        this.notify('move')
        return { ok: true, swappedWith: occupant.id }
      }
      item.position = this.firstFreeCell(target, s)
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

  /** 改变项的跨度（文件夹卡片 / 大图标）。撞其它图标 → 挤开；撞小组件占格 → 本项自动挪到最近可用位置。永不拒绝 */
  resizeItem(id: string, span: DesktopSpan): MoveResult {
    const item = this.items.get(id)
    if (!item) {
      return { ok: false }
    }
    const blockedHit = (rect: Rect) => {
      for (let c = rect.col; c < rect.col + rect.w; c++) {
        for (let r = rect.row; r < rect.row + rect.h; r++) {
          if (this.blockedCells.has(`${c},${r}`)) {
            return true
          }
        }
      }
      return false
    }
    let pos = item.position as GridPosition
    let rect: Rect = { col: pos.col, row: pos.row, w: span.w, h: span.h }
    // 目标区域压到小组件 → 本项挪到最近能放下 span 的空位
    if (blockedHit(rect)) {
      pos = this.firstFreeCell(pos, span)
      item.position = pos
      rect = { col: pos.col, row: pos.row, w: span.w, h: span.h }
    }
    // 挡路的图标逐个迁到空位
    for (const x of this.list()) {
      if (x.id === id) {
        continue
      }
      if (this.rectsOverlap(this.rectOf(x), rect)) {
        x.position = this.firstFreeCell({ col: 0, row: 0 }, spanOf(x))
      }
    }
    item.span = span
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
      // 子项快照放回桌面（空位自动避让）
      for (const child of folder.children ?? []) {
        this.addToGrid(child)
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

  /** 图标移入文件夹（子项快照存入 children） */
  addToFolder(folderId: string, childId: string): boolean {
    const folder = this.items.get(folderId)
    const child = this.items.get(childId)
    if (!folder || folder.type !== 'folder' || !child || childId === folderId) {
      return false
    }
    if (child.type === 'folder') {
      return false // 文件夹套文件夹：首版不支持
    }
    folder.children = folder.children ?? []
    if (folder.children.some(c => c.id === childId)) {
      return false
    }
    folder.children.push({ ...child })
    this.items.delete(childId)
    this.notify('folder')
    return true
  }

  /** 子项移出文件夹：回到桌面网格空位（可指定起点） */
  removeFromFolder(folderId: string, childId: string, start?: GridPosition): boolean {
    const folder = this.items.get(folderId)
    if (!folder || folder.type !== 'folder') {
      return false
    }
    const children = folder.children ?? []
    const idx = children.findIndex(c => c.id === childId)
    if (idx < 0) {
      return false
    }
    const [child] = children.splice(idx, 1)
    this.addToGrid(child, start)
    this.notify('folder')
    return true
  }

  /** 文本文件写内容（随 desktop.layout 持久化） */
  writeFile(id: string, content: string): boolean {
    const item = this.items.get(id)
    if (!item) {
      return false
    }
    item.content = content
    this.notify('file')
    return true
  }

  /** 把游离项放回网格：给定位被占/被遮挡 → 自动找空位 */
  private addToGrid(child: DesktopItem, start?: GridPosition) {
    const pos = (start ?? child.position) as GridPosition | undefined
    const s = child.span ?? { w: 1, h: 1 }
    const rect: Rect = pos
      ? { col: pos.col, row: pos.row, w: s.w, h: s.h }
      : { col: 0, row: 0, w: s.w, h: s.h }
    const restored: DesktopItem = { ...child, position: this.collides(rect, child.id) ? this.firstFreeCell(pos ?? { col: 0, row: 0 }, s) : (pos ?? { col: 0, row: 0 }) }
    this.items.set(restored.id, restored)
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

  /** 网格占用探测：从起点向下找第一个放下 span 且不撞遮挡/他项的空位（列优先，macOS 惯例） */
  private firstFreeCell(start: GridPosition, span: DesktopSpan = { w: 1, h: 1 }): GridPosition {
    const startCol = Math.max(0, start.col)
    let col = startCol
    let row = start.row
    for (let guard = 0; guard < 10000; guard++) {
      if (!this.collides({ col, row, w: span.w, h: span.h })) {
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
    const movable = this.list().filter(x => x.id !== excludeId && x.span == null).map(x => ({ x, p: x.position as GridPosition })).sort((a, b) => (a.p.col - b.p.col) || (a.p.row - b.p.row))
    for (const { x, p } of movable) {
      // 在目标之后的项整体后移一行
      if (p.row >= start.row) {
        ;(x.position as GridPosition) = { col: p.col, row: p.row + 1 }
      }
    }
  }

  /** 重排：全部项（含文件夹/多格卡）按列优先放进不撞遮挡与他项的格子 */
  private reassignCells(items: DesktopItem[]) {
    const next = new Map<string, DesktopItem>()
    const placed: Rect[] = []
    const free = (rect: Rect) => {
      for (let c = rect.col; c < rect.col + rect.w; c++) {
        for (let r = rect.row; r < rect.row + rect.h; r++) {
          if (this.blockedCells.has(`${c},${r}`)) {
            return false
          }
        }
      }
      return !placed.some(p => this.rectsOverlap(p, rect))
    }
    for (const item of items) {
      const s = spanOf(item)
      let col = 0
      let row = 0
      let guard = 0
      for (; guard < 10000; guard++) {
        if (free({ col, row, w: s.w, h: s.h })) {
          break
        }
        row++
        if (this.rowsPerColumn > 0 && row >= this.rowsPerColumn) {
          row = 0
          col++
        }
      }
      if (guard >= 10000) {
        col = (item.position as GridPosition).col
        row = (item.position as GridPosition).row // 放不下：原地保留
      }
      placed.push({ col, row, w: s.w, h: s.h })
      next.set(item.id, { ...item, position: { col, row } })
    }
    this.items = next
  }

  private notify(reason: DesktopChangeReason) {
    this.listeners.forEach(fn => fn(reason))
  }
}
