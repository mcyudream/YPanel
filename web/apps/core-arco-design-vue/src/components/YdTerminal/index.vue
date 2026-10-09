<script setup lang="ts">
import type { TerminalConnState, TerminalEndpoint } from './types'
import { useReconnectingWs } from '@/composables/useReconnectingWs'
import { i18n } from '@/locales'
import { defaultRetries, buildTerminalWSURL } from './types'
import XtermEngine from './XtermEngine.vue'

// YdTerminal 终端会话组件（xterm 引擎）：
// - host 协议：入站 JSON {type:input|resize}，出站 binary（pty 原始输出），支持 resize；
// - exec 协议（容器 exec）：双向裸文本帧，无 resize；
// - 断线自动重开新会话（host 重试 3 次；exec 会话结束即终止），重连后清空并重新握手尺寸；
// - 自动重试耗尽断死后按 Enter 手动重连（其余按键丢弃，避免待发命令的首键被吞）。
const props = withDefaults(defineProps<{
  endpoint: TerminalEndpoint
  /** 是否聚焦（切换标签时置 true） */
  active?: boolean
}>(), {
  active: true,
})

const emits = defineEmits<{
  state: [state: TerminalConnState, text?: string]
  /** 键入字节（同步输入广播用） */
  input: [data: string]
  /** 引擎上报的尺寸（状态栏显示用） */
  size: [cols: number, rows: number]
  /** shell 经 OSC 7 上报的当前目录（文件树跟随用） */
  cwd: [path: string]
}>()

const appAccountStore = useAppAccountStore()

const engineRef = useTemplateRef<{ write: (d: string | Uint8Array) => void, note: (t: string) => void, reset: () => void, focus: () => void }>('engine')

// 主题跟随实际生效的暗色（html.dark）：经典面板由 fa settings 驱动、桌面工作台由 webos
// 设置驱动，两套状态各自切换 html.dark——监听 class 变化统一口径，避免「webos 深色下终端白底」
const isDark = ref(document.documentElement.classList.contains('dark'))
let themeObserver: MutationObserver | null = null

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark')
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})

const theme = computed<'dark' | 'light'>(() => isDark.value ? 'dark' : 'light')

const lastSize = reactive({ cols: 80, rows: 24 })
let connectedOnce = false
let gaveUp = false
let conn: ReturnType<typeof useReconnectingWs> | null = null

function setState(state: TerminalConnState, text?: string) {
  emits('state', state, text)
}

function sendResize() {
  if (!conn) {
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
  // 断死状态：回车手动重连；其余按键丢弃，不上抛不同步广播
  if (gaveUp) {
    if (data === '\r') {
      reconnect()
    }
    return
  }
  if (conn) {
    conn.send(props.endpoint.kind === 'exec' ? data : JSON.stringify({ type: 'input', data }))
  }
  emits('input', data)
}

function connect() {
  gaveUp = false
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
      gaveUp = true
      engineRef.value?.note(i18n.global.t('components.ydTerminal.pressEnterRetry', { reason }))
      setState('closed', reason)
    },
  })
}

// 断死后的手动重连：关旧连接重建新会话（重连成功即清屏并重新握手尺寸）
function reconnect() {
  engineRef.value?.note(i18n.global.t('components.ydTerminal.reconnecting'))
  conn?.close()
  conn = null
  connect()
}

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
    <XtermEngine
      ref="engine"
      :theme="theme"
      @input="onEngineInput"
      @size="onEngineSize"
      @cwd="emits('cwd', $event)"
    />
  </div>
</template>
