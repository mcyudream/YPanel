import api from '../index'

// M68 可复用中间件实例（非数据库类共享中间件；Neo4j/S3 在数据库域）
export interface MiddlewareInstance {
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

export interface MiddlewareScanItem {
  type: string
  label: string
  containerName: string
  image: string
  version: string
  networks: string[]
  composeDir?: string
  composeProject?: string
  user?: string
  reused: boolean
  reusable: boolean
  adoptable: boolean
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
  label: string
  minReuseVersion: string
  reusable: boolean
  injectPrefix: string
  declaredPrefix: string
}

export default {
  list: () => api.get('api/v1/middleware/instances'),
  types: () => api.get('api/v1/middleware/types'),
  create: (data: { name: string, type: string, host: string, port?: number, user?: string, password?: string, remark?: string }) => api.post('api/v1/middleware/instances', data),
  update: (id: number, data: { host?: string, port?: number, user?: string, password?: string, remark?: string }) => api.put(`api/v1/middleware/instances/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/middleware/instances/${id}`),
  test: (id: number) => api.post(`api/v1/middleware/instances/${id}/test`),
  reveal: (id: number) => api.get(`api/v1/middleware/instances/${id}/reveal`),
  versionRefresh: (id: number, node?: string) => api.post(`api/v1/middleware/instances/${id}/version?node=${node || 'local'}`),
  scan: (node?: string) => api.get(`api/v1/middleware/scan?node=${node || 'local'}`),
  adopt: (data: { node?: string, containerName: string, name?: string }) => api.post('api/v1/middleware/adopt', data),
  reuseOptions: (type: string, prefix?: string) => api.get(`api/v1/middleware/reuse?type=${type}&prefix=${prefix || ''}`),
}
