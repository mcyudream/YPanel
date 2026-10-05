import api from '../index'

export default {
  get: async () => {
    const res = await api.get('api/v1/settings')
    return res.data as Record<string, string>
  },
  put: async (data: Record<string, string>) => {
    const res = await api.put('api/v1/settings', data)
    return res.data as Record<string, string>
  },
}
