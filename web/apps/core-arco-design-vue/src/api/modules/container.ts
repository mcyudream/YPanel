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

export interface ContainerCreateReq {
  name: string
  image: string
  cmd?: string[]
  env?: string[]
  ports?: { host: string, container: string, proto: string }[]
  mounts?: string[]
  restart?: string
  network?: string
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
  // 创建容器（返回新容器 id）
  create: async (req: ContainerCreateReq) => {
    const res = await api.post('api/v1/docker/containers', req, { timeout: 300000 })
    return (res.data as { id: string }).id
  },
  remove: (id: string, force = false) =>
    api.delete(`api/v1/docker/containers/${encodeURIComponent(id)}?force=${force}`),
  // 容器详情（docker inspect 原始 JSON）
  inspect: async (id: string) => {
    const res = await api.get(`api/v1/docker/containers/${encodeURIComponent(id)}/inspect`, { silent: true })
    return res.data as Record<string, any>
  },
  // 实时资源占用（CPU%/内存）
  stats: async (id: string) => {
    const res = await api.get(`api/v1/docker/containers/${encodeURIComponent(id)}/stats`, { silent: true })
    return res.data as Record<string, any>
  },
  // exec 终端 WS 地址（token 经 query 传递）
  execWSURL: (id: string, token: string, cmd = '/bin/sh') =>
    `api/v1/docker/containers/${encodeURIComponent(id)}/exec?cmd=${encodeURIComponent(cmd)}&token=${encodeURIComponent(token)}`,
}
