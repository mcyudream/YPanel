// B7 前端：事件总线订阅（WS 单例）。收到事件后通过回调分发给页面/全局提示。
import { ref } from 'vue'

export interface BusEvent {
  topic: string
  type: string
  title?: string
  payload?: any
  ts: string
}

const events = ref<BusEvent[]>([])
let connected = false
const listeners: Array<(e: BusEvent) => void> = []
let ws: WebSocket | null = null
let retryTimer: ReturnType<typeof setTimeout> | null = null
let retries = 0

function connect() {
  if (connected) {
    return
  }
  connected = true
  const token = localStorage.getItem('token') || ''
  if (!token) {
    connected = false
    return
  }
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${location.host}${wsBase()}/api/v1/ws?token=${encodeURIComponent(token)}`)
  ws.onopen = () => {
    retries = 0
  }
  ws.onmessage = (ev) => {
    try {
      const e = JSON.parse(ev.data as string) as BusEvent
      events.value.push(e)
      if (events.value.length > 100) {
        events.value.splice(0, events.value.length - 100)
      }
      listeners.forEach(fn => fn(e))
    }
    catch {}
  }
  ws.onclose = () => {
    connected = false
    if (retryTimer) {
      clearTimeout(retryTimer)
      retryTimer = null
    }
    if (retries < 10) {
      retries++
      retryTimer = setTimeout(() => {
        retryTimer = null
        connect()
      }, Math.min(30000, 2000 * 2 ** retries))
    }
  }
  ws.onerror = () => {}
}

function onEvent(fn: (e: BusEvent) => void) {
  connect()
  listeners.push(fn)
  return () => {
    const i = listeners.indexOf(fn)
    if (i >= 0) {
      listeners.splice(i, 1)
    }
  }
}

import { wsBase } from '@/utils/format'

export function useEventBus() {
  return { events, onEvent, connect }
}
