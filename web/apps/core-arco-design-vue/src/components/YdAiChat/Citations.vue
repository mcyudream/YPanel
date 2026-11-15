<script setup lang="ts">
// YdAiCitations：知识库引用来源折叠块（列出命中条目标题，点击展开正文，样式对齐深度思考块）。
import { ref } from 'vue'

defineProps<{
  knowledge: Array<{
    title: string
    body?: string
  }>
}>()
const expanded = ref(true)
const openIdx = ref<number>(-1)
function toggleBody(i: number) {
  openIdx.value = openIdx.value === i ? -1 : i
}
</script>

<template>
  <div class="mb-2 border border-border/60 rounded-md bg-muted/30 w-full overflow-hidden">
    <button
      type="button"
      class="text-xs text-muted-foreground px-2.5 py-1.5 flex gap-1.5 w-full cursor-pointer transition-colors items-center hover:bg-accent/40"
      @click="expanded = !expanded"
    >
      <FaIcon name="i-lucide:book-open" class="text-[11px]" />
      <span class="font-medium">{{ $t('components.ydAiChat.citations') }}</span>
      <span class="text-[10px] opacity-70">{{ $t('components.ydAiChat.citationCount', { n: knowledge.length }) }}</span>
      <FaIcon
        :name="expanded ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'"
        class="text-[11px] ml-auto"
      />
    </button>
    <div v-show="expanded" class="px-2.5 py-1.5 border-t border-border/40">
      <div v-for="(k, i) in knowledge" :key="i" class="mb-1 last:mb-0">
        <button
          type="button"
          class="text-xs text-foreground/80 px-1 py-0.5 text-left rounded flex gap-1.5 w-full cursor-pointer transition-colors items-center hover:bg-accent/40"
          @click="toggleBody(i)"
        >
          <FaIcon
            :name="openIdx === i ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'"
            class="text-[10px] text-muted-foreground shrink-0"
          />
          <span class="truncate">{{ k.title }}</span>
        </button>
        <div
          v-if="openIdx === i && k.body"
          class="text-[11px] text-muted-foreground leading-relaxed ml-5 mt-0.5 px-2 py-1.5 rounded bg-background/60 whitespace-pre-wrap break-words"
        >
          {{ k.body }}
        </div>
      </div>
    </div>
  </div>
</template>
