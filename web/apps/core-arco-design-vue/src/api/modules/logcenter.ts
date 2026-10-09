import api from '../index'
import type { ContainerItem } from './container'

// M33 日志中心：容器日志跨节点服务端搜索
export interface LogSearchItem {
  ts: string
  container: string
  id: string
  level?: string
  line: string
  node?: string // 跨节点查询时由 core 归并侧标注
}

export interface LogSearchResp {
  items: LogSearchItem[]
  truncated: boolean
  scanned: number
  errors?: string[]
}

export interface LogSearchBody {
  containers: string[]
  since: string
  until: string
  pattern: string
  negate: boolean
  perLimit: number
  totalLimit: number
  nodes: string[]
}

export default {
  search: async (body: LogSearchBody) => {
    const res = await api.post('api/v1/logs/search', body, { timeout: 90000 })
    return res.data as LogSearchResp
  },
  // 指定节点的容器列表（core 侧 ?node= 代理）
  containers: async (node: string) => {
    const suffix = node && node !== 'local' ? `&node=${encodeURIComponent(node)}` : ''
    const res = await api.get(`api/v1/docker/containers?1=1${suffix}`, { silent: true })
    return res.data as ContainerItem[]
  },
}

// ---- M33 P2 集中存储（VictoriaLogs，多节点自动发现聚合）----

export interface VLInstance {
  nodeId: string
  nodeName: string
  online: boolean
  found: boolean
  container?: string
  port?: string
  error?: string
}

export interface CentralQueryItem {
  ts: string
  stream: string
  msg: string
  node?: string // 多节点聚合时的来源节点标
}

export interface CentralQueryResult {
  items: CentralQueryItem[]
  truncated: boolean
}

export interface CentralStreamValue {
  value: string
  hits: number
}

export const centralApi = {
  // 各节点 VL 自动发现状态（面板自动管理，无手工地址）
  status: async () => {
    const res = await api.get('api/v1/logs/central/status', { silent: true })
    return res.data as { instances: VLInstance[] }
  },
  query: async (body: { query: string, start: string, end: string, limit: number, offset?: number }) => {
    const res = await api.post('api/v1/logs/central/query', body, { timeout: 90000 })
    return res.data as CentralQueryResult
  },
  streams: async (body: { field?: string, start?: string, end?: string, limit?: number }) => {
    const res = await api.post('api/v1/logs/central/streams', body, { silent: true })
    return res.data as CentralStreamValue[]
  },
  // 直方图 + 命中总数（VL hits：total 即命中数，v1.53 无 count 路径）
  hits: async (body: { query: string, start: string, end: string, step?: string }) => {
    const res = await api.post('api/v1/logs/central/hits', body, { timeout: 60000 })
    return res.data as { total: number, timestamps: string[], values: number[], series: { fields: Record<string, any>, timestamps: string[], values: number[], total: number }[] }
  },
  // 保留策略（VL 自动清理过期数据）
  retention: async () => {
    const res = await api.get('api/v1/logs/central/retention', { silent: true })
    return res.data as { nodeId: string, nodeName: string, found: boolean, period: string, error?: string }[]
  },
  setRetention: async (period: string) => {
    const res = await api.put('api/v1/logs/central/retention', { period }, { timeout: 300000 })
    return res.data as { applied: string[], failures: string[] }
  },
}

// ---- 查询历史 / 常用查询（SQLite） ----

export interface LogSearchQueryItem {
  id: number
  queryText: string
  containersJson: string // 过滤条件 JSON（{container_name:[],host:[],image:[]}）
  keyword: string
  isRegex: boolean
  rangeType: string
  startAt: string | null
  endAt: string | null
  pinned: boolean
  remark: string
  createdAt: string
}

export const historyApi = {
  list: async () => {
    const res = await api.get('api/v1/logs/central/history', { silent: true })
    return res.data as LogSearchQueryItem[]
  },
  record: async (item: Omit<LogSearchQueryItem, 'id' | 'pinned' | 'remark' | 'createdAt'>) => {
    const res = await api.post('api/v1/logs/central/history', item)
    return res.data as LogSearchQueryItem
  },
  pin: async (id: number, pinned: boolean, remark = '') => {
    await api.put(`api/v1/logs/central/history/${id}/pin`, { pinned, remark })
  },
  remove: async (id: number) => {
    await api.delete(`api/v1/logs/central/history/${id}`)
  },
  clearAll: async () => {
    await api.delete('api/v1/logs/central/history/0?all=1')
  },
}
