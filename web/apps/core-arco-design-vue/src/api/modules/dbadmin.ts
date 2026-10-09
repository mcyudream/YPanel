import api from '../index'

// 类型与后端 core/internal/dbdriver 对齐
export interface DbAdminInstance {
  id: number
  name: string
  type: 'mysql' | 'postgres' | 'redis' | 'mongo'
  origin: string
  host: string
  port: number
  running: boolean
}

export interface DbDatabaseInfo { name: string, sizeMb: number, charset?: string }

export interface DbTableInfo { name: string, rows: number, sizeMb: number, comment?: string, kind?: string }

export interface DbColumnInfo {
  name: string
  dataType: string
  nullable: boolean
  key?: string
  default?: string | null
  comment?: string
  extra?: string
}

export interface DbIndexInfo { name: string, columns: string[], unique: boolean, primary: boolean }

export interface DbQueryResult { columns: string[], rows: unknown[][], elapsed: string }

export interface DbPagedResult { columns: string[], rows: unknown[][], total: number, offset: number, limit: number, elapsed: string }

export interface BrowseReq {
  db: string
  schema?: string
  table: string
  sortCol?: string
  sortDir?: string
  offset: number
  limit: number
}

export interface RowEditReq { db: string, schema?: string, table: string, pk?: Record<string, unknown>, values?: Record<string, unknown> }

export interface RedisKeyItem { name: string, type: string }

export interface RedisZSetItem { member: string, score: number }

export interface RedisStreamEntry { id: string, fields: Record<string, string> }

export interface RedisKeyDetail {
  name: string
  type: string
  ttl: number
  length: number
  string?: string
  hash?: Record<string, string>
  list?: string[]
  set?: string[]
  zset?: RedisZSetItem[]
  stream?: RedisStreamEntry[]
  binary?: boolean
  preview: boolean
}

export interface MongoIndex {
  name: string
  keys: Record<string, number>
  unique: boolean
  sizeMb: number
  accesses: number
  accessedAt?: string
}

export interface MongoDocPage { docs: string[], total: number, skip: number, limit: number }

export interface DbAuditItem {
  id: number
  username: string
  instanceId: number
  instanceName: string
  dbType: string
  kind: string
  target: string
  detail: string
  success: boolean
  createdAt: string
}

const GET = 'api/v1/plugin/db-admin'
const q = (params: Record<string, string | number | undefined>) => {
  const u = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== '') u.set(k, String(v)) })
  const s = u.toString()
  return s ? `?${s}` : ''
}

export default {
  instances: async () => {
    const res = await api.get(`${GET}/instances`, { silent: true })
    return res.data as DbAdminInstance[]
  },
  ping: async (id: number) => {
    const res = await api.get(`${GET}/${id}/ping`, { silent: true, timeout: 8000 })
    return (res.data as { ms: number }).ms
  },
  databases: async (id: number) => {
    const res = await api.get(`${GET}/${id}/databases`, { silent: true })
    return res.data as DbDatabaseInfo[]
  },
  schemas: async (id: number, db: string) => {
    const res = await api.get(`${GET}/${id}/schemas${q({ db })}`, { silent: true })
    return res.data as string[]
  },
  tables: async (id: number, db: string, schema?: string) => {
    const res = await api.get(`${GET}/${id}/tables${q({ db, schema })}`, { silent: true })
    return res.data as DbTableInfo[]
  },
  columns: async (id: number, db: string, table: string, schema?: string) => {
    const res = await api.get(`${GET}/${id}/columns${q({ db, table, schema })}`, { silent: true })
    return res.data as DbColumnInfo[]
  },
  ddl: async (id: number, db: string, table: string, schema?: string) => {
    const res = await api.get(`${GET}/${id}/ddl${q({ db, table, schema })}`, { silent: true })
    return (res.data as { ddl: string }).ddl
  },
  indexes: async (id: number, db: string, table: string, schema?: string) => {
    const res = await api.get(`${GET}/${id}/indexes${q({ db, table, schema })}`, { silent: true })
    return res.data as DbIndexInfo[]
  },
  primaryKey: async (id: number, db: string, table: string, schema?: string) => {
    const res = await api.get(`${GET}/${id}/pk${q({ db, table, schema })}`, { silent: true })
    return res.data as string[]
  },
  query: async (id: number, db: string, sql: string, limit?: number) => {
    const res = await api.post(`${GET}/${id}/query`, { db, sql, limit }, { timeout: 60000 })
    return res.data as DbQueryResult
  },
  browse: async (id: number, req: BrowseReq) => {
    const res = await api.post(`${GET}/${id}/browse`, req, { timeout: 60000 })
    return res.data as DbPagedResult
  },
  rowInsert: async (id: number, req: RowEditReq) => {
    await api.post(`${GET}/${id}/rows/insert`, req)
  },
  rowUpdate: async (id: number, req: RowEditReq) => {
    await api.post(`${GET}/${id}/rows/update`, req)
  },
  rowDelete: async (id: number, req: RowEditReq) => {
    await api.post(`${GET}/${id}/rows/delete`, req)
  },
  indexCreate: async (id: number, req: { db: string, table: string, schema?: string, name?: string, columns: string[], unique: boolean }) => {
    await api.post(`${GET}/${id}/indexes/create`, req)
  },
  indexDrop: async (id: number, req: { db: string, table: string, schema?: string, name: string }) => {
    await api.post(`${GET}/${id}/indexes/drop`, req)
  },
  redisKeys: async (id: number, db: string, pattern: string, cursor: string, count = 100) => {
    const res = await api.get(`${GET}/${id}/redis/keys${q({ db, pattern, cursor, count })}`, { silent: true })
    return res.data as { keys: RedisKeyItem[], cursor: string }
  },
  redisKey: async (id: number, db: string, name: string) => {
    const res = await api.get(`${GET}/${id}/redis/key${q({ db, name })}`, { silent: true })
    return res.data as RedisKeyDetail
  },
  redisKeyWrite: async (id: number, req: { db: string, name: string, type: string, value: unknown }) => {
    await api.post(`${GET}/${id}/redis/key`, req)
  },
  redisKeyDelete: async (id: number, req: { db: string, name: string }) => {
    await api.post(`${GET}/${id}/redis/key/delete`, req)
  },
  redisKeyTTL: async (id: number, req: { db: string, name: string, ttl: number }) => {
    await api.post(`${GET}/${id}/redis/key/ttl`, req)
  },
  redisExec: async (id: number, db: string, command: string) => {
    const res = await api.post(`${GET}/${id}/redis/exec`, { db, command }, { timeout: 15000 })
    return (res.data as { output: string }).output
  },
  mongoDocs: async (id: number, req: { db: string, collection: string, filter?: string, project?: string, sort?: string, dir?: string, skip: number, limit: number }) => {
    const res = await api.post(`${GET}/${id}/mongo/docs`, req, { timeout: 60000 })
    return res.data as MongoDocPage
  },
  mongoAggregate: async (id: number, req: { db: string, collection: string, stages: string[], maxDocs?: number }) => {
    const res = await api.post(`${GET}/${id}/mongo/aggregate`, req, { timeout: 60000 })
    return res.data as MongoDocPage
  },
  mongoDoc: async (id: number, db: string, collection: string, docId: string) => {
    const res = await api.get(`${GET}/${id}/mongo/doc${q({ db, collection, id: docId })}`, { silent: true })
    return (res.data as { doc: string }).doc
  },
  mongoDocInsert: async (id: number, req: { db: string, collection: string, doc: string }) => {
    await api.post(`${GET}/${id}/mongo/doc`, req)
  },
  mongoDocUpdate: async (id: number, req: { db: string, collection: string, id: string, doc: string }) => {
    await api.post(`${GET}/${id}/mongo/doc/update`, req)
  },
  mongoDocDelete: async (id: number, req: { db: string, collection: string, id: string }) => {
    await api.post(`${GET}/${id}/mongo/doc/delete`, req)
  },
  mongoIndexes: async (id: number, db: string, collection: string) => {
    const res = await api.get(`${GET}/${id}/mongo/indexes${q({ db, collection })}`, { silent: true })
    return res.data as MongoIndex[]
  },
  mongoIndexCreate: async (id: number, req: { db: string, collection: string, name?: string, keys: Record<string, number>, unique: boolean }) => {
    await api.post(`${GET}/${id}/mongo/indexes/create`, req)
  },
  mongoIndexDrop: async (id: number, req: { db: string, collection: string, name: string }) => {
    await api.post(`${GET}/${id}/mongo/indexes/drop`, req)
  },
  mongoCollection: async (id: number, req: { db: string, action: 'create' | 'drop' | 'rename', name: string, to?: string }) => {
    await api.post(`${GET}/${id}/mongo/collection`, req)
  },
  importSQL: async (id: number, req: { db: string, content?: string, srcPath?: string }) => {
    const res = await api.post(`${GET}/${id}/sql/import`, req, { timeout: 3600000 })
    return res.data as { file: string, output: string }
  },
  mongoImport: async (id: number, req: { db: string, collection: string, content: string, format?: string }) => {
    const res = await api.post(`${GET}/${id}/mongo/import`, req, { timeout: 300000 })
    return res.data as { imported: number }
  },
  audits: async (instanceId?: number, page = 1, size = 20) => {
    const res = await api.get(`${GET}/audits${q({ instanceId, page, size })}`, { silent: true })
    return res.data as { items: DbAuditItem[], total: number }
  },
}
