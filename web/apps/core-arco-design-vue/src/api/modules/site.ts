import api from '../index'

export interface ProxyRule {
  prefix: string
  target: string
  ws?: boolean
}

export interface SiteItem {
  id: number
  name: string
  type: 'static' | 'proxy'
  domain: string
  domains: string[]
  port: number
  proxyPass: string
  indexFiles: string
  logsEnabled: boolean
  certDomain: string
  certId: number
  certNotAfter: string | null
  groupId: number
  groupName: string
  remark: string
  runDir: string
  enabled: boolean
  onDisk: boolean
  nginxRunning: boolean
  createdAt: string
}

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

export interface SiteWaf {
  denyIps: string[]
  allowIps: string[]
  denyUAs: string[]
  rateEnable: boolean
  rate: number
  burst: number
}

export interface RewriteTemplate {
  name: string
  content: string
}

export interface SiteExtConfig {
  rewriteName: string
  rewriteContent: string
  customLocations: { comment: string, content: string }[]
  errorPage404: string
  cacheEnable: boolean
  cacheDuration: string
}

export const extApi = {
  rewriteTemplates: async () => {
    const res = await api.get('api/v1/sites/rewrite-templates', { silent: true })
    return res.data as RewriteTemplate[]
  },
  getExt: async (id: number) => {
    const res = await api.get(`api/v1/sites/${id}/ext`, { silent: true })
    return res.data as SiteExtConfig
  },
  updateExt: (id: number, ext: SiteExtConfig) => api.put(`api/v1/sites/${id}/ext`, ext),
}

export const wafApi = {
  get: async (id: number) => {
    const res = await api.get(`api/v1/sites/${id}/waf`, { silent: true })
    return res.data as SiteWaf
  },
  update: (id: number, waf: SiteWaf) => api.put(`api/v1/sites/${id}/waf`, waf),
}

export default {
  status: async () => {
    const res = await api.get('api/v1/nginx/status', { silent: true })
    return res.data as { installed: boolean, running: boolean, sites: number, mode?: 'container' | 'host' }
  },
  install: () => api.post('api/v1/nginx/install'),
  adoptHost: async () => {
    const res = await api.post('api/v1/nginx/adopt-host', null, { timeout: 60000 })
    return res.data as { detected: boolean, mode?: string, hint?: string }
  },
  setMode: (mode: 'container' | 'host') => api.put('api/v1/nginx/mode', { mode }),
  getOne: async (id: number | string) => {
    const res = await api.get(`api/v1/sites/${id}/detail`, { silent: true })
    return res.data as SiteItem
  },
  list: async () => {
    const res = await api.get('api/v1/sites', { silent: true })
    return res.data as SiteItem[]
  },
  siteLogs: async (id: number, type = 'access', tail = 200) => {
    const res = await api.get(`api/v1/sites/${id}/logs?type=${type}&tail=${tail}`, { silent: true })
    return (res.data as { content: string }).content
  },
  create: (data: { name: string, type: string, domain: string, extraDomains?: string[], port?: number, proxyRules?: ProxyRule[], proxyPass?: string, indexFiles?: string, runtimeId?: number, groupId?: number, remark?: string, runDir?: string }) =>
    api.post('api/v1/sites', data),
  updateMeta: (id: number, data: { groupId?: number, remark?: string }) => api.put(`api/v1/sites/${id}/meta`, data),
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
