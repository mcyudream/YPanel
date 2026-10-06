import api from '../index'

export interface DbInstance {
  id: number
  name: string
  type: 'mysql' | 'postgres' | 'redis' | 'mongo'
  port: number
  user: string
  composeProject: string
  running: boolean
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

export interface DbBackup {
  name: string
  sizeMb: number
  modTime: string
  path: string
}

export default {
  remoteAccess: (id: number, enable: boolean) =>
    api.post(`api/v1/database/instances/${id}/remote`, { enable }),
  list: async () => {
    const res = await api.get('api/v1/database/instances', { silent: true })
    return res.data as DbInstance[]
  },
  create: (data: { name: string, type: string, port?: number, password?: string }) =>
    api.post('api/v1/database/instances', data),
  remove: (id: number, purge: boolean) => api.delete(`api/v1/database/instances/${id}?purge=${purge}`),
  start: (id: number) => api.post(`api/v1/database/instances/${id}/start`),
  stop: (id: number) => api.post(`api/v1/database/instances/${id}/stop`),
  reveal: async (id: number) => {
    const res = await api.get(`api/v1/database/instances/${id}/reveal`)
    return res.data as { name: string, type: string, port: number, user: string, password: string, host: string }
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
  createBackup: async (id: number) => {
    const res = await api.post(`api/v1/database/instances/${id}/backups`)
    return res.data as { file?: string, output?: string }
  },
  deleteBackup: (id: number, file: string) => api.delete(`api/v1/database/instances/${id}/backups?file=${encodeURIComponent(file)}`),
  restoreBackup: (id: number, file: string) => api.post(`api/v1/database/instances/${id}/backups/restore?file=${encodeURIComponent(file)}`),
  backupDownloadURL: (path: string, token: string) =>
    `api/v1/files/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}`,
}
