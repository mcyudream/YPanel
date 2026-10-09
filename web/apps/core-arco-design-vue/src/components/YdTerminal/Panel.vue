<script setup lang="ts">
import type { NodeItem } from '@/api/modules/node'
import type { TerminalConnState } from './types'
import { useLocalStorage } from '@vueuse/core'
import { i18n } from '@/locales'
import YdTerminal from './index.vue'
import StatusBar from './StatusBar.vue'

// 终端面板（host 协议多标签）：顶部工具栏 + 标签 + 终端 + 底部状态栏。
// 侧栏开合与节点选择由上层（Workspace）持有，通过 props/emits 协作。
defineOptions({
  name: 'YdTerminalPanel',
})

const props = defineProps<{
  /** 新建会话使用的节点 */
  node: string
  nodes: NodeItem[]
  treeVisible: boolean
  monitorVisible: boolean
  /** 挂载后首个会话就绪时粘贴一次的内容（如跨窗拖入的文件路径），粘贴后清空 */
  initialPaste?: string
}>()

const emits = defineEmits<{
  'update:node': [node: string]
  'toggle-tree': []
  'toggle-monitor': []
  /** 活动标签的当前目录（跟随开关开启时；空串=不跟随），供文件树定位 */
  cwd: [path: string]
}>()

interface TermTab {
  id: number
  title: string
  node: string
  state: TerminalConnState
  stateText?: string
  size: { cols: number, rows: number }
  /** 该会话 shell 最近上报的目录（OSC 7） */
  cwd: string
}

let uid = 0
const tabs = ref<TermTab[]>([])
const activeId = ref(0)
const syncInput = ref(false)
const followTerm = useLocalStorage('ypanel.terminal.follow', true)
const termRefs = new Map<number, InstanceType<typeof YdTerminal>>()

const activeTab = computed(() => tabs.value.find(t => t.id === activeId.value))

/** 向活跃终端写入文本（等效键入，不带回车）——跨窗口拖文件填路径用 */
function pasteText(text: string) {
  const t = tabs.value.find(t => t.id === activeId.value)
  if (t) {
    termRefs.get(t.id)?.sendRaw(text)
  }
}

defineExpose({ pasteText })

// 跟随目录变化（cwd 上报 / 开关切换 / 切换标签）统一向上游汇报
function emitFollowCwd() {
  emits('cwd', followTerm.value ? (activeTab.value?.cwd ?? '') : '')
}

function onTabCwd(id: number, path: string) {
  const t = tabs.value.find(t => t.id === id)
  if (!t) {
    return
  }
  t.cwd = path
  if (id === activeId.value) {
    emitFollowCwd()
  }
}

watch(followTerm, emitFollowCwd)
watch(activeId, emitFollowCwd)

function setRef(id: number) {
  return (el: any) => {
    if (el) {
      termRefs.set(id, el)
    }
    else {
      termRefs.delete(id)
    }
  }
}

function nodeLabel(id: string) {
  const n = props.nodes.find(n => n.id === id)
  return n ? (n.hostname || n.name || id) : id
}

function createTab() {
  uid++
  tabs.value.push({
    id: uid,
    title: i18n.global.t('components.ydTerminal.tabTitle', { uid }),
    node: props.node,
    state: 'connecting',
    size: { cols: 80, rows: 24 },
    cwd: '',
  })
  activeId.value = uid
}

/** 待粘贴内容：等首个会话连接就绪（state=ready）后写入一次 */
const pendingPaste = ref(props.initialPaste ?? '')

watch(() => activeTab.value?.state, (state) => {
  if (state === 'connected' && pendingPaste.value) {
    const text = pendingPaste.value
    pendingPaste.value = ''
    pasteText(text)
  }
})

function closeTab(id: number) {
  const idx = tabs.value.findIndex(t => t.id === id)
  if (idx === -1) {
    return
  }
  termRefs.delete(id)
  tabs.value.splice(idx, 1)
  if (activeId.value === id && tabs.value.length) {
    activeId.value = tabs.value[Math.max(0, idx - 1)].id
  }
  if (!tabs.value.length) {
    createTab()
  }
}

// 同步输入：任一会话键入广播到全部会话
function onTabInput(id: number, data: string) {
  if (!syncInput.value) {
    return
  }
  for (const t of tabs.value) {
    if (t.id !== id) {
      termRefs.get(t.id)?.sendRaw(data)
    }
  }
}

function onTabState(id: number, state: TerminalConnState, text?: string) {
  const t = tabs.value.find(t => t.id === id)
  if (t) {
    t.state = state
    t.stateText = text
  }
}

// 引擎切换（activeTab），并把终端上报尺寸写入标签状态
function onTabSize(id: number, cols: number, rows: number) {
  const t = tabs.value.find(t => t.id === id)
  if (t) {
    t.size.cols = cols
    t.size.rows = rows
  }
}

// 新建终端时跟随当前节点（已有会话保持原节点）
watch(() => props.node, () => {
  if (!tabs.value.length) {
    createTab()
  }
})

onMounted(() => createTab())
</script>

<template>
  <div class="flex size-full min-h-0 min-w-0 flex-col overflow-hidden">
    <!-- 顶部工具栏：节点 + 标签 + 面板开关 -->
    <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-1.5 text-[13px]">
      <FaSelect
        :model-value="props.node"
        :options="[{ label: $t('components.ydTerminal.localNode'), value: 'local' }, ...props.nodes.filter(n => n.id !== 'local').map(n => ({ label: n.hostname || n.name || n.id, value: n.id }))]"
        class="w-28 shrink-0"
        :disabled="props.nodes.filter(n => n.id !== 'local').length === 0"
        @update:model-value="emits('update:node', String($event))"
      />
      <FaButton variant="ghost" size="icon-sm" class="size-5! shrink-0" :title="$t('components.ydTerminal.newTab')" @click="createTab">
        <FaIcon name="i-lucide:plus" class="text-xs" />
      </FaButton>

      <!-- 标签 -->
      <div class="min-w-0 flex flex-1 items-center gap-0.5 overflow-x-auto">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          class="group inline-flex shrink-0 cursor-pointer items-center gap-1 rounded px-2 py-0.5 transition-colors"
          :class="activeId === t.id ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
          :title="nodeLabel(t.node)"
          @click="activeId = t.id"
        >
          <YdMorphIcon name="square-terminal" :size="12" />
          <span class="max-w-28 truncate">{{ t.title }}</span>
          <span v-if="t.node !== 'local'" class="text-[10px] opacity-60">@{{ nodeLabel(t.node) }}</span>
          <span
            class="inline-flex size-3.5 shrink-0 cursor-pointer items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-accent"
            :title="$t('common.close')"
            @click.stop="closeTab(t.id)"
          >
            <FaIcon name="i-lucide:x" class="text-[9px]" />
          </span>
        </button>
      </div>

      <div class="ml-auto flex shrink-0 items-center gap-0.5">
        <label class="flex cursor-pointer items-center gap-1 rounded px-1.5 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-accent/50" :title="$t('components.ydTerminal.followTip')">
          <input v-model="followTerm" type="checkbox" class="size-3">
          {{ $t('components.ydTerminal.follow') }}
        </label>
        <label class="flex cursor-pointer items-center gap-1 rounded px-1.5 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-accent/50" :title="$t('components.ydTerminal.syncTip')">
          <input v-model="syncInput" type="checkbox" class="size-3">
          {{ $t('components.ydTerminal.sync') }}
        </label>
        <FaButton variant="ghost" size="icon-sm" class="size-5!" :title="props.treeVisible ? $t('components.ydTerminal.collapseTree') : $t('components.ydTerminal.expandTree')" @click="emits('toggle-tree')">
          <FaIcon :name="props.treeVisible ? 'i-lucide:panel-left-close' : 'i-lucide:panel-left'" class="text-xs" />
        </FaButton>
        <FaButton variant="ghost" size="icon-sm" class="size-5!" :title="props.monitorVisible ? $t('components.ydTerminal.collapseMonitor') : $t('components.ydTerminal.expandMonitor')" @click="emits('toggle-monitor')">
          <FaIcon :name="props.monitorVisible ? 'i-lucide:panel-right-close' : 'i-lucide:panel-right'" class="text-xs" />
        </FaButton>
      </div>
    </div>

    <!-- 终端区（多实例 keep-alive：v-show 切换，保持会话与滚动位置） -->
    <div class="relative min-h-0 min-w-0 flex-1">
      <div
        v-for="t in tabs"
        :key="t.id"
        v-show="activeId === t.id"
        class="absolute inset-0"
      >
        <YdTerminal
          :ref="setRef(t.id)"
          :endpoint="{ kind: 'host', node: t.node }"
          :active="activeId === t.id"
          @input="onTabInput(t.id, $event)"
          @size="(c: number, r: number) => onTabSize(t.id, c, r)"
          @state="(s, text) => onTabState(t.id, s, text)"
          @cwd="(p: string) => onTabCwd(t.id, p)"
        />
      </div>
    </div>

    <!-- 底部状态栏 -->
    <StatusBar
      :state="activeTab?.state ?? 'closed'"
      :state-text="activeTab?.stateText"
      :node-label="activeTab ? nodeLabel(activeTab.node) : props.node"
      :size="activeTab?.size"
    />
  </div>
</template>
