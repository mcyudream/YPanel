import api from '../index'

export interface StoreApp {
  id: number
  key: string
  name: string
  title: string
  description: string
  readMe: string
  iconUrl: string
  tags: string
  versionsJson: string
}

export interface StoreVersion {
  id: string
  name: string
  downloadUrl: string
  formFields: { envKey: string, label: Record<string, string>, default?: unknown, type?: string, required?: boolean }[]
}

export interface StoreInstall {
  id: number
  key: string
  name: string
  version: string
  composeProject: string
  createdAt: string
}

export const storeApi = {
  list: async (search = '', tag = '') => {
    const q = `?search=${encodeURIComponent(search)}&tag=${encodeURIComponent(tag)}`
    const res = await api.get(`api/v1/store/apps${q}`, { silent: true })
    return res.data as StoreApp[]
  },
  get: async (key: string) => {
    const res = await api.get(`api/v1/store/apps/${encodeURIComponent(key)}`, { silent: true })
    return res.data as { app: StoreApp, versions: StoreVersion[] }
  },
  sync: async (force = false) => {
    const res = await api.post(`api/v1/store/sync${force ? '?force=1' : ''}`, null, { timeout: 300000 })
    return res.data as { total: number, created: number, skipped?: boolean }
  },
  installed: async () => {
    const res = await api.get('api/v1/store/installed', { silent: true })
    return res.data as StoreInstall[]
  },
  install: async (data: { key: string, version: string, name: string, params: Record<string, string> }) => {
    const res = await api.post('api/v1/store/install', data, { timeout: 600000 })
    return res.data as { project: string, logs: string }
  },
  uninstall: (project: string) => api.delete(`api/v1/store/install/${encodeURIComponent(project)}`),
  reinstall: async (key: string, version: string, name: string, params: Record<string, string>) => {
    const res = await api.post('api/v1/store/install', { key, version, name, params }, { timeout: 600000 })
    return res.data as { project: string, logs: string }
  },
}
