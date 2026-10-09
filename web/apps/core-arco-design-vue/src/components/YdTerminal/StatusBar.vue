<script setup lang="ts">
import type { TerminalConnState } from './types'
import { i18n } from '@/locales'

// 终端状态栏：连接状态 / 节点 / 尺寸，VS Code 式纤细单行。
const props = defineProps<{
  state: TerminalConnState
  stateText?: string
  nodeLabel: string
  size?: { cols: number, rows: number }
  extra?: string
}>()

const stateMeta = computed(() => {
  switch (props.state) {
    case 'connected':
      return { dot: 'bg-emerald-500', text: i18n.global.t('components.ydTerminal.stateConnected') }
    case 'connecting':
      return { dot: 'bg-amber-500 animate-pulse', text: i18n.global.t('components.ydTerminal.stateConnecting') }
    case 'error':
      return { dot: 'bg-red-500', text: props.stateText || i18n.global.t('components.ydTerminal.stateError') }
    default:
      return { dot: 'bg-muted-foreground/60', text: props.stateText || i18n.global.t('components.ydTerminal.stateDisconnected') }
  }
})
</script>

<template>
  <div class="flex h-6 shrink-0 items-center gap-3 border-t bg-muted/40 px-2 text-[11px] text-muted-foreground">
    <span class="inline-flex items-center gap-1.5">
      <span class="inline-block size-1.5 rounded-full" :class="stateMeta.dot" />
      {{ stateMeta.text }}
    </span>
    <span class="inline-flex items-center gap-1">
      <YdMorphIcon name="server" :size="11" />
      {{ nodeLabel }}
    </span>
    <span v-if="size" class="tabular-nums">
      {{ size.cols }}×{{ size.rows }}
    </span>
    <span v-if="extra" class="truncate" :title="extra">{{ extra }}</span>
    <slot />
  </div>
</template>
