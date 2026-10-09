import api from '../index'

// 监控探针（M29）—— /api/v1/probes*

export interface MonitorProbe {
  id: number
  name: string
  targetType: 'site' | 'container' | 'http' | 'tcp'
  targetRef: string
  method: string
  expectStatus: string
  keyword: string
  intervalSec: number
  timeoutSec: number
  retries: number
  webhookUrl: string
  webhookType: string
  enabled: boolean
  status: 'up' | 'down' | 'paused' | ''
  lastCheckedAt: string | null
  lastDownAt: string | null
  lastError: string
  createdAt: string
}

export interface MonitorProbeInput {
  name: string
  targetType: MonitorProbe['targetType']
  targetRef: string
  method?: string
  expectStatus?: string
  keyword?: string
  intervalSec?: number
  timeoutSec?: number
  retries?: number
  webhookUrl?: string
  webhookType?: string
}

export interface SiteOption { id: number, name: string, domain: string, certDomain?: string }
export interface ContainerOption { id: string, name: string, state: string, image: string }

export const probeApi = {
  list: async () => {
    const res = await api.get('api/v1/probes', { silent: true })
    return res.data as MonitorProbe[]
  },
  create: async (data: MonitorProbeInput) => {
    const res = await api.post('api/v1/probes', data)
    return res.data as MonitorProbe
  },
  update: async (id: number, data: MonitorProbeInput) => {
    const res = await api.put(`api/v1/probes/${id}`, data)
    return res.data as MonitorProbe
  },
  remove: async (id: number) => {
    await api.delete(`api/v1/probes/${id}`)
  },
  setEnabled: async (id: number, enabled: boolean) => {
    await api.post(`api/v1/probes/${id}/${enabled ? 'enable' : 'disable'}`)
  },
  test: async (data: MonitorProbeInput) => {
    const res = await api.post('api/v1/probes/test', data, { timeout: 120000 })
    return res.data as { ok: boolean, detail: string, checkedAt: string }
  },
}
