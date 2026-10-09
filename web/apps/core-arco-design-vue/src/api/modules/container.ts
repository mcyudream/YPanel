import api from '../index'
import { makeNodeApi, withNode } from '../dockerNode'

// M55 容器域节点路由：全部调用自动附当前节点（setDockerNode 切换）
const napi = makeNodeApi(api)

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
  // M23 结构化创建表单扩展
  entrypoint?: string[]
  workdir?: string
  tty?: boolean
  labels?: Record<string, string>
  privileged?: boolean
  memoryMB?: number
  cpus?: number
}

// compose 项目内定位容器的 label（agent List 已透出）
export const LabelComposeProject = 'com.docker.compose.project'
export const LabelComposeService = 'com.docker.compose.service'

export default {
  list: async () => {
    const res = await napi.get('api/v1/docker/containers', { silent: true })
    return res.data as ContainerItem[]
  },
  action: (id: string, action: 'start' | 'stop' | 'restart') => napi.post(`api/v1/docker/containers/${id}/${action}`),
  // 日志流地址（follow=1 为持续流；timestamps=0 关闭时间戳前缀）
  logsURL: (id: string, token: string, tail = 500, follow = false, timestamps = true) =>
    withNode(`api/v1/docker/containers/${id}/logs?tail=${tail}&follow=${follow ? 1 : 0}&timestamps=${timestamps ? 1 : 0}&token=${encodeURIComponent(token)}`),
  // 创建容器（返回新容器 id）
  create: async (req: ContainerCreateReq) => {
    const res = await napi.post('api/v1/docker/containers', req, { timeout: 300000 })
    return (res.data as { id: string }).id
  },
  // 编辑保存：删除并按新参数重建同名容器（compose 管理的容器被后端拒绝，返回新容器 id）
  recreate: async (id: string, req: ContainerCreateReq) => {
    const res = await napi.post(`api/v1/docker/containers/${encodeURIComponent(id)}/recreate`, req, { timeout: 300000 })
    return (res.data as { id: string }).id
  },
  // 资源限制/重启策略热更新（docker update，免重建）
  updateResources: (id: string, req: { memoryMB?: number, cpus?: number, restart?: string }) =>
    napi.post(`api/v1/docker/containers/${encodeURIComponent(id)}/update`, req),
  remove: (id: string, force = false, volumes = false) =>
    napi.delete(`api/v1/docker/containers/${encodeURIComponent(id)}?force=${force}&v=${volumes}`),
  // 容器详情（docker inspect 原始 JSON）
  inspect: async (id: string) => {
    const res = await napi.get(`api/v1/docker/containers/${encodeURIComponent(id)}/inspect`, { silent: true })
    return res.data as Record<string, any>
  },
  // 实时资源占用（docker stats 单次采样）
  stats: async (id: string) => {
    const res = await napi.get(`api/v1/docker/containers/${encodeURIComponent(id)}/stats`, { silent: true })
    return res.data as Record<string, any>
  },
  // 容器可写层宿主目录（overlay2 UpperDir；供无法启动的容器经宿主文件通道直读/修复）
  rootfs: async (id: string) => {
    const res = await napi.get(`api/v1/docker/containers/${encodeURIComponent(id)}/rootfs`, { silent: true })
    return (res.data as { path: string }).path
  },
  // 最近一次启动/重启失败原因（内存留存，重启服务或启动成功后清除；无记录返回 null）
  lastOpErr: async (id: string) => {
    const res = await napi.get(`api/v1/docker/containers/${encodeURIComponent(id)}/last-op-err`, { silent: true })
    return (res.data as { action: string, message: string, at: string } | null) || null
  },
  // exec 终端 WS 地址（token 经 query 传递）
  execWSURL: (id: string, token: string, cmd = '/bin/sh') =>
    withNode(`api/v1/docker/containers/${encodeURIComponent(id)}/exec?cmd=${encodeURIComponent(cmd)}&token=${encodeURIComponent(token)}`),
}
