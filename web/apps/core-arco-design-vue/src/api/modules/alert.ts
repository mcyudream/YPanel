import api from '../index'

export interface AlertRule {
  id: number
  name: string
  metric: 'cpu' | 'memory' | 'disk'
  threshold: number
  webhookUrl: string
  webhookType: 'feishu' | 'dingtalk' | 'wecom' | 'generic'
  enabled: boolean
}

export default {
  list: async () => {
    const res = await api.get('api/v1/alert/rules', { silent: true })
    return res.data as AlertRule[]
  },
  create: (data: { name: string, metric: string, threshold: number, webhookUrl: string, webhookType: string }) =>
    api.post('api/v1/alert/rules', data),
  update: (id: number, data: { enabled?: boolean, threshold?: number, webhookUrl?: string }) =>
    api.put(`api/v1/alert/rules/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/alert/rules/${id}`),
}
