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
}
