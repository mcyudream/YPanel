<script setup lang="ts">
// YdAiReasoning：深度思考折叠块（流式展开，完成后可折叠，对齐参考图"深度思考"块）。
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  reasoning: string
  streaming?: boolean
}>(), {
  streaming: false,
})

const expanded = ref(true)

// 流式进行中保持展开；结束后自动折叠
watch(() => props.streaming, (v) => {
  if (!v) {
    expanded.value = false
  }
})

const chars = computed(() => props.reasoning.length)
</script>

<template>
  <div class="mb-2 overflow-hidden rounded-md border border-border/60 bg-muted/30">
    <button
      type="button"
      class="flex w-full cursor-pointer items-center gap-1.5 px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent/40"
      @click="expanded = !expanded"
    >
      <FaIcon name="i-lucide:brain" class="text-[11px]" />
      <span class="font-medium">深度思考</span>
      <span v-if="streaming" class="flex items-center gap-1">
        · <span class="animate-pulse">思考中</span>
      </span>
      <span v-else-if="chars" class="text-[10px] opacity-70">{{ chars }} 字</span>
      <FaIcon
        :name="expanded ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'"
        class="ml-auto text-[11px]"
      />
    </button>
    <div
      v-show="expanded"
      class="max-h-60 overflow-y-auto whitespace-pre-wrap border-t border-border/40 px-2.5 py-2 text-xs leading-relaxed text-muted-foreground"
    >
      {{ reasoning || '（空）' }}
    </div>
  </div>
</template>
