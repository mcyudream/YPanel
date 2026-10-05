import api from '../index'

export interface PortBinding {
  hostIp: string
  hostPort: string
  containerPort: string
  proto: string
}

export interface ContainerItem {
  id: string
  name: string
  image: string
  state: 'running' | 'exited' | 'paused' | 'created' | 'restarting' | 'dead'
  status: string
  command: string
  created: string
  ports: PortBinding[]
  labels?: Record<string, string>
}

export default {
  list: async () => {
    const res = await api.get('api/v1/docker/containers', { silent: true })
    return res.data as ContainerItem[]
  },
  action: (id: string, action: 'start' | 'stop' | 'restart') => api.post(`api/v1/docker/containers/${id}/${action}`),
  // 日志流地址（follow=1 为持续流）
  logsURL: (id: string, token: string, tail = 500, follow = false) =>
    `api/v1/docker/containers/${id}/logs?tail=${tail}&follow=${follow ? 1 : 0}&token=${encodeURIComponent(token)}`,
}
