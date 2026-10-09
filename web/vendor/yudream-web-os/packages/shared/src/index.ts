/** 几何类型 */
export interface Size {
  width: number
  height: number
}

export interface Position {
  x: number
  y: number
}

export interface Rect extends Position, Size {}

/** 网格坐标（栅格模式） */
export interface GridPosition {
  col: number
  row: number
}

/** 自由坐标（像素） */
export type FreePosition = Position

/** 深度 Partial：override 深合并用 */
export type DeepPartial<T> = T extends (infer U)[]
  ? U[]
  : T extends object
    ? { [K in keyof T]?: DeepPartial<T[K]> }
    : T

/** 数值钳制 */
export function clamp(v: number, min: number, max: number): number {
  return Math.min(Math.max(v, min), max)
}

/**
 * VFS 路径清洗：解析相对段、吞掉 `.` 与 `..`，杜绝越权（防 ../ 穿越）。
 * 始终返回以 / 开头的绝对路径。
 */
export function normalizePath(path: string): string {
  const p = path.replaceAll('\\', '/').trim()
  const parts = p.split('/')
  const out: string[] = []
  for (const seg of parts) {
    if (!seg || seg === '.') {
      continue
    }
    if (seg === '..') {
      out.pop()
      continue
    }
    out.push(seg)
  }
  return `/${out.join('/')}`
}

/** 解析相对路径（基于基准目录） */
export function resolvePath(base: string, relative: string): string {
  if (relative.startsWith('/')) {
    return normalizePath(relative)
  }
  return normalizePath(`${base}/${relative}`)
}

/** 父目录（根目录返回自身） */
export function dirname(path: string): string {
  const p = normalizePath(path)
  const idx = p.lastIndexOf('/')
  return idx <= 0 ? '/' : p.slice(0, idx)
}

/** 文件名 */
export function basename(path: string): string {
  const p = normalizePath(path)
  return p.slice(p.lastIndexOf('/') + 1)
}

/** 扩展名（小写、不含点；无扩展名返回 ''） */
export function extname(path: string): string {
  const name = basename(path)
  const idx = name.lastIndexOf('.')
  return idx <= 0 ? '' : name.slice(idx + 1).toLowerCase()
}

/**
 * 深合并（override 用）：数组与普通对象递归合并，其余按 src 覆盖；src 为 undefined 保留 dst。
 */
export function deepMerge<T>(dst: T, src: DeepPartial<T> | undefined): T {
  if (src === undefined) {
    return dst
  }
  if (Array.isArray(dst) || Array.isArray(src)) {
    return src as T
  }
  if (typeof dst === 'object' && typeof src === 'object' && dst !== null && src !== null) {
    const out: Record<string, unknown> = { ...(dst as Record<string, unknown>) }
    for (const [k, v] of Object.entries(src as Record<string, unknown>)) {
      if (v === undefined) {
        continue
      }
      out[k] = k in (dst as Record<string, unknown>)
        ? deepMerge((dst as Record<string, unknown>)[k], v as DeepPartial<unknown>)
        : v
    }
    return out as T
  }
  return src as T
}

/** 短随机 id */
export function shortId(prefix = ''): string {
  const rand = Math.random().toString(36).slice(2, 8)
  return prefix ? `${prefix}-${rand}` : rand
}

/** 单调递增 id 工厂 */
export function createIdFactory(start = 1): () => number {
  let n = start
  return () => n++
}

/** 防抖（persist 300ms 合并写） */
export function debounce<A extends unknown[]>(fn: (...args: A) => void, ms: number): ((...args: A) => void) & { cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | null = null
  const wrapped = (...args: A) => {
    if (timer) {
      clearTimeout(timer)
    }
    timer = setTimeout(() => {
      timer = null
      fn(...args)
    }, ms)
  }
  wrapped.cancel = () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }
  return wrapped
}
