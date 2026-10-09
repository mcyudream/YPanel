/**
 * 事件总线（core 零依赖实现）。事件命名空间统一 webos:*。
 */
export type EventHandler<T = unknown> = (payload: T) => void

export interface EventBus<Events extends object> {
  on: <K extends keyof Events>(event: K, handler: EventHandler<Events[K]>) => () => void
  once: <K extends keyof Events>(event: K, handler: EventHandler<Events[K]>) => () => void
  off: <K extends keyof Events>(event: K, handler: EventHandler<Events[K]>) => void
  emit: <K extends keyof Events>(event: K, payload: Events[K]) => void
  clear: () => void
}

export function createEventBus<Events extends object>(): EventBus<Events> {
  const handlers = new Map<keyof Events, Set<EventHandler<never>>>()

  function on<K extends keyof Events>(event: K, handler: EventHandler<Events[K]>): () => void {
    let set = handlers.get(event)
    if (!set) {
      set = new Set()
      handlers.set(event, set)
    }
    set.add(handler as EventHandler<never>)
    return () => off(event, handler)
  }

  function once<K extends keyof Events>(event: K, handler: EventHandler<Events[K]>): () => void {
    const wrap: EventHandler<Events[K]> = (p) => {
      off(event, wrap)
      handler(p)
    }
    return on(event, wrap)
  }

  function off<K extends keyof Events>(event: K, handler: EventHandler<Events[K]>): void {
    handlers.get(event)?.delete(handler as EventHandler<never>)
  }

  function emit<K extends keyof Events>(event: K, payload: Events[K]): void {
    handlers.get(event)?.forEach(h => (h as EventHandler<Events[K]>)(payload))
  }

  function clear(): void {
    handlers.clear()
  }

  return { on, once, off, emit, clear }
}

/** WebOS 全局事件表（webos:* 命名空间） */
export interface YwBusEvents {
  'webos:window:open': { windowId: string, appId: string }
  'webos:window:close': { windowId: string, appId: string }
  'webos:window:focus': { windowId: string, appId: string }
  'webos:window:blur': { windowId: string, appId: string }
  'webos:window:move': { windowId: string, bounds: { x: number, y: number, width: number, height: number } }
  'webos:window:resize': { windowId: string, bounds: { x: number, y: number, width: number, height: number } }
  'webos:window:state-change': { windowId: string, state: string }
  'webos:app:launch': { appId: string }
  'webos:app:terminate': { appId: string }
  'webos:app:relaunch': { appId: string }
  'webos:vfs:change': { path: string, type: 'create' | 'write' | 'remove' | 'move' }
  'webos:desktop:change': { reason: string }
  'webos:dock:change': { reason: string }
  'webos:registry:change': { appId: string, action: 'register' | 'unregister' | 'override' }
  'webos:system:theme-change': { mode: 'light' | 'dark' }
  'webos:system:wallpaper-change': { src: string | null }
}

export type YwEventBus = EventBus<YwBusEvents>
