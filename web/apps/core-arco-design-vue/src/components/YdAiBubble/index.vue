<script setup lang="ts">
// YdAiBubble：AI 对话气泡（Markdown 正文/深度思考折叠/工具块分段渲染/内联交互/操作栏）。
import { computed } from 'vue'
import { marked } from 'marked'
import YdAiProcess from '@/components/YdAiChat/Process.vue'
import YdAiReasoning from '@/components/YdAiChat/Reasoning.vue'
import YdAiCitations from '@/components/YdAiChat/Citations.vue'
import YdAiToolBlock from '@/components/YdAiChat/ToolBlock.vue'

export interface AiBubbleSegment {
  type: 'text' | 'tool' | 'question'
  text?: string
  name?: string
  args?: string
  summary?: string
  output?: string
  status?: string
  risk?: string
  askId?: string
  done?: boolean
  answer?: string
  questions?: Array<{ question: string, header?: string, multiSelect?: boolean, options?: Array<{ label: string, description?: string }> }>
}

const props = withDefaults(defineProps<{
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  showActions?: boolean
  reasoning?: string
  steps?: Array<{ type: string, name?: string, detail?: string }>
  knowledge?: Array<{ title: string, body?: string }>
  /** ZCode 式分段序列：文本与工具块按时间序交错（有值时优先于 content/steps 渲染） */
  segments?: AiBubbleSegment[]
}>(), {
  pending: false,
  showActions: false,
  reasoning: '',
  steps: () => [],
  knowledge: () => [],
  segments: () => [],
})

const emit = defineEmits<{ regenerate: [] }>()

const loading = computed(() => props.pending && !props.content)
// 用户气泡：深色实底（与助手明显区分）、自适应内容宽度、超宽换行
const bubbleClass = computed(() => props.role === 'user'
  ? 'w-fit max-w-full break-words bg-primary text-primary-foreground whitespace-pre-wrap [&_p]:my-0'
  : 'ai-md bg-muted/50 border border-border/60 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-muted/70 [&_pre]:p-2 [&_pre]:text-xs [&_code]:text-xs [&_ul]:list-disc [&_ol]:list-decimal [&_li]:ml-4 [&_h1]:my-2 [&_h2]:my-2 [&_h3]:my-2 [&_h1]:font-medium [&_h2]:font-medium [&_h3]:font-medium [&_p]:my-1 [&_a]:text-primary [&_a]:underline [&_table]:w-full [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1')

function renderMd(text?: string) {
  try {
    return marked.parse(text || '', { async: false, breaks: true }) as string
  }
  catch {
    return text || ''
  }
}

const html = computed(() => {
  if (props.role === 'user') {
    return ''
  }
  return renderMd(props.content)
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
    <div class="flex min-w-0 flex-1 flex-col items-start" :class="role === 'user' ? 'items-end' : ''">
      <!-- 深度思考折叠块 -->
      <YdAiReasoning
        v-if="role === 'assistant' && (reasoning || pending)"
        class="w-full"
        :reasoning="reasoning"
        :streaming="pending && !content"
      />
      <!-- ZCode 式分段渲染：文本与工具块按时间序交错 -->
      <div v-if="role === 'assistant' && segments.length" class="flex w-full flex-col gap-2">
        <template v-for="(seg, si) in segments" :key="si">
          <div
            v-if="seg.type === 'text' && seg.text"
            class="w-full rounded-lg border border-border/60 bg-muted/50 px-3 py-2 text-sm leading-relaxed ai-md [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-muted/70 [&_pre]:p-2 [&_pre]:text-xs [&_code]:text-xs [&_ul]:list-disc [&_ol]:list-decimal [&_li]:ml-4 [&_h1]:my-2 [&_h2]:my-2 [&_h3]:my-2 [&_h1]:font-medium [&_h2]:font-medium [&_h3]:font-medium [&_p]:my-1 [&_a]:text-primary [&_a]:underline [&_table]:w-full [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1"
          >
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div v-html="renderMd(seg.text)" />
          </div>
          <YdAiToolBlock
            v-else-if="seg.type === 'tool'"
            :seg="{ name: seg.name || '', args: seg.args || '', summary: seg.summary || '', output: seg.output || '', status: seg.status || '', risk: seg.risk, askId: seg.askId, done: seg.done }"
          />
          <div
            v-else-if="seg.type === 'question'"
            class="w-full rounded-lg border border-primary/30 bg-primary/5 px-3 py-2 text-xs"
          >
            <div class="flex items-center gap-1.5 font-medium">
              <FaIcon name="i-lucide:help-circle" class="text-primary" />
              {{ $t('components.ydAiChat.askUserWithCount', { n: (seg.questions || []).length }) }}
            </div>
            <div class="mt-1 space-y-0.5 text-muted-foreground">
              <div v-for="(q, qi) in seg.questions || []" :key="qi">
                {{ qi + 1 }}. {{ q.question }}
              </div>
            </div>
            <div v-if="seg.answer" class="mt-1 whitespace-pre-wrap border-t border-primary/20 pt-1 text-muted-foreground">
              {{ seg.answer }}
            </div>
            <div v-else-if="!seg.done" class="mt-1 text-primary">
              {{ $t('components.ydAiChat.answerBelow') }}
            </div>
          </div>
        </template>
      </div>
      <template v-else>
        <!-- 工具/步骤时间线（历史会话兼容渲染） -->
        <YdAiProcess
          v-if="role === 'assistant' && steps.length"
          class="w-full"
          :steps="steps"
          :streaming="pending && !content"
        />
        <div
          class="rounded-lg px-3 py-2 text-sm leading-relaxed"
          :class="bubbleClass"
        >
          <span v-if="loading" class="text-muted-foreground">{{ $t('components.ydAiChat.thinking') }}…</span>
          <template v-else-if="role === 'assistant'">
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div class="ai-md" v-html="html" />
            <span v-if="pending" class="ml-0.5 inline-block h-4 w-1.5 animate-pulse bg-foreground/60 align-middle" />
          </template>
          <template v-else>
            {{ content }}
          </template>
        </div>
      </template>
      <!-- 知识库引用来源（回答正文下方，可展开正文） -->
      <YdAiCitations
        v-if="role === 'assistant' && knowledge.length"
        class="w-full"
        :knowledge="knowledge"
      />
      <!-- 操作栏 -->
      <div v-if="showActions && !loading && content" class="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
        <button type="button" class="cursor-pointer hover:text-foreground" @click="copyText">
          <FaIcon name="i-lucide:copy" class="mr-0.5" /> {{ $t('common.copy') }}
        </button>
        <button type="button" class="cursor-pointer hover:text-foreground" @click="emit('regenerate')">
          <FaIcon name="i-lucide:refresh-cw" class="mr-0.5" /> {{ $t('components.ydAiChat.regenerate') }}
        </button>
      </div>
    </div>
  </div>
</template>
