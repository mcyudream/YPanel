import api from '../index'
import { makeNodeIdApi } from '../hostNode'

// M55 主机域节点路由：自动附当前 nodeId（setHostNode 切换）
const hnapi = makeNodeIdApi(api)

export interface FirewallStatus {
  available: boolean
  backend?: 'ufw' | 'firewalld' | 'none'
  enabled?: boolean
  running?: boolean
  hint?: string
  rules?: { raw: string }[]
  ports?: { port: string, proto: string, number: number }[]
  siteManaged?: { port: number, proto: string, sites: string }[]
}

export interface Fail2banJail {
  name: string
  banned: string[]
  total: number
}

export const fail2banApi = {
  status: async () => {
    const res = await hnapi.get('api/v1/fail2ban/status', { silent: true })
    return res.data as { available: boolean, hint?: string, jails?: Fail2banJail[] }
  },
  install: async () => hnapi.post('api/v1/fail2ban/install'),
  unban: (jail: string, ip: string) => hnapi.post('api/v1/fail2ban/unban', { jail, ip }),
  ban: (jail: string, ip: string) => hnapi.post('api/v1/fail2ban/ban', { jail, ip }),
}

export default {
  status: async () => {
    const res = await hnapi.get('api/v1/firewall/status', { silent: true })
    return res.data as FirewallStatus
  },
  allow: (port: string, proto = 'tcp') => hnapi.post('api/v1/firewall/allow', { port, proto }),
  deleteRule: (number: number) => hnapi.delete(`api/v1/firewall/rules/${number}`),
  enable: () => hnapi.post('api/v1/firewall/enable'),
  disable: () => hnapi.post('api/v1/firewall/disable'),
}
