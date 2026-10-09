import api from '../index'

// 磁盘空间保护（Disk Guard）API（M29）

export interface DiskGuardConfig {
  enabled: boolean
  thresholdGB: number
  exclude: string[]
}

export interface GuardStoppedItem {
  id: string
  name: string
  restartPolicy: string
}

export interface DiskGuardEvent {
  id: number
  nodeId: string
  nodeName: string
  triggeredAt: string
  restoredAt: string | null
  freeBytes: number
  thresholdBytes: number
  containers: GuardStoppedItem[]
  remark: string
}

export interface DiskGuardNodeDisk {
  mountpoint: string
  fsType: string
  total: number
  used: number
  free: number
  usagePercent: number
}

export interface DiskGuardNodeStatus {
  nodeId: string
  nodeName: string
  online: boolean
  triggered: boolean
  agentTooOld: boolean
  disks: DiskGuardNodeDisk[]
}

export interface DiskGuardStatus {
  config: DiskGuardConfig
  events: DiskGuardEvent[]
  nodes: DiskGuardNodeStatus[]
}

export const diskGuardApi = {
  status: async () => {
    const res = await api.get('api/v1/diskguard/status', { silent: true })
    return res.data as DiskGuardStatus
  },
  updateConfig: async (data: DiskGuardConfig) => {
    const res = await api.put('api/v1/diskguard/config', data)
    return res.data as DiskGuardConfig
  },
  restore: async (eventId = 0) => {
    const res = await api.post('api/v1/diskguard/restore', { eventId })
    return res.data as { started: string[], failed: string[] }
  },
}
