<script setup lang="ts">
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { useReconnectingWs } from '@/composables/useReconnectingWs'
import { useHtmlDark } from '@/composables/useHtmlDark'
import { i18n } from '@/locales'
import { Pane } from 'splitpanes'

// 底部终端面板：xterm 多实例 + useReconnectingWs（PTY 不发心跳）。
// currentContainer 非空时新会话为容器 exec（裸文本帧 + JSON resize 控制帧）；
// exec 会话支持重连与更换/自定义 shell（重开会话）；宿主会话断线自动重连；
// 断死（自动重试耗尽/exec 结束）后按 Enter 重开会话。
const props = defineProps<{
  /** 面板方向：决定容器 class */
  horizontal: boolean
}>()

const appAccountStore = useAppAccountStore()
const store = useFileEditorStore()

const shellOptions = [
  { label: '/bin/sh', value: '/bin/sh' },
  { label: '/bin/bash', value: '/bin/bash' },
  { label: '/bin/ash', value: '/bin/ash' },
  { label: i18n.global.t('files.editor.customShell'), value: '__custom__' },
]

interface TermSession {
  id: number
  title: string
  node: string
  containerId?: string
  /** 容器 exec 的 shell/命令（宿主会话为空） */
  cmd: string
  conn: { send: (data: string | ArrayBuffer) => void, close: () => void } | null
  /** 自动重试耗尽/exec 结束后的断死态：回车重开会话 */
  dead: boolean
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

const activeSession = computed(() => sessions.value.find(s => s.id === activeId.value))

// 自定义 shell 输入弹窗
const customPrompt = reactive({
  visible: false,
  value: '',
  session: null as TermSession | null,
})

function wsBase() {
  if (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) {
    return '/proxy'
  }
  return ''
}

const htmlDark = useHtmlDark()

function termTheme() {
  // xterm 默认前景为白色：浅色背景必须显式给深色前景，否则白字白底
  return htmlDark.value
    ? { background: '#1c1c1a', foreground: '#d4d4d4', cursor: '#d4d4d4', cursorAccent: '#1c1c1a' }
    : { background: '#ffffff', foreground: '#1f2328', cursor: '#1f2328', cursorAccent: '#ffffff' }
}

function sessionURL(s: TermSession) {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  if (s.containerId) {
    return `${proto}://${location.host}${wsBase()}/api/v1/docker/containers/${encodeURIComponent(s.containerId)}/exec?cmd=${encodeURIComponent(s.cmd)}&token=${encodeURIComponent(appAccountStore.token)}`
  }
  return `${proto}://${location.host}${wsBase()}/api/v1/terminal?token=${encodeURIComponent(appAccountStore.token)}&node=${encodeURIComponent(s.node)}`
}

// 建/重建会话连接（url 动态读取 s.cmd，重连与换 shell 都走这里）
function makeConn(s: TermSession) {
  return useReconnectingWs({
    url: () => sessionURL(s),
    heartbeatMs: 0,
    // 容器 exec 会话结束即终止（手动重连）；宿主终端断线自动重连
    maxRetries: s.containerId ? 0 : 3,
    onMessage: data => s.term.write(typeof data === 'string' ? data : new Uint8Array(data as ArrayBuffer)),
    onOpen: () => {
      s.dead = false
      requestAnimationFrame(() => {
        try {
          s.fit.fit()
          s.send(JSON.stringify({ type: 'resize', cols: s.term.cols, rows: s.term.rows }))
        }
        catch {}
      })
    },
    onGiveUp: (reason) => {
      s.dead = true
      s.term.write(`\r\n\x1b[31m[${reason} · ${i18n.global.t('files.editor.reconnectHint')}]\x1b[0m\r\n`)
    },
  })
}

function createSession() {
  uid++
  const id = uid
  const node = store.currentNode
  const containerId = store.currentContainer || undefined
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: termTheme(),
  })
  const fit = new FitAddon()
  term.loadAddon(fit)

  const s: TermSession = {
    id,
    title: containerId ? i18n.global.t('files.editor.containerTerminal', { n: id }) : i18n.global.t('files.editor.terminal', { n: id }),
    node,
    containerId,
    cmd: '/bin/sh',
    conn: null,
    dead: false,
    term,
    fit,
    send: (d: string) => s.conn?.send(d),
    container: null,
    ro: null,
  }
  s.conn = makeConn(s)
  // 容器 exec 为裸文本帧；宿主终端为 JSON 控制帧
  term.onData((data) => {
    // 断死状态：回车重开会话（exec 会话即按原命令重跑，与工具栏重连按钮同语义）；其余按键丢弃
    if (s.dead) {
      if (data === '\r') {
        restartSession(s)
      }
      return
    }
    s.send(containerId ? data : JSON.stringify({ type: 'input', data }))
  })

  sessions.value.push(s)
  activeId.value = id

  nextTick(() => mountSession(s))
}

// 重连当前会话：关闭旧连接、清屏、按当前 shell 重开（服务端新 PTY）
function restartSession(s: TermSession) {
  s.dead = false
  s.conn?.close()
  s.conn = null
  try {
    s.term.reset()
  }
  catch {}
  s.conn = makeConn(s)
  s.term.focus()
}

// 更换 shell（exec 会话）：更新 cmd 并重开
function changeShell(s: TermSession, value: string) {
  if (value === '__custom__') {
    customPrompt.session = s
    customPrompt.value = s.cmd === '/bin/sh' ? '' : s.cmd
    customPrompt.visible = true
    // 还原选择器显示（确定后再落到实际 cmd）
    return
  }
  if (value === s.cmd) {
    return
  }
  s.cmd = value
  restartSession(s)
}

function submitCustomShell() {
  const s = customPrompt.session
  const v = customPrompt.value.trim()
  customPrompt.visible = false
  if (!s || !v) {
    return
  }
  s.cmd = v
  restartSession(s)
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
  s.conn?.close()
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

// 新建终端时跟随当前节点/容器上下文（已有会话保持不变）

onMounted(() => createSession())

onBeforeUnmount(() => {
  sessions.value.forEach((s) => {
    s.ro?.disconnect()
    s.conn?.close()
    s.term.dispose()
  })
  sessions.value = []
  store.terminalSessions = 0
})

// 会话数回写 store：关闭确认（断开提示）与贴边按钮徽标消费
watch(() => sessions.value.length, (n) => {
  store.terminalSessions = n
}, { immediate: true })
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
      <span class="mr-1 text-xs font-medium text-muted-foreground">{{ $t('files.editor.terminalLabel') }}</span>
      <button
        v-for="s in sessions"
        :key="s.id"
        type="button"
        class="group inline-flex cursor-pointer items-center gap-1 rounded px-2 py-0.5 transition-colors"
        :class="activeId === s.id ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
        :title="s.containerId ? $t('files.editor.containerSession', { cmd: s.cmd }) : s.node"
        @click="activeId = s.id"
      >
        {{ s.title }}
        <span v-if="s.containerId" class="text-[10px] opacity-60">· {{ $t('files.editor.containerBadge') }}</span>
        <span v-else-if="s.node !== 'local'" class="text-[10px] opacity-60">@{{ s.node }}</span>
        <span
          class="inline-flex size-3.5 cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-accent"
          :title="$t('common.close')"
          @click.stop="closeSession(s.id)"
        >
          <FaIcon name="i-lucide:x" class="text-[9px]" />
        </span>
      </button>
      <FaButton variant="ghost" size="icon-sm" class="ml-1 size-5!" :title="$t('files.editor.newTerminal')" @click="createSession">
        <FaIcon name="i-lucide:plus" class="text-xs" />
      </FaButton>
      <div class="ml-auto flex items-center gap-0.5">
        <!-- 容器会话：shell 选择 -->
        <FaSelect
          v-if="activeSession?.containerId"
          :model-value="activeSession.cmd"
          :options="shellOptions"
          class="h-6! w-32 shrink-0 text-xs!"
          :title="$t('files.editor.shellTitle')"
          @update:model-value="changeShell(activeSession, String($event))"
        />
        <FaButton
          variant="ghost"
          size="icon-sm"
          class="size-5!"
          :title="$t('files.editor.reconnect')"
          @click="activeSession && restartSession(activeSession)"
        >
          <FaIcon name="i-lucide:rotate-cw" class="text-xs" />
        </FaButton>
        <FaButton variant="ghost" size="icon-sm" class="size-5!" :title="$t('files.editor.movePanel')" @click="store.layout.terminalSide = store.layout.terminalSide === 'bottom' ? 'right' : 'bottom'">
          <FaIcon :name="store.layout.terminalSide === 'bottom' ? 'i-lucide:panel-right' : 'i-lucide:panel-bottom'" class="text-xs" />
        </FaButton>
        <FaButton variant="ghost" size="icon-sm" class="size-5!" :title="$t('files.editor.closePanel')" @click="store.toggleTerminal()">
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

    <!-- 自定义 shell 输入 -->
    <FaModal v-model="customPrompt.visible" :title="$t('files.editor.customCmdTitle')" class="max-w-lg!" :destroy-on-close="true">
      <div class="space-y-2">
        <div class="text-xs text-muted-foreground">
          {{ $t('files.editor.customCmdHint') }}
        </div>
        <FaInput v-model="customPrompt.value" placeholder="/usr/bin/zsh" class="w-full" @keyup.enter="submitCustomShell" />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="customPrompt.visible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton @click="submitCustomShell">
          {{ $t('files.editor.customCmdOk') }}
        </FaButton>
      </template>
    </FaModal>
  </Pane>
</template>

<style scoped>
/* xterm.css 给 .xterm-viewport 硬编码黑底（macOS 滚动条兼容），亮色主题下滚动条槽露出黑边：透明化 */
:deep(.xterm-viewport) {
  background-color: transparent;
}
</style>
