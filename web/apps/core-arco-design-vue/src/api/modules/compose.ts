import api from '../index'
import { makeNodeApi, withNode } from '../dockerNode'

// M55 容器域节点路由：全部调用自动附当前节点（setDockerNode 切换）
const napi = makeNodeApi(api)

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

// M26 P1：项目服务拓扑（depends_on 结构 + 容器实时状态）
export interface ComposeTopologyNode {
  name: string
  image: string
  state?: string
  health?: string
  ports?: string[]
}

export interface ComposeTopologyEdge {
  from: string
  to: string
  condition?: string
}

export interface ComposeTopology {
  name: string
  dir: string
  nodes: ComposeTopologyNode[]
  edges: ComposeTopologyEdge[]
}

// M26 P2：源码构建生成 compose 项目
export interface Src2Suggestion {
  dir: string
  marker: string
  lang: string
  version?: string
  startCmd?: string
  buildCmd?: string
  pkgMgr?: string
  packaging?: string
  modules?: string[]
  port: number
  hostPort: number
  service: string
}

export interface Src2PreviewResp {
  commit: string
  suggestions: Src2Suggestion[]
}

export interface Src2ServiceSpec {
  name: string
  dir: string
  lang: string
  version?: string
  hostPort: number
  containerPort: number
  startCmd?: string
  module?: string
  buildCmd?: string
  distDir?: string
}

export interface Src2BuildReq {
  name: string
  gitUrl: string
  branch?: string
  credentialId?: number
  services: Src2ServiceSpec[]
}

export default {
  list: async () => {
    const res = await napi.get('api/v1/compose/projects', { silent: true })
    return res.data as ComposeProject[]
  },
  config: async (name: string, dir?: string) => {
    const res = await napi.get(`api/v1/compose/config?name=${encodeURIComponent(name)}&dir=${encodeURIComponent(dir || '')}`)
    return res.data as { name: string, dir: string, file: string, content: string }
  },
  write: (name: string, content: string) => napi.post('api/v1/compose/config', { name, content }),
  up: async (name: string, dir?: string) => {
    const res = await napi.post('api/v1/compose/up', { name, dir })
    return (res.data as { output?: string }).output || ''
  },
  down: async (name: string, dir?: string) => {
    const res = await napi.post('api/v1/compose/down', { name, dir })
    return (res.data as { output?: string }).output || ''
  },
  // 单服务操作：start / stop / restart / pull / up（按当前编排定义重建该服务）
  serviceAction: async (name: string, service: string, action: 'start' | 'stop' | 'restart' | 'pull' | 'up', dir?: string) => {
    const res = await napi.post('api/v1/compose/service-action', { name, service, action, dir }, { timeout: 300000 })
    return (res.data as { output?: string }).output || ''
  },
  // 项目服务拓扑（M26 P1）
  topology: async (name: string, dir?: string) => {
    const res = await napi.get(`api/v1/compose/topology?name=${encodeURIComponent(name)}&dir=${encodeURIComponent(dir || '')}`)
    return res.data as ComposeTopology
  },
  // M26 P2：源码构建（预检 + 创建构建任务；产物为普通 compose 项目）
  // 源码预检 SSE 流地址（fetch + Authorization 头流式读取；data 行 JSON：log/done/error）
  src2PreviewStreamURL: (gitUrl: string, branch?: string, credentialId?: number) =>
    withNode(`api/v1/compose/src2compose/preview/stream?gitUrl=${encodeURIComponent(gitUrl)}&branch=${encodeURIComponent(branch || '')}&credentialId=${credentialId || 0}`),
  src2Create: async (data: Src2BuildReq) => {
    const res = await napi.post('api/v1/compose/src2compose', data, { timeout: 30000 })
    return res.data as { taskId: number }
  },
  // 删除托管项目（down + 移除编排目录，含其下全部数据，调用方须先确认）
  deleteProject: (name: string) => napi.delete(`api/v1/compose/projects/${encodeURIComponent(name)}`),
  logsURL: (name: string, dir: string | undefined, token: string, tail = 500, follow = false, service = '') =>
    `api/v1/compose/logs?name=${encodeURIComponent(name)}&dir=${encodeURIComponent(dir || '')}&tail=${tail}&follow=${follow ? 1 : 0}&service=${encodeURIComponent(service)}&token=${encodeURIComponent(token)}`,
}
