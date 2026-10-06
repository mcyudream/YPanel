import api from '../index'

export interface RuntimeItem {
  id: number
  name: string
  version: string
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
  remove: (id: number) => api.delete(`api/v1/runtimes/${id}`),
  start: (id: number) => api.post(`api/v1/runtimes/${id}/start`),
  stop: (id: number) => api.post(`api/v1/runtimes/${id}/stop`),
}
