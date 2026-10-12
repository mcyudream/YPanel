<script setup lang="ts">
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { copyText, readClipboardText } from '@/utils/clipboard'
import { i18n } from '@/locales'
import TouchKeybar from './TouchKeybar.vue'

// xterm 引擎：全仿真终端（透明逐键 + TUI 支持），vim/htop 等全屏程序的兜底引擎。
// 移动端（mode=mobile）：触屏点按手势内聚焦弹软键盘 + 底部按键条（Ctrl/Esc/Tab）+ 软键盘弹起时
// 高度钳制到可视区（visualViewport 压缩即收高，ResizeObserver 随动 refit）。
// 选择与复制：右键菜单（复制/粘贴/全选/清屏）+ Ctrl+C 有选区即复制（无选区仍 SIGINT）、
// Ctrl+Shift+C/V、Ctrl+Ins/Shift+Ins；http 非安全上下文读剪贴板不可用时粘贴降级弹窗手动粘贴。
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

const hasSelection = ref(false)

// ---- 右键菜单（复制依赖实时选区态，经 onSelectionChange 维护） ----
interface TermMenuItem {
  label: string
  icon: string
  variant?: 'default' | 'destructive'
  disabled?: boolean
  handle: () => void
}

async function copySelection() {
  if (!term?.hasSelection()) {
    return
  }
  const ok = await copyText(term.getSelection())
  if (!ok) {
    useFaToast().error(i18n.global.t('components.ydTerminal.copyFailed'))
  }
}

async function pasteFromClipboard() {
  const text = await readClipboardText()
  if (text === null) {
    // http 非安全上下文等读不到剪贴板：弹窗手动粘贴
    pasteText.value = ''
    pasteVisible.value = true
    return
  }
  if (text) {
    term?.paste(text)
  }
}

const contextItems = computed<TermMenuItem[][]>(() => [[
  { label: i18n.global.t('common.copy'), icon: 'i-lucide:copy', disabled: !hasSelection.value, handle: () => { void copySelection() } },
  { label: i18n.global.t('components.ydTerminal.paste'), icon: 'i-lucide:clipboard-paste', handle: () => { void pasteFromClipboard() } },
  { label: i18n.global.t('components.ydTerminal.selectAll'), icon: 'i-lucide:text-select', handle: () => term?.selectAll() },
  { label: i18n.global.t('components.ydTerminal.clearScreen'), icon: 'i-lucide:eraser', handle: () => term?.clear() },
]])

// ---- 手动粘贴降级弹窗 ----
const pasteVisible = ref(false)
const pasteText = ref('')

function confirmPaste() {
  if (pasteText.value) {
    term?.paste(pasteText.value)
  }
  pasteVisible.value = false
}

// Ctrl+C 有选区=复制（不发 SIGINT）、无选区=默认中断；Ctrl+Shift+C/V 与 Ins 系列走剪贴板。
// Ctrl+Shift+C 在 Chrome 被开发者工具快捷键截胡收不到，属浏览器层限制，右键菜单兜底。
function keyHandler(ev: KeyboardEvent): boolean {
  if (ev.type !== 'keydown' || !term) {
    return true
  }
  const plainCtrl = ev.ctrlKey && !ev.shiftKey && !ev.altKey && !ev.metaKey
  if (ev.ctrlKey && ev.code === 'KeyC' && !ev.altKey) {
    if (term.hasSelection()) {
      void copySelection()
      return false
    }
    return plainCtrl // 无选区放行默认 SIGINT；Ctrl+Shift+C 无选区无意义
  }
  if ((ev.ctrlKey && !ev.altKey && ev.code === 'KeyV') || (ev.shiftKey && ev.code === 'Insert')) {
    void pasteFromClipboard()
    return false
  }
  if (ev.ctrlKey && ev.code === 'Insert') {
    void copySelection()
    return false
  }
  return true
}

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
  term.onSelectionChange(() => {
    hasSelection.value = !!term?.hasSelection()
  })
  term.attachCustomKeyEventHandler(keyHandler)
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
    <!-- pointerup 手势绑在外层包裹：FaContextMenu(reka Slot) 转发会吞触发元素自身的 listener -->
    <div class="flex min-h-0 min-w-0 flex-1 flex-col" @pointerup="onHostPointerUp">
      <FaContextMenu :items="contextItems">
        <div ref="host" class="yd-xterm size-full p-0.5" />
      </FaContextMenu>
    </div>
    <TouchKeybar :send="data => emits('input', data)" :refocus="focus" />
  </div>

  <!-- 手动粘贴降级弹窗（http 非安全上下文读不到剪贴板） -->
  <FaModal v-model="pasteVisible" :title="$t('components.ydTerminal.pasteTitle')" class="max-w-lg!" :destroy-on-close="true">
    <FaInput v-model="pasteText" :placeholder="$t('components.ydTerminal.pastePlaceholder')" class="w-full" @keyup.enter="confirmPaste" />
    <template #footer>
      <FaButton variant="outline" @click="pasteVisible = false">
        {{ $t('common.cancel') }}
      </FaButton>
      <FaButton @click="confirmPaste">
        {{ $t('components.ydTerminal.paste') }}
      </FaButton>
    </template>
  </FaModal>
</template>

<style>
/* xterm.css 给 .xterm-viewport 硬编码黑底（macOS 滚动条兼容），v6 DOM 渲染器不再以
   内联主题色覆盖，滚动条槽会露出黑底形成"黑框"：透明化，随宿主背景自适应明暗 */
.yd-xterm .xterm .xterm-viewport {
  background-color: transparent;
}
</style>
