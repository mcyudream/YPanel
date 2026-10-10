import api from '../index'

export interface NatRule {
  id: number
  nodeId: string
  name: string
  protocol: 'tcp' | 'udp'
  ipFamily: 4 | 6
  listenPort: number
  listenPortEnd: number
  targetIp: string
  targetPort: number
  targetPortEnd: number
  iface: string
  destIp: string
  srcSpec: string
  enabled: boolean
  sort: number
  createdAt: string
  updatedAt: string
}

export interface NatExternalRule {
  id: string
  chain: string
  family: 4 | 6
  spec: string
  source: 'manual' | 'docker' | 'firewalld' | 'ufw' | 'libvirt' | 'k8s' | 'custom'
  proto: string
  iface: string
  destIp: string
  dportStart: number
  dportEnd: number
  toIp: string
  toPortStart: number
  toPortEnd: number
  comment: string
  importable: boolean
  reason: string
}

export interface NatExternalList {
  available: boolean
  rules: NatExternalRule[]
}

export interface NatInterface {
  name: string
  family: 4 | 6
  addr: string
  internal: boolean
}

export interface NatPortOccupy {
  port: number
  proto: string
  process: string
}

export interface NatSaveResult {
  rule: NatRule
  warnings: string[]
}

export interface NatImportItem {
  ruleId: string
  name?: string
}

export interface NatImportResult {
  rules: NatRule[]
  warnings: string[]
}

export interface NatApplyResult {
  warnings: string[]
}

export default {
  list: async (nodeId: string) => {
    const res = await api.get('api/v1/nat/forwards', { params: { nodeId }, silent: true })
    return res.data as NatRule[]
  },
  save: async (data: Partial<NatRule>) => {
    const res = await api.post('api/v1/nat/forwards', data)
    return res.data as NatSaveResult
  },
  remove: (id: number) => api.delete(`api/v1/nat/forwards/${id}`),
  enable: (id: number) => api.post(`api/v1/nat/forwards/${id}/enable`),
  disable: (id: number) => api.post(`api/v1/nat/forwards/${id}/disable`),
  interfaces: async (nodeId: string, family?: 4 | 6) => {
    const res = await api.get('api/v1/nat/interfaces', { params: { nodeId, family: family ?? '' }, silent: true })
    return res.data as NatInterface[]
  },
  checkPort: async (nodeId: string, protocol: string, listenPort: number, listenPortEnd: number) => {
    const res = await api.post('api/v1/nat/check-port', { nodeId, protocol, listenPort, listenPortEnd })
    return (res.data as { occupied: NatPortOccupy[] }).occupied
  },
  apply: async (nodeId: string) => {
    const res = await api.post('api/v1/nat/apply', { nodeId })
    return res.data as NatApplyResult
  },
  external: async (nodeId: string) => {
    const res = await api.get('api/v1/nat/external', { params: { nodeId }, silent: true })
    return res.data as NatExternalList
  },
  import: async (nodeId: string, items: NatImportItem[]) => {
    const res = await api.post('api/v1/nat/import', { nodeId, items })
    return res.data as NatImportResult
  },
}
