<script setup lang="ts">
// YdAiMessageList：消息列表容器（自动滚动到底部）。
import type { AiChatMessage } from '@/composables/useAiChat'
import { nextTick, ref, watch } from 'vue'

const props = defineProps<{
  messages: AiChatMessage[]
}>()

const listRef = ref<HTMLElement>()

watch(() => props.messages.map(m => m.content.length).join(','), async () => {
  await nextTick()
  const el = listRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}, { immediate: true })
</script>

<template>
  <div ref="listRef" class="flex-1 space-y-4 overflow-y-auto p-4">
    <slot />
  </div>
</template>
