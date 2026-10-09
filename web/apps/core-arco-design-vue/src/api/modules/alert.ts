import api from '../index'

export interface AlertRule {
  id: number
  name: string
  metric: 'cpu' | 'memory' | 'disk' | 'load' | 'network' | 'cert_expiry' | 'site_expiry' | 'log'
  threshold: number
  webhookUrl: string
  webhookType: 'feishu' | 'dingtalk' | 'wecom' | 'generic' | 'telegram' | 'bark' | 'email'
  silentStart: string
  silentEnd: string
  query?: string
  windowSec?: number
  enabled: boolean
}

export interface SmtpSettings {
  host: string
  port: number
  user: string
  pass: string
  from: string
  ssl: boolean
  to: string
  hasPass?: boolean
}

export default {
  list: async () => {
    const res = await api.get('api/v1/alert/rules', { silent: true })
    return res.data as AlertRule[]
  },
  create: (data: { name: string, metric: string, threshold: number, webhookUrl?: string, webhookType: string, silentStart?: string, silentEnd?: string, query?: string, windowSec?: number }) =>
    api.post('api/v1/alert/rules', data),
  update: (id: number, data: { enabled?: boolean, threshold?: number, webhookUrl?: string, silentStart?: string, silentEnd?: string, query?: string, windowSec?: number }) =>
    api.put(`api/v1/alert/rules/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/alert/rules/${id}`),
  // M37：SMTP 邮件通道
  getSmtp: async () => {
    const res = await api.get('api/v1/alert/smtp', { silent: true })
    return res.data as SmtpSettings
  },
  putSmtp: (data: SmtpSettings) => {
    const { hasPass: _hasPass, ...body } = data
    return api.put('api/v1/alert/smtp', body)
  },
  testSmtp: async (to: string) => {
    const res = await api.post('api/v1/alert/smtp/test', { to })
    return res.data as { ok: boolean }
  },
}
