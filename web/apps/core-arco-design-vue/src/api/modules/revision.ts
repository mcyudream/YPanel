import api from '../index'

export interface ConfigRevisionMeta {
  id: number
  scope: string
  trigger: string // save / rollback
  note: string
  author: string
  size: number
  createdAt: string
}

export interface ConfigRevision extends ConfigRevisionMeta {
  content: string
}

// 受管配置版本快照（M23）：compose 项目文件、daemon.json 等面板写盘前自动快照。
export const revisionApi = {
  list: async (node: string, path: string) => {
    const res = await api.get(`api/v1/config-revisions?node=${encodeURIComponent(node)}&path=${encodeURIComponent(path)}`, { silent: true })
    return res.data as ConfigRevisionMeta[]
  },
  get: async (id: number) => {
    const res = await api.get(`api/v1/config-revisions/${id}`, { silent: true })
    return res.data as ConfigRevision
  },
  // 回滚：服务端写盘并留 rollback 快照
  restore: async (id: number) => {
    const res = await api.post(`api/v1/config-revisions/${id}/restore`)
    return res.data as ConfigRevision
  },
}
