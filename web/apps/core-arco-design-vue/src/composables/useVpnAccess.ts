// 组网访问跳转：EasyTier 运行时取本机虚拟 IP，容器端口一键跳转（M28 组网延伸）。
// 模块级缓存 60s：status 需 agent exec docker inspect + easytier-cli，跨页面复用避免反复执行。
import { onMounted, ref } from 'vue'

import { vpnApi } from '@/api/modules/vpn'

const cacheMs = 60_000
let cachedIp = ''
let cachedAt = 0

/** 当前组网虚拟 IP（未安装/未运行/探测失败均为空串） */
const vpnIp = ref('')

/** 刷新组网虚拟 IP；force=true 跳过 60s 缓存（组网配置变更后使用）。 */
async function refreshVpnIp(force = false): Promise<void> {
  if (!force && cachedAt && Date.now() - cachedAt < cacheMs) {
    vpnIp.value = cachedIp
    return
  }
  let ip = ''
  try {
    const st = await vpnApi.status()
    if (st.running && st.virtualIp) {
      ip = st.virtualIp
    }
  }
  catch {
    // silent：组网未安装等场景下状态接口失败不影响容器页
  }
  cachedIp = ip
  cachedAt = Date.now()
  vpnIp.value = ip
}

export function useVpnAccess() {
  onMounted(() => {
    void refreshVpnIp()
  })

  /**
   * 端口经组网虚拟 IP 的浏览器跳转地址；不可跳返回空串。
   * 仅 tcp 且宿主端口存在；docker-proxy 只监听绑定地址，仅全网卡绑定（0.0.0.0/::/空）对虚拟 IP 可达。
   */
  function portJumpUrl(hostIp: string, hostPort: string | number, proto: string): string {
    const port = String(hostPort || '').trim()
    if (!vpnIp.value || !port || proto !== 'tcp') {
      return ''
    }
    const ip = (hostIp || '').trim()
    if (ip && ip !== '0.0.0.0' && ip !== '::' && ip !== '[::]') {
      return ''
    }
    return `${port === '443' ? 'https' : 'http'}://${vpnIp.value}:${port}`
  }

  return { vpnIp, portJumpUrl, refreshVpnIp }
}
