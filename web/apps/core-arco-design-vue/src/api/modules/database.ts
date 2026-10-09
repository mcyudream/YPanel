import api from '../index'

export interface DbInstance {
  ownerId?: number
  id: number
  name: string
  type: 'mysql' | 'postgres' | 'redis' | 'mongo'
  origin: 'container' | 'external' | 'store'
  host: string
  port: number
  user: string
  remark: string
  composeProject: string
  running: boolean
  adoptable?: boolean
  createdAt: string
}

export interface DbDatabase {
  name: string
  sizeMb: number
  charset?: string
}

export interface DbUser {
  name: string
  host: string
  extra: string
}

export interface MigratePreview {
  srcType: string
  targetType: string
  targetName: string
  compatible: boolean
  dbs: { name: string, sizeMb: number, charset?: string }[]
}

export interface DbBackup {
  name: string
  sizeMb: number
  modTime: string
  path: string
}

export default {
  migratePreview: async (src: number, target: number) => {
    const res = await api.get(`api/v1/plugin/db-admin/migrate/preview?src=${src}&target=${target}`, { silent: true })
    return res.data as MigratePreview
  },
  migrateStart: (req: { src: number, target: number, dbs: string[] }) =>
    api.post('api/v1/plugin/db-admin/migrate/start', req),
  backupImport: (id: number, filename: string, content: string) =>
    api.post(`api/v1/database/instances/${id}/backups/import`, { filename, content }, { timeout: 300000 }),
  remoteAccess: (id: number, enable: boolean) =>
    api.post(`api/v1/database/instances/${id}/remote`, { enable }),
  // M54-P3 属主分配（0=公共；仅数据范围不受限账号可操作）
  setOwner: (id: number, ownerId: number) =>
    api.put(`api/v1/database/instances/${id}/owner`, { ownerId }),
  remoteAccessStatus: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/remote`, { silent: true })
    return (res.data as { enabled: boolean }).enabled
  },
  list: async () => {
    const res = await api.get('api/v1/database/instances', { silent: true })
    return res.data as DbInstance[]
  },
  create: (data: { name: string, type: string, port?: number, password?: string }) =>
    api.post('api/v1/database/instances', data),
  createExternal: (data: { name: string, type: string, host?: string, port: number, user?: string, password: string, remark?: string }) =>
    api.post('api/v1/database/instances/external', data, { timeout: 30000 }),
  adopt: (project: string) =>
    api.post('api/v1/database/instances/adopt', { project }, { timeout: 30000 }),
  remove: (id: number, data = false, backups = false) => api.delete(`api/v1/database/instances/${id}?data=${data}&backups=${backups}`),
  start: (id: number) => api.post(`api/v1/database/instances/${id}/start`),
  stop: (id: number) => api.post(`api/v1/database/instances/${id}/stop`),
  reveal: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/reveal`)
    return res.data as { name: string, type: string, port: number, user: string, password: string, host: string, container?: string, innerPort?: string, networks?: string, lanIp?: string, mapPort?: string }
  },
  databases: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/databases`, { silent: true })
    return res.data as DbDatabase[]
  },
  createDatabase: (id: number, name: string, charset?: string) =>
    api.post(`api/v1/database/instances/${id}/databases`, { name, charset }),
  dropDatabase: (id: number, name: string) => api.delete(`api/v1/database/instances/${id}/databases/${encodeURIComponent(name)}`),
  users: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/users`, { silent: true })
    return res.data as DbUser[]
  },
  createUser: (id: number, data: { name: string, host?: string, password: string }) =>
    api.post(`api/v1/database/instances/${id}/users`, data),
  dropUser: (id: number, name: string, host?: string) =>
    api.delete(`api/v1/database/instances/${id}/users/${encodeURIComponent(name)}?host=${encodeURIComponent(host || '')}`),
  changeUserPassword: (id: number, name: string, password: string, host?: string) =>
    api.put(`api/v1/database/instances/${id}/users/${encodeURIComponent(name)}/password?host=${encodeURIComponent(host || '')}`, { password }),
  backups: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/backups`, { silent: true })
    return res.data as DbBackup[]
  },
  createBackup: async (id: number, opts?: { storageAccountId?: number, keep?: number }) => {
    const body = opts && (opts.storageAccountId || opts.keep) ? opts : undefined
    const res = await api.post(`api/v1/database/instances/${id}/backups`, body)
    return res.data as { file?: string, output?: string, remoteKey?: string }
  },
  deleteBackup: (id: number, file: string) => api.delete(`api/v1/database/instances/${id}/backups?file=${encodeURIComponent(file)}`),
  restoreBackup: (id: number, file: string) => api.post(`api/v1/database/instances/${id}/backups/restore?file=${encodeURIComponent(file)}`),
  backupDownloadURL: (path: string, token: string) =>
    `api/v1/files/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}`,
}

// ---- M39：MySQL 管理深化 ----
export interface GrantRow {
  user: string
  host: string
  privs: string[]
}

export interface DbKV {
  name: string
  value: string
}

export const dbTuningApi = {
  grantMatrix: async (id: number, db: string) => {
    const res = await api.get(`api/v1/database/instances/${id}/privileges?db=${encodeURIComponent(db)}`, { silent: true })
    return res.data as GrantRow[]
  },
  setPrivileges: (id: number, data: { db: string, user: string, host: string, privs: string[], grant: boolean }) =>
    api.put(`api/v1/database/instances/${id}/privileges`, data),
  variables: async (id: number, filter = '') => {
    const res = await api.get(`api/v1/database/instances/${id}/variables?filter=${encodeURIComponent(filter)}`, { silent: true })
    return res.data as DbKV[]
  },
  setVariable: (id: number, name: string, value: string) => api.put(`api/v1/database/instances/${id}/variables`, { name, value }),
  status: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/status`, { silent: true })
    return res.data as Record<string, number>
  },
}
