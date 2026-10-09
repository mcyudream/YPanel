<script setup lang="ts">
// 网站详情承载：webos 隐藏应用，launchOptions.id 来自网站列表「配置」动作。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import Detail from '@/views/sites/detail.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const siteId = computed(() => Number(props.win?.launchOptions?.id ?? 0))
</script>

<template>
  <Detail v-if="siteId" :id="siteId" />
</template>
