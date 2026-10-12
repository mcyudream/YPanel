import api from '../index'

export interface OnlineSource {
  source: 'github' | 'gitee' | 'gitlab'
  reachable: boolean
  version?: string
  name?: string
  notes?: string
  published?: string
  assetUrl?: string
  assetSize?: number
  sumUrl?: string
  error?: string
}

export interface OnlineCheck {
  currentVersion: string
  currentIsDev: boolean
  latest?: string
  updatable: boolean
  sources: OnlineSource[]
}

export const selfUpdateApi = {
  // 在线检查更新（GitHub/Gitee 并行探测）
  check: async () => {
    const res = await api.get('api/v1/system/update/check', { silent: true })
    return res.data as OnlineCheck
  },
  // 一键升级（任务化：秒回 taskId，下载/校验/应用进度见任务中心，应用后面板自动重启）
  upgrade: async (source: 'github' | 'gitee' | 'gitlab') => {
    const res = await api.post('api/v1/system/update/upgrade', { source })
    return res.data as { taskId: number }
  },
}
