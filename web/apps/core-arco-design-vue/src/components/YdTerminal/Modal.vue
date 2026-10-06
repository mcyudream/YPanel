<script setup lang="ts">
import type { SplitpanesResizedPayload } from 'splitpanes'
import type { FileEntry } from '@/api/modules/file'
import type { TerminalConnState, TerminalEngine } from './types'
import { useFileEditorStore } from '@/store/modules/fileEditor'
import { useLocalStorage } from '@vueuse/core'
import { Pane, Splitpanes } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import YdFileTree from '../YdFileTree/index.vue'
import YdTerminal from './index.vue'
import StatusBar from './StatusBar.vue'

// 终端弹窗（容器 exec 用）：FaModal + 左侧容器文件树（可折叠拖宽）+ 单会话终端 + 状态栏。
// 容器 exec 为裸文本帧协议且无 resize，默认 xterm 引擎（TUI 安全）。
const props = withDefaults(defineProps<{
  open: boolean
  title: string
  containerId: string
  cmd?: string
}>(), {
  cmd: '/bin/sh',
})

const emits = defineEmits<{
  'update:open': [open: boolean]
}>()

const fileEditorStore = useFileEditorStore()

const engine = ref<TerminalEngine>('xterm')
const shell = ref(props.cmd)
const treeVisible = useLocalStorage('ypanel.terminal.modalTree', true)
const treeWidth = useLocalStorage('ypanel.terminal.modalTreeWidth', 28)

const endpoint = computed(() => ({
  kind: 'exec' as const,
  containerId: props.containerId,
  cmd: shell.value,
}))

const state = ref<TerminalConnState>('connecting')
const stateText = ref<string | undefined>()

function switchEngine() {
  engine.value = engine.value === 'xterm' ? 'vwt' : 'xterm'
}

const shellOptions = [
  { label: '/bin/sh', value: '/bin/sh' },
  { label: '/bin/bash', value: '/bin/bash' },
  { label: '/bin/ash', value: '/bin/ash' },
]

function onTreeResized(e: SplitpanesResizedPayload) {
  const size = Math.round(e.panes[0]?.size * 10) / 10
  if (size) {
    treeWidth.value = size
  }
}

// 双击/打开文件 → 编辑器工作台（容器维度，保存写回容器）
function openInEditor(entry: FileEntry) {
  fileEditorStore.openWorkspace(entry.path, 'local', props.containerId)
}
</script>

<template>
  <FaModal
    :model-value="props.open"
    :title="props.title"
    class="max-w-5xl!"
    :footer="false"
    :destroy-on-close="true"
    @update:model-value="emits('update:open', Boolean($event))"
  >
    <div class="flex h-[560px] flex-col overflow-hidden rounded-md border bg-background">
      <!-- 迷你工具栏 -->
      <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-2 text-[13px]">
        <YdMorphIcon name="square-terminal" :size="13" class="shrink-0 text-muted-foreground" />
        <FaSelect
          v-model="shell"
          :options="shellOptions"
          class="w-28 shrink-0"
          title="切换 Shell（将重开会话）"
        />
        <div class="ml-auto flex items-center gap-0.5">
          <FaButton variant="ghost" size="icon-sm" class="size-5!" :title="treeVisible ? '收起文件树' : '展开文件树'" @click="treeVisible = !treeVisible">
            <FaIcon :name="treeVisible ? 'i-lucide:panel-left-close' : 'i-lucide:panel-left'" class="text-xs" />
          </FaButton>
          <FaButton variant="ghost" size="sm" class="h-5! px-1.5! text-[11px]" title="切换终端引擎" @click="switchEngine">
            {{ engine === 'xterm' ? 'xterm' : 'vwt' }}
          </FaButton>
        </div>
      </div>

      <!-- 文件树 | 终端 -->
      <Splitpanes class="yd-term-modal min-h-0 flex-1" @resized="onTreeResized">
        <Pane
          v-if="treeVisible"
          :size="treeWidth"
          min-size="15"
          max-size="45"
          class="border-r bg-muted/20"
        >
          <YdFileTree :container-id="props.containerId" hide-filter @open-file="openInEditor" />
        </Pane>
        <Pane min-size="30" class="min-w-0">
          <div class="relative size-full min-h-0">
            <YdTerminal
              v-if="props.open"
              :key="`${props.containerId}:${shell}:${engine}`"
              :endpoint="endpoint"
              :engine="engine"
              @state="(s, text) => { state = s; stateText = text }"
            />
          </div>
        </Pane>
      </Splitpanes>

      <StatusBar :state="state" :state-text="stateText" node-label="容器 exec" :engine="engine" />
    </div>
  </FaModal>
</template>

<style>
.yd-term-modal .splitpanes__splitter {
  position: relative;
  width: 5px;
  margin: 0 -2px;
  z-index: 5;
  cursor: col-resize;
}
.yd-term-modal .splitpanes__splitter::before {
  content: '';
  position: absolute;
  inset: 0 2px;
  background: transparent;
  transition: background 0.15s;
}
.yd-term-modal .splitpanes__splitter:hover::before,
.yd-term-modal .splitpanes--dragging .splitpanes__splitter::before {
  background: oklch(var(--primary));
}
</style>
