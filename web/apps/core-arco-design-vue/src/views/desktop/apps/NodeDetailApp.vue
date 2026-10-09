<script setup lang="ts">
// 节点详情承载：webos 隐藏应用（dock/启动台不可见，multiInstance）。
// launchOptions.id 优先于路由参数（与容器/应用/站点详情承载一致）。
import type { WindowInstance } from '@yudream/yudream-webos-core'
import Detail from '@/views/nodes/detail.vue'

const props = defineProps<{
  win?: WindowInstance & { launchOptions?: Record<string, unknown> }
}>()

const nodeId = computed(() => String(props.win?.launchOptions?.id ?? ''))
</script>

<template>
  <Detail v-if="nodeId" :id="nodeId" />
</template>
