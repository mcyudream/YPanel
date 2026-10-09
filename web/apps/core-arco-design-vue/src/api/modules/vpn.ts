import api from '../index'

// EasyTier 组网（工具域）—— /api/v1/vpn/easytier/*

export interface VpnProxyNetwork {
  cidr: string
  remap?: string
  enabled: boolean
}

export interface VpnConfig {
  rawMode: boolean
  raw?: string
  instanceName: string
  hostname: string
  networkName: string
  networkSecret: string
  virtualIp: string
  dhcp: boolean
  peers: string[]
  listeners: string[]
  proxyNetworks: VpnProxyNetwork[]
  exitNodes: string[]
  enableExitNode: boolean
  latencyFirst: boolean
  devName: string
}

export interface VpnStatus {
  installed: boolean
  running: boolean
  project: string
  version: string
  virtualIp: string
  peerCount: number
  networkName: string
  configured: boolean
  hasTun: boolean
}

export interface VpnTable {
  columns: string[]
  rows: string[][]
}

export const vpnApi = {
  status: async () => {
    const res = await api.get('api/v1/vpn/easytier/status', { silent: true })
    return res.data as VpnStatus
  },
  getConfig: async () => {
    const res = await api.get('api/v1/vpn/easytier/config', { silent: true })
    return res.data as { config: VpnConfig, rendered: string }
  },
  saveConfig: async (data: VpnConfig) => {
    const res = await api.put('api/v1/vpn/easytier/config', data, { timeout: 300000 })
    return res.data as VpnStatus
  },
  install: async (data: Partial<Pick<VpnConfig, 'networkName' | 'networkSecret' | 'virtualIp' | 'dhcp'>> = {}) => {
    const res = await api.post('api/v1/vpn/easytier/install', data)
    return res.data as { taskId: number }
  },
  apply: async () => {
    const res = await api.post('api/v1/vpn/easytier/apply', null, { timeout: 300000 })
    return res.data as VpnStatus
  },
  peers: async () => {
    const res = await api.get('api/v1/vpn/easytier/peers', { silent: true })
    return res.data as VpnTable
  },
  routes: async () => {
    const res = await api.get('api/v1/vpn/easytier/routes', { silent: true })
    return res.data as VpnTable
  },
}
