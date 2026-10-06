<script setup lang="ts">
import type { VwtMessage } from './types'
import { Terminal as VwtTerminal } from 'vue-web-terminal'

// vue-web-terminal 引擎（行模式）：
// - 用户在组件输入行编辑命令（历史/提示为 vwt 原生能力），回车整行下发 `cmd\n`；
// - pty 输出（含回显）以 ansi 消息流式 appendMessage（超限轮换新消息，防单条无限膨胀）；
// - Ctrl+C/D 等控制键即时下发控制字节；
// - vwt 的 ansi 解析只保留着色码，\r/光标控制被过滤，因此交互式全屏程序请切 xterm 引擎。
const props = defineProps<{
  theme: 'dark' | 'light'
}>()

const emits = defineEmits<{
  input: [data: string]
  size: [cols: number, rows: number]
}>()

const termRef = useTemplateRef<InstanceType<typeof VwtTerminal>>('term')
const rootRef = useTemplateRef<HTMLDivElement>('root')

// 每实例全局唯一 name（TerminalApi 按 name 寻址）
const name = `ydt-vwt-${Date.now().toString(36)}-${Math.floor(Math.random() * 1e8).toString(36)}`

// ---- 输出流式回显（轮换 ansi 消息） ----
// vwt 的 ansi 解析只翻译 SGR 着色码，其余序列（OSC 标题、\e[K 擦行、\e[?2004h 括号粘贴等）
// 会原样漏成可见乱码：这里清洗为仅保留 \n 与着色码。
function sanitizeAnsi(s: string): string {
  return s
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
    .replace(/\x1b\[([0-9;?]*)([ -/]*)([@-~])/g, (m, _p1, _p2, fin: string) => (fin === 'm' ? m : ''))
    .replace(/[\x00-\x08\x0b-\x1f\x7f]/g, '')
}

const ROTATE_BYTES = 160 * 1024
let targetFresh = true
let targetBytes = 0
let buf = ''
let flushScheduled = false
const decoder = new TextDecoder('utf-8')

function write(data: string | Uint8Array) {
  buf += sanitizeAnsi(typeof data === 'string' ? data : decoder.decode(data, { stream: true }))
  if (!flushScheduled) {
    flushScheduled = true
    nextTick(flushBuf)
  }
}

function flushBuf() {
  flushScheduled = false
  const term = termRef.value
  if (!term || !buf) {
    return
  }
  if (targetFresh) {
    term.pushMessage({ type: 'ansi', content: buf } satisfies VwtMessage)
    targetFresh = false
    targetBytes = buf.length
  }
  else {
    term.appendMessage(buf)
    targetBytes += buf.length
  }
  buf = ''
  if (targetBytes > ROTATE_BYTES) {
    targetFresh = true
  }
  try {
    term.jumpToBottom(false)
  }
  catch {}
}

// 系统提示（连接状态等），不进入输出流
function note(text: string) {
  targetFresh = true
  termRef.value?.pushMessage({ type: 'ansi', class: 'system', content: `\r\n[${text}]\r\n` } satisfies VwtMessage)
}

function reset() {
  buf = ''
  targetFresh = true
  try {
    termRef.value?.clearLog(false)
  }
  catch {}
}

// ---- 行执行：整行下发，远端回显与输出经 write() 流式回来 ----
function onExecCmd(_key: string, command: string, success: () => void) {
  const cmd = command.trim()
  // 本地特殊命令：清屏（shell 侧照发，远端同样清屏）
  if (cmd === 'clear' || cmd === 'cls') {
    try {
      termRef.value?.clearLog(false)
    }
    catch {}
  }
  emits('input', `${command}\n`)
  success()
}

// ---- 控制键拦截（不进输入行，直接下发字节） ----
function onKeydown(e: KeyboardEvent) {
  if (e.isComposing || e.key === 'Process') {
    return
  }
  if (!(e.ctrlKey && !e.altKey && !e.metaKey)) {
    return
  }
  const k = e.key.toUpperCase()
  if (k === 'C' || k === 'D') {
    e.preventDefault()
    try {
      termRef.value?.setCommand('')
    }
    catch {}
    emits('input', k === 'C' ? '\x03' : '\x04')
  }
}

// ---- 尺寸估算（host 协议 resize 用；exec 协议忽略） ----
// 用真实 DOM 探针量字符宽/行高，比 elementInfo 估算更贴近实际渲染。
function measure() {
  const term = termRef.value
  const el = rootRef.value
  if (!term || !el) {
    return
  }
  try {
    const probe = document.createElement('span')
    probe.textContent = 'M'.repeat(20)
    probe.style.cssText = 'position:absolute;visibility:hidden;white-space:pre;font:13px Menlo, Monaco, "Courier New", monospace'
    el.appendChild(probe)
    const charW = probe.getBoundingClientRect().width / 20 || 7
    probe.remove()
    const rowH = 19
    const cols = Math.max(20, Math.floor((el.clientWidth - 24) / charW))
    const rows = Math.max(6, Math.floor(el.clientHeight / rowH) - 2)
    emits('size', cols, rows)
  }
  catch {}
}

let ro: ResizeObserver | null = null
let roRaf = 0

function onInitComplete() {
  nextTick(() => {
    measure()
    try {
      termRef.value?.focus()
    }
    catch {}
  })
  if (rootRef.value && !ro) {
    ro = new ResizeObserver(() => {
      cancelAnimationFrame(roRaf)
      roRaf = requestAnimationFrame(measure)
    })
    ro.observe(rootRef.value)
  }
}

function focus() {
  try {
    termRef.value?.focus(true)
  }
  catch {}
}

onBeforeUnmount(() => {
  ro?.disconnect()
  cancelAnimationFrame(roRaf)
})

defineExpose({ write, note, reset, focus, measure })
</script>

<template>
  <div ref="root" class="vwt-host size-full min-h-0 min-w-0 overflow-hidden">
    <VwtTerminal
      ref="term"
      :name="name"
      :theme="props.theme"
      context=""
      context-suffix=""
      :show-header="false"
      :init-log="[]"
      :enable-default-command="false"
      :enable-input-tips="false"
      :enable-help-box="false"
      :enable-fold="false"
      :log-size-limit="600"
      :line-space="8"
      cursor-style="bar"
      scroll-mode="auto"
      class="size-full!"
      @exec-cmd="onExecCmd"
      @on-keydown="onKeydown"
      @init-complete="onInitComplete"
    >
      <!-- 隐藏 vwt 的命令行回显行：远端 pty 回显已含真实提示符，避免重复 -->
      <template #cmdLine>
        <span style="display: none" />
      </template>
    </VwtTerminal>
  </div>
</template>

<style>
/* vwt 主题变量按宿主字号收窄（默认 16px 偏大），跟随 fa 终端字号 */
.vwt-host {
  --t-font-size: 13px;
  --t-font-height: 19px;
  --t-cmd-tips-border-radius: 6px;
}
</style>
