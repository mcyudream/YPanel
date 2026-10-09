import api from '../index'

export interface RuntimeItem {
  id: number
  name: string
  type: string
  version: string
  origin: 'container' | 'external'
  fcgiAddr: string
  remark: string
  image: string
  containerName: string
  composeProject: string
  status: 'running' | 'stopped' | 'building' | 'creating' | 'error'
  message: string
  extensions: string[]
  codeDir?: string
  startCmd?: string
  pkgMgr?: string
  autoInstall?: boolean
  ports?: { host: number, container: number, protocol?: string }[]
  running: boolean
  createdAt: string
}

export interface PHPExtensionItem {
  name: string
  desc: string
  installed: boolean
}

export interface PHPExtensionsResp {
  catalog: PHPExtensionItem[]
  installed: string[]
}

export interface PHPQuickConfig {
  memoryLimit: string
  uploadMaxSize: string
  maxExecutionTime: string
  disableFunctions: string[]
}

export interface CreateResult {
  runtime: RuntimeItem
  taskId: number
}

function qs(params: Record<string, any>) {
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') {
      sp.set(k, String(v))
    }
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

export default {
  list: async () => {
    const res = await api.get('api/v1/runtimes', { silent: true })
    return res.data as RuntimeItem[]
  },
  detail: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/${id}`, { silent: true })
    return res.data as RuntimeItem
  },
  create: async (data: {
    name: string
    type?: string
    version: string
    extensions?: string[]
    remark?: string
    codeDir?: string
    startCmd?: string
    autoInstall?: boolean
    pkgMgr?: string
    ports?: { host: number, container: number, protocol?: string }[]
  }) => {
    const res = await api.post('api/v1/runtimes', data, { timeout: 30000 })
    return res.data as CreateResult
  },
  attachExternal: (data: { name: string, version?: string, fcgiAddr: string, remark?: string }) =>
    api.post('api/v1/runtimes/external', data, { timeout: 30000 }),
  remove: (id: number) => api.delete(`api/v1/runtimes/${id}`),
  start: (id: number) => api.post(`api/v1/runtimes/${id}/start`),
  stop: (id: number) => api.post(`api/v1/runtimes/${id}/stop`),
  restart: (id: number) => api.post(`api/v1/runtimes/${id}/restart`),
  phpCatalog: async () => {
    const res = await api.get('api/v1/runtimes/php/catalog', { silent: true })
    return res.data as {
      catalog: PHPExtensionItem[]
      templates: { name: string, desc: string, extensions: string[] }[]
    }
  },
  phpExtensions: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/php/extensions${qs({ id })}`, { silent: true })
    return res.data as PHPExtensionsResp
  },
  phpExtensionInstall: async (id: number, name: string) => {
    const res = await api.post('api/v1/runtimes/php/extensions/install', { id, name }, { timeout: 30000 })
    return res.data as { taskId: number }
  },
  phpExtensionUninstall: async (id: number, name: string) => {
    const res = await api.post('api/v1/runtimes/php/extensions/uninstall', { id, name }, { timeout: 30000 })
    return res.data as { taskId: number }
  },
  phpConfig: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/php/config${qs({ id })}`, { silent: true })
    return res.data as PHPQuickConfig
  },
  phpConfigSave: (id: number, data: Partial<PHPQuickConfig>) =>
    api.post(`api/v1/runtimes/php/config${qs({ id })}`, data, { timeout: 60000 }),
  fpmConfig: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/php/fpm-config${qs({ id })}`, { silent: true })
    return res.data as Record<string, string>
  },
  fpmConfigSave: (id: number, params: Record<string, string>) =>
    api.post('api/v1/runtimes/php/fpm-config', { id, params }, { timeout: 60000 }),
  fpmStatus: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/php/fpm-status${qs({ id })}`, { silent: true, timeout: 20000 })
    return res.data as { items: { key: string, value: string }[] }
  },
  nodeModules: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/node/modules${qs({ id })}`, { silent: true })
    return res.data as { modules: { name: string, version: string, dev: boolean }[], pkgMgr: string, packageJson: boolean }
  },
  nodeModuleOperate: async (id: number, operate: 'install' | 'uninstall' | 'update', module: string, pkgManager?: string) => {
    const res = await api.post('api/v1/runtimes/node/modules/operate', { id, operate, module, pkgManager }, { timeout: 30000 })
    return res.data as { taskId: number }
  },
  rebuild: async (id: number) => {
    const res = await api.post(`api/v1/runtimes/${id}/rebuild`, {}, { timeout: 30000 })
    return res.data as { taskId: number }
  },
  slowLog: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/php/slow-log${qs({ id })}`, { silent: true, timeout: 20000 })
    return res.data as { log: string, empty: boolean }
  },
  slowLogClear: (id: number) => api.post('api/v1/runtimes/php/slow-log/clear', { id }, { timeout: 30000 }),
  supervisorList: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/php/supervisor${qs({ id })}`, { silent: true, timeout: 20000 })
    return res.data as { name: string, status: string, detail: string }[]
  },
  supervisorUpsert: (id: number, data: { name: string, command: string, autoStart?: boolean, autoRestart?: boolean }) =>
    api.post(`api/v1/runtimes/php/supervisor${qs({ id })}`, data, { timeout: 30000 }),
  supervisorOperate: (id: number, name: string, action: 'start' | 'stop' | 'restart') =>
    api.post('api/v1/runtimes/php/supervisor/operate', { id, name, action }, { timeout: 30000 }),
  supervisorRemove: (id: number, name: string) =>
    api.delete(`api/v1/runtimes/php/supervisor${qs({ id, name })}`, { timeout: 30000 }),
  supervisorLog: async (id: number, name: string) => {
    const res = await api.get(`api/v1/runtimes/php/supervisor/log${qs({ id, name })}`, { silent: true, timeout: 20000 })
    return res.data as { log: string }
  },
  backupsList: async (id: number) => {
    const res = await api.get(`api/v1/runtimes/${id}/backups`, { silent: true })
    return res.data as { file: string, size: string, at: string }[]
  },
  backupsCreate: async (id: number) => {
    const res = await api.post(`api/v1/runtimes/${id}/backups`, {}, { timeout: 30000 })
    return res.data as { taskId: number, file: string }
  },
  backupsRestore: async (id: number, file: string) => {
    const res = await api.post(`api/v1/runtimes/${id}/backups/restore`, { file }, { timeout: 30000 })
    return res.data as { taskId: number }
  },
  backupsRemove: (id: number, file: string) =>
    api.delete(`api/v1/runtimes/${id}/backups${qs({ file })}`, { timeout: 30000 }),
}
