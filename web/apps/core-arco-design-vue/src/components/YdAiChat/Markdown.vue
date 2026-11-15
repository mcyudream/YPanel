<script setup lang="ts">
import { useHtmlDark } from '@/composables/useHtmlDark'
import { aiHighlight, aiMarkdown } from '@/utils/aiMarkdown'

const props = defineProps<{
  text: string
  streaming?: boolean
}>()
const isDark = useHtmlDark()
const highlights = shallowRef<Record<number, string>>({})
const blocks = computed(() => aiMarkdown(props.text || '', !props.streaming))

let version = 0
watch([() => props.text, () => props.streaming, isDark], async () => {
  const current = ++version
  highlights.value = {}
  if (props.streaming || !blocks.value.some(b => b.code !== undefined)) {
    return
  }
  const result: Record<number, string> = {}
  for (const [i, b] of blocks.value.entries()) {
    if (b.code !== undefined && b.code.length < 100000) {
      result[i] = await aiHighlight(b.code, b.language, isDark.value)
    }
  }
  if (current === version) {
    highlights.value = result
  }
}, { immediate: true })
function copy(code: string) {
  void navigator.clipboard.writeText(code)
}
function download(code: string, language: string) {
  const url = URL.createObjectURL(new Blob([code], { type: 'text/plain;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `code.${language.replace(/[^\w-]/g, '') || 'txt'}`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
</script>

<template>
  <div class="yd-ai-markdown">
    <template v-for="(block, i) in blocks" :key="i">
      <div v-if="block.code !== undefined" class="my-3 border rounded-xl bg-muted/30 min-w-0 overflow-hidden">
        <div class="text-xs text-muted-foreground px-3 py-1 border-b flex gap-2 items-center">
          <span class="font-mono mr-auto">{{ block.language }}</span>
          <FaButton size="sm" variant="ghost" :title="$t('aiAssistant.copyCode')" @click="copy(block.code)">
            <FaIcon name="i-lucide:copy" />
          </FaButton>
          <FaButton size="sm" variant="ghost" :title="$t('aiAssistant.download')" @click="download(block.code, block.language)">
            <FaIcon name="i-lucide:download" />
          </FaButton>
        </div>
        <!-- Only sanitized Monaco spans enter this HTML sink. -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <pre v-if="highlights[i]" class="text-xs leading-6 p-3 overflow-x-auto" v-html="highlights[i]" />
        <pre v-else class="text-xs leading-6 p-3 overflow-x-auto"><code>{{ block.code }}</code></pre>
      </div>
      <!-- eslint-disable-next-line vue/no-v-html -->
      <div v-else v-html="block.html" />
    </template>
  </div>
</template>

<style scoped>
.yd-ai-markdown {
  min-width: 0;
  font-size: 14px;
  line-height: 1.85;
  overflow-wrap: anywhere;
}

.yd-ai-markdown :deep(p) {
  margin: 0.5em 0;
}

.yd-ai-markdown :deep(h1),
.yd-ai-markdown :deep(h2),
.yd-ai-markdown :deep(h3) {
  margin: 1em 0 0.5em;
  font-weight: 600;
}

.yd-ai-markdown :deep(ul) {
  padding-left: 1.5em;
  list-style: disc;
}

.yd-ai-markdown :deep(ol) {
  padding-left: 1.5em;
  list-style: decimal;
}

.yd-ai-markdown :deep(a) {
  color: oklch(var(--primary));
  text-decoration: underline;
}

.yd-ai-markdown :deep(table) {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}

.yd-ai-markdown :deep(th),
.yd-ai-markdown :deep(td) {
  padding: 6px 10px;
  border: 1px solid oklch(var(--border));
}

.yd-ai-markdown :deep(blockquote) {
  padding-left: 1em;
  color: oklch(var(--muted-foreground));
  border-left: 3px solid oklch(var(--primary) / 30%);
}

.yd-ai-markdown :deep(pre) {
  overflow-wrap: normal;
  white-space: pre;
}

.yd-ai-markdown :deep(:not(pre) > code) {
  padding: 2px 4px;
  font-size: 0.9em;
  background: oklch(var(--muted) / 60%);
  border-radius: 4px;
}
</style>
