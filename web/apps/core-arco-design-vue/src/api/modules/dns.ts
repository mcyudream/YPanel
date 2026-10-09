import api from '../index'

export type DnsRecordType = 'address' | 'cname' | 'txt'

export interface DnsPort53Occupy {
  port: number
  proto: string
  addr: string // 本地绑定地址（0.0.0.0 / :: / 127.0.0.53 等）
  process: string
}

export interface DnsRecord {
  id: number
  type: DnsRecordType
  domain: string
  target: string
  enabled: boolean
  comment: string
  sort: number
  createdAt: string
  updatedAt: string
}

export interface DnsSiteAlignEntry {
  domain: string
  siteName: string
}

export interface DnsOverview {
  deployed: boolean
  container: string // running / exited / ... / missing
  image: string
  dirReady: boolean
  port53: DnsPort53Occupy[]
  port53Conflict: boolean
  port53Note: string
  listenIp: string
  upstreams: string[]
  recordCount: number
  siteAlign: boolean
  siteAlignIp: string // 生效指向（未单设时=监听 IP）
  siteAlignCount: number
  siteAlignSites: number
  siteAlignPreview: DnsSiteAlignEntry[]
  deployNodeId: string
  nodeMatch: boolean
}

export interface DnsConfig {
  deployNodeId: string
  listenIp: string
  upstreams: string[]
  cacheSize: number
  image: string
  siteAlign: boolean
  siteAlignIp: string
}

export default {
  overview: async (nodeId: string) => {
    const res = await api.get('api/v1/dns/overview', { params: { nodeId }, silent: true })
    return res.data as DnsOverview
  },
  list: async () => {
    const res = await api.get('api/v1/dns/records', { silent: true })
    return res.data as DnsRecord[]
  },
  save: async (data: Partial<DnsRecord>) => {
    const res = await api.post('api/v1/dns/records', data)
    return res.data as DnsRecord
  },
  remove: (id: number) => api.delete(`api/v1/dns/records/${id}`),
  enable: (id: number) => api.post(`api/v1/dns/records/${id}/enable`),
  disable: (id: number) => api.post(`api/v1/dns/records/${id}/disable`),
  getSettings: async () => {
    const res = await api.get('api/v1/dns/settings', { silent: true })
    return res.data as DnsConfig
  },
  saveSettings: async (cfg: Partial<DnsConfig>) => {
    const res = await api.put('api/v1/dns/settings', cfg)
    return res.data as DnsConfig
  },
  deploy: async (nodeId: string) => {
    const res = await api.post('api/v1/dns/deploy', { nodeId })
    return res.data as { warnings: string[] }
  },
  undeploy: async (nodeId: string, removeFiles: boolean) => {
    const res = await api.post('api/v1/dns/undeploy', { nodeId, removeFiles })
    return res.data
  },
  apply: async (nodeId: string) => {
    const res = await api.post('api/v1/dns/apply', { nodeId })
    return res.data
  },
  check: async (nodeId: string, domain: string) => {
    const res = await api.post('api/v1/dns/check', { nodeId, domain })
    return (res.data as { output: string }).output
  },
  interfaces: async (nodeId: string) => {
    const res = await api.get('api/v1/dns/interfaces', { params: { nodeId }, silent: true })
    return res.data as { name: string, family: 4 | 6, addr: string, internal: boolean }[]
  },
}
