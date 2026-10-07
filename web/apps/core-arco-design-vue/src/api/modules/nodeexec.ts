import api from '../index'

export interface NodeExecResult {
  nodeId: string
  name: string
  ok: boolean
  output: string
  error?: string
}

export const nodeExecApi = {
  exec: async (nodeIds: string[], command: string, timeoutSecs = 60) => {
    const res = await api.post('api/v1/nodes/exec', { nodeIds, command, timeoutSecs }, { timeout: 300000 })
    return res.data as NodeExecResult[]
  },
}

export interface ProcessItem {
  pid: number
  name: string
  cpu: number
  mem: number
  memRss: number
  user: string
  cmdline: string
}

export interface ServiceItem {
  name: string
  load: string
  active: string
  desc: string
}

// GET 类的 node 查询串（无既有 query 时以 ?1=1 占位）
function nodeQS(node?: string, extra?: Record<string, string>) {
  const parts: string[] = ['1=1']
  if (node && node !== 'local') {
    parts.push(`node=${encodeURIComponent(node)}`)
  }
  for (const [k, v] of Object.entries(extra ?? {})) {
    if (v) {
      parts.push(`${k}=${encodeURIComponent(v)}`)
    }
  }
  return `?${parts.join('&')}`
}

// POST 类的 node 查询串（?node=xxx 或空串）
function nodeQ2(node?: string) {
  return node && node !== 'local' ? `?node=${encodeURIComponent(node)}` : ''
}

export type ProcessSort = 'cpu' | 'mem' | 'rss' | 'pid' | 'name'

export const procApi = {
  processes: async (opts?: { node?: string, sort?: ProcessSort, order?: 'asc' | 'desc', limit?: number }) => {
    const qs = nodeQS(opts?.node, {
      sort: opts?.sort ?? '',
      order: opts?.order ?? '',
      limit: opts?.limit ? String(opts.limit) : '',
    })
    const res = await api.get(`api/v1/processes${qs}`, { silent: true })
    return res.data as ProcessItem[]
  },
  kill: (pid: number, node?: string) => api.post(`api/v1/processes/kill${nodeQ2(node)}`, { pid }),
  services: async (node?: string) => {
    const res = await api.get(`api/v1/services${nodeQS(node)}`, { silent: true })
    return res.data as ServiceItem[]
  },
  serviceAction: (name: string, action: 'start' | 'stop' | 'restart', node?: string) =>
    api.post(`api/v1/services/${encodeURIComponent(name)}/${action}${nodeQ2(node)}`),
}
