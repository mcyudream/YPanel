<script setup lang="ts">
// YdAiBubble：AI 对话气泡（参考 YDBubble 规格封装：Markdown 正文/思考中/光标/操作栏）。
import { computed } from 'vue'
import { marked } from 'marked'

const props = withDefaults(defineProps<{
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  showActions?: boolean
}>(), {
  pending: false,
  showActions: false,
})

const emit = defineEmits<{ regenerate: [] }>()

const loading = computed(() => props.pending && !props.content)
const bubbleClass = computed(() => props.role === 'user'
  ? 'bg-primary/10 whitespace-pre-wrap'
  : 'ai-md bg-muted/50 border border-border/60 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-muted/70 [&_pre]:p-2 [&_code]:text-xs [&_ul]:list-disc [&_ol]:list-decimal [&_li]:ml-4 [&_h1]:my-2 [&_h2]:my-2 [&_h3]:my-2 [&_h1]:font-medium [&_h2]:font-medium [&_h3]:font-medium [&_p]:my-1 [&_a]:text-primary [&_a]:underline [&_table]:w-full [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1')

const html = computed(() => {
  if (props.role === 'user') {
    return ''
  }
  try {
    return marked.parse(props.content || '', { async: false, breaks: true }) as string
  }
  catch {
    return props.content
  }
})

async function copyText() {
  try {
    await navigator.clipboard.writeText(props.content)
  }
  catch {}
}
defineExpose({ copyText })
</script>

<template>
  <div class="flex gap-2" :class="role === 'user' ? 'flex-row-reverse' : ''">
    <div
      class="mt-1 flex size-7 shrink-0 items-center justify-center rounded-full"
      :class="role === 'user' ? 'bg-primary/15 text-primary' : 'bg-muted text-foreground'"
    >
      <FaIcon :name="role === 'user' ? 'i-lucide:user' : 'i-ri:sparkling-2-line'" class="text-sm" />
    </div>
    <div
      class="max-w-85% rounded-lg px-3 py-2 text-sm leading-relaxed"
      :class="bubbleClass"
    >
      <span v-if="loading" class="text-muted-foreground">正在思考…</span>
      <template v-else-if="role === 'assistant'">
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="ai-md" v-html="html" />
        <span v-if="pending" class="ml-0.5 inline-block h-4 w-1.5 animate-pulse bg-foreground/60 align-middle" />
      </template>
      <template v-else>
        {{ content }}
      </template>
    </div>
  </div>
</template>
