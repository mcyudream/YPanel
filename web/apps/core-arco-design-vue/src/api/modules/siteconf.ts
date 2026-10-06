import api from '../index'

export interface SiteDomainConf {
  name: string
  domain: string
  domains: string[]
  certDomain: string
}

export interface SiteDefaultsConf {
  indexFiles: string
  errorPage404: string
}

export interface SiteProxyRule {
  prefix: string
  target: string
  ws: boolean
}

export interface SiteProxyConf {
  rules: SiteProxyRule[]
  cacheEnable: boolean
  cacheDuration: string
}

export interface SiteRewriteConf {
  rewriteName: string
  rewriteContent: string
}

export interface SiteHTTPSConf {
  enable: boolean
  certDomain: string
  httpRedirect: boolean
}

function base(id: number | string, domain: string) {
  return `api/v1/sites/${id}/conf/${domain}`
}

export const siteConfApi = {
  getDomain: async (id: number | string) => {
    const res = await api.get(base(id, 'domain'), { silent: true })
    return res.data as SiteDomainConf
  },
  updateDomain: async (id: number | string, domains: string[]) => {
    const res = await api.put(base(id, 'domain'), { domains })
    return res.data as SiteDomainConf
  },
  getDefaults: async (id: number | string) => {
    const res = await api.get(base(id, 'defaults'), { silent: true })
    return res.data as SiteDefaultsConf
  },
  updateDefaults: async (id: number | string, conf: SiteDefaultsConf) => {
    const res = await api.put(base(id, 'defaults'), conf)
    return res.data as SiteDefaultsConf
  },
  getProxy: async (id: number | string) => {
    const res = await api.get(base(id, 'proxy'), { silent: true })
    return res.data as SiteProxyConf
  },
  updateProxy: async (id: number | string, conf: SiteProxyConf) => {
    const res = await api.put(base(id, 'proxy'), conf)
    return res.data as SiteProxyConf
  },
  getRewrite: async (id: number | string) => {
    const res = await api.get(base(id, 'rewrite'), { silent: true })
    return res.data as SiteRewriteConf
  },
  updateRewrite: async (id: number | string, conf: SiteRewriteConf) => {
    const res = await api.put(base(id, 'rewrite'), conf)
    return res.data as SiteRewriteConf
  },
  getHTTPS: async (id: number | string) => {
    const res = await api.get(base(id, 'https'), { silent: true })
    return res.data as SiteHTTPSConf
  },
  enableHTTPS: async (id: number | string) => {
    const res = await api.post(base(id, 'https'))
    return res.data as SiteHTTPSConf
  },
  disableHTTPS: async (id: number | string) => {
    const res = await api.delete(base(id, 'https'))
    return res.data as SiteHTTPSConf
  },
}
