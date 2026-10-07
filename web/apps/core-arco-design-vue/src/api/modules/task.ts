import api from '../index'

export interface AppTask {
  id: number
  type: string // store-install / store-uninstall / image-pull
  title: string
  ref: string
  status: 'running' | 'success' | 'failed'
  logText: string
  error: string
  createdAt: string
  updatedAt: string
}

export const taskApi = {
  list: async (q: { type?: string, status?: string, page?: number, pageSize?: number } = {}) => {
    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) {
      if (v !== undefined && v !== '') {
        params.set(k, String(v))
      }
    }
    const res = await api.get(`api/v1/tasks?${params.toString()}`, { silent: true })
    return res.data as { total: number, page: number, pageSize: number, items: Omit<AppTask, 'logText' | 'error'>[] }
  },
  get: async (id: number) => {
    const res = await api.get(`api/v1/tasks/${id}`, { silent: true })
    return res.data as AppTask
  },
  remove: (id: number) => api.delete(`api/v1/tasks/${id}`),
  clear: async () => {
    const res = await api.delete('api/v1/tasks')
    return res.data as { cleared: number }
  },
}
