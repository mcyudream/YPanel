import { defineStore } from 'pinia'

// 桌面工作台窗口管理：组件内窗口层（z-order/拖拽位置/最小化/最大化）。
// 应用清单与窗口实例分开维护；内容组件按 key 懒加载。

export interface WorkbenchApp {
  key: string
  title: string
  icon: string
  width: number
  height: number
}

export interface WorkbenchWindow {
  id: number
  key: string
  title: string
  icon: string
  x: number
  y: number
  width: number
  height: number
  minimized: boolean
  maximized: boolean
  z: number
}

export const useWorkbenchStore = defineStore('workbench', () => {
  // 应用注册表（内容组件在桌面壳中按 key 映射）
  const apps: WorkbenchApp[] = [
    { key: 'overview', title: '主机概览', icon: 'gauge', width: 900, height: 600 },
    { key: 'file', title: '文件管理', icon: 'folder-open', width: 960, height: 620 },
    { key: 'container', title: '容器管理', icon: 'container', width: 960, height: 600 },
    { key: 'terminal', title: '终端', icon: 'square-terminal', width: 880, height: 560 },
    { key: 'database', title: '数据库', icon: 'database', width: 960, height: 620 },
    { key: 'sites', title: '网站', icon: 'globe', width: 960, height: 600 },
    { key: 'compose', title: 'Compose 编排', icon: 'layers', width: 960, height: 600 },
    { key: 'cron', title: '计划任务', icon: 'calendar-clock', width: 920, height: 580 },
    { key: 'icons', title: '图标库', icon: 'shapes', width: 900, height: 560 },
  ]

  const windows = ref<WorkbenchWindow[]>([])
  const topZ = ref(10)
  const nextId = ref(1)
  const focusedId = ref(0)

  function open(key: string) {
    const app = apps.find(a => a.key === key)
    if (!app) {
      return
    }
    // 已打开 → 聚焦并还原
    const existing = windows.value.find(w => w.key === key)
    if (existing) {
      existing.minimized = false
      focus(existing.id)
      return
    }
    const id = nextId.value++
    const vw = window.innerWidth
    const vh = window.innerHeight
    const w = Math.min(app.width, vw - 80)
    const h = Math.min(app.height, vh - 140)
    topZ.value++
    windows.value.push({
      id,
      key,
      title: app.title,
      icon: app.icon,
      x: Math.max(20, (vw - w) / 2 + ((id % 5) - 2) * 30),
      y: Math.max(20, (vh - h) / 2 - 30 + ((id % 4) - 1) * 24),
      width: w,
      height: h,
      minimized: false,
      maximized: false,
      z: topZ.value,
    })
    focus(id)
  }

  function focus(id: number) {
    topZ.value++
    const w = windows.value.find(i => i.id === id)
    if (w) {
      w.z = topZ.value
    }
    focusedId.value = id
  }

  function close(id: number) {
    windows.value = windows.value.filter(w => w.id !== id)
  }

  function toggleMinimize(id: number) {
    const w = windows.value.find(i => i.id === id)
    if (w) {
      w.minimized = !w.minimized
      if (!w.minimized) {
        focus(id)
      }
    }
  }

  function toggleMaximize(id: number) {
    const w = windows.value.find(i => i.id === id)
    if (w) {
      w.maximized = !w.maximized
      focus(id)
    }
  }

  function move(id: number, x: number, y: number) {
    const w = windows.value.find(i => i.id === id)
    if (w) {
      w.x = x
      w.y = y
    }
  }

  // F20：窗口布局持久化（localStorage）——刷新/重登后恢复窗口位置与尺寸
  const LS_KEY = 'ypanel.workbench.layout'

  function persist() {
    try {
      const snapshot = windows.value.map(w => ({
        key: w.key, x: w.x, y: w.y, width: w.width, height: w.height,
        minimized: w.minimized, maximized: w.maximized, z: w.z,
      }))
      localStorage.setItem(LS_KEY, JSON.stringify({ topZ: topZ.value, nextId: nextId.value, windows: snapshot }))
    }
    catch {}
  }

  function restore() {
    try {
      const raw = localStorage.getItem(LS_KEY)
      if (!raw) {
        return
      }
      const saved = JSON.parse(raw) as { topZ?: number, nextId?: number, windows?: Array<{ key: string, x: number, y: number, width: number, height: number, minimized?: boolean, maximized?: boolean, z?: number }> }
      if (!saved.windows?.length) {
        return
      }
      const restored: WorkbenchWindow[] = []
      for (const w of saved.windows) {
        const app = apps.find(a => a.key === w.key)
        if (!app) {
          continue
        }
        const vw = window.innerWidth
        const vh = window.innerHeight
        restored.push({
          id: w.z ?? restored.length + 1,
          key: w.key,
          title: app.title,
          icon: app.icon,
          x: Math.max(0, Math.min(w.x, vw - 80)),
          y: Math.max(0, Math.min(w.y, vh - 80)),
          width: Math.min(w.width, vw - 40),
          height: Math.min(w.height, vh - 80),
          minimized: !!w.minimized,
          maximized: !!w.maximized,
          z: w.z ?? restored.length + 1,
        })
      }
      if (restored.length) {
        windows.value = restored
        nextId.value = Math.max(saved.nextId ?? 1, ...restored.map(w => w.id + 1))
        topZ.value = Math.max(10, saved.topZ ?? 10)
        const visible = restored.find(w => !w.minimized)
        if (visible) {
          focusedId.value = visible.id
        }
      }
    }
    catch {}
  }

  restore()

  // 任何窗口状态变化（移动/缩放/开关/最小化）都自动持久化
  watch(windows, () => persist(), { deep: true })

  return { apps, windows, focusedId, open, focus, close, toggleMinimize, toggleMaximize, move, persist }
})
