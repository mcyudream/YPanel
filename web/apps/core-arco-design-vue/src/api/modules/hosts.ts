import api from '../index'

export interface HostRecord {
  id: number
  ip: string
  hostnames: string
  comment: string
  enabled: boolean
  sort: number
  createdAt: string
  updatedAt: string
}

export type HostNodeState = 'match' | 'notApplied' | 'drift' | 'offline' | 'error'

export interface HostNodeStatus {
  nodeId: string
  targeted: boolean
  state: HostNodeState
  message: string
}

export interface HostApplyResult {
  nodeId: string
  ok: boolean
  message: string
}

export default {
  list: async () => {
    const res = await api.get('api/v1/hosts/records', { silent: true })
    return res.data as HostRecord[]
  },
  save: async (data: Partial<HostRecord>) => {
    const res = await api.post('api/v1/hosts/records', data)
    return res.data as HostRecord
  },
  remove: (id: number) => api.delete(`api/v1/hosts/records/${id}`),
  enable: (id: number) => api.post(`api/v1/hosts/records/${id}/enable`),
  disable: (id: number) => api.post(`api/v1/hosts/records/${id}/disable`),
  status: async () => {
    const res = await api.get('api/v1/hosts/status', { silent: true })
    return res.data as HostNodeStatus[]
  },
  apply: async (nodeIds: string[]) => {
    const res = await api.post('api/v1/hosts/apply', { nodeIds })
    return (res.data as { results: HostApplyResult[] }).results
  },
  removeNode: async (nodeId: string) => {
    await api.post('api/v1/hosts/remove', { nodeId })
  },
}
