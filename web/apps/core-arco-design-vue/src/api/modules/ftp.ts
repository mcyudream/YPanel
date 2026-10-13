// M53 FTP（vsftpd）管理域 API
import api from '../index'
import { makeNodeIdApi } from '../hostNode'

const hnapi = makeNodeIdApi(api)

export interface FtpStatus {
  installed: boolean
  running: boolean
  port: number
  pasvMin: number
  pasvMax: number
}

export default {
  status: async () => {
    const res = await hnapi.get('api/v1/ftp/status', { silent: true })
    return res.data as FtpStatus
  },
  // 安装（任务化：秒回 taskId，apt 输出实时进任务日志）
  install: async (nodeId?: string) => {
    const res = await hnapi.post(`api/v1/ftp/install${nodeId ? `?nodeId=${encodeURIComponent(nodeId)}` : ''}`)
    return res.data as { taskId: number }
  },
  power: (action: 'start' | 'stop' | 'restart') => hnapi.post('api/v1/ftp/power', { action }),
  setPort: (port: number, pasvMin: number, pasvMax: number) =>
    hnapi.post('api/v1/ftp/port', { port, pasvMin, pasvMax }),
  getConfig: async () => {
    const res = await hnapi.get('api/v1/ftp/config')
    return res.data.content as string
  },
  putConfig: (content: string) => hnapi.put('api/v1/ftp/config', { content }),
  // ---------- 账号管理（M69） ----------
  id: number
  nodeId: string
  name: string
  directory: string
  readOnly: boolean
  enabled: boolean
  remark: string
  systemUserExists?: boolean
  createdAt: string
}

export default {
  // ...（status/install/power/setPort/getConfig/putConfig 见上）
  accounts: {
    list: async (nodeId?: string) => {
      const res = await hnapi.get(`api/v1/ftp/accounts${nodeId ? `?nodeId=${encodeURIComponent(nodeId)}` : ''}`, { silent: true })
      return (res.data?.items ?? res.data) as FtpAccount[]
    },
    create: (data: { nodeId?: string, name: string, directory: string, password: string, readOnly?: boolean, remark?: string }) =>
      hnapi.post('api/v1/ftp/accounts', { nodeId, ...data }),
    update: (id: number, data: { directory?: string, password?: string, readOnly?: boolean, enabled?: boolean, remark?: string }) =>
      hnapi.put(`api/v1/ftp/accounts/${id}`, data),
    remove: (id: number, purgeDir?: boolean) =>
      hnapi.delete(`api/v1/ftp/accounts/${id}${purgeDir !== undefined ? `?purgeDir=${purgeDir}` : ''}`),
    test: (id: number, password: string) => hnapi.post(`api/v1/ftp/accounts/${id}/test`, { password }),
  },
}
}