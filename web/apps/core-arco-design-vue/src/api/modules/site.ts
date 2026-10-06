import api from '../index'

export interface SiteItem {
  id: number
  name: string
  type: 'static' | 'proxy'
  domain: string
  port: number
  proxyPass: string
  certDomain: string
  enabled: boolean
  onDisk: boolean
  nginxRunning: boolean
  createdAt: string
}

export default {
  status: async () => {
    const res = await api.get('api/v1/nginx/status', { silent: true })
    return res.data as { installed: boolean, running: boolean, sites: number }
  },
  install: () => api.post('api/v1/nginx/install'),
  list: async () => {
    const res = await api.get('api/v1/sites', { silent: true })
    return res.data as SiteItem[]
  },
  create: (data: { name: string, type: string, domain: string, port?: number, proxyPass?: string }) =>
    api.post('api/v1/sites', data),
  remove: (id: number, purge: boolean) => api.delete(`api/v1/sites/${id}?purge=${purge}`),
  enable: (id: number) => api.post(`api/v1/sites/${id}/enable`),
  disable: (id: number) => api.post(`api/v1/sites/${id}/disable`),
  config: async (id: number) => {
    const res = await api.get(`api/v1/sites/${id}/config`, { silent: true })
    return (res.data as { content: string }).content
  },
  updateConfig: (id: number, content: string) => api.put(`api/v1/sites/${id}/config`, { content }),
  issueSelfSigned: (id: number) => api.post(`api/v1/sites/${id}/cert/selfsigned`),
}
