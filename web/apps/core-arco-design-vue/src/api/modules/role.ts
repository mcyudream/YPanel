import api from '../index'

// M54 RBAC：角色与权限点（与后端 rbac 包 / dto.RoleInfo 对齐）

export interface Perm {
  key: string
  group: string
}

export interface CatalogGroup {
  key: string
  perms: Perm[]
}

export interface RoleInfo {
  id: number
  key: string
  name: string
  builtin: boolean
  scopeAllNodes: boolean
  dataScope: 'all' | 'assigned'
  remark: string
}

export interface RoleItem extends RoleInfo {
  perms: string[]
}

export default {
  catalog: async () => {
    const res = await api.get('api/v1/rbac/catalog')
    return res.data as CatalogGroup[]
  },
  list: async () => {
    const res = await api.get('api/v1/rbac/roles')
    return res.data as RoleItem[]
  },
  create: (data: { key: string, name: string, remark?: string, dataScope?: string, perms: string[] }) =>
    api.post('api/v1/rbac/roles', data),
  update: (id: number, data: { name?: string, remark?: string, dataScope?: string, perms?: string[] }) =>
    api.put(`api/v1/rbac/roles/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/rbac/roles/${id}`),
}
