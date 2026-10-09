import api from '../index'

export interface OnlineSource {
  source: 'github' | 'gitee'
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
  // 一键升级（下载→sha256 校验→替换→重启）
  upgrade: async (source: 'github' | 'gitee') => {
    const res = await api.post('api/v1/system/update/upgrade', { source }, { timeout: 600000 })
    return res.data as { message: string, version?: string }
  },
}
