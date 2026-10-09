// M53 FTP（vsftpd）管理域 API
import api from '../index'

export interface FtpStatus {
  installed: boolean
  running: boolean
  port: number
  pasvMin: number
  pasvMax: number
}

export default {
  status: async () => {
    const res = await api.get('api/v1/ftp/status', { silent: true })
    return res.data as FtpStatus
  },
  install: async () => api.post('api/v1/ftp/install'),
  power: (action: 'start' | 'stop' | 'restart') => api.post('api/v1/ftp/power', { action }),
  setPort: (port: number, pasvMin: number, pasvMax: number) =>
    api.post('api/v1/ftp/port', { port, pasvMin, pasvMax }),
  getConfig: async () => {
    const res = await api.get('api/v1/ftp/config')
    return res.data.content as string
  },
  putConfig: (content: string) => api.put('api/v1/ftp/config', { content }),
}
