import api from '../index'
import { makeNodeApi } from '../dockerNode'

// M55 容器域节点路由：全部调用自动附当前节点（setDockerNode 切换）
const napi = makeNodeApi(api)

export interface DockerImage {
  id: string
  tags: string[]
  sizeMb: number
  createdAt: number
}

export interface DockerNetwork {
  id: string
  name: string
  driver: string
  scope: string
  subnet: string
}

export interface DockerVolume {
  name: string
  driver: string
  mountpoint: string
}

export const dockerExtApi = {
  images: async () => {
    const res = await napi.get('api/v1/docker/images', { silent: true })
    return res.data as DockerImage[]
  },
  pull: async (ref: string) => {
    const res = await napi.post('api/v1/docker/images/pull', { ref }, { timeout: 60000 })
    return res.data as { taskId: number }
  },
  removeImage: (id: string, force = false) =>
    napi.delete(`api/v1/docker/images/${encodeURIComponent(id)}?force=${force}`),
  pruneImages: async () => {
    const res = await napi.post('api/v1/docker/images/prune')
    return (res.data as { output: string }).output
  },
  networks: async () => {
    const res = await napi.get('api/v1/docker/networks', { silent: true })
    return res.data as DockerNetwork[]
  },
  createNetwork: (name: string, driver = 'bridge') => napi.post('api/v1/docker/networks', { name, driver }),
  removeNetwork: (name: string) => napi.delete(`api/v1/docker/networks/${encodeURIComponent(name)}`),
  volumes: async () => {
    const res = await napi.get('api/v1/docker/volumes', { silent: true })
    return res.data as DockerVolume[]
  },
  createVolume: (name: string) => napi.post('api/v1/docker/volumes', { name }),
  removeVolume: (name: string) => napi.delete(`api/v1/docker/volumes/${encodeURIComponent(name)}`),
  pruneVolumes: async () => {
    const res = await napi.post('api/v1/docker/volumes/prune')
    return (res.data as { output: string }).output
  },
  pruneContainers: async () => {
    const res = await napi.post('api/v1/docker/containers/prune')
    return (res.data as { output: string }).output
  },
  daemonConfig: async () => {
    const res = await api.get('api/v1/docker/daemon-config', { silent: true })
    return (res.data as { content: string }).content
  },
  updateDaemonConfig: (content: string) => api.put('api/v1/docker/daemon-config', { content }, { timeout: 300000 }),
  // ---- 镜像仓库（~/.docker/config.json auths）----
  registries: async () => {
    const res = await api.get('api/v1/docker/registry', { silent: true })
    return res.data as { registry: string, username: string, auth?: string }[]
  },
  setRegistry: (registry: string, username: string, password: string) =>
    api.put('api/v1/docker/registry', { registry, username, password }),
  removeRegistry: (registry: string) => api.delete(`api/v1/docker/registry?registry=${encodeURIComponent(registry)}`),
}
