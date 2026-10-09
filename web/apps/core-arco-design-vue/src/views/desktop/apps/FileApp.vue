<script setup lang="ts">
// 文件管理承载：webos 隐藏参数窗（多实例）。launchOptions 携带初始目录/节点。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import FileIndex from '@/views/file_management/index.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const dir = computed(() => {
  const v = props.win?.launchOptions?.dir
  return typeof v === 'string' && v ? v : undefined
})
const node = computed(() => {
  const v = props.win?.launchOptions?.node
  return typeof v === 'string' && v ? v : undefined
})
</script>

<template>
  <FileIndex :initial-dir="dir" :initial-node="node" />
</template>
