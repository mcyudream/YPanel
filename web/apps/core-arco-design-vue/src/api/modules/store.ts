import api from '../index'

export interface StoreFormValue { label: string, value: string }

export interface StoreFormField {
  envKey: string
  label: Record<string, string>
  default?: unknown
  type?: string // text / number / password / select / service / apps
  rule?: string // paramPort / paramCommon / paramComplexity...
  required?: boolean
  random?: boolean // 安装时随机生成（密码/名称）
  edit?: boolean // false = 只读
  disabled?: boolean
  description?: string
  values?: StoreFormValue[]
}

export function isPortField(f: StoreFormField) {
  return f.rule === 'paramPort' || f.rule === 'paramPortRange' || (f.envKey || '').toUpperCase().includes('PORT')
}

export interface StoreVersion {
  id: string
  name: string
  downloadUrl?: string
  localDir?: string
  releaseNotes?: string
  formFields: StoreFormField[]
}

export interface StoreApp {
  id: number
  sourceId: number
  key: string
  name: string
  title: string
  description: string
  readMe: string
  iconUrl: string
  tags: string
  kind: 'app' | 'service' | 'middleware' | string
  author: string
  arch: string
  versionsJson: string
  reverseProxy: string
  website: string
  sourceUrl: string
  document: string
  latestVersion: string
  lastModified: number
}

export interface StoreInstall {
  id: number
  sourceId: number
  key: string
  name: string
  version: string
  composeProject: string
  remark: string
  createdAt: string
}

export interface StoreInstallInfo {
  id: number
  sourceId: number
  key: string
  name: string
  remark: string
  appName: string
  iconUrl: string
  version: string
  latestVersion: string
  upgradable: boolean
  composeProject: string
  running: boolean
  ports: number[]
  params: Record<string, string>
  createdAt: string
}

export interface StoreAppItem extends StoreApp {
  installed: boolean
  upgradable: boolean
  latestVer: string
  installInfo?: StoreInstall
}

export interface StoreAppList {
  total: number
  page: number
  pageSize: number
  items: StoreAppItem[]
}

export interface StoreSource {
  id: number
  name: string
  type: 'onepanel' | 'yp-url' | 'yp-git' | string
  url: string
  branch: string
  enabled: boolean
  builtin: boolean
  remark: string
  status: 'pending' | 'ok' | 'error' | string
  message: string
  appCount: number
  lastSyncAt?: string
}

export interface StoreTag { name: string, count: number }

export interface StoreListQuery {
  search?: string
  tag?: string
  sourceId?: number
  kind?: string
  status?: 'all' | 'installed' | 'notInstalled' | 'upgradable' | string
  orderBy?: 'name' | 'lastModified' | string
  order?: 'asc' | 'desc' | string
  page?: number
  pageSize?: number
}

export const storeApi = {
  list: async (q: StoreListQuery = {}) => {
    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) {
      if (v !== undefined && v !== null && v !== '') {
        params.set(k, String(v))
      }
    }
    const res = await api.get(`api/v1/store/apps?${params.toString()}`, { silent: true })
    return res.data as StoreAppList
  },
  get: async (sourceId: number, key: string) => {
    const res = await api.get(`api/v1/store/apps/${sourceId}/${encodeURIComponent(key)}`, { silent: true })
    return res.data as { app: StoreApp, versions: StoreVersion[], installed: boolean, install: StoreInstall }
  },
  tags: async () => {
    const res = await api.get('api/v1/store/tags', { silent: true })
    return res.data as StoreTag[]
  },
  iconUrl: (sourceId: number, key: string) => `api/v1/store/apps/${sourceId}/${encodeURIComponent(key)}/icon`,
  sync: async (force = false) => {
    const res = await api.post(`api/v1/store/sync${force ? '?force=1' : ''}`, null, { timeout: 300000 })
    return res.data as Record<string, string>
  },
  // 源管理
  sources: async () => {
    const res = await api.get('api/v1/store/sources', { silent: true })
    return res.data as StoreSource[]
  },
  createSource: async (data: { name: string, type: string, url: string, branch?: string, authToken?: string, remark?: string }) => {
    const res = await api.post('api/v1/store/sources', data)
    return res.data as StoreSource
  },
  updateSource: async (id: number, data: Partial<{ name: string, type: string, url: string, branch: string, authToken: string, remark: string }>) => {
    const res = await api.put(`api/v1/store/sources/${id}`, data)
    return res.data as StoreSource
  },
  setSourceEnabled: async (id: number, enabled: boolean) => {
    const res = await api.post(`api/v1/store/sources/${id}/${enabled ? 'enable' : 'disable'}`)
    return res.data
  },
  deleteSource: async (id: number) => {
    const res = await api.delete(`api/v1/store/sources/${id}`)
    return res.data
  },
  syncSource: async (id: number, force = true) => {
    const res = await api.post(`api/v1/store/sources/${id}/sync${force ? '?force=1' : ''}`, null, { timeout: 300000 })
    return res.data as { source: string, total: number }
  },
  installed: async () => {
    const res = await api.get('api/v1/store/installed', { silent: true })
    return res.data as StoreInstallInfo[]
  },
  installedAction: (project: string, action: 'start' | 'stop' | 'restart' | 'rebuild') =>
    api.post(`api/v1/store/installed/${encodeURIComponent(project)}/${action}`, null, { timeout: 300000 }),
  installEnv: async (project: string) => {
    const res = await api.get(`api/v1/store/installed/${encodeURIComponent(project)}/env`, { silent: true })
    return res.data as Record<string, string>
  },
  saveInstallEnv: (project: string, content: string) =>
    api.put(`api/v1/store/installed/${encodeURIComponent(project)}/env`, { content }, { timeout: 300000 }),
  install: async (data: { sourceId: number, key: string, version: string, name: string, params: Record<string, string>, domain?: string, network?: string, createNetwork?: boolean, timezone?: string, extraHosts?: string[], mountHostsFile?: boolean, externalDB?: { instanceId: number, database?: string, user?: string, createIfMissing?: boolean } }) => {
    const res = await api.post('api/v1/store/install', data, { timeout: 60000 })
    return res.data as { taskId: number, project: string }
  },
  uninstall: async (project: string, opts: { purgeData?: boolean, rmi?: boolean, cascadeDB?: boolean } = {}) => {
    const q = new URLSearchParams()
    if (opts.purgeData) {
      q.set('purgeData', 'true')
    }
    if (opts.rmi) {
      q.set('rmi', 'true')
    }
    if (opts.cascadeDB) {
      q.set('cascadeDB', 'true')
    }
    const qs = q.toString() ? `?${q.toString()}` : ''
    const res = await api.delete(`api/v1/store/install/${encodeURIComponent(project)}${qs}`)
    return res.data as { taskId: number, project: string }
  },
}
