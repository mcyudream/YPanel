import api from '../index'

export interface SiteDomainConf {
  name: string
  domain: string
  domains: string[]
  certDomain: string
  certNotice?: string
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
  certId: number
  httpMode: 'redirect' | 'both' | 'deny'
  hsts: boolean
  hstsSubdomain: boolean
  tlsVersions: string[]
  ciphers: string
  http2: boolean
}

export interface SiteHTTPSUpdate {
  certId?: number
  selfSigned?: boolean
  disable?: boolean
  httpMode: 'redirect' | 'both' | 'deny'
  hsts: boolean
  hstsSubdomain: boolean
  tlsVersions: string[]
  ciphers: string
  http2: boolean
}

export interface SiteRunDirConf {
  root: string
  hostRoot: string
  runDir: string
  subdirs: string[]
}

// M50 监听端口配置域
export interface SitePortConf {
  port: number
  mode: 'container' | 'host'
  applied: boolean
}

function base(id: number | string, domain: string) {
  return `api/v1/sites/${id}/conf/${domain}`
}

export const siteConfApi = {
  getDomain: async (id: number | string) => {
    const res = await api.get(base(id, 'domain'), { silent: true })
    return res.data as SiteDomainConf
  },
  updateDomain: async (id: number | string, domains: string[], primary?: string) => {
    const res = await api.put(base(id, 'domain'), { domains, primary: primary || '' })
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
  updateHTTPS: async (id: number | string, conf: SiteHTTPSUpdate) => {
    const res = await api.put(base(id, 'https'), conf)
    return res.data as SiteHTTPSConf
  },
  // B23 网站目录（运行目录）
  getRunDir: async (id: number | string) => {
    const res = await api.get(base(id, 'rundir'), { silent: true })
    return res.data as SiteRunDirConf
  },
  updateRunDir: async (id: number | string, runDir: string) => {
    await api.put(base(id, 'rundir'), { runDir })
  },
  // M50 监听端口（非 80/443 保存后自动追加容器映射 / 联动防火墙放行）
  getPort: async (id: number | string) => {
    const res = await api.get(base(id, 'port'), { silent: true })
    return res.data as SitePortConf
  },
  updatePort: async (id: number | string, port: number) => {
    const res = await api.put(base(id, 'port'), { port })
    return res.data as SitePortConf
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
