import api from '../index'

export interface UserInfo {
  id: number
  username: string
  nickname: string
  role: 'admin' | 'user'
  lastLoginAt?: string | null
}

export interface LoginLogItem {
  id: number
  username: string
  ip: string
  userAgent: string
  success: boolean
  message: string
  createdAt: string
}

export interface PageResp<T> {
  total: number
  items: T[]
}

export default {
  list: async (page = 1, pageSize = 20) => {
    const res = await api.get(`api/v1/users?page=${page}&pageSize=${pageSize}`)
    return res.data as PageResp<UserInfo>
  },
  create: (data: { username: string, password: string, nickname?: string, role: 'admin' | 'user' }) =>
    api.post('api/v1/users', data),
  update: (id: number, data: { password?: string, nickname?: string, role?: 'admin' | 'user', status?: 0 | 1 }) =>
    api.put(`api/v1/users/${id}`, data),
  remove: (id: number) => api.delete(`api/v1/users/${id}`),
  loginLogs: async (page = 1, pageSize = 20) => {
    const res = await api.get(`api/v1/audit/logins?page=${page}&pageSize=${pageSize}`)
    return res.data as PageResp<LoginLogItem>
  },
}
