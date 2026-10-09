<script setup lang="ts">
// 容器详情承载：webos 隐藏应用（dock/启动台不可见，multiInstance）。
// 窗口内容组件接收 win prop，launchOptions.id 来自容器列表「开新窗」动作。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import Detail from '@/views/container/detail.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const containerId = computed(() => String(props.win?.launchOptions?.id ?? ''))
</script>

<template>
  <Detail v-if="containerId" :id="containerId" />
</template>
