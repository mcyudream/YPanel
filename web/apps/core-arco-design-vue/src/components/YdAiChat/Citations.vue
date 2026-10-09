<script setup lang="ts">
// YdAiCitations：知识库引用来源折叠块（列出命中条目标题，点击展开正文，样式对齐深度思考块）。
import { ref } from 'vue'

const props = defineProps<{
  knowledge: Array<{ title: string, body?: string }>
}>()

const expanded = ref(true)
const openIdx = ref<number>(-1)

function toggleBody(i: number) {
  openIdx.value = openIdx.value === i ? -1 : i
}
</script>

<template>
  <div class="mb-2 w-full overflow-hidden rounded-md border border-border/60 bg-muted/30">
    <button
      type="button"
      class="flex w-full cursor-pointer items-center gap-1.5 px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent/40"
      @click="expanded = !expanded"
    >
      <FaIcon name="i-lucide:book-open" class="text-[11px]" />
      <span class="font-medium">{{ $t('components.ydAiChat.citations') }}</span>
      <span class="text-[10px] opacity-70">{{ $t('components.ydAiChat.citationCount', { n: knowledge.length }) }}</span>
      <FaIcon
        :name="expanded ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'"
        class="ml-auto text-[11px]"
      />
    </button>
    <div v-show="expanded" class="border-t border-border/40 px-2.5 py-1.5">
      <div v-for="(k, i) in knowledge" :key="i" class="mb-1 last:mb-0">
        <button
          type="button"
          class="flex w-full cursor-pointer items-center gap-1.5 rounded px-1 py-0.5 text-left text-xs text-foreground/80 transition-colors hover:bg-accent/40"
          @click="toggleBody(i)"
        >
          <FaIcon
            :name="openIdx === i ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right'"
            class="shrink-0 text-[10px] text-muted-foreground"
          />
          <span class="truncate">{{ k.title }}</span>
        </button>
        <div
          v-if="openIdx === i && k.body"
          class="ml-5 mt-0.5 whitespace-pre-wrap break-words rounded bg-background/60 px-2 py-1.5 text-[11px] leading-relaxed text-muted-foreground"
        >
          {{ k.body }}
        </div>
      </div>
    </div>
  </div>
</template>
