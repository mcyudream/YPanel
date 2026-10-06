import api from '../index'

export interface SecuritySettings {
  twoFaEnabled: boolean
  safeEntry: string
  allowedIps: string
  sessionHours: number
}

export interface TwoFASetupResp {
  secret: string
  otpauthUri: string
}

export const securityApi = {
  get: async () => {
    const res = await api.get('api/v1/security/settings', { silent: true })
    return res.data as SecuritySettings
  },
  update: async (data: { safeEntry: string, allowedIps: string, sessionHours: number }) => {
    const res = await api.put('api/v1/security/settings', data)
    return res.data as SecuritySettings
  },
  twoFAStatus: async () => {
    const res = await api.get('api/v1/auth/2fa/status', { silent: true })
    return (res.data as { enabled: boolean }).enabled
  },
  twoFASetup: async () => {
    const res = await api.post('api/v1/auth/2fa/setup')
    return res.data as TwoFASetupResp
  },
  twoFADisable: () => api.post('api/v1/auth/2fa/disable'),
}
