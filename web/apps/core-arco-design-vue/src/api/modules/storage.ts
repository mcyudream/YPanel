import api from '../index'

// ---- 远程备份存储账号（M34） ----
export interface StorageAccount {
  id: number
  name: string
  type: 's3' | 'webdav' | 'sftp'
  endpoint: string
  region: string
  bucket: string
  accessKey: string
  backupPath: string
  useSSL: boolean
  remark: string
  hasSecret: boolean
  createdAt: string
}

export interface StorageAccountInput {
  name: string
  type: 's3' | 'webdav' | 'sftp'
  endpoint: string
  region?: string
  bucket?: string
  accessKey?: string
  secret?: string // 留空 = 保留原值（编辑时）
  backupPath?: string
  useSSL?: boolean
  remark?: string
}

export interface StorageObject {
  name: string
  sizeMb: number
  modTime: string
}

export interface BackupUploadOpts {
  storageAccountId?: number
  keep?: number
}

export const storageApi = {
  list: async () => {
    const res = await api.get('api/v1/storage-accounts', { silent: true })
    return res.data as StorageAccount[]
  },
  create: (data: StorageAccountInput) => api.post('api/v1/storage-accounts', data),
  update: (id: number, data: StorageAccountInput) => api.put(`api/v1/storage-accounts/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/storage-accounts/${id}`),
  test: async (id: number) => {
    const res = await api.post(`api/v1/storage-accounts/${id}/test`)
    return res.data as { ok: boolean, latencyMs: number }
  },
  objects: async (id: number, category = '') => {
    const res = await api.get(`api/v1/storage-accounts/${id}/objects?category=${encodeURIComponent(category)}`, { silent: true })
    return res.data as StorageObject[]
  },
  deleteObject: (id: number, key: string) =>
    api.delete(`api/v1/storage-accounts/${id}/object?key=${encodeURIComponent(key)}`),
  fetch: async (id: number, key: string, destDir: string) => {
    const res = await api.post(`api/v1/storage-accounts/${id}/fetch`, { key, destDir })
    return res.data as { file: string, path: string }
  },
}

// ---- 目录 / 编排备份 ----
export const backupApi = {
  dirBackup: (data: { srcDir: string, name?: string } & BackupUploadOpts) => api.post('api/v1/backups/dir', data),
  composeBackup: (data: { project: string } & BackupUploadOpts) => api.post('api/v1/backups/compose', data),
}
