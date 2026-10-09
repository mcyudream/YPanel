import api from '../index'

export interface DockerUsageItem {
  name: string
  size: number
  sub?: string
}

export interface DockerUsage {
  imagesTotalSize: number
  imagesCount: number
  containersRwSize: number
  containersCount: number
  volumesTotalSize: number
  volumesCount: number
  buildCacheSize: number
  buildCacheCount: number
  networksCount: number
  hostPortsCount: number
  containerItems: DockerUsageItem[]
  imageItems: DockerUsageItem[]
  volumeItems: DockerUsageItem[]
  collectedAt: string
}

export default {
  // system df 由 daemon 实算 size，镜像多时较慢：懒加载、勿进高频轮询
  usage: async () => {
    const res = await api.get('api/v1/docker/usage', { silent: true })
    return res.data as DockerUsage
  },
  buildCachePrune: async () => {
    const res = await api.post('api/v1/docker/buildcache/prune')
    return res.data as { freed: number }
  },
}
