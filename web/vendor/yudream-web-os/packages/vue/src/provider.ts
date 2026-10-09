import type { PersistenceAdapter, WebOSConfig, YwEventBus, YwUiAdapter } from '@yudream/yudream-webos-core'
import type { App, InjectionKey } from 'vue'
import type { YwSettingsState } from './system/settings'
import {
  AppRegistry,
  createEventBus,
  DesktopModel,
  DockModel,
  LocalStoragePersistence,
  MemoryPersistence,
  resolveConfig,
  VFS,
  WidgetStore,
  WindowManager,
} from '@yudream/yudream-webos-core'
import { inject } from 'vue'
import { builtinAdapter } from './adapter/builtin'
import { createSettingsState } from './system/settings'

/** WebOS 服务容器（一次装配，全部 composables 共享） */
export interface WebOSInstance {
  bus: YwEventBus
  registry: AppRegistry
  wm: WindowManager
  desktop: DesktopModel
  dock: DockModel
  widgets: WidgetStore
  vfs: VFS
  persist: PersistenceAdapter
  config: WebOSConfig
  /** UI 适配器（菜单/对话框/提示） */
  ui: YwUiAdapter
  /** 系统设置状态（壁纸/主题/强调色） */
  settings: YwSettingsState
  /** 持久化 scope 前缀 */
  storagePrefix: string
  /** 应用组件表（AppDefinition.component 载荷，ui 层渲染） */
  appComponents: Map<string, unknown>
  /** 打开应用（归一化入口） */
  openApp: (appId: string, launchOptions?: Record<string, unknown>) => void
  /** 打开 URL（内置浏览器接管或外链） */
  openUrl: (url: string, target?: 'browser' | 'external') => void
}

export const WEBOS_KEY: InjectionKey<WebOSInstance> = Symbol('webos-instance')
export const YW_UI_ADAPTER_KEY: InjectionKey<YwUiAdapter> = Symbol('yw-ui-adapter')

export interface CreateWebOSOptions {
  ui?: YwUiAdapter
  /** 应用定义列表（内置应用 + 宿主应用同通道） */
  apps?: Parameters<AppRegistry['registerAll']>[0]
  /** 运行前预覆盖（appId → patch） */
  overrides?: Record<string, Parameters<AppRegistry['override']>[1]>
  persist?: { adapter?: PersistenceAdapter | 'indexeddb' | 'localstorage' | 'memory', prefix?: string }
  desktop?: Partial<WebOSConfig['desktop']>
  dock?: Partial<WebOSConfig['dock']>
  widgets?: Partial<WebOSConfig['widgets']>
  menubar?: Partial<WebOSConfig['menubar']>
  windows?: Partial<WebOSConfig['windows']>
}

/** 构造 WebOS 服务容器（不依赖 Vue app） */
export function createWebOSInstance(options: CreateWebOSOptions = {}): WebOSInstance {
  const config = resolveConfig({
    desktop: options.desktop,
    dock: options.dock,
    widgets: options.widgets,
    menubar: options.menubar,
    windows: options.windows,
  })

  const bus = createEventBus<any>()
  const registry = new AppRegistry()
  const wm = new WindowManager()
  // 事件回调占位：wmWatch 在装配完成后覆写并桥接到 bus
  ;(wm as unknown as { events: Record<string, ((w: unknown) => void) | undefined> }).events = {}
  const desktop = new DesktopModel()
  desktop.arrangeMode = config.desktop.arrangeMode
  desktop.collision = config.desktop.collision
  const dock = new DockModel({
    position: config.dock.position,
    magnification: config.dock.magnification,
    autohide: config.dock.autohide,
    iconSize: config.dock.iconSize,
  })
  const widgets = new WidgetStore()
  widgets.placement = config.widgets.placement
  const vfs = new VFS()

  // 持久化后端
  let persist: PersistenceAdapter
  const pa = options.persist?.adapter
  if (typeof pa === 'object') {
    persist = pa
  }
  else if (pa === 'memory') {
    persist = new MemoryPersistence()
  }
  else {
    // indexeddb / localstorage：scope 持久走 localStorage 防抖实现（VFS 自身用 IndexedDBAdapter）
    persist = new LocalStoragePersistence(undefined, options.persist?.prefix ?? config.persist.prefix)
  }

  const instance: WebOSInstance = {
    bus,
    registry,
    wm,
    desktop,
    dock,
    widgets,
    vfs,
    persist,
    config,
    ui: options.ui ?? builtinAdapter,
    settings: createSettingsState(),
    storagePrefix: options.persist?.prefix ?? config.persist.prefix,
    appComponents: new Map(),
    openApp: () => {},
    openUrl: () => {},
  }

  // 注册应用：spec 登记窗口约束、组件表、小组件
  if (options.apps?.length) {
    registry.registerAll(options.apps)
    for (const app of options.apps) {
      if (app.component !== undefined) {
        instance.appComponents.set(app.id, app.component)
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
    }
  }
  for (const [id, patch] of Object.entries(options.overrides ?? {})) {
    registry.override(id, patch)
  }

  // 窗口事件 → webos:* 总线
  wmWatch(wm, bus)

  // 打开应用归一化
  instance.openApp = (appId, launchOptions) => {
    const app = registry.get(appId)
    if (!app) {
      instance.ui.message('error', `应用不存在：${appId}`)
      return
    }
    bus.emit('webos:app:launch', { appId })
    app.onLaunch?.({ appId, launchOptions, bus, events: undefined as never })
    wm.open(appId, { launchOptions })
  }
  instance.openUrl = (url, target = 'browser') => {
    if (target === 'external') {
      window.open(url, '_blank', 'noopener')
      return
    }
    instance.openApp('browser', { url })
  }

  registry.onChange((appId, action) => bus.emit('webos:registry:change', { appId, action }))

  // ── scope 持久化装配（§8）：dock / widgets / desktop.layout ──
  // 宿主可通过 persist.adapter 替换存储后端（IndexedDB / 远程 / 自定义），全部 scope 自动走它
  // key = scope 名；prefix 隔离由 PersistenceAdapter 内部管理（§8：key 前缀即 scope）
  const scopeKey = (s: string) => s
  void persist.get(scopeKey('dock')).then((saved: any) => {
    if (saved?.pinned) {
      dock.pinned = saved.pinned
    }
    if (typeof saved?.iconSize === 'number') {
      dock.iconSize = saved.iconSize
    }
    if (saved?.position) {
      dock.position = saved.position
    }
    if (typeof saved?.autohide === 'boolean') {
      dock.autohide = saved.autohide
    }
  })
  dock.onChange(() => {
    void persist.set(scopeKey('dock'), {
      pinned: [...dock.pinned],
      position: dock.position,
      autohide: dock.autohide,
      iconSize: dock.iconSize,
      showRecent: dock.showRecent,
    })
  })

  void persist.get(scopeKey('widgets')).then((saved: any) => {
    if (Array.isArray(saved?.instances)) {
      widgets.loadInstances(saved.instances)
    }
  })
  widgets.onChange(() => {
    void persist.set(scopeKey('widgets'), { instances: widgets.listInstances() })
  })

  void persist.get(scopeKey('desktop.layout')).then((saved: any) => {
    if (Array.isArray(saved)) {
      for (const item of saved) {
        desktop.add(item)
      }
    }
  })
  desktop.onChange((reason) => {
    void persist.set(scopeKey('desktop.layout'), desktop.list().map(x => ({
      id: x.id,
      type: x.type,
      refId: x.refId,
      name: x.name,
      icon: x.icon,
      position: x.position,
      children: x.children,
    })))
    void reason
  })

  return instance
}

/** WindowManager 回调 → bus 桥接 */
function wmWatch(wm: WindowManager, bus: YwEventBus) {
  const raw = (wm as unknown as { events?: Record<string, ((w: any) => void) | undefined> }).events
  if (!raw) {
    return
  }
  const orig = { ...raw }
  raw.onOpen = (w) => {
    bus.emit('webos:window:open', { windowId: w.id, appId: w.appId })
    orig.onOpen?.(w)
  }
  raw.onClose = (w) => {
    bus.emit('webos:window:close', { windowId: w.id, appId: w.appId })
    orig.onClose?.(w)
  }
  raw.onFocus = (w) => {
    bus.emit('webos:window:focus', { windowId: w.id, appId: w.appId })
    orig.onFocus?.(w)
  }
  raw.onBlur = (w) => {
    bus.emit('webos:window:blur', { windowId: w.id, appId: w.appId })
    orig.onBlur?.(w)
  }
  raw.onMove = (w) => {
    bus.emit('webos:window:move', { windowId: w.id, bounds: w.bounds })
    orig.onMove?.(w)
  }
  raw.onResize = (w) => {
    bus.emit('webos:window:resize', { windowId: w.id, bounds: w.bounds })
    orig.onResize?.(w)
  }
  raw.onStateChange = (w) => {
    bus.emit('webos:window:state-change', { windowId: w.id, state: w.state })
    orig.onStateChange?.(w)
  }
}

/** Vue 插件：createWebOS(options) → app.use() */
const flushWiredApps = new WeakSet<App>()
export function createWebOS(options: CreateWebOSOptions = {}) {
  const instance = createWebOSInstance(options)
  return {
    instance,
    install(app: App) {
      app.provide(WEBOS_KEY, instance)
      app.provide(YW_UI_ADAPTER_KEY, instance.ui)
      // 防抖写入在页面隐藏/卸载时可能来不及触发（遮挡/后台窗口的定时器节流可达分钟级），
      // visibilitychange/pagehide 时强制落盘，避免「改完立刻刷新丢数据」
      if (!flushWiredApps.has(app)) {
        flushWiredApps.add(app)
        const flush = () => instance.persist.flushNow?.()
        document.addEventListener('visibilitychange', () => {
          if (document.visibilityState === 'hidden') {
            flush()
          }
        })
        window.addEventListener('pagehide', flush)
      }
    },
  }
}

/** 取服务容器（Provider 内） */
export function useWebOS(): WebOSInstance {
  const inst = inject(WEBOS_KEY)
  if (!inst) {
    throw new Error('[webos] WebOS 未安装：请先 app.use(createWebOS(...)) 并在 <WebOSProvider> 内使用')
  }
  return inst
}
