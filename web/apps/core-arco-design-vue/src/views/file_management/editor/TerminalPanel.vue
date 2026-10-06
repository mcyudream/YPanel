<script setup lang="ts">
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { useReconnectingWs } from '@/composables/useReconnectingWs'

// 底部终端面板：xterm 多实例 + useReconnectingWs（PTY 不发心跳；断线自动重开新会话并提示）。
const props = defineProps<{
  /** 面板方向：决定容器 class */
  horizontal: boolean
}>()

const appAccountStore = useAppAccountStore()
const appSettingsStore = useAppSettingsStore()
const store = useFileEditorStore()

interface TermSession {
  id: number
  title: string
  node: string
  term: Terminal
  fit: FitAddon
  send: (data: string) => void
  container: HTMLElement | null
  ro: ResizeObserver | null
}

let uid = 0
const sessions = ref<TermSession[]>([])
const activeId = ref(0)
const hostRef = useTemplateRef<HTMLDivElement>('host')

function wsBase() {
  if (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) {
    return '/proxy'
  }
  return ''
}

function termTheme() {
  return appSettingsStore.settings.theme.colorScheme === 'dark'
    ? { background: '#1c1c1a' }
    : { background: '#ffffff' }
}

function createSession() {
  uid++
  const id = uid
  const node = store.currentNode
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: termTheme(),
  })
  const fit = new FitAddon()
  term.loadAddon(fit)

  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const ws = useReconnectingWs({
    url: () =>
      `${proto}://${location.host}${wsBase()}/api/v1/terminal?token=${encodeURIComponent(appAccountStore.token)}&node=${encodeURIComponent(node)}`,
    heartbeatMs: 0,
    onMessage: data => term.write(typeof data === 'string' ? data : new Uint8Array(data as ArrayBuffer)),
    onOpen: () => {
      requestAnimationFrame(() => {
        try {
          fit.fit()
          ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
        }
        catch {}
      })
    },
    onGiveUp: reason => term.write(`\r\n\x1b[31m[${reason}]\x1b[0m\r\n`),
    maxRetries: 3,
  })

  term.onData(data => ws.send(JSON.stringify({ type: 'input', data })))

  const s: TermSession = {
    id,
    title: `终端 ${id}`,
    node,
    term,
    fit,
    send: (d: string) => ws.send(d),
    container: null,
    ro: null,
  }
  sessions.value.push(s)
  activeId.value = id

  nextTick(() => mountSession(s))
}

function mountSession(s: TermSession) {
  const el = hostRef.value?.querySelector<HTMLElement>(`[data-term-id="${s.id}"]`)
  if (!el || s.container) {
    return
  }
  s.container = el
  s.term.open(el)
  s.ro = new ResizeObserver(() => {
    try {
      s.fit.fit()
      s.send(JSON.stringify({ type: 'resize', cols: s.term.cols, rows: s.term.rows }))
    }
    catch {}
  })
  s.ro.observe(el)
  s.term.focus()
}

function closeSession(id: number) {
  const idx = sessions.value.findIndex(s => s.id === id)
  if (idx === -1) {
    return
  }
  const s = sessions.value[idx]
  s.ro?.disconnect()
  sessions.value.splice(idx, 1)
  if (activeId.value === id && sessions.value.length) {
    activeId.value = sessions.value[Math.max(0, idx - 1)].id
  }
  if (!sessions.value.length) {
    createSession()
  }
}

watch(activeId, () => {
  const s = sessions.value.find(x => x.id === activeId.value)
  if (s?.container) {
    requestAnimationFrame(() => {
      try {
        s.fit.fit()
      }
      catch {}
      s.term.focus()
    })
  }
})

// 新建终端时跟随当前节点（已有会话保持原节点不变）

onMounted(() => createSession())

onBeforeUnmount(() => {
  sessions.value.forEach((s) => {
    s.ro?.disconnect()
    s.term.dispose()
  })
  sessions.value = []
})
</script>

<template>
  <Pane
    :size="store.layout.terminalSize"
    min-size="10"
    max-size="80"
    class="flex min-h-0 min-w-0 flex-col overflow-hidden border-t bg-background"
    :class="props.horizontal ? '' : 'border-l'"
  >
    <!-- 面板头 -->
    <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-2 text-[13px]">
      <YdMorphIcon name="square-terminal" :size="14" class="text-muted-foreground" />
      <span class="mr-1 text-xs font-medium text-muted-foreground">终端</span>
      <button
        v-for="s in sessions"
        :key="s.id"
        type="button"
        class="group inline-flex cursor-pointer items-center gap-1 rounded px-2 py-0.5 transition-colors"
        :class="activeId === s.id ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
        @click="activeId = s.id"
      >
        {{ s.title }}
        <span v-if="s.node !== 'local'" class="text-[10px] opacity-60">@{{ s.node }}</span>
        <span
          class="inline-flex size-3.5 cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-accent"
          title="关闭"
          @click.stop="closeSession(s.id)"
        >
          <FaIcon name="i-lucide:x" class="text-[9px]" />
        </span>
      </button>
      <FaButton variant="ghost" size="icon-sm" class="ml-1 size-5!" title="新建终端" @click="createSession">
        <FaIcon name="i-lucide:plus" class="text-xs" />
      </FaButton>
      <div class="ml-auto flex items-center gap-0.5">
        <FaButton variant="ghost" size="icon-sm" class="size-5!" title="切到底部/右侧" @click="store.layout.terminalSide = store.layout.terminalSide === 'bottom' ? 'right' : 'bottom'">
          <FaIcon :name="store.layout.terminalSide === 'bottom' ? 'i-lucide:panel-right' : 'i-lucide:panel-bottom'" class="text-xs" />
        </FaButton>
        <FaButton variant="ghost" size="icon-sm" class="size-5!" title="关闭面板 (Ctrl+J)" @click="store.toggleTerminal()">
          <FaIcon name="i-lucide:x" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 终端容器 -->
    <div ref="host" class="relative min-h-0 flex-1 p-0.5">
      <div
        v-for="s in sessions"
        :key="s.id"
        v-show="activeId === s.id"
        :data-term-id="s.id"
        class="size-full min-h-0"
      />
    </div>
  </Pane>
</template>
