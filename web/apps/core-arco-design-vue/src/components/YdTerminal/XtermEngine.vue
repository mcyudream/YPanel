<script setup lang="ts">
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'

// xterm 引擎：全仿真终端（透明逐键 + TUI 支持），vim/htop 等全屏程序的兜底引擎。
const props = defineProps<{
  theme: 'dark' | 'light'
}>()

const emits = defineEmits<{
  input: [data: string]
  size: [cols: number, rows: number]
}>()

const hostRef = useTemplateRef<HTMLDivElement>('host')

let term: Terminal | null = null
let fit: FitAddon | null = null
let ro: ResizeObserver | null = null
let roRaf = 0

function termTheme() {
  return props.theme === 'dark'
    ? { background: '#1c1c1a' }
    : { background: '#ffffff' }
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
})

onBeforeUnmount(() => {
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
  <div ref="host" class="size-full min-h-0 min-w-0 p-0.5" />
</template>
