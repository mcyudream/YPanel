/**
 * 持久化抽象 + localStorage 实现（默认 IndexedDB，fallback localStorage）。
 * 所有写入防抖合并 300ms；schema 带 version，升级走 migrations。
 */
import { debounce } from '@yudream/yudream-webos-shared'

/**
 * JSON 安全克隆：持久化值的最终形态就是 JSON（flush 走 JSON.stringify），克隆语义与之对齐。
 *  不用 structuredClone——宿主（Vue reactive 等 Proxy 包装对象）传入时会抛 DataCloneError。
 */
function jsonClone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export interface PersistenceAdapter {
  get: <T>(key: string) => Promise<T | null>
  set: (key: string, value: unknown) => Promise<void>
  remove: (key: string) => Promise<void>
  clear: (scope?: string) => Promise<void>
  /** 立即落盘（页面隐藏/卸载等防抖来不及触发的场景）；不支持者可缺省 */
  flushNow?: () => void
}

/** 带 schema 版本的存储格式 */
export interface VersionedData<T = unknown> {
  version: number
  data: T
}

/** 内存实现（测试/降级） */
export class MemoryPersistence implements PersistenceAdapter {
  private store = new Map<string, unknown>()

  async get<T>(key: string): Promise<T | null> {
    return this.getSync<T>(key)
  }

  /** 同步读取（防抖 flush 等同步路径用） */
  getSync<T>(key: string): T | null {
    return (this.store.get(key) as T) ?? null
  }

  async set(key: string, value: unknown): Promise<void> {
    this.store.set(key, jsonClone(value))
  }

  async remove(key: string): Promise<void> {
    this.store.delete(key)
  }

  async clear(scope?: string): Promise<void> {
    if (!scope) {
      this.store.clear()
      return
    }
    for (const key of [...this.store.keys()]) {
      if (key.startsWith(scope)) {
        this.store.delete(key)
      }
    }
  }
}

/** localStorage 实现（写入防抖 300ms） */
export class LocalStoragePersistence implements PersistenceAdapter {
  private mem = new MemoryPersistence()
  private debouncedFlush: ((keys?: string[]) => void) & { cancel: () => void }
  private dirty = new Set<string>()

  constructor(private storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> = localStorage, private prefix = 'webos', debounceMs = 300) {
    this.debouncedFlush = debounce(() => this.flush(), debounceMs)
  }

  private fullKey(key: string): string {
    return `${this.prefix}.${key}`
  }

  async get<T>(key: string): Promise<T | null> {
    // storage 访问失败（隐私模式/iframe 限制）静默降级到内存
    try {
      const raw = this.storage.getItem(this.fullKey(key))
      if (raw !== null) {
        return JSON.parse(raw) as T
      }
    }
    catch {
      /* 降级到 mem */
    }
    return this.mem.get<T>(key)
  }

  async set(key: string, value: unknown): Promise<void> {
    await this.mem.set(key, value)
    this.dirty.add(key)
    this.debouncedFlush()
  }

  async remove(key: string): Promise<void> {
    await this.mem.remove(key)
    this.dirty.add(key)
    this.debouncedFlush()
  }

  async clear(scope?: string): Promise<void> {
    await this.mem.clear(scope)
    for (const key of [...this.dirty]) {
      if (!scope || key.startsWith(scope)) {
        this.storage.removeItem(this.fullKey(key))
        this.dirty.delete(key)
      }
    }
    if (!scope) {
      for (const key of [...this.dirty]) {
        this.storage.removeItem(this.fullKey(key))
      }
    }
  }

  /** 立即落盘（beforeunload 用） */
  flushNow(): void {
    this.debouncedFlush.cancel()
    this.flush()
  }

  private flush(): void {
    for (const key of this.dirty) {
      const value = this.mem.getSync(key)
      if (value === null) {
        this.storage.removeItem(this.fullKey(key))
      }
      else {
        this.storage.setItem(this.fullKey(key), JSON.stringify(value))
      }
    }
    this.dirty.clear()
  }
}

/** 版本迁移：migrations[fromVersion] = (data) => nextData */
export async function migrate<T>(
  adapter: PersistenceAdapter,
  key: string,
  currentVersion: number,
  migrations: Record<number, (data: unknown) => unknown>,
  fallback: T,
): Promise<T> {
  const raw = await adapter.get<VersionedData<T> | T>(key)
  if (!raw) {
    return fallback
  }
  const versioned = raw as VersionedData<T>
  if (typeof versioned !== 'object' || versioned === null || !('version' in versioned)) {
    // 无版本旧数据：按 0 处理
    let data: unknown = raw
    for (let v = 0; v < currentVersion; v++) {
      data = migrations[v]?.(data) ?? data
    }
    await adapter.set(key, { version: currentVersion, data })
    return data as T
  }
  let version: number = versioned.version
  let data: unknown = versioned.data
  while (version < currentVersion) {
    data = migrations[version]?.(data) ?? data
    version++
  }
  if (version !== versioned.version) {
    await adapter.set(key, { version, data })
  }
  return data as T
}
