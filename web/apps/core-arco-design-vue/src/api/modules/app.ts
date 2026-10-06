import api from '../index'

// 后端契约类型（与 shared/dto 对齐）
export interface YPanelUserInfo {
  id: number
  username: string
  nickname: string
  role: 'admin' | 'user'
  lastLoginAt?: string | null
}

export interface YPanelLoginResp {
  token: string
  expireAt: string
  user: YPanelUserInfo
}

export default {
  // 路由表（YPanel 固定使用 frontend 模式，此存根仅为类型完备）
  routeList: async () => ({ code: 0, message: 'ok', data: [] }),

  // 登录：字段映射为 fa 账号 store 期望的 {token, account, avatar}
  login: async (data: {
    account: string
    password: string
    otpCode?: string
  }) => {
    const entry = localStorage.getItem('login_entry') ?? ''
    const res = await api.post('api/v1/auth/login', {
      username: data.account,
      password: data.password,
      otpCode: data.otpCode ?? '',
    }, entry ? { headers: { 'X-Safe-Entry': entry } } : undefined)
    return {
      ...res,
      data: {
        token: res.data.token,
        account: res.data.user.username,
        avatar: '',
        role: res.data.user.role,
      },
    }
  },

  // 权限：由角色推导（M0 权限模型：admin 全量 / user 只读业务）
  permission: async () => {
    const res = await api.get('api/v1/auth/me')
    return {
      ...res,
      data: {
        permissions: res.data.role === 'admin' ? ['admin'] : ['user'],
      },
    }
  },

  // 修改密码
  passwordEdit: async (data: {
    password: string
    newPassword: string
  }) => {
    await api.put('api/v1/auth/password', {
      oldPassword: data.password,
      newPassword: data.newPassword,
    })
  },
}
