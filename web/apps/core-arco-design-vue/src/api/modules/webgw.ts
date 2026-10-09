// M51 内网浏览器：代理会话管理（gw 网关的服务端配对，第二端口 /s/{sid}/*）。
import api from '../index'

export interface WebGwSessionInfo {
  sid: string
  target: string
  expiresAt: string
}

export default {
  create: async (url: string) => {
    const res = await api.post('api/v1/webgw/session', { url })
    return res.data as WebGwSessionInfo
  },
  get: async (sid: string) => {
    const res = await api.get(`api/v1/webgw/session/${sid}`, { silent: true })
    return res.data as WebGwSessionInfo
  },
  remove: (sid: string) => api.delete(`api/v1/webgw/session/${sid}`),
}

// ---- M50：节点目标发现 ----
export interface WebGwTarget {
  name: string
  container: string
  url: string
  hostPort: string
  reachable: boolean
}

export const webgwTargetsApi = {
  list: async (node: string) => {
    const res = await api.get(`api/v1/webgw/targets?node=${encodeURIComponent(node)}`, { silent: true })
    return res.data as { node: string, ip: string, targets: WebGwTarget[] }
  },
}
