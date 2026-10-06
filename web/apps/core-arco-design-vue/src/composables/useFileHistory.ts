// 文件本地历史（IndexedDB）：每次保存自动快照，可 diff 预览与恢复。
// 按存储键 node::path 归档，每键限量（超出删最旧）。

export interface FileHistoryEntry {
  id?: number // 自增主键
  key: string // `${node}::${path}`
  ts: number
  content: string
  encoding: string
  eol: 'lf' | 'crlf'
  size: number
}

const DB_NAME = 'ypanel-file-history'
const DB_VERSION = 1
const STORE = 'snapshots'
const MAX_PER_KEY = 30

let dbPromise: Promise<IDBDatabase> | null = null

function openDB(): Promise<IDBDatabase> {
  if (!dbPromise) {
    dbPromise = new Promise((resolve, reject) => {
      const req = indexedDB.open(DB_NAME, DB_VERSION)
      req.onupgradeneeded = () => {
        const db = req.result
        if (!db.objectStoreNames.contains(STORE)) {
          const store = db.createObjectStore(STORE, { keyPath: 'id', autoIncrement: true })
          store.createIndex('key_ts', ['key', 'ts'])
        }
      }
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
    })
  }
  return dbPromise
}

function tx(db: IDBDatabase, mode: IDBTransactionMode) {
  return db.transaction(STORE, mode).objectStore(STORE)
}

function reqAs<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

/** 追加一条快照，并裁剪该键超过上限的旧记录。 */
export async function pushFileHistory(entry: Omit<FileHistoryEntry, 'id' | 'ts'>) {
  const db = await openDB()
  const store = tx(db, 'readwrite')
  await reqAs(store.add({ ...entry, ts: Date.now() }))
  // 裁剪：按 key+ts 倒序，跳过 MAX_PER_KEY 条后删除
  const range = IDBKeyRange.bound([entry.key, 0], [entry.key, Number.MAX_SAFE_INTEGER])
  const all = await reqAs(store.index('key_ts').getAll(range))
  if (all.length > MAX_PER_KEY) {
    const cursorReq = store.index('key_ts').openCursor(range, 'prev')
    let seen = 0
    await new Promise<void>((resolve) => {
      cursorReq.onsuccess = () => {
        const cursor = cursorReq.result
        if (!cursor) {
          resolve()
          return
        }
        seen++
        if (seen > MAX_PER_KEY) {
          cursor.delete()
        }
        cursor.continue()
      }
      cursorReq.onerror = () => resolve()
    })
  }
}

/** 列出某文件的历史快照（新→旧）。 */
export async function listFileHistory(key: string): Promise<FileHistoryEntry[]> {
  const db = await openDB()
  const range = IDBKeyRange.bound([key, 0], [key, Number.MAX_SAFE_INTEGER])
  const all = await reqAs(tx(db, 'readonly').index('key_ts').getAll(range))
  return all.sort((a, b) => b.ts - a.ts)
}
