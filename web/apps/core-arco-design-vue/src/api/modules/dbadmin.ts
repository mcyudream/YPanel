import api from '../index'

export interface DbAdminInstance {
  id: number
  name: string
  type: 'mysql' | 'postgres' | 'redis' | 'mongo'
  port: number
  running: boolean
}

export interface DbAdminTable {
  name: string
  rows: number
  sizeMb: number
}

export interface DbAdminQueryResult {
  columns: string[]
  rows: unknown[][]
  elapsed: string
}

export default {
  instances: async () => {
    const res = await api.get('api/v1/plugin/db-admin/instances', { silent: true })
    return res.data as DbAdminInstance[]
  },
  databases: async (id: number) => {
    const res = await api.get(`api/v1/plugin/db-admin/${id}/databases`, { silent: true })
    return res.data as { name: string, sizeMb: number }[]
  },
  tables: async (id: number, db: string) => {
    const res = await api.get(`api/v1/plugin/db-admin/${id}/tables?db=${encodeURIComponent(db)}`, { silent: true })
    return res.data as DbAdminTable[]
  },
  query: async (id: number, db: string, sql: string) => {
    const res = await api.post(`api/v1/plugin/db-admin/${id}/query`, { db, sql }, { timeout: 60000 })
    return res.data as DbAdminQueryResult
  },
}
