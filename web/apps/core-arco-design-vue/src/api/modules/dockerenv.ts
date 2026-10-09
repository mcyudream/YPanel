import api from '../index'

// ---- 多 Docker 环境（M35） ----
export interface DockerEnvironment {
  id: number
  name: string
  type: 'local' | 'tcp'
  endpoint: string
  tlsEnable: boolean
  remark: string
  hasTLSKey: boolean
  builtin?: boolean
  createdAt?: string
}

export interface EnvContainer {
  id: string
  name: string
  image: string
  state: string
  status: string
  ports: string[]
}

export interface EnvImage {
  id: string
  tags: string[]
  sizeMb: number
  created: number
}

export interface EnvInput {
  name: string
  type: 'local' | 'tcp'
  endpoint?: string
  tlsEnable?: boolean
  tlsCa?: string
  tlsCert?: string
  tlsKey?: string // 留空 = 保留原值
  remark?: string
}

export const dockerEnvApi = {
  list: async () => {
    const res = await api.get('api/v1/docker/environments', { silent: true })
    return res.data as DockerEnvironment[]
  },
  create: (data: EnvInput) => api.post('api/v1/docker/environments', data),
  update: (id: number, data: EnvInput) => api.put(`api/v1/docker/environments/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/docker/environments/${id}`),
  test: async (id: number) => {
    const res = await api.post(`api/v1/docker/environments/${id}/test`)
    return res.data as { ok: boolean, latencyMs: number, apiVersion?: string }
  },
  containers: async (id: number) => {
    const res = await api.get(`api/v1/docker/environments/${id}/containers`, { silent: true })
    return res.data as EnvContainer[]
  },
  images: async (id: number) => {
    const res = await api.get(`api/v1/docker/environments/${id}/images`, { silent: true })
    return res.data as EnvImage[]
  },
  containerAction: (id: number, name: string, action: string) =>
    api.post(`api/v1/docker/environments/${id}/containers/${encodeURIComponent(name)}/${action}`),
  imageRemove: (id: number, image: string, force = false) =>
    api.delete(`api/v1/docker/environments/${id}/images/${encodeURIComponent(image)}?force=${force}`),
}

// ---- 镜像生命周期（M35） ----
export interface ImageUpdateCheck {
  image: string
  ok: boolean
  error?: string
  localDigests?: string[]
  remoteDigest?: string
  hasUpdate?: boolean
}

export const dockerImgApi = {
  build: async (data: { contextDir: string, dockerfile?: string, tag: string }) => {
    const res = await api.post('api/v1/docker/images/build', data)
    return res.data as { tag?: string, logFile?: string, output?: string }
  },
  save: async (images: string[], name?: string) => {
    const res = await api.post('api/v1/docker/images/save', { images, name })
    return res.data as { file: string, name: string }
  },
  load: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return api.post('api/v1/docker/images/load', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
  tag: (src: string, dst: string) => api.post('api/v1/docker/images/tag', { src, dst }),
  checkUpdates: async (images: string[]) => {
    const res = await api.post('api/v1/docker/images/check-updates', { images })
    return res.data as ImageUpdateCheck[]
  },
  commit: (container: string, image: string) => api.post(`api/v1/docker/containers/${encodeURIComponent(container)}/commit`, { image }),
  containerBackup: async (container: string) => {
    const res = await api.post(`api/v1/docker/containers/${encodeURIComponent(container)}/backup`)
    return res.data as { files: string[], volumes: string[], dir: string }
  },
  swarmStatus: async () => {
    const res = await api.get('api/v1/docker/swarm-status', { silent: true })
    return res.data as { state: string, controlAvailable: boolean, clusterId?: string }
  },
}
