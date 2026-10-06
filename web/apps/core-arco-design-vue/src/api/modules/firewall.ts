import api from '../index'

export interface FirewallStatus {
  available: boolean
  enabled?: boolean
  hint?: string
  rules?: { raw: string }[]
}

export default {
  status: async () => {
    const res = await api.get('api/v1/firewall/status', { silent: true })
    return res.data as FirewallStatus
  },
  allow: (port: string, proto = 'tcp') => api.post('api/v1/firewall/allow', { port, proto }),
  deleteRule: (number: number) => api.delete(`api/v1/firewall/rules/${number}`),
  enable: () => api.post('api/v1/firewall/enable'),
  disable: () => api.post('api/v1/firewall/disable'),
}
