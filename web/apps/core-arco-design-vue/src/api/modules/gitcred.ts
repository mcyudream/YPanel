import api from '../index'

// M26 P2：私有仓库凭据库（secret 只写不读，列表不回传）
export interface GitCredential {
  id: number
  name: string
  type: 'token' | 'ssh'
  host: string
  username: string
  remark: string
  createdAt: string
}

export default {
  list: async () => {
    const res = await api.get('api/v1/git/credentials', { silent: true })
    return res.data as GitCredential[]
  },
  create: (data: { name: string, type: 'token' | 'ssh', host: string, username?: string, secret: string, remark?: string }) =>
    api.post('api/v1/git/credentials', data),
  update: (id: number, data: { name?: string, host?: string, username?: string, secret?: string, remark?: string }) =>
    api.put(`api/v1/git/credentials/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/git/credentials/${id}`),
  // 向导预览：给定 git 地址将命中的凭据（null = 无匹配，匿名克隆）
  match: async (url: string) => {
    const res = await api.get(`api/v1/git/credentials/match?url=${encodeURIComponent(url)}`, { silent: true })
    return res.data as { id: number, name: string, type: string, host: string } | null
  },
}
