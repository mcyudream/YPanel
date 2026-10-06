import api from '../index'

export interface NodeItem {
  id: string
  name: string
  remote: boolean
  addr?: string
  hostname?: string
  os?: string
  arch?: string
  version?: string
  online: boolean
  lastSeenAt?: string
}

export default {
  list: async () => {
    const res = await api.get('api/v1/nodes', { silent: true })
    return res.data as NodeItem[]
  },
  pairingCode: async () => {
    const res = await api.post('api/v1/nodes/pairing-code')
    return res.data as { code: string, expireIn: number }
  },
  remove: (id: string) => api.delete(`api/v1/nodes/${id}`),
}
