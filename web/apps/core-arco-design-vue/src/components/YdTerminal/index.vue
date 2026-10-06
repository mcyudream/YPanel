<script setup lang="ts">
import type { TerminalConnState, TerminalEngine, TerminalEndpoint } from './types'
import { useReconnectingWs } from '@/composables/useReconnectingWs'
import { defaultRetries, buildTerminalWSURL } from './types'
import VwtEngine from './VwtEngine.vue'
import XtermEngine from './XtermEngine.vue'

// YdTerminal 核心：双引擎 + 双协议的终端会话组件。
// - host 协议：入站 JSON {type:input|resize}，出站 binary（pty 原始输出），支持 resize；
// - exec 协议（容器 exec）：双向裸文本帧，无 resize；
// - 断线自动重开新会话（PTY 不保活），重连后清空引擎并重新握手尺寸。
const props = withDefaults(defineProps<{
  endpoint: TerminalEndpoint
  engine?: TerminalEngine
  /** 是否聚焦（切换标签时置 true） */
  active?: boolean
}>(), {
  engine: 'vwt',
  active: true,
})

const emits = defineEmits<{
  state: [state: TerminalConnState, text?: string]
  /** 键入字节（同步输入广播用） */
  input: [data: string]
  /** 引擎上报的尺寸（状态栏显示用） */
  size: [cols: number, rows: number]
}>()

const appAccountStore = useAppAccountStore()
const appSettingsStore = useAppSettingsStore()

const engineRef = useTemplateRef<{ write: (d: string | Uint8Array) => void, note: (t: string) => void, reset: () => void, focus: () => void }>('engine')

const theme = computed<'dark' | 'light'>(() => appSettingsStore.settings.theme.colorScheme === 'dark' ? 'dark' : 'light')

const lastSize = reactive({ cols: 80, rows: 24 })
let connectedOnce = false
let conn: ReturnType<typeof useReconnectingWs> | null = null

function setState(state: TerminalConnState, text?: string) {
  emits('state', state, text)
}

function sendResize() {
  if (props.endpoint.kind !== 'host' || !conn) {
    return
  }
  conn.send(JSON.stringify({ type: 'resize', cols: lastSize.cols, rows: lastSize.rows }))
}

function onEngineSize(cols: number, rows: number) {
  if (cols === lastSize.cols && rows === lastSize.rows) {
    return
  }
  lastSize.cols = cols
  lastSize.rows = rows
  sendResize()
  emits('size', cols, rows)
}

function onEngineInput(data: string) {
  if (conn) {
    conn.send(props.endpoint.kind === 'exec' ? data : JSON.stringify({ type: 'input', data }))
  }
  emits('input', data)
}

function connect() {
  setState('connecting')
  conn = useReconnectingWs({
    url: () => buildTerminalWSURL(props.endpoint, appAccountStore.token),
    heartbeatMs: 0,
    maxRetries: defaultRetries(props.endpoint),
    onOpen: () => {
      if (connectedOnce) {
        // 重连 = 服务端新 PTY 会话：清屏后重新握手尺寸
        engineRef.value?.reset()
      }
      connectedOnce = true
      setState('connected')
      requestAnimationFrame(sendResize)
    },
    onMessage: (data) => {
      setState('connected')
      engineRef.value?.write(typeof data === 'string' ? data : new Uint8Array(data as ArrayBuffer))
    },
    onGiveUp: (reason) => {
      engineRef.value?.note(reason)
      setState('closed', reason)
    },
  })
}

// 引擎切换 = 重开会话（两端状态一致）
watch(() => props.engine, () => {
  connectedOnce = false
  conn?.close()
  conn = null
  nextTick(connect)
})

// endpoint 变化（如容器切换/shell 切换）= 重开会话。
// 注意按序列化 key 比较：父组件模板常内联新对象字面量，按引用观察会导致渲染即重连的死循环。
const endpointKey = computed(() => props.endpoint.kind === 'exec'
  ? `exec:${props.endpoint.containerId}:${props.endpoint.cmd || ''}`
  : `host:${props.endpoint.node || 'local'}`)

watch(endpointKey, () => {
  connectedOnce = false
  conn?.close()
  conn = null
  nextTick(connect)
})

watch(() => props.active, (v) => {
  if (v) {
    requestAnimationFrame(() => engineRef.value?.focus())
  }
})

onMounted(connect)

onBeforeUnmount(() => {
  conn?.close()
  conn = null
})

// 供外部注入字节（同步输入广播）：走协议封装但不回发 input 事件
function sendRaw(data: string) {
  if (!conn) {
    return
  }
  conn.send(props.endpoint.kind === 'exec' ? data : JSON.stringify({ type: 'input', data }))
}

defineExpose({
  focus: () => engineRef.value?.focus(),
  sendRaw,
  size: lastSize,
})
</script>

<template>
  <div class="relative size-full min-h-0 min-w-0 overflow-hidden bg-background">
    <component
      :is="props.engine === 'vwt' ? VwtEngine : XtermEngine"
      ref="engine"
      :theme="theme"
      @input="onEngineInput"
      @size="onEngineSize"
    />
  </div>
</template>
