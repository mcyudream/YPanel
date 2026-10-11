<script setup lang="ts">
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import TouchKeybar from './TouchKeybar.vue'

// xterm 引擎：全仿真终端（透明逐键 + TUI 支持），vim/htop 等全屏程序的兜底引擎。
// 移动端（mode=mobile）：触屏点按手势内聚焦弹软键盘 + 底部按键条（Ctrl/Esc/Tab）+ 软键盘弹起时
// 高度钳制到可视区（visualViewport 压缩即收高，ResizeObserver 随动 refit）。
const props = defineProps<{
  theme: 'dark' | 'light'
}>()

const emits = defineEmits<{
  input: [data: string]
  size: [cols: number, rows: number]
  /** shell 经 OSC 7 上报的当前目录（bash rcfile 集成注入） */
  cwd: [path: string]
}>()

const hostRef = useTemplateRef<HTMLDivElement>('host')
const rootRef = useTemplateRef<HTMLDivElement>('root')

let term: Terminal | null = null
let fit: FitAddon | null = null
let ro: ResizeObserver | null = null
let roRaf = 0

function termTheme() {
  // xterm 默认前景为白色：浅色背景必须显式给深色前景，否则白字白底
  return props.theme === 'dark'
    ? { background: '#1c1c1a', foreground: '#d4d4d4', cursor: '#d4d4d4', cursorAccent: '#1c1c1a' }
    : { background: '#ffffff', foreground: '#1f2328', cursor: '#1f2328', cursorAccent: '#ffffff' }
}

// 软键盘钳制：键盘弹起（可视高度被压缩 >120px）时把引擎根高度钉到键盘上沿，避免输入行被盖
const vv = typeof window !== 'undefined' ? window.visualViewport : null
function vvFit() {
  const root = rootRef.value
  if (!root) {
    return
  }
  const kbOpen = !!vv && window.innerHeight - vv.height > 120 && document.body.dataset.mode === 'mobile'
  if (kbOpen && vv) {
    const top = root.getBoundingClientRect().top
    root.style.height = `${Math.max(160, Math.round(vv.height - top))}px`
  }
  else if (root.style.height) {
    root.style.height = ''
  }
}

// 触屏点按：在用户手势内聚焦隐藏 textarea，iOS 才会弹软键盘（程序化 focus 不算手势）
function onHostPointerUp(ev: PointerEvent) {
  if (ev.pointerType === 'touch') {
    term?.focus()
  }
}

onMounted(() => {
  const el = hostRef.value
  if (!el) {
    return
  }
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: termTheme(),
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(el)
  term.onData((data) => {
    emits('input', data)
  })
  // OSC 7：shell 提示符处上报 cwd（payload 形如 file://host/path），返回 false 不阻断后续处理器
  term.parser.registerOscHandler(7, (data) => {
    try {
      const u = new URL(data)
      if (u.protocol === 'file:') {
        emits('cwd', decodeURIComponent(u.pathname))
      }
    }
    catch {}
    return false
  })
  try {
    fit.fit()
  }
  catch {}
  emitSize()
  ro = new ResizeObserver(() => {
    cancelAnimationFrame(roRaf)
    roRaf = requestAnimationFrame(() => {
      try {
        fit?.fit()
      }
      catch {}
      emitSize()
    })
  })
  ro.observe(el)
  term.focus()
  vv?.addEventListener('resize', vvFit)
  vv?.addEventListener('scroll', vvFit)
  // 首算 fit 可能落在等宽字体就绪前（字宽偏大→列数偏少留白），字体就绪后重排一次
  document.fonts?.ready.then(() => {
    requestAnimationFrame(() => {
      try {
        fit?.fit()
      }
      catch {}
      emitSize()
    })
  }).catch(() => {})
})

onBeforeUnmount(() => {
  vv?.removeEventListener('resize', vvFit)
  vv?.removeEventListener('scroll', vvFit)
  ro?.disconnect()
  cancelAnimationFrame(roRaf)
  term?.dispose()
  term = null
})

watch(() => props.theme, () => {
  if (term) {
    term.options.theme = termTheme()
  }
})

function emitSize() {
  if (term) {
    emits('size', term.cols, term.rows)
  }
}

function write(data: string | Uint8Array) {
  term?.write(data)
}

function note(text: string) {
  term?.write(`\r\n\x1b[2m[${text}]\x1b[0m\r\n`)
}

function reset() {
  term?.reset()
}

function focus() {
  term?.focus()
}

defineExpose({ write, note, reset, focus })
</script>

<template>
  <div ref="root" class="flex size-full min-h-0 min-w-0 flex-col">
    <div ref="host" class="yd-xterm min-h-0 min-w-0 flex-1 p-0.5" @pointerup="onHostPointerUp" />
    <TouchKeybar :send="data => emits('input', data)" :refocus="focus" />
  </div>
</template>

<style>
/* xterm.css 给 .xterm-viewport 硬编码黑底（macOS 滚动条兼容），v6 DOM 渲染器不再以
   内联主题色覆盖，滚动条槽会露出黑底形成"黑框"：透明化，随宿主背景自适应明暗 */
.yd-xterm .xterm .xterm-viewport {
  background-color: transparent;
}
</style>
