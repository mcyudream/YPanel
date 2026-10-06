import api from '../index'

export interface RuntimeItem {
  id: number
  name: string
  version: string
  origin: 'container' | 'external'
  fcgiAddr: string
  remark: string
  composeProject: string
  running: boolean
  createdAt: string
}

export default {
  list: async () => {
    const res = await api.get('api/v1/runtimes', { silent: true })
    return res.data as RuntimeItem[]
  },
  create: (data: { name: string, version: string }) => api.post('api/v1/runtimes', data),
  attachExternal: (data: { name: string, version?: string, fcgiAddr: string, remark?: string }) =>
    api.post('api/v1/runtimes/external', data, { timeout: 30000 }),
  remove: (id: number) => api.delete(`api/v1/runtimes/${id}`),
  start: (id: number) => api.post(`api/v1/runtimes/${id}/start`),
  stop: (id: number) => api.post(`api/v1/runtimes/${id}/stop`),
}
