/**
 * ui 层兼容 stores：同名 API 包装 WebOSInstance（core 服务），
 * 使组件从旧 Pinia 架构平滑消费新绑定层。非 Pinia——reactive 对象内 ref/computed 自动解包。
 *
 * 单例契约：store 按 WebOSInstance 缓存（WeakMap），同一实例内所有组件共享同一份
 * 状态与 bus 订阅；直接调用工厂函数才会得到独立副本（仅测试场景使用）。
 */
import type { AppDefinition, SnapZone, WindowInstance } from '@yudream/yudream-webos-core'

import type { WebOSInstance, YwSettingsState } from '@yudream/yudream-webos-vue'
import * as core0 from '@yudream/yudream-webos-core'
import { useSystemSettings, useWebOS } from '@yudream/yudream-webos-vue'

import { computed, defineAsyncComponent, markRaw, reactive, ref, watch } from 'vue'

let require0Cache: ReturnType<typeof require0Impl> | null = null
function require0Impl() {
  return core0
}
function require0() {
  return require0Cache ?? (require0Cache = require0Impl())
}

/** 异步 loader → 异步组件缓存（keyed by loader 引用，避免每次渲染重建组件导致反复挂载） */
const asyncCompCache = new WeakMap<() => unknown, unknown>()

/** 注册载荷归一化：函数 loader 归一化为 defineAsyncComponent（缓存），组件原样返回 */
function normalizeComponent(comp: unknown): unknown {
  if (typeof comp !== 'function') {
    return comp
  }
  let cached = asyncCompCache.get(comp as () => unknown)
  if (!cached) {
    cached = markRaw(defineAsyncComponent(comp as () => Promise<any>))
    asyncCompCache.set(comp as () => unknown, cached)
  }
  return cached
}

/** ── windows ── */
function createWindowsStore(os: WebOSInstance) {
  const { wm, appComponents, bus, config, persist } = os
  const viewport = ref({ ...wm.getViewport() })

  /** 窗口内容载荷（windowId → 载荷）；恢复的窗口由 reconcile 从注册表重建 */
  const payloads = reactive(new Map<string, unknown>())

  const persistSession = () => {
    void persist.set(`windows.session`, wm.serialize())
  }

  const store = reactive({
    wm,
    state: {} as { windows: WindowInstance[] },
    viewport,
    windows: new Proxy([] as WindowInstance[], {}) as WindowInstance[],
    visible: [] as WindowInstance[],
    focusedId: null as string | null,
    runningApps: [] as string[],
    setViewport(v: { width: number, height: number, menubarHeight?: number }) {
      viewport.value = { ...v }
      wm.setViewport(v)
    },
    restore() {
      void persist.get<WindowInstance[]>(`windows.session`).then((snaps) => {
        if (snaps?.length) {
          wm.restoreSession(snaps as never)
          sync()
        }
      })
    },
    openComponent(payload: { appKey: string, component?: unknown, title: string, singleton?: boolean, multiInstance?: boolean, defaultSize?: { width: number, height: number } }) {
      // 单例聚焦
      const existing = wm.windowsOfApp(payload.appKey)[0]
      if (payload.singleton && existing) {
        wm.focus(existing.id)
        sync()
        return existing.zIndex
      }
      const win = wm.open(payload.appKey, { title: payload.title })
      if (payload.component !== undefined) {
        payloads.set(win.id, { appId: payload.appKey, component: normalizeComponent(payload.component), title: payload.title })
      }
      void payload.multiInstance
      void payload.defaultSize
      sync()
      return win.id
    },
    focus(id: string) {
      wm.focus(id)
      sync()
    },
    close(id: string) {
      wm.close(id)
      payloads.delete(id)
      sync()
    },
    toggleMinimize(id: string) {
      const win = wm.get(id)
      if (win?.state === 'minimized') {
        wm.restore(id)
      }
      else {
        wm.minimize(id)
      }
      sync()
    },
    toggleMaximize(id: string) {
      wm.maximize(id)
      sync()
    },
    moveTo(id: string, x: number, y: number) {
      wm.move(id, { x, y })
      sync()
    },
    resizeTo(id: string, rect: { x: number, y: number, width: number, height: number }) {
      wm.resize(id, rect)
      sync()
    },
    snap(id: string, zone: SnapZone) {
      wm.snap(id, zone)
      sync()
    },
    tileAll() {
      wm.tileAll()
      sync()
    },
    payloadOf(id: string) {
      const win = wm.get(id)
      if (!win) {
        return undefined
      }
      // ① 窗口级载荷（openComponent / reconcile 写入）优先
      const attached = payloads.get(id)
      if (attached !== undefined) {
        return attached as { appId: string, component: unknown, title: string }
      }
      // ② 注册表组件载荷兜底（恢复会话 / 未走 openComponent 的路径），归一化后回写缓存
      const comp = appComponents.get(win.appId)
      if (comp === undefined) {
        return undefined
      }
      const payload = { appId: win.appId, component: normalizeComponent(comp), title: win.title }
      payloads.set(id, payload)
      return payload
    },
    attachPayload(id: string, payload: { appId: string, component?: unknown, title: string }) {
      payloads.set(id, {
        appId: payload.appId,
        title: payload.title,
        component: payload.component !== undefined ? normalizeComponent(payload.component) : undefined,
      })
    },
  })

  /** persist=false 用于创建时的首次同步：此时恢复流程尚未发生，写入会把空列表固化为会话 */
  function sync(persist = true) {
    if (persist) {
      persistSession()
    }
    const list = wm.list()
    // 关键：浅拷贝每个窗口（wm 直接 mutate bounds，不拷贝新引用 Vue 检测不到）
    store.windows = list.map(w => ({ ...w, bounds: { ...w.bounds } })) as unknown as WindowInstance[]
    store.visible = store.windows.filter(w => w.state !== 'minimized')
    store.focusedId = wm.focusedWindow()?.id ?? null
    store.runningApps = [...new Set(list.map(w => w.appId))]
    store.state = { windows: store.windows }
  }

  // 订阅 bus 事件同步（webos:window:*）——单例下只挂一次
  for (const ev of ['webos:window:open', 'webos:window:close', 'webos:window:focus', 'webos:window:blur', 'webos:window:move', 'webos:window:resize', 'webos:window:state-change'] as const) {
    bus.on(ev, () => sync())
  }

  void config
  sync(false)
  return store
}

const windowsStores = new WeakMap<WebOSInstance, ReturnType<typeof createWindowsStore>>()
export function useWindowsStore(): ReturnType<typeof createWindowsStore> {
  const os = useWebOS()
  let s = windowsStores.get(os)
  if (!s) {
    s = createWindowsStore(os)
    windowsStores.set(os, s)
  }
  return s
}

/** ── apps ── */
function createAppsStore(os: WebOSInstance) {
  const { registry, openApp, ui, wm, appComponents, widgets } = os
  const appsRef = ref<AppDefinition[]>(registry.list())
  registry.onChange(() => {
    appsRef.value = registry.list()
  })

  const store = reactive({
    state: {} as { apps: Record<string, AppDefinition>, disabled: string[] },
    apps: appsRef,
    dock: computed(() => registry.dockApps()),
    launchpad: computed(() => registry.launchpadApps()),
    all: {} as Record<string, AppDefinition>,
    register(app: AppDefinition) {
      registry.register(app)
      // 运行时注册：登记窗口规格、组件载荷与附带小组件（与 createWebOSInstance 初始化同构）
      if (app.component !== undefined) {
        appComponents.set(app.id, app.component)
      }
      for (const w of app.widgets ?? []) {
        widgets.register(w)
      }
      wm.registerSpec(app.id, {
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
    },
    unregister(id: string) {
      registry.unregister(id)
    },
    setEnabled(id: string, enabled: boolean) {
      if (!enabled) {
        // 简化：禁用 = 从启动台隐藏（dock 保留）——完整 enable/disable 状态在持久化层
        void id
        void enabled
      }
    },
    openApp(id: string): boolean {
      if (!registry.get(id)) {
        ui.message('error', '应用不存在')
        return false
      }
      openApp(id)
      return true
    },
  })

  const syncAll = () => {
    const map: Record<string, AppDefinition> = {}
    for (const app of registry.list()) {
      map[app.id] = app
    }
    store.all = map
    store.state = { apps: map, disabled: [] }
  }
  registry.onChange(syncAll)
  syncAll()
  return store
}

const appsStores = new WeakMap<WebOSInstance, ReturnType<typeof createAppsStore>>()
export function useAppsStore(): ReturnType<typeof createAppsStore> {
  const os = useWebOS()
  let s = appsStores.get(os)
  if (!s) {
    s = createAppsStore(os)
    appsStores.set(os, s)
  }
  return s
}

/** ── theme（系统设置包装） ── */
function createThemeStore(os: WebOSInstance) {
  const settings = os.settings as unknown as YwSettingsState
  const { persist, bus } = os
  const sys = useSystemSettings()
  const store = reactive({
    mode: computed(() => settings.mode),
    effectiveMode: computed(() => sys.effectiveMode.value),
    systemDark: computed(() => settings.systemDark),
    wallpaper: computed(() => settings.wallpaper),
    accent: computed(() => settings.accent),
    setMode(m: 'light' | 'dark' | 'system') {
      sys.setMode(m)
      applyNow()
    },
    setSystemDark(v: boolean) {
      settings.systemDark = v
      applyNow()
    },
    setWallpaper(w: YwSettingsState['wallpaper']) {
      sys.setWallpaper(w)
      applyNow()
    },
    setAccent(color: string | null) {
      sys.setAccent(color)
      applyNow()
    },
    apply: applyNow,
  })

  function applyNow() {
    const root = document.documentElement
    const dark = settings.mode === 'system' ? settings.systemDark : settings.mode === 'dark'
    root.classList.toggle('dark', dark)
    root.style.colorScheme = dark ? 'dark' : 'light'
    if (settings.accent) {
      const { hexToOklchChannels } = require0()
      const ch = settings.accent.startsWith('#') ? hexToOklchChannels(settings.accent) : settings.accent
      root.style.setProperty('--yw-primary', ch)
      root.style.setProperty('--yw-ring', ch)
    }
    else {
      root.style.removeProperty('--yw-primary')
      root.style.removeProperty('--yw-ring')
    }
    bus.emit?.('webos:system:theme-change', { mode: dark ? 'dark' : 'light' })
  }

  watch(() => settings.mode, applyNow)
  void persist
  return store
}

const themeStores = new WeakMap<WebOSInstance, ReturnType<typeof createThemeStore>>()
export function useThemeStore(): ReturnType<typeof createThemeStore> {
  const os = useWebOS()
  let s = themeStores.get(os)
  if (!s) {
    s = createThemeStore(os)
    themeStores.set(os, s)
  }
  return s
}

/** ── browser（多标签会话状态保留在此层） ── */
export interface BrowserSession {
  url: string
  history: string[]
  index: number
  loading: boolean
}

const DEFAULT_SANDBOX = ['allow-scripts', 'allow-same-origin', 'allow-forms', 'allow-popups-to-escape-sandbox']

/** 地址栏输入解析：URL / 补协议 / 搜索引擎回退 */
export function resolveInput(input: string, searchEngine: string): string {
  const v = input.trim()
  if (!v) {
    return ''
  }
  if (/^https?:\/\//i.test(v)) {
    return v
  }
  if (!/\s/.test(v) && /^[\w-]+(?:\.[\w-]+)+(?:\/\S*)?$/.test(v)) {
    return `https://${v}`
  }
  return searchEngine.replace('{q}', encodeURIComponent(v))
}

export interface YwBrowserOptions {
  searchEngine?: string
  sandbox?: string[]
  allow?: string[]
  disabled?: boolean
  openExternalTargets?: string[]
}

function createBrowserStore(os: WebOSInstance) {
  const { openApp, wm } = os
  const options = ref<YwBrowserOptions>({})
  const sessions = reactive(new Map<string, BrowserSession>())

  const store = reactive({
    options,
    sessions,
    configure(opts: YwBrowserOptions) {
      options.value = { ...opts }
    },
    sessionOf(windowId: string): BrowserSession {
      let s = sessions.get(windowId)
      if (!s) {
        s = { url: '', history: [], index: -1, loading: false }
        sessions.set(windowId, s)
      }
      return s
    },
    navigate(windowId: string, url: string) {
      const s = store.sessionOf(windowId)
      if (!url || s.url === url) {
        return
      }
      s.history = [...s.history.slice(0, s.index + 1), url]
      s.index = s.history.length - 1
      s.url = url
      s.loading = true
    },
    back(windowId: string) {
      const s = store.sessionOf(windowId)
      if (s.index > 0) {
        s.index--
        s.url = s.history[s.index]!
        s.loading = true
      }
    },
    forward(windowId: string) {
      const s = store.sessionOf(windowId)
      if (s.index < s.history.length - 1) {
        s.index++
        s.url = s.history[s.index]!
        s.loading = true
      }
    },
    reload(windowId: string) {
      const s = store.sessionOf(windowId)
      if (s.url) {
        s.loading = true
      }
    },
    setLoading(windowId: string, loading: boolean) {
      store.sessionOf(windowId).loading = loading
    },
    canBack(id: string) {
      return store.sessionOf(id).index > 0
    },
    canForward(id: string) {
      return store.sessionOf(id).index < store.sessionOf(id).history.length - 1
    },
    openUrl(url: string, target: 'browser' | 'external' = 'browser') {
      if (target === 'external') {
        window.open(url, '_blank', 'noopener')
        return
      }
      const existing = wm.list().find(w => w.appId === 'browser')
      if (existing) {
        wm.focus(existing.id)
        store.navigate(existing.id, url)
      }
      else {
        const win = wm.open('browser')
        store.navigate(win.id, url)
      }
    },
  })

  void openApp
  void DEFAULT_SANDBOX
  return store
}

const browserStores = new WeakMap<WebOSInstance, ReturnType<typeof createBrowserStore>>()
export function useBrowserStore(): ReturnType<typeof createBrowserStore> {
  const os = useWebOS()
  let s = browserStores.get(os)
  if (!s) {
    s = createBrowserStore(os)
    browserStores.set(os, s)
  }
  return s
}

/** 浏览器窗口内容组件引用（apps 层注册） */
let browserComponent: unknown = null
export function registerBrowserComponent(comp: unknown) {
  browserComponent = comp
}
export function resolveBrowserComponent(): unknown {
  return browserComponent
}
