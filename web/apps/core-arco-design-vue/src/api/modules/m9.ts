import api from '../index'

// ---- 通知中心 ----
export interface NotificationItem {
  id: number
  level: 'info' | 'success' | 'warning' | 'error'
  title: string
  content: string
  read: boolean
  createdAt: string
}

export const notificationApi = {
  list: async (limit = 50) => {
    const res = await api.get(`api/v1/notifications?limit=${limit}`, { silent: true })
    return res.data as NotificationItem[]
  },
  unread: async () => {
    const res = await api.get('api/v1/notifications/unread', { silent: true })
    return (res.data as { count: number }).count
  },
  markRead: (id?: number) => api.post(`api/v1/notifications/read${id ? `?id=${id}` : ''}`),
}

// ---- 面板备份 ----
export interface PanelBackup {
  name: string
  sizeMb: number
  modTime: string
  path: string
}

export const panelBackupApi = {
  list: async () => {
    const res = await api.get('api/v1/panel/backups', { silent: true })
    return res.data as PanelBackup[]
  },
  create: async () => {
    const res = await api.post('api/v1/panel/backups')
    return res.data as { file?: string }
  },
  remove: (file: string) => api.delete(`api/v1/panel/backups?file=${encodeURIComponent(file)}`),
  restoreHint: async () => {
    const res = await api.get('api/v1/panel/backups/restore-hint', { silent: true })
    return (res.data as { hint: string }).hint
  },
  downloadURL: (path: string, token: string) =>
    `api/v1/files/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}`,
}

// ---- 站点识别 ----
export interface DiscoveredSite {
  file: string
  domain: string
  port: string
  type: 'static' | 'proxy'
  proxyPass: string
  root: string
}

export const siteDiscoveryApi = {
  scan: async () => {
    const res = await api.get('api/v1/sites/scan', { silent: true })
    return res.data as { sites: DiscoveredSite[], containers: { name: string, image: string, ports: string }[] }
  },
  adopt: (data: { file: string, domain: string, type: string, proxyPass?: string }) =>
    api.post('api/v1/sites/adopt', data),
}

// ---- 操作审计 ----
export interface AuditLogItem {
  id: number
  username: string
  method: string
  path: string
  ip: string
  success: boolean
  createdAt: string
}

export const auditApi = {
  list: async (page = 1, pageSize = 20) => {
    const res = await api.get(`api/v1/audit/ops?page=${page}&pageSize=${pageSize}`, { silent: true })
    return res.data as { total: number, items: AuditLogItem[] }
  },
}
