/**
 * 桌面网格拖拽：pointer 拖动桌面项，落点换算网格坐标 → DesktopModel.moveTo（swap/shift）。
 * 关键契约：**移动超过阈值（5px）才激活拖拽**——单击/双击不产生 ghost、不提交 moveTo，
 * 否则 pointerup 的同位 moveTo 会触发 onChange 重建 DOM 并吞掉浏览器的 dblclick。
 * 跨格卡（span>1）：ghost 中心吸附指针（居中跟手），落点按 ghost 中心所在格换算基列。
 */
import type { DesktopItem, DesktopModel } from '@yudream/yudream-webos-core'

export interface DesktopDragState {
  id: string
  pointerId: number
  offsetX: number
  offsetY: number
  /** ghost 当前像素位置（相对桌面容器） */
  x: number
  y: number
}

export interface DesktopGridMetrics {
  cellWidth: number
  cellHeight: number
  gap: number
  /** 可用区域尺寸（桌面容器减去内边距） */
  width: number
  height: number
  /** gravity=right 时列从右侧起算 */
  gravity: 'top-right' | 'top-left'
}

export interface DesktopDragHandle {
  onPointerDown: (ev: PointerEvent, item: DesktopItem) => void
  attach: () => void
  detach: () => void
  /** 允许挂载后注入（Vue mounted 时才有 DOM） */
  container: HTMLElement | null
  model: DesktopModel | null
}

/** 拖拽激活阈值（px）：低于此位移视为点击 */
const DRAG_THRESHOLD = 5

/** 落点格：ghost 中心所在格 → 模型基列（rtl 镜像下按跨度居中换算） */
function dropCell(options: { metrics: () => DesktopGridMetrics, model: DesktopModel | null }, s: DesktopDragState): { col: number, row: number } {
  const m = options.metrics()
  const it = options.model?.get(s.id)
  const sp = it ? (it.span ?? { w: 1, h: 1 }) : { w: 1, h: 1 }
  const Wpx = sp.w * 84 + (sp.w - 1) * 4
  const Hpx = sp.h * 92 + (sp.h - 1) * 4
  const cf = (s.x + Wpx / 2) / (m.cellWidth + m.gap)
  const rf = (s.y + Hpx / 2) / (m.cellHeight + m.gap)
  const cols = Math.max(1, Math.floor((m.width + m.gap) / (m.cellWidth + m.gap)))
  const rows = Math.max(1, Math.floor((m.height + m.gap) / (m.cellHeight + m.gap)))
  const base = m.gravity === 'top-right'
    ? Math.round(cols - 1 - cf - (sp.w - 1) / 2)
    : Math.round(cf - (sp.w - 1) / 2)
  return {
    col: Math.max(0, Math.min(cols - sp.w, base)),
    row: Math.max(0, Math.min(rows - sp.h, Math.round(rf - (sp.h - 1) / 2))),
  }
}

export function bindDesktopDrag(options: {
  container: HTMLElement | null
  model: DesktopModel | null
  metrics: () => DesktopGridMetrics
  getState: () => DesktopDragState | null
  setState: (s: DesktopDragState | null) => void
  /** 拖拽结束提交后回调（持久化） */
  onCommit?: (id: string) => void
  /** 落点命中文件夹 → 返回 folderId（拖入归组，不再换位） */
  resolveDropTarget?: (id: string, col: number, row: number) => string | null
  /** 拖拽悬停/结束的文件夹高亮 */
  setHoverFolder?: (id: string | null) => void
  /** 拖拽过程中的落点格指示（模型坐标） */
  setDropHint?: (hint: { col: number, row: number } | null) => void
}) {
  /** 激活前的待定起点（pointerdown 记录，超阈值才转正为拖拽） */
  let pending: { id: string, pointerId: number, startX: number, startY: number, item: DesktopItem } | null = null

  function onPointerDown(ev: PointerEvent, item: DesktopItem) {
    if (ev.button !== 0 || !options.container) {
      return
    }
    pending = { id: item.id, pointerId: ev.pointerId, startX: ev.clientX, startY: ev.clientY, item }
    options.setState(null)
  }

  function onPointerMove(ev: PointerEvent) {
    // 激活拖拽：pending + 位移超阈值
    if (pending && pending.pointerId === ev.pointerId) {
      const dist = Math.hypot(ev.clientX - pending.startX, ev.clientY - pending.startY)
      if (dist < DRAG_THRESHOLD || !options.container) {
        return
      }
      const containerRect = options.container.getBoundingClientRect()
      const itemEl = document.elementFromPoint(pending.startX, pending.startY)?.closest('.yw-desktop-cell')
      const itemRect = itemEl?.getBoundingClientRect()
      // 跨格卡：中心吸附指针（居中跟手）；1×1 保持抓取点锚定
      const it0 = options.model?.get(pending.id)
      const sp0 = it0 ? (it0.span ?? { w: 1, h: 1 }) : { w: 1, h: 1 }
      const anchorX = itemRect ? (sp0.w > 1 ? itemRect.width / 2 : pending.startX - itemRect.left) : 42
      const anchorY = itemRect ? (sp0.h > 1 ? itemRect.height / 2 : pending.startY - itemRect.top) : 46
      options.setState({
        id: pending.id,
        pointerId: pending.pointerId,
        offsetX: anchorX,
        offsetY: anchorY,
        x: ev.clientX - containerRect.left - anchorX,
        y: ev.clientY - containerRect.top - anchorY,
      })
      pending = null
    }

    const s = options.getState()
    if (!s || s.pointerId !== ev.pointerId) {
      return
    }
    if (!options.container) {
      return
    }
    const containerRect = options.container.getBoundingClientRect()
    s.x = ev.clientX - containerRect.left - s.offsetX
    s.y = ev.clientY - containerRect.top - s.offsetY
    // 悬停文件夹高亮 + 落点格指示
    const drop = dropCell(options, s)
    if (options.resolveDropTarget && options.setHoverFolder) {
      options.setHoverFolder(options.resolveDropTarget(s.id, drop.col, drop.row))
    }
    options.setDropHint?.(drop)
  }

  function onPointerUp(ev: PointerEvent) {
    pending = null
    const s = options.getState()
    if (!s || s.pointerId !== ev.pointerId) {
      return
    }
    options.setHoverFolder?.(null)
    options.setDropHint?.(null)
    if (!options.container) {
      options.setState(null)
      return
    }
    const drop = dropCell(options, s)
    // 落点命中文件夹 → 移入归组（不换位）
    const folderId = options.resolveDropTarget?.(s.id, drop.col, drop.row)
    if (folderId) {
      options.model?.addToFolder(folderId, s.id)
      options.setState(null)
      options.onCommit?.(s.id)
      return
    }
    options.model?.moveTo(s.id, { col: drop.col, row: drop.row })
    options.setState(null)
    options.onCommit?.(s.id)
  }

  function attach() {
    // move/up 挂 window：拖出容器/图标后事件仍可达（目标是其它兄弟层时不会冒泡进容器）
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    window.addEventListener('pointercancel', onPointerUp)
  }

  function detach() {
    window.removeEventListener('pointermove', onPointerMove)
    window.removeEventListener('pointerup', onPointerUp)
    window.removeEventListener('pointercancel', onPointerUp)
  }

  const handle: DesktopDragHandle = { onPointerDown, attach, detach, container: null, model: null }
  Object.defineProperty(handle, 'container', { get: () => options.container, set: v => (options.container = v) })
  Object.defineProperty(handle, 'model', { get: () => options.model, set: v => (options.model = v) })
  return handle
}
