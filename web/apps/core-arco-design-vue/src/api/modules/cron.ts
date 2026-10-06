import api from '../index'

export interface CronTask {
  id: number
  name: string
  cron: string
  command: string
  enabled: boolean
  timeoutSecs: number
  lastRunAt?: string | null
  lastSuccess?: boolean | null
  createdAt: string
}

export interface CronTaskLog {
  id: number
  taskId: number
  taskName: string
  trigger: string
  startAt: string
  endAt?: string | null
  durationMs: number
  success: boolean
  output: string
}

export interface PageResp<T> {
  total: number
  items: T[]
}

export default {
  list: async (page = 1, pageSize = 50) => {
    const res = await api.get(`api/v1/cron/tasks?page=${page}&pageSize=${pageSize}`)
    return res.data as PageResp<CronTask>
  },
  create: (data: { name: string, cron: string, command: string, timeoutSecs?: number }) =>
    api.post('api/v1/cron/tasks', data),
  update: (id: number, data: { name?: string, cron?: string, command?: string, timeoutSecs?: number, enabled?: boolean }) =>
    api.put(`api/v1/cron/tasks/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/cron/tasks/${id}`),
  run: (id: number) => api.post(`api/v1/cron/tasks/${id}/run`),
  logs: async (taskId?: number, page = 1, pageSize = 20) => {
    const q = `?page=${page}&pageSize=${pageSize}${taskId ? `&taskId=${taskId}` : ''}`
    const res = await api.get(`api/v1/cron/logs${q}`)
    return res.data as PageResp<CronTaskLog>
  },
}
