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

export const procApi = {
  processes: async () => {
    const res = await api.get('api/v1/processes', { silent: true })
    return res.data as ProcessItem[]
  },
  kill: (pid: number) => api.post('api/v1/processes/kill', { pid }),
  services: async () => {
    const res = await api.get('api/v1/services', { silent: true })
    return res.data as ServiceItem[]
  },
  serviceAction: (name: string, action: 'start' | 'stop' | 'restart') =>
    api.post(`api/v1/services/${encodeURIComponent(name)}/${action}`),
}
