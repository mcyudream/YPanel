import api from '../index'

export interface ComposeServiceState {
  name: string
  image: string
  state: string
}

export interface ComposeProject {
  name: string
  dir: string
  managed: boolean
  running: number
  total: number
  services: ComposeServiceState[]
}

export default {
  list: async () => {
    const res = await api.get('api/v1/compose/projects', { silent: true })
    return res.data as ComposeProject[]
  },
  config: async (name: string, dir?: string) => {
    const res = await api.get(`api/v1/compose/config?name=${encodeURIComponent(name)}&dir=${encodeURIComponent(dir || '')}`)
    return res.data as { name: string, dir: string, file: string, content: string }
  },
  write: (name: string, content: string) => api.post('api/v1/compose/config', { name, content }),
  up: async (name: string, dir?: string) => {
    const res = await api.post('api/v1/compose/up', { name, dir })
    return (res.data as { output?: string }).output || ''
  },
  down: async (name: string, dir?: string) => {
    const res = await api.post('api/v1/compose/down', { name, dir })
    return (res.data as { output?: string }).output || ''
  },
  logsURL: (name: string, dir: string | undefined, token: string, tail = 500, follow = false, service = '') =>
    `api/v1/compose/logs?name=${encodeURIComponent(name)}&dir=${encodeURIComponent(dir || '')}&tail=${tail}&follow=${follow ? 1 : 0}&service=${encodeURIComponent(service)}&token=${encodeURIComponent(token)}`,
}
