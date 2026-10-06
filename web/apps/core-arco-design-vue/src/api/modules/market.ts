import api from '../index'

export interface MarketAppParam {
  key: string
  label: string
  default: string
  isPassword?: boolean
}

export interface MarketApp {
  id: string
  name: string
  category: string
  description: string
  params: MarketAppParam[]
  source: string
}

export default {
  list: async () => {
    const res = await api.get('api/v1/market/apps', { silent: true })
    return res.data as MarketApp[]
  },
  installed: async () => {
    const res = await api.get('api/v1/market/installed', { silent: true })
    return res.data as { name: string, running: number, total: number }[]
  },
  install: async (appId: string, params: Record<string, string>) => {
    const res = await api.post('api/v1/market/install', { appId, params })
    return (res.data as { output?: string }).output || ''
  },
}
