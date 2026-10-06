// YdTerminal：xterm 引擎 + 双协议（host=JSON 控制帧 / exec=容器 exec 裸文本帧）。

export type TerminalConnState = 'connecting' | 'connected' | 'closed' | 'error'

export type TerminalEndpoint =
  | { kind: 'host', node?: string }
  | { kind: 'exec', containerId: string, cmd?: string }

function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}

// 各协议的 WS 地址（token 经 query 传递，与现有后端一致）
export function buildTerminalWSURL(endpoint: TerminalEndpoint, token: string): string {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const auth = `token=${encodeURIComponent(token)}`
  if (endpoint.kind === 'exec') {
    const cmd = endpoint.cmd || '/bin/sh'
    return `${proto}://${location.host}${wsBase()}/api/v1/docker/containers/${encodeURIComponent(endpoint.containerId)}/exec?cmd=${encodeURIComponent(cmd)}&${auth}`
  }
  const node = endpoint.node && endpoint.node !== 'local' ? `&node=${encodeURIComponent(endpoint.node)}` : ''
  return `${proto}://${location.host}${wsBase()}/api/v1/terminal?${auth}${node}`
}

// 主机终端默认重试 3 次（断线自动重开会话）；容器 exec 会话结束即终止
export function defaultRetries(endpoint: TerminalEndpoint): number {
  return endpoint.kind === 'exec' ? 0 : 3
}
