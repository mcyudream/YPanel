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
  // B1：ACME 签发（DNS API 验证）
  issueACME: async (id: number | string, domain: string) => {
    const res = await api.post(`api/v1/sites/${id}/cert/acme`, { domain }, { timeout: 600000 })
    return res.data as { domain: string, certDomain: string, issuer: string }
  },
}

// ---- S20 第二批配置域 ----

export interface SiteAntiLeech {
  enable: boolean
  validReferers: string[]
  allowNone: boolean
  allowBlocked: boolean
  returnCode: number
}

export interface SiteAuthBasicUser {
  user: string
  password: string
}

export interface SiteAuthBasic {
  enable: boolean
  realm: string
  users: SiteAuthBasicUser[]
}

export interface SiteCORS {
  enable: boolean
  allowOrigins: string[]
  allowMethods: string[]
  allowHeaders: string[]
  allowCredentials: boolean
  maxAge: number
}

export interface SiteRedirect {
  enable: boolean
  target: string
  code: number
}

export interface SiteRealIP {
  enable: boolean
  trustedProxies: string[]
  header: string
}

export interface SiteLimitConn {
  enable: boolean
  connPerIP: number
}

export interface SiteUpstream {
  address: string
  weight: number
}

export interface SiteLoadBalance {
  enable: boolean
  strategy: string
  upstreams: SiteUpstream[]
}

export interface SiteExtraConf {
  antiLeech?: SiteAntiLeech
  authBasic?: SiteAuthBasic
  cors?: SiteCORS
  redirect?: SiteRedirect
  realIP?: SiteRealIP
  limitConn?: SiteLimitConn
  loadBalance?: SiteLoadBalance
}

function put<TReq, TRes>(id: number | string, domain: string, body: TReq) {
  return api.put(base(id, domain), body).then((res) => res.data as TRes)
}

export const siteExtraApi = {
  getAntiLeech: async (id: number | string) => (await api.get(base(id, 'antileech'), { silent: true })).data as SiteAntiLeech,
  updateAntiLeech: (id: number | string, c: SiteAntiLeech) => put<SiteAntiLeech, SiteAntiLeech>(id, 'antileech', c),
  getAuthBasic: async (id: number | string) => (await api.get(base(id, 'authbasic'), { silent: true })).data as SiteAuthBasic,
  updateAuthBasic: (id: number | string, c: SiteAuthBasic) => put<SiteAuthBasic, SiteAuthBasic>(id, 'authbasic', c),
  getCORS: async (id: number | string) => (await api.get(base(id, 'cors'), { silent: true })).data as SiteCORS,
  updateCORS: (id: number | string, c: SiteCORS) => put<SiteCORS, SiteCORS>(id, 'cors', c),
  getRedirect: async (id: number | string) => (await api.get(base(id, 'redirect'), { silent: true })).data as SiteRedirect,
  updateRedirect: (id: number | string, c: SiteRedirect) => put<SiteRedirect, SiteRedirect>(id, 'redirect', c),
  getRealIP: async (id: number | string) => (await api.get(base(id, 'realip'), { silent: true })).data as SiteRealIP,
  updateRealIP: (id: number | string, c: SiteRealIP) => put<SiteRealIP, SiteRealIP>(id, 'realip', c),
  getLimitConn: async (id: number | string) => (await api.get(base(id, 'limitconn'), { silent: true })).data as SiteLimitConn,
  updateLimitConn: (id: number | string, c: SiteLimitConn) => put<SiteLimitConn, SiteLimitConn>(id, 'limitconn', c),
  getLoadBalance: async (id: number | string) => (await api.get(base(id, 'loadbalance'), { silent: true })).data as SiteLoadBalance,
  updateLoadBalance: (id: number | string, c: SiteLoadBalance) => put<SiteLoadBalance, SiteLoadBalance>(id, 'loadbalance', c),
}
