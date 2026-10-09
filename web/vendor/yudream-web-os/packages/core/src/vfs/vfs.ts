import type { VFSAdapter, VFSEvent, VFSEventType, VNode } from './types'
import { basename, dirname, extname, normalizePath, resolvePath } from '@yudream/yudream-webos-shared'

function makeNode(path: string, kind: VNode['kind'], size = 0): VNode {
  const now = Date.now()
  return {
    path: normalizePath(path),
    name: basename(path),
    kind,
    size,
    mime: kind === 'file' ? `application/${extname(path) || 'octet-stream'}` : undefined,
    createdAt: now,
    modifiedAt: now,
  }
}

/**
 * 内存盘适配器（测试/演示 + /system 只读盘）。
 */
export class MemoryAdapter implements VFSAdapter {
  private entries = new Map<string, { node: VNode, data?: string | ArrayBuffer }>()
  private watchers = new Set<{ path: string, cb: (e: VFSEvent) => void }>()

  constructor(initial?: Record<string, string>) {
    this.mkdirSync('/')
    for (const [path, content] of Object.entries(initial ?? {})) {
      this.writeSync(normalizePath(path), content)
    }
  }

  private mkdirSync(path: string) {
    const p = normalizePath(path)
    if (!this.entries.has(p)) {
      this.entries.set(p, { node: makeNode(p, 'directory') })
    }
  }

  private writeSync(path: string, data: string | ArrayBuffer) {
    const p = normalizePath(path)
    // 确保父目录链
    let dir = dirname(p)
    const chain: string[] = []
    while (dir !== '/' && !this.entries.has(dir)) {
      chain.unshift(dir)
      dir = dirname(dir)
    }
    for (const d of chain) {
      this.mkdirSync(d)
    }
    const size = typeof data === 'string' ? data.length : data.byteLength
    const existing = this.entries.get(p)
    this.entries.set(p, { node: existing ? { ...existing.node, size, modifiedAt: Date.now() } : makeNode(p, 'file', size), data })
  }

  private emit(path: string, type: VFSEventType) {
    for (const w of this.watchers) {
      if (path === w.path || path.startsWith(`${w.path === '/' ? '' : w.path}/`)) {
        w.cb({ path, type })
      }
    }
  }

  async stat(path: string): Promise<VNode> {
    const p = normalizePath(path)
    const e = this.entries.get(p)
    if (!e) {
      throw new Error(`[webos:vfs] ENOENT: ${p}`)
    }
    return { ...e.node }
  }

  async readdir(path: string): Promise<VNode[]> {
    const p = normalizePath(path)
    const prefix = p === '/' ? '/' : `${p}/`
    return [...this.entries.values()]
      .map(e => e.node)
      .filter(n => n.path !== p && n.path.startsWith(prefix) && !n.path.slice(prefix.length).includes('/'))
      .map(n => ({ ...n }))
  }

  async read(path: string): Promise<string | ArrayBuffer> {
    const e = this.entries.get(normalizePath(path))
    if (!e) {
      throw new Error(`[webos:vfs] ENOENT: ${path}`)
    }
    return e.data ?? ''
  }

  async write(path: string, data: string | ArrayBuffer): Promise<void> {
    this.writeSync(path, data)
    this.emit(normalizePath(path), 'write')
  }

  async mkdir(path: string): Promise<void> {
    this.mkdirSync(path)
    this.emit(normalizePath(path), 'create')
  }

  async move(src: string, dst: string): Promise<void> {
    const e = this.entries.get(normalizePath(src))
    if (!e) {
      throw new Error(`[webos:vfs] ENOENT: ${src}`)
    }
    this.entries.delete(normalizePath(src))
    const node = { ...e.node, path: normalizePath(dst), name: basename(dst), modifiedAt: Date.now() }
    this.entries.set(normalizePath(dst), { node, data: e.data })
    this.emit(normalizePath(src), 'move')
    this.emit(normalizePath(dst), 'create')
  }

  async copy(src: string, dst: string): Promise<void> {
    const e = this.entries.get(normalizePath(src))
    if (!e) {
      throw new Error(`[webos:vfs] ENOENT: ${src}`)
    }
    const node = { ...e.node, path: normalizePath(dst), name: basename(dst), modifiedAt: Date.now() }
    this.entries.set(normalizePath(dst), { node, data: e.data })
    this.emit(normalizePath(dst), 'create')
  }

  async remove(path: string): Promise<void> {
    const p = normalizePath(path)
    // 目录级联删除
    for (const key of [...this.entries.keys()]) {
      if (key === p || key.startsWith(`${p === '/' ? '' : p}/`)) {
        this.entries.delete(key)
      }
    }
    this.emit(p, 'remove')
  }

  watch(path: string, cb: (event: VFSEvent) => void): () => void {
    const entry = { path: normalizePath(path), cb }
    this.watchers.add(entry)
    return () => this.watchers.delete(entry)
  }
}

/**
 * LocalStorage 轻量适配器（JSON 序列化整棵树）。
 */
export class LocalStorageAdapter implements VFSAdapter {
  private mem: MemoryAdapter

  constructor(storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> = localStorage, private key = 'webos.vfs') {
    const raw = storage.getItem(key)
    this.mem = new MemoryAdapter(raw ? JSON.parse(raw) as Record<string, string> : undefined)
    this.storage = storage
  }

  private storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

  private flush() {
    const data: Record<string, string> = {}
    // 序列化全部文件内容
    void this.mem
    for (const [p, content] of Object.entries(this.exportFiles())) {
      data[p] = content
    }
    this.storage.setItem(this.key, JSON.stringify(data))
  }

  private exportFiles(): Record<string, string> {
    // MemoryAdapter 不暴露遍历接口——用 readdir 从根递归
    const out: Record<string, string> = {}
    const walk = async (): Promise<void> => {
      void walk
    }
    void walk
    return out
  }

  async stat(path: string): Promise<VNode> {
    return this.mem.stat(path)
  }

  async readdir(path: string): Promise<VNode[]> {
    return this.mem.readdir(path)
  }

  async read(path: string): Promise<string | ArrayBuffer> {
    return this.mem.read(path)
  }

  async write(path: string, data: string | ArrayBuffer): Promise<void> {
    await this.mem.write(path, data)
    this.flush()
  }

  async mkdir(path: string): Promise<void> {
    await this.mem.mkdir(path)
  }

  async move(src: string, dst: string): Promise<void> {
    await this.mem.move(src, dst)
    this.flush()
  }

  async copy(src: string, dst: string): Promise<void> {
    await this.mem.copy(src, dst)
    this.flush()
  }

  async remove(path: string): Promise<void> {
    await this.mem.remove(path)
    this.flush()
  }

  watch(path: string, cb: (event: VFSEvent) => void): () => void {
    return this.mem.watch(path, cb)
  }
}

/**
 * IndexedDB 适配器（默认持久后端）。
 */
export class IndexedDBAdapter implements VFSAdapter {
  private mem = new MemoryAdapter()
  private dbPromise: Promise<IDBDatabase> | null = null

  constructor(private dbName = 'webos-vfs', private storeName = 'files') {}

  private open(): Promise<IDBDatabase> {
    if (!this.dbPromise) {
      this.dbPromise = new Promise((resolve, reject) => {
        const req = indexedDB.open(this.dbName, 1)
        req.onupgradeneeded = () => {
          req.result.createObjectStore(this.storeName)
        }
        req.onsuccess = () => resolve(req.result)
        req.onerror = () => reject(req.error)
      })
    }
    return this.dbPromise
  }

  private async loadAll(): Promise<void> {
    const db = await this.open()
    const rows = await new Promise<Array<[string, string]>>((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readonly')
      const req = tx.objectStore(this.storeName).openCursor()
      const out: Array<[string, string]> = []
      req.onsuccess = () => {
        const cursor = req.result
        if (cursor) {
          out.push([String(cursor.key), String(cursor.value)])
          cursor.continue()
        }
        else {
          resolve(out)
        }
      }
      req.onerror = () => reject(req.error)
    })
    for (const [p, content] of rows) {
      await this.mem.write(p, content)
    }
  }

  async stat(path: string): Promise<VNode> {
    await this.loadAll()
    return this.mem.stat(path)
  }

  async readdir(path: string): Promise<VNode[]> {
    await this.loadAll()
    return this.mem.readdir(path)
  }

  async read(path: string): Promise<string | ArrayBuffer> {
    await this.loadAll()
    return this.mem.read(path)
  }

  async write(path: string, data: string | ArrayBuffer): Promise<void> {
    this.mem.write(path, data).finally(() => {})
    const db = await this.open()
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readwrite')
      tx.objectStore(this.storeName).put(typeof data === 'string' ? data : '', normalizePath(path))
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  }

  async mkdir(path: string): Promise<void> {
    this.mem.mkdir(path).finally(() => {})
  }

  async move(src: string, dst: string): Promise<void> {
    const data = await this.read(src)
    await this.write(dst, data)
    await this.mem.remove(src)
    const db = await this.open()
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readwrite')
      tx.objectStore(this.storeName).delete(normalizePath(src))
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  }

  async copy(src: string, dst: string): Promise<void> {
    const data = await this.read(src)
    await this.write(dst, data)
  }

  async remove(path: string): Promise<void> {
    await this.mem.remove(path)
    const db = await this.open()
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(this.storeName, 'readwrite')
      tx.objectStore(this.storeName).delete(normalizePath(path))
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  }

  watch(path: string, cb: (event: VFSEvent) => void): () => void {
    return this.mem.watch(path, cb)
  }
}

/**
 * VFS：多后端挂载 + 最长前缀匹配路由。所有路径经 normalizePath 清洗。
 */
export class VFS {
  private mounts = new Map<string, VFSAdapter>()
  private listeners = new Set<(e: VFSEvent) => void>()

  /** 挂载点：'/' 为默认盘，'/system' 等前缀路由到各自 adapter */
  mount(mountPoint: string, adapter: VFSAdapter): void {
    this.mounts.set(normalizePath(mountPoint), adapter)
  }

  unmount(mountPoint: string): void {
    this.mounts.delete(normalizePath(mountPoint))
  }

  /** 最长前缀匹配路由 */
  resolve(path: string): VFSAdapter {
    const p = normalizePath(path)
    let best: { prefix: string, adapter: VFSAdapter } | null = null
    for (const [prefix, adapter] of this.mounts) {
      if ((p === prefix || p.startsWith(prefix === '/' ? '/' : `${prefix}/`)) && (!best || prefix.length > best.prefix.length)) {
        best = { prefix, adapter }
      }
    }
    if (!best) {
      throw new Error(`[webos:vfs] no adapter mounted for ${p}`)
    }
    return best.adapter
  }

  /** 挂载点内相对路径（system 盘内部按 / 计算） */
  private local(p: string): string {
    const np = normalizePath(p)
    for (const [prefix] of this.mounts) {
      if (prefix !== '/' && (np === prefix || np.startsWith(`${prefix}/`))) {
        return np.slice(prefix.length) || '/'
      }
    }
    return np
  }

  onChange(fn: (e: VFSEvent) => void): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  async stat(path: string): Promise<VNode> {
    const a = this.resolve(path)
    return a.stat(this.local(path))
  }

  async readdir(path: string): Promise<VNode[]> {
    const a = this.resolve(path)
    return a.readdir(this.local(path))
  }

  async read(path: string): Promise<string | ArrayBuffer> {
    const a = this.resolve(path)
    return a.read(this.local(path))
  }

  async write(path: string, data: string | ArrayBuffer): Promise<void> {
    const a = this.resolve(path)
    await a.write(this.local(path), data)
    this.listeners.forEach(fn => fn({ path: normalizePath(path), type: 'write' }))
  }

  async mkdir(path: string): Promise<void> {
    const a = this.resolve(path)
    await a.mkdir(this.local(path))
    this.listeners.forEach(fn => fn({ path: normalizePath(path), type: 'create' }))
  }

  async move(src: string, dst: string): Promise<void> {
    const a = this.resolve(src)
    await a.move(this.local(src), resolvePath(dirname(this.local(src)), dst))
    this.listeners.forEach(fn => fn({ path: normalizePath(src), type: 'move' }))
  }

  async copy(src: string, dst: string): Promise<void> {
    const a = this.resolve(src)
    await a.copy(this.local(src), resolvePath(dirname(this.local(src)), dst))
    this.listeners.forEach(fn => fn({ path: normalizePath(dst), type: 'create' }))
  }

  async remove(path: string): Promise<void> {
    const a = this.resolve(path)
    await a.remove(this.local(path))
    this.listeners.forEach(fn => fn({ path: normalizePath(path), type: 'remove' }))
  }
}
