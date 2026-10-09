// F6：可重连 WebSocket 封装（指数退避 + 心跳）。
// 终端/容器 exec 等 PTY 场景复用；组件卸载自动关闭。
import { onBeforeUnmount } from 'vue'
import { i18n } from '@/locales'

export interface ReconnectingWsOptions {
  /** 连接地址 */
  url: () => string
  /** 收到消息 */
  onMessage: (data: string | ArrayBuffer) => void
  /** 连接建立（每次重连后也会回调） */
  onOpen?: () => void
  /** 最终放弃（达到最大重试） */
  onGiveUp?: (reason: string) => void
  /** 最大重试次数，默认 5 */
  maxRetries?: number
  /** 心跳间隔 ms（0=禁用，默认 30000；发送 ping 文本帧） */
  heartbeatMs?: number
}

export function useReconnectingWs(options: ReconnectingWsOptions) {
  const maxRetries = options.maxRetries ?? 5
  const heartbeatMs = options.heartbeatMs ?? 30000

  let ws: WebSocket | null = null
  let retries = 0
  let closedByUser = false
  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  function stopHeartbeat() {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  function startHeartbeat() {
    if (heartbeatMs <= 0) {
      return
    }
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send('ping')
      }
    }, heartbeatMs)
  }

  function connect() {
    if (closedByUser) {
      return
    }
    ws = new WebSocket(options.url())
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      retries = 0
      startHeartbeat()
      options.onOpen?.()
    }
    ws.onmessage = (ev) => {
      options.onMessage(ev.data)
    }
    ws.onclose = () => {
      stopHeartbeat()
      if (closedByUser) {
        return
      }
      if (retries >= maxRetries) {
        options.onGiveUp?.(i18n.global.t('components.useReconnectingWs.gaveUp', { n: retries }))
        return
      }
      const delay = Math.min(30000, 1000 * 2 ** retries)
      retries++
      reconnectTimer = setTimeout(connect, delay)
    }
    ws.onerror = () => {
      // onclose 会跟随触发，重连逻辑在 onclose 统一处理
    }
  }

  function send(data: string | ArrayBuffer) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  }

  function close() {
    closedByUser = true
    stopHeartbeat()
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }
    ws?.close()
    ws = null
  }

  onBeforeUnmount(close)

  connect()

  return { send, close }
}
