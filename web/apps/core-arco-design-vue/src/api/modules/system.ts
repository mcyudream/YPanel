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
}

export interface MetricSample {
  at: string
  cpuPercent: number
  memPercent: number
  rxSpeedBps: number
  txSpeedBps: number
  load1: number
}

export default {
  overview: async () => {
    const res = await api.get('api/v1/system/overview')
    return res.data as SystemOverview
  },
  history: async (seconds = 600) => {
    const res = await api.get(`api/v1/system/history?seconds=${seconds}`)
    return res.data as MetricSample[]
  },
}
