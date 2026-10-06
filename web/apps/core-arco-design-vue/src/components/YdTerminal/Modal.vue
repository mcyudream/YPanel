<script setup lang="ts">
import type { TerminalConnState, TerminalEngine } from './types'
import YdTerminal from './index.vue'
import StatusBar from './StatusBar.vue'

// 终端弹窗（容器 exec 用）：FaModal + 单会话 YdTerminal + 迷你工具栏（引擎/Shell 切换）+ 状态栏。
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

const engine = ref<TerminalEngine>('xterm')
const shell = ref(props.cmd)

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
</script>

<template>
  <FaModal
    :model-value="props.open"
    :title="props.title"
    class="max-w-4xl!"
    :footer="false"
    :destroy-on-close="true"
    @update:model-value="emits('update:open', Boolean($event))"
  >
    <div class="flex h-[540px] flex-col overflow-hidden rounded-md border">
      <!-- 迷你工具栏 -->
      <div class="flex h-8 shrink-0 items-center gap-1 border-b bg-muted/40 px-2 text-[13px]">
        <YdMorphIcon name="square-terminal" :size="13" class="text-muted-foreground" />
        <FaSelect
          v-model="shell"
          :options="shellOptions"
          class="w-28 shrink-0"
          title="切换 Shell（将重开会话）"
        />
        <div class="ml-auto flex items-center gap-0.5">
          <FaButton variant="ghost" size="sm" class="h-5! px-1.5! text-[11px]" title="切换终端引擎" @click="switchEngine">
            {{ engine === 'xterm' ? 'xterm' : 'vwt' }}
          </FaButton>
        </div>
      </div>

      <div class="relative min-h-0 flex-1">
        <YdTerminal
          v-if="props.open"
          :key="`${props.containerId}:${shell}:${engine}`"
          :endpoint="endpoint"
          :engine="engine"
          @state="(s, text) => { state = s; stateText = text }"
        />
      </div>

      <StatusBar :state="state" :state-text="stateText" node-label="容器 exec" :engine="engine" />
    </div>
  </FaModal>
</template>
