// M53 SSH 管理域 API：sshd 配置（节点维度）+ 密钥统一管理 + 失败登录聚合
import api from '../index'

export interface SshConfig {
  port: number
  passwordAuth: boolean
  pubkeyAuth: boolean
  permitRootLogin: string
}

export interface KeyDeployStatus {
  nodeId: string
  nodeName: string
  deployed: boolean
  reachable: boolean
}

export interface SshKey {
  name: string
  type: string
  fingerprint: string
  comment: string
  publicKey: string
  deployedOn: KeyDeployStatus[]
}

export interface SshAttempt {
  ip: string
  count: number
  lastAt?: string
  banned: boolean
}

export default {
  getConfig: async (nodeId = 'local') => {
    const res = await api.get('api/v1/ssh/config', { silent: true, params: { nodeId } })
    return res.data as SshConfig
  },
  setConfig: async (patch: Partial<SshConfig>, nodeId = 'local') => {
    const res = await api.put('api/v1/ssh/config', patch, { params: { nodeId } })
    return res.data as SshConfig
  },
  listKeys: async (nodeId = 'local') => {
    const res = await api.get('api/v1/ssh/keys', { silent: true, params: { nodeId } })
    return (res.data || []) as SshKey[]
  },
  generateKey: async (data: { nodeId?: string, type?: string, name?: string, comment?: string }) => {
    const res = await api.post('api/v1/ssh/keys/generate', data)
    return res.data as SshKey
  },
  importKey: async (data: { nodeId?: string, name?: string, publicKey: string }) => {
    const res = await api.post('api/v1/ssh/keys/import', data)
    return res.data as SshKey
  },
  deleteKey: async (data: { nodeId?: string, name: string, withPrivate?: boolean }) => {
    await api.post('api/v1/ssh/keys/delete', data)
  },
  deployKey: async (data: { keyNode?: string, name: string, targetNode: string }) => {
    const res = await api.post('api/v1/ssh/keys/deploy', data)
    return res.data as { result: 'deployed' | 'exists' }
  },
  undeployKey: async (data: { keyNode?: string, name: string, targetNode: string }) => {
    const res = await api.post('api/v1/ssh/keys/undeploy', data)
    return res.data as { result: 'revoked' | 'absent' }
  },
  attempts: async (nodeId = 'local', limit = 100) => {
    const res = await api.get('api/v1/ssh/attempts', { silent: true, params: { nodeId, limit } })
    return (res.data || []) as SshAttempt[]
  },
}
