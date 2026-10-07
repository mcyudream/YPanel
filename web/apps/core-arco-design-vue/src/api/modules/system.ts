import api from '../index'

export interface SystemOverview {
  hostname: string
  os: string
  platform: string
  kernelVersion: string
  arch: string
  uptime: number
  cpu: { logicalCount: number, physicalCount: number, modelName: string, usagePercent: number, perCore: number[] }
  memory: { total: number, used: number, available: number, usagePercent: number }
  swap: { total: number, used: number, available: number, usagePercent: number }
  disks: { mountpoint: string, fsType: string, total: number, used: number, free: number, usagePercent: number }[]
  network: { rxTotal: number, txTotal: number, rxSpeedBps: number, txSpeedBps: number }
  load: { load1: number, load5: number, load15: number }
  collectedAt: string
  recentNotifications?: import('./ops').NotificationItem[]
}

export interface MetricSample {
  at: string
  cpuPercent: number
  memPercent: number
  rxSpeedBps: number
  txSpeedBps: number
  load1: number
}

export interface MetricRecord {
  at: string
  nodeId?: string
  cpu: number
  mem: number
  swap?: number
  rxSpeed: number
  txSpeed: number
  load1: number
}

// node 为节点 id（默认 local，core 端 ?node= 路由）
function nodeQ(node?: string) {
  return node && node !== 'local' ? `&node=${encodeURIComponent(node)}` : ''
}

export default {
  overview: async (node?: string) => {
    const res = await api.get(`api/v1/system/overview?1=1${nodeQ(node)}`)
    return res.data as SystemOverview
  },
  history: async (seconds = 600, node?: string) => {
    const res = await api.get(`api/v1/system/history?seconds=${seconds}${nodeQ(node)}`)
    return res.data as MetricSample[]
  },
  // 历史监控持久化查询（全节点 30 天，60s 粒度）
  historyPersisted: async (seconds = 3600, node?: string) => {
    const res = await api.get(`api/v1/system/history/persisted?seconds=${seconds}${nodeQ(node)}`, { silent: true })
    return res.data as MetricRecord[]
  },
}
