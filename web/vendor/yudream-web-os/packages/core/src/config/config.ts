import type { DeepPartial } from '@yudream/yudream-webos-shared'
import { deepMerge } from '@yudream/yudream-webos-shared'

/** WebOS 全局配置 schema */
export interface WebOSConfig {
  desktop: {
    arrangeMode: 'grid' | 'free'
    /** 桌面项投影到 VFS /Desktop */
    bindVFS: boolean
    collision: 'swap' | 'shift'
  }
  dock: {
    position: 'bottom' | 'left' | 'right'
    magnification: boolean
    autohide: boolean
    showRecent: boolean
    iconSize: number
  }
  widgets: {
    placement: 'sidebar' | 'desktop'
  }
  menubar: {
    showControlCenter: boolean
    showClock: boolean
  }
  windows: {
    snapToEdge: boolean
    sessionRestore: boolean
  }
  persist: {
    adapter: 'indexeddb' | 'localstorage' | 'memory'
    prefix: string
  }
}

export const defaultConfig: WebOSConfig = {
  desktop: { arrangeMode: 'grid', bindVFS: true, collision: 'swap' },
  dock: { position: 'bottom', magnification: true, autohide: false, showRecent: true, iconSize: 48 },
  widgets: { placement: 'sidebar' },
  menubar: { showControlCenter: true, showClock: true },
  windows: { snapToEdge: true, sessionRestore: true },
  persist: { adapter: 'indexeddb', prefix: 'webos' },
}

/** 配置合并：默认值 ← 宿主覆盖（深合并） */
export function resolveConfig(override?: DeepPartial<WebOSConfig>): WebOSConfig {
  return deepMerge(defaultConfig, override)
}
