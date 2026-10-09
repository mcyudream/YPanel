import api from '../index'

export interface DiskUsageTreeItem {
  name: string
  path: string
  size: number
  isDir: boolean
}

export interface DiskUsageTree {
  path: string
  total: number
  items: DiskUsageTreeItem[]
  collectedAt: string
}

export default {
  // du 语义实算 + agent 侧 24h 缓存（refresh=1 绕过重算并回写缓存）：勿高频轮询
  tree: async (path: string, refresh = false) => {
    const res = await api.get(`api/v1/disk/usage?path=${encodeURIComponent(path)}${refresh ? '&refresh=1' : ''}`, { silent: true })
    return res.data as DiskUsageTree
  },
}
