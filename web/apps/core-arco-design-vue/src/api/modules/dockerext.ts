import api from '../index'

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
    const res = await api.get('api/v1/docker/images', { silent: true })
    return res.data as DockerImage[]
  },
  pull: async (ref: string) => {
    const res = await api.post('api/v1/docker/images/pull', { ref }, { timeout: 600000 })
    return (res.data as { output: string }).output
  },
  removeImage: (id: string, force = false) =>
    api.delete(`api/v1/docker/images/${encodeURIComponent(id)}?force=${force}`),
  pruneImages: async () => {
    const res = await api.post('api/v1/docker/images/prune')
    return (res.data as { output: string }).output
  },
  networks: async () => {
    const res = await api.get('api/v1/docker/networks', { silent: true })
    return res.data as DockerNetwork[]
  },
  createNetwork: (name: string, driver = 'bridge') => api.post('api/v1/docker/networks', { name, driver }),
  removeNetwork: (name: string) => api.delete(`api/v1/docker/networks/${encodeURIComponent(name)}`),
  volumes: async () => {
    const res = await api.get('api/v1/docker/volumes', { silent: true })
    return res.data as DockerVolume[]
  },
  createVolume: (name: string) => api.post('api/v1/docker/volumes', { name }),
  removeVolume: (name: string) => api.delete(`api/v1/docker/volumes/${encodeURIComponent(name)}`),
  pruneVolumes: async () => {
    const res = await api.post('api/v1/docker/volumes/prune')
    return (res.data as { output: string }).output
  },
  pruneContainers: async () => {
    const res = await api.post('api/v1/docker/containers/prune')
    return (res.data as { output: string }).output
  },
  daemonConfig: async () => {
    const res = await api.get('api/v1/docker/daemon-config', { silent: true })
    return (res.data as { content: string }).content
  },
  updateDaemonConfig: (content: string) => api.put('api/v1/docker/daemon-config', { content }, { timeout: 300000 }),
}
