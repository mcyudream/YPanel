import api from '../index'

// 中间件清单包含本域登记、数据库域引用与只读发现条目；ID 必须结合 source 使用。
export interface MiddlewareInstance {
  key: string
  source: 'middleware' | 'database' | 'discovered'
  managedBy: 'middleware' | 'database'
  nodeId: string
  image: string
  networks: string[]
  state: string
  adoptable: boolean
  id: number
  name: string
  type: string
  label?: string
  origin: 'container' | 'external'
  host: string
  port: number
  user: string
  version: string
  containerName: string
  composeProject: string
  composeDir?: string
  ownerId: number
  remark: string
  minReuseVersion?: string
  reusable?: boolean
  createdAt: string
}

export interface MiddlewareType {
  type: string
  label: string
  defaultPort: number
  minReuseVersion: string
  injectPrefix: string
}

export interface MiddlewareList {
  items: MiddlewareInstance[]
  total: number
  discoveryError?: string
}

export interface MiddlewareScanItem extends MiddlewareInstance {
  reused: boolean
}

export interface MiddlewareReuseOption {
  id: number
  name: string
  type: string
  origin: string
  host: string
  port: number
  version: string
  containerName: string
  nodeId: string
  label: string
  minReuseVersion: string
  reusable: boolean
  injectPrefix: string
  declaredPrefix: string
}

export default {
  list: (node = 'local') => api.get<MiddlewareList>('api/v1/middleware/instances', { params: { node }, silent: true }),
  types: () => api.get<MiddlewareType[]>('api/v1/middleware/types'),
  create: (data: { name: string, type: string, host: string, port?: number, user?: string, password?: string, remark?: string }) => api.post('api/v1/middleware/instances', data),
  update: (id: number, data: { host?: string, port?: number, user?: string, password?: string, remark?: string }) => api.put(`api/v1/middleware/instances/${id}`, data),
  remove: (id: number, node?: string) => api.delete(`api/v1/middleware/instances/${id}`, { params: { node: node || undefined } }),
  test: (id: number, node?: string) => api.post(`api/v1/middleware/instances/${id}/test`, undefined, { params: { node: node || undefined } }),
  reveal: (id: number) => api.get(`api/v1/middleware/instances/${id}/reveal`),
  versionRefresh: (id: number, node?: string) => api.post<{ version: string }>(`api/v1/middleware/instances/${id}/version`, undefined, { params: { node: node || undefined } }),
  scan: (node = 'local') => api.get<MiddlewareScanItem[]>('api/v1/middleware/scan', { params: { node } }),
  adopt: (data: { node?: string, containerName: string, name?: string }) => api.post('api/v1/middleware/adopt', data, { params: { node: data.node || 'local' } }),
  reuseOptions: (type: string, prefix?: string) => api.get(`api/v1/middleware/reuse?type=${type}&prefix=${prefix || ''}`),
}
