import api from '../index'

// M52 Docker/Compose 一键安装 + 镜像加速器

export interface DockerPrecheck {
  nodeId: string
  supported: boolean
  reason?: string
  family: 'debian' | 'rhel' | 'unknown'
  distro: string
  version: string
  codename?: string
  prettyName?: string
  arch?: string
  systemd: boolean
  dockerInstalled: boolean
  dockerVersion?: string
  dockerRunning: boolean
  composeInstalled: boolean
  composeVersion?: string
  action: 'full' | 'compose-only' | 'none'
  note?: string
}

export interface InstallStep {
  name: string
  cmd: string
  timeout: number
}

export interface InstallDryRunResp {
  dryRun: true
  precheck: DockerPrecheck
  steps: InstallStep[]
  syntaxOk: boolean
  syntaxOut: string
}

export interface SourceTest {
  source: string
  url: string
  status: string
  ok: boolean
  detail?: string
}

export const dockerInstallApi = {
  // 预检（source 非空附带连通性测试）
  precheck: async (source = '') => {
    const res = await api.post('api/v1/docker/install/precheck', source ? { source } : {}, { silent: true })
    return res.data as { precheck: DockerPrecheck, sourceTest?: SourceTest }
  },
  // dryRun=true 只生成步骤 + 语法校验；否则创建安装任务
  install: async (req: { source: string, dryRun?: boolean, configureMirror?: boolean, mirrors?: string[] }) => {
    const res = await api.post('api/v1/docker/install', req)
    return res.data as { taskId: number } & Partial<InstallDryRunResp>
  },
  // 当前镜像加速器（daemon.json registry-mirrors）
  mirrors: async () => {
    const res = await api.get('api/v1/docker/registry-mirrors', { silent: true })
    return res.data.mirrors as string[]
  },
  // 保存加速器：创建应用任务（合并写 daemon.json + 重启 docker；agent 侧失败自动回滚），
  // 返回 taskId 由调用方轮询任务确认结果（重启可达分钟级，不再同步等待）
  setMirrors: async (mirrors: string[]) => {
    const res = await api.post('api/v1/docker/registry-mirrors', { mirrors })
    return res.data as { taskId: number }
  },
}
