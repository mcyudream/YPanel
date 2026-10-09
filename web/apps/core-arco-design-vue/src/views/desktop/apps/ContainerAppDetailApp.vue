<script setup lang="ts">
// Compose 应用详情承载：webos 隐藏应用，launchOptions.project 来自应用列表卡片。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import Detail from '@/views/container/app-detail.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const project = computed(() => String(props.win?.launchOptions?.project ?? ''))
</script>

<template>
  <Detail v-if="project" :project="project" />
</template>
