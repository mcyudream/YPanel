/**
 * Pointer Events 拖拽/缩放统一封装。
 * 拖拽：目标元素 pointerdown 后监听 window 级 pointermove/pointerup。
 * 手写实现（规范强约束：禁止引入 sortablejs/dnd 库）。
 */
import type { Ref } from 'vue'

export interface PointerDragOptions {
  /** 每次移动回调（相对起点的位移） */
  onMove: (dx: number, dy: number, ev: PointerEvent) => void
  onEnd?: (dx: number, dy: number, ev: PointerEvent) => void
}

/** 生成 pointerdown 处理器；返回的 start 绑定到把手元素 */
export function usePointerDrag(options: PointerDragOptions) {
  function start(ev: PointerEvent) {
    if (ev.button !== 0) {
      return
    }
    ev.preventDefault()
    ev.stopPropagation()
    const sx = ev.clientX
    const sy = ev.clientY
    let dx = 0
    let dy = 0

    const onMove = (e: PointerEvent) => {
      dx = e.clientX - sx
      dy = e.clientY - sy
      options.onMove(dx, dy, e)
    }
    const onUp = (e: PointerEvent) => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      options.onEnd?.(dx, dy, e)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  return { start }
}

/** 缩放方向 */
export type ResizeDirection = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

export interface PointerResizeOptions {
  direction: ResizeDirection
  /** 起始矩形 */
  rect: { x: number, y: number, width: number, height: number }
  minWidth?: number
  minHeight?: number
  onResize: (rect: { x: number, y: number, width: number, height: number }, ev: PointerEvent) => void
  onEnd?: (rect: { x: number, y: number, width: number, height: number }, ev: PointerEvent) => void
}

/** 缩放处理器：根据方向调整矩形，min 尺寸约束 */
export function usePointerResize(options: PointerResizeOptions) {
  function start(ev: PointerEvent) {
    if (ev.button !== 0) {
      return
    }
    ev.preventDefault()
    ev.stopPropagation()
    document.body.style.userSelect = 'none'
    const { direction, rect: r0 } = options
    const minW = options.minWidth ?? 240
    const minH = options.minHeight ?? 160
    const sx = ev.clientX
    const sy = ev.clientY
    let last = { ...r0 }

    const onMove = (e: PointerEvent) => {
      const dx = e.clientX - sx
      const dy = e.clientY - sy
      let { x, y, width, height } = r0

      if (direction.includes('e')) {
        width = Math.max(minW, r0.width + dx)
      }
      if (direction.includes('s')) {
        height = Math.max(minH, r0.height + dy)
      }
      if (direction.includes('w')) {
        width = Math.max(minW, r0.width - dx)
        x = r0.x + (r0.width - width)
      }
      if (direction.includes('n')) {
        height = Math.max(minH, r0.height - dy)
        y = r0.y + (r0.height - height)
      }

      last = { x, y, width, height }
      options.onResize(last, e)
    }
    const onUp = (e: PointerEvent) => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      document.body.style.userSelect = ''
      options.onEnd?.(last, e)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  return { start }
}

/** 视口尺寸跟踪（窗口管理机依赖） */
export function useViewport(target?: Ref<{ width: number, height: number } | undefined>) {
  function sync() {
    target?.value?.valueOf()
    return { width: window.innerWidth, height: window.innerHeight }
  }
  return { sync }
}
