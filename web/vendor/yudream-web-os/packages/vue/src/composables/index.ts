/**
 * Composables：宿主自定义能力的入口。全部薄层，包装 WebOSInstance 上的 core 服务。
 */
import type { AppDefinition, DesktopModel, SnapZone, WindowManager, YwBusEvents } from '@yudream/yudream-webos-core'
import type { ShallowRef } from 'vue'
import { mergeMenus } from '@yudream/yudream-webos-core'
import { computed, onScopeDispose, reactive, ref, shallowRef, triggerRef } from 'vue'
import { useWebOS } from '../provider'
import { applySettings } from '../system/settings'

/** 窗口管理器 + 响应式窗口列表（按 zIndex 升序） */
export function useWindowManager() {
  const { wm } = useWebOS()
  const windows: ShallowRef<ReturnType<WindowManager['list']>> = shallowRef([])
  const focusedId = ref<string | null>(null)

  let raf = 0
  const sync = () => {
    if (raf) {
      return
    }
    raf = requestAnimationFrame(() => {
      raf = 0
      windows.value = wm.list()
      focusedId.value = wm.focusedWindow()?.id ?? null
    })
  }
  const offs = [
    wm['events' as never] ? () => {} : () => {},
  ]
  void offs
  // 直接订阅 wm 内部回调（events 对象可变引用）
  const evts = (wm as unknown as { events: Record<string, ((w: any) => void) | undefined> }).events
  const hooks: Array<keyof typeof evts> = ['onOpen', 'onClose', 'onFocus', 'onBlur', 'onMove', 'onResize', 'onStateChange']
  const originals = new Map<keyof typeof evts, ((w: any) => void) | undefined>()
  for (const key of hooks) {
    originals.set(key, evts[key])
    evts[key] = (w) => {
      originals.get(key)?.(w)
      sync()
    }
  }
  onScopeDispose(() => {
    for (const key of hooks) {
      evts[key] = originals.get(key)
    }
  })

  return {
    wm,
    windows,
    focusedId,
    open: wm.open.bind(wm),
    close: (id: string) => {
      wm.close(id)
      sync()
    },
    focus: (id: string) => {
      wm.focus(id)
      sync()
    },
    blur: wm.blur.bind(wm),
    minimize: (id: string) => {
      wm.minimize(id)
      sync()
    },
    maximize: (id: string) => {
      wm.maximize(id)
      sync()
    },
    fullscreen: (id: string) => {
      wm.fullscreen(id)
      sync()
    },
    restore: (id: string) => {
      wm.restore(id)
      sync()
    },
    snap: (id: string, zone: SnapZone) => {
      wm.snap(id, zone)
      sync()
    },
    tileAll: () => {
      wm.tileAll()
      sync()
    },
  }
}

/** 应用注册表 */
export function useAppRegistry() {
  const os = useWebOS()
  const { registry, openApp } = os
  const apps = ref<AppDefinition[]>(registry.list())
  registry.onChange(() => {
    apps.value = registry.list()
    triggerRef(apps)
  })

  /** 运行时注册：登记窗口规格、组件载荷与附带小组件（与 createWebOSInstance 初始化同构） */
  function register(app: AppDefinition) {
    registry.register(app)
    if (app.component !== undefined) {
      os.appComponents.set(app.id, app.component)
    }
    for (const w of app.widgets ?? []) {
      os.widgets.register(w)
    }
    os.wm.registerSpec(app.id, {
      appId: app.id,
      title: app.name,
      icon: app.icon,
      defaultSize: app.defaultSize,
      minSize: app.minSize,
      maxSize: app.maxSize,
      resizable: app.resizable,
      frameless: app.frameless,
      singleton: app.singleton,
      multiInstance: app.multiInstance,
      defaultPosition: app.defaultPosition,
    })
  }

  return {
    registry,
    apps,
    openApp,
    register,
    registerAll: registry.registerAll.bind(registry),
    unregister: registry.unregister.bind(registry),
    override: registry.override.bind(registry),
    get: registry.get.bind(registry),
  }
}

/** 桌面布局 */
export function useDesktop() {
  const { desktop } = useWebOS()
  const items = ref(desktop.list())
  desktop.onChange(() => {
    items.value = desktop.list()
  })
  return {
    desktop,
    items,
    add: (item: Parameters<DesktopModel['add']>[0]) => {
      const r = desktop.add(item)
      return r
    },
    remove: (id: string) => desktop.remove(id),
    moveTo: (id: string, pos: any) => desktop.moveTo(id, pos),
    createFolder: (name: string, ids: string[]) => desktop.createFolder(name, ids),
    dissolveFolder: (id: string) => desktop.dissolveFolder(id),
    sortBy: (c: 'name' | 'kind' | 'dateModified') => desktop.sortBy(c),
    autoArrange: () => desktop.autoArrange(),
  }
}

/** Dock */
export function useDock() {
  const { dock, config } = useWebOS()
  const pinned = ref([...dock.pinned])
  dock.onChange(() => {
    pinned.value = [...dock.pinned]
  })
  return {
    dock,
    config: config.dock,
    pinned,
    pin: (id: string, index?: number) => dock.pin(id, index),
    unpin: (id: string) => dock.unpin(id),
    reorder: (from: number, to: number) => dock.reorder(from, to),
    setBadge: (id: string, badge: string | number | null) => dock.setBadge(id, badge),
  }
}

/** VFS */
export function useVFS() {
  const { vfs } = useWebOS()
  return { vfs }
}

/** Finder 导航状态（导航栈 + 选中集，视图无关） */
export function useFinder() {
  const state = reactive({
    path: '/',
    history: ['/'],
    historyIndex: 0,
    view: 'icon' as 'icon' | 'list' | 'column' | 'gallery',
    selection: [] as string[],
  })
  return {
    state,
    navigate(path: string) {
      state.path = path
      state.history = [...state.history.slice(0, state.historyIndex + 1), path]
      state.historyIndex = state.history.length - 1
      state.selection = []
    },
    back() {
      if (state.historyIndex > 0) {
        state.historyIndex--
        state.path = state.history[state.historyIndex]!
      }
    },
    forward() {
      if (state.historyIndex < state.history.length - 1) {
        state.historyIndex++
        state.path = state.history[state.historyIndex]!
      }
    },
    setView(v: typeof state.view) {
      state.view = v
    },
    select(ids: string[]) {
      state.selection = ids
    },
  }
}

/** 小组件实例管理 */
export function useWidgets() {
  const { widgets } = useWebOS()
  const instances = ref(widgets.listInstances())
  // editing 是普通类字段（非响应式）：经 onChange 手动同步，否则编辑态永不更新
  const editing = ref(widgets.editing)
  widgets.onChange(() => {
    instances.value = widgets.listInstances()
    editing.value = widgets.editing
  })
  return {
    widgets,
    instances,
    editing,
    setEditing: (v: boolean) => widgets.setEditing(v),
    add: widgets.addInstance.bind(widgets),
    remove: widgets.removeInstance.bind(widgets),
    move: widgets.moveInstance.bind(widgets),
    resize: widgets.resizeInstance.bind(widgets),
    setConfig: widgets.setConfig.bind(widgets),
  }
}

/** 菜单栏状态（聚焦应用的应用菜单合并结果） */
export function useMenuBar() {
  const { registry, bus } = useWebOS()
  const systemMenus = ref<any[]>([])
  const focusedAppId = ref<string | null>(null)
  // 聚焦应用变化 → 重算合并菜单
  bus.on('webos:window:focus', ({ appId }) => {
    focusedAppId.value = appId
  })
  bus.on('webos:window:close', ({ appId }) => {
    if (focusedAppId.value === appId) {
      focusedAppId.value = null
    }
  })
  const mergedMenus = computed(() => {
    const app = focusedAppId.value ? registry.get(focusedAppId.value) : undefined
    return mergeMenus({ systemMenus: systemMenus.value, app: app?.menus })
  })
  return {
    systemMenus,
    focusedAppId,
    mergedMenus,
    setSystemMenus: (menus: any[]) => { systemMenus.value = menus },
  }
}

/** 系统设置（壁纸/主题/强调色），持久化 scope: system */
export function useSystemSettings() {
  const { settings, persist, bus } = useWebOS()
  const effectiveMode = computed<'light' | 'dark'>(() =>
    settings.mode === 'system' ? (settings.systemDark ? 'dark' : 'light') : settings.mode,
  )

  async function load() {
    const saved = await persist.get<{ mode?: typeof settings.mode, accent?: string | null, wallpaper?: typeof settings.wallpaper }>(`system`)
    if (saved) {
      if (saved.mode) {
        settings.mode = saved.mode
      }
      if (saved.accent !== undefined) {
        settings.accent = saved.accent
      }
      if (saved.wallpaper !== undefined) {
        settings.wallpaper = saved.wallpaper
      }
      // 恢复后必须重新应用（否则持久化的主题/强调色不生效直到下次变更）
      applyNow()
    }
  }

  async function save() {
    await persist.set(`system`, {
      mode: settings.mode,
      accent: settings.accent,
      wallpaper: settings.wallpaper,
    })
  }

  function applyNow() {
    applySettings(settings)
    bus.emit('webos:system:theme-change', { mode: settings.mode === 'system' ? (settings.systemDark ? 'dark' : 'light') : settings.mode })
  }

  function setMode(mode: typeof settings.mode) {
    settings.mode = mode
    void save()
    applyNow()
  }

  function setAccent(color: string | null) {
    settings.accent = color
    void save()
    applyNow()
  }

  function setWallpaper(w: typeof settings.wallpaper) {
    settings.wallpaper = w
    void save()
    applyNow()
    bus.emit('webos:system:wallpaper-change', { src: w?.src ?? null })
  }

  /** 应用设置到 DOM（供 Provider/宿主手动刷新） */
  function apply() {
    // 局部 import 避免循环；实际同包直接引用
    applySettings(settings)
  }

  return { settings, effectiveMode, load, setMode, setAccent, setWallpaper, apply }
}

/** 订阅全部 webos:* 事件 */
export function useWebOSEvents() {
  const { bus } = useWebOS()
  function on<K extends keyof YwBusEvents>(event: K, handler: (payload: YwBusEvents[K]) => void): () => void {
    return bus.on(event, handler)
  }
  function once<K extends keyof YwBusEvents>(event: K, handler: (payload: YwBusEvents[K]) => void): () => void {
    return bus.once(event, handler)
  }
  return { on, once }
}
