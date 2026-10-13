import api from '../index'

export interface NodeItem {
  id: string
  name: string
  remote: boolean
  addr?: string
  hostname?: string
  os?: string
  arch?: string
  version?: string
  online: boolean
  lastSeenAt?: string
}

/**
 * 节点身份显示名（全站唯一口径）：name（配对唯一名）优先，缺失回落 id。
 * 禁止用 hostname 作身份 label——hostname 可重复（多台机器常同名），见 docs/exp/frontend.md。
 */
export function nodeDisplayName(n?: Pick<NodeItem, 'id' | 'name'> | null): string {
  if (!n) {
    return ''
  }
  return n.name || n.id
}

export default {
  list: async () => {
    const res = await api.get('api/v1/nodes', { silent: true })
    return res.data as NodeItem[]
  },
  pairingCode: async () => {
    const res = await api.post('api/v1/nodes/pairing-code')
    return res.data as { code: string, expireIn: number }
  },
  remove: (id: string) => api.delete(`api/v1/nodes/${id}`),
  // M54 节点 agent 更新：一键更新（任务化，返回 taskId）
  upgradeAgent: async (id: string, source: '' | 'github' | 'gitee' | 'gitlab' = '') => {
    const res = await api.post(`api/v1/nodes/${id}/upgrade-agent`, { source }, { timeout: 600000 })
    return res.data as { taskId: number }
  },
}

// ---- M43：服务器资产 ----
export interface NodeAsset {
  expireDate: string | null
  monthlyPrice: string
  trafficQuotaGB: number
  assetRemark: string
}

export const nodeAssetApi = {
  update: (id: number | string, data: NodeAsset) => api.put(`api/v1/nodes/${id}/asset`, data),
  // M54 节点 agent 更新：检查 + 一键更新（任务化，返回 taskId）
  checkAgentUpdate: async (nodeId: string) => {
    const res = await api.get(`api/v1/nodes/agent-update/check?node=${encodeURIComponent(nodeId)}`, { silent: true })
    return res.data as { nodeId: string, agentVersion: string, latest?: string, updatable: boolean, reason?: string }
  },
  upgradeAgent: async (id: string, source: '' | 'github' | 'gitee' | 'gitlab' = '') => {
    const res = await api.post(`api/v1/nodes/${id}/upgrade-agent`, { source }, { timeout: 600000 })
    return res.data as { taskId: number }
  },
}
