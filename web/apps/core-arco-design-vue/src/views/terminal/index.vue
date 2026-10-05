<script setup lang="ts">
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'

defineOptions({
  name: 'TerminalIndex',
})

interface TermTab {
  id: number
  title: string
  term: Terminal
  fit: FitAddon
  ws: WebSocket
  container: HTMLElement
  resizeObserver: ResizeObserver
}

const appAccountStore = useAppAccountStore()
const appSettingsStore = useAppSettingsStore()

let uid = 0
const tabs = ref<TermTab[]>([])
const activeId = ref<number>(0)

function wsBase() {
  if (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) {
    return '/proxy'
  }
  return ''
}

function createTab() {
  uid++
  const id = uid
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: appSettingsStore.settings.theme.colorScheme === 'dark'
      ? { background: '#1c1c1a' }
      : { background: '#ffffff' },
  })
  const fit = new FitAddon()
  term.loadAddon(fit)

  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const ws = new WebSocket(`${proto}://${location.host}${wsBase()}/api/v1/terminal?token=${encodeURIComponent(appAccountStore.token)}`)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    term.onData((data) => {
      ws.send(JSON.stringify({ type: 'input', data }))
    })
    // 首次自适应尺寸
    requestAnimationFrame(() => {
      try {
        fit.fit()
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
      catch {}
    })
  }
  ws.onmessage = (ev) => {
    term.write(new Uint8Array(ev.data as ArrayBuffer))
  }
  ws.onclose = (ev) => {
    term.write(`\r\n\x1b[33m[连接已断开${ev.reason ? `：${ev.reason}` : ''}]\x1b[0m\r\n`)
  }
  ws.onerror = () => {
    term.write('\r\n\x1b[31m[连接错误]\x1b[0m\r\n')
  }

  const tab: TermTab = {
    id,
    title: `终端 ${id}`,
    term,
    fit,
    ws,
    container: null!,
    resizeObserver: null!,
  }
  tabs.value.push(tab)
  activeId.value = id

  // 挂载到 DOM（等 nextTick 容器出现）
  nextTick(() => {
    const el = document.getElementById(`term-container-${id}`)
    if (!el) {
      return
    }
    tab.container = el
    term.open(el)
    tab.resizeObserver = new ResizeObserver(() => {
      try {
        fit.fit()
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
        }
      }
      catch {}
    })
    tab.resizeObserver.observe(el)
    term.focus()
  })
}

function closeTab(id: number) {
  const idx = tabs.value.findIndex(t => t.id === id)
  if (idx === -1) {
    return
  }
  const tab = tabs.value[idx]
  tab.resizeObserver?.disconnect()
  tab.ws.close()
  tab.term.dispose()
  tabs.value.splice(idx, 1)
  if (activeId.value === id && tabs.value.length) {
    activeId.value = tabs.value[Math.max(0, idx - 1)].id
  }
  if (!tabs.value.length) {
    createTab()
  }
}

watch(activeId, () => {
  // 切换标签后重新适配尺寸
  const tab = tabs.value.find(t => t.id === activeId.value)
  if (tab?.container) {
    requestAnimationFrame(() => {
      try {
        tab.fit.fit()
      }
      catch {}
    })
  }
})

onMounted(() => createTab())

onBeforeUnmount(() => {
  tabs.value.forEach((t) => {
    t.resizeObserver?.disconnect()
    t.ws.close()
    t.term.dispose()
  })
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="square-terminal" :size="24" />
          <span>终端</span>
        </div>
      </template>
      <template #description>
        <span>服务器交互式终端（WebSocket + PTY）</span>
      </template>
      <FaButton size="sm" @click="createTab">
        <FaIcon name="i-lucide:plus" class="mr-1" /> 新建终端
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <!-- 标签栏 -->
      <div class="mb-2 flex flex-wrap items-center gap-1">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          class="group inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-3 py-1 text-sm transition-colors"
          :class="activeId === t.id
            ? 'border-primary bg-primary/10 text-foreground'
            : 'border-transparent text-muted-foreground hover:bg-accent/50'"
          @click="activeId = t.id"
        >
          <YdMorphIcon name="square-terminal" :size="14" />
          {{ t.title }}
          <span
            class="ml-1 inline-flex size-4 cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-accent"
            title="关闭"
            @click.stop="closeTab(t.id)"
          >
            <FaIcon name="i-lucide:x" class="text-[10px]" />
          </span>
        </button>
      </div>

      <!-- 终端容器 -->
      <div class="relative rounded-lg border bg-background p-1">
        <div
          v-for="t in tabs"
          :key="t.id"
          v-show="activeId === t.id"
          :id="`term-container-${t.id}`"
          class="h-[calc(100vh-320px)] min-h-96 w-full"
        />
      </div>
    </FaPageMain>
  </div>
</template>
