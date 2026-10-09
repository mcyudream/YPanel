<script setup lang="ts">
// 内网浏览器承载（M42）：webos 隐藏参数窗（多实例）。launchOptions.url 携带初始地址。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import { computed } from 'vue'
import YdWebBrowser from '@/components/YdWebBrowser/index.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const url = computed(() => {
  const v = props.win?.launchOptions?.url
  return typeof v === 'string' && v ? v : undefined
})
</script>

<template>
  <YdWebBrowser :url="url" />
</template>
