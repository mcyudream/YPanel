<script setup lang="ts">
// 进程与服务承载：webos 应用窗（launchOptions.node 指定初始节点，供节点域下钻）。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import Processes from '@/views/processes/index.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const node = computed(() => {
  const v = props.win?.launchOptions?.node
  return typeof v === 'string' && v && v !== 'local' ? v : undefined
})
</script>

<template>
  <Processes :initial-node="node" />
</template>
