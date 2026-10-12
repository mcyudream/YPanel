import api from '../index'

// B23 证书库（对齐 1Panel 证书页）：证书 / DNS 账户 / ACME 账户 / 站点分组。

// parseAltDomains 后端 altDomains 落库形态（JSON 数组字符串）→ 数组；兼容空串/已解析数组。
export function parseAltDomains(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw as string[]
  }
  if (typeof raw === 'string' && raw) {
    try {
      const v = JSON.parse(raw)
      return Array.isArray(v) ? v : []
    }
    catch {
      return []
    }
  }
  return []
}

export interface Certificate {
  id: number
  certName: string
  domain: string
  // 后端落库为 JSON 数组字符串（如 ["a.com"]），渲染用 parseAltDomains 解析
  altDomains: string
  provider: 'acme' | 'selfsigned' | 'upload'
  issuer: string
  remark: string
  autoRenew: boolean
  notAfter: string | null
  status: 'ok' | 'expiring' | 'expired' | 'error'
  acmeAccountId: number
  dnsAccountId: number
  issueLog: string
  sites: string[]
  createdAt: string
}

export interface DnsAccount {
  id: number
  name: string
  provider: 'aliyun' | 'dnspod' | 'cloudflare'
  accessKey: string
  createdAt: string
}

export interface AcmeAccount {
  id: number
  email: string
  caType: 'letsencrypt' | 'zerossl' | 'buypass'
  keyType: string
  createdAt: string
}

export interface SiteGroup {
  id: number
  name: string
  isDefault: boolean
  sites: number
}

export const certApi = {
  list: async () => {
    const res = await api.get('api/v1/certs', { silent: true })
    return res.data as Certificate[]
  },
  // 签发（任务化：秒回 taskId，进度/结果见任务中心）
  issue: async (data: { domain: string, altDomains?: string, remark?: string, autoRenew?: boolean, acmeAccountId?: number, dnsAccountId?: number, deploySiteIds?: number[] }) => {
    const res = await api.post('api/v1/certs/issue', data)
    return res.data as { taskId: number }
  },
  upload: (data: { certName?: string, domain: string, remark?: string, certPem: string, keyPem: string }) =>
    api.post('api/v1/certs/upload', data),
  selfSigned: (data: { domain: string, remark?: string, days?: number }) =>
    api.post('api/v1/certs/selfsigned', data),
  detail: async (id: number) => {
    const res = await api.get(`api/v1/certs/${id}`, { silent: true })
    return res.data as { cert: Certificate, altDomains: string[], text: string }
  },
  update: (id: number, data: { remark?: string, autoRenew?: boolean }) => api.put(`api/v1/certs/${id}`, data),
  // 续签（任务化：秒回 taskId）
  renew: async (id: number) => {
    const res = await api.post(`api/v1/certs/${id}/renew`, {})
    return res.data as { taskId: number }
  },
  remove: (id: number) => api.delete(`api/v1/certs/${id}`),

  dnsAccounts: async () => {
    const res = await api.get('api/v1/certs/dns-accounts', { silent: true })
    return res.data as DnsAccount[]
  },
  createDnsAccount: (data: { name: string, provider: string, accessKey: string, secret: string }) =>
    api.post('api/v1/certs/dns-accounts', data),
  updateDnsAccount: (id: number, data: { name?: string, provider?: string, accessKey?: string, secret?: string }) =>
    api.put(`api/v1/certs/dns-accounts/${id}`, data),
  removeDnsAccount: (id: number) => api.delete(`api/v1/certs/dns-accounts/${id}`),

  acmeAccounts: async () => {
    const res = await api.get('api/v1/certs/acme-accounts', { silent: true })
    return res.data as AcmeAccount[]
  },
  // 注册 ACME 账户（任务化：秒回 taskId，注册输出见任务日志）
  createAcmeAccount: async (data: { email: string, caType: string, keyType?: string }) => {
    const res = await api.post('api/v1/certs/acme-accounts', data)
    return res.data as { taskId: number, accountId: number }
  },
  removeAcmeAccount: (id: number) => api.delete(`api/v1/certs/acme-accounts/${id}`),
}

export const siteGroupApi = {
  list: async () => {
    const res = await api.get('api/v1/site-groups', { silent: true })
    return res.data as SiteGroup[]
  },
  create: (name: string) => api.post('api/v1/site-groups', { name }),
  rename: (id: number, name: string) => api.put(`api/v1/site-groups/${id}`, { name }),
  remove: (id: number) => api.delete(`api/v1/site-groups/${id}`),
  setDefault: (id: number) => api.post(`api/v1/site-groups/${id}/default`),
}
