<script setup lang="ts">
// 终端承载：webos 多实例窗（Dock 拖文件到终端图标时经 launchOptions.dropPaths 携带路径，
// 窗口内首个会话连接就绪后自动填入）。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import TerminalIndex from '@/views/terminal/index.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

/** 拖入的文件路径清单 → 引号包裹（含空格加双引号）拼为待粘贴串 */
const initialPaste = computed(() => {
  const paths = props.win?.launchOptions?.dropPaths
  if (!Array.isArray(paths) || !paths.length) {
    return ''
  }
  return paths
    .map(p => (typeof p === 'string' && /[\s';]/.test(p) ? `"${p.replaceAll('"', '\"')}"` : p))
    .join(' ')
})
</script>

<template>
  <TerminalIndex :initial-paste="initialPaste" />
</template>
