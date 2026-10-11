<script setup lang="ts">
// YdAiBubble：AI 对话气泡（M60.1 参考 YuDream-Admin 去气泡化）——
// 助手消息透明排版流（Markdown 正文/深度思考折叠/工具块分段渲染/内联交互），
// 用户消息浅主色气泡（右下小尾巴、无头像）；操作栏 icon+文字 ghost 悬停主色。
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
  /** 仅最后一条 assistant 消息为 true：重新生成按钮只出现在这里（M60） */
  canRegenerate?: boolean
  /** 流异常中断（未收到 done 事件）：气泡下方灰字提示（M60） */
  interrupted?: boolean
  reasoning?: string
  steps?: Array<{ type: string, name?: string, detail?: string }>
  knowledge?: Array<{ title: string, body?: string }>
  /** ZCode 式分段序列：文本与工具块按时间序交错（有值时优先于 content/steps 渲染） */
  segments?: AiBubbleSegment[]
}>(), {
  pending: false,
  showActions: false,
  canRegenerate: false,
  interrupted: false,
  reasoning: '',
  steps: () => [],
  knowledge: () => [],
  segments: () => [],
})

const emit = defineEmits<{ regenerate: [] }>()

const loading = computed(() => props.pending && !props.content)

// Markdown 正文子元素排版（透明流式排版，参考 YuDream-Admin yd-markdown）
const MD_CLS = '[&_pre]:overflow-auto [&_pre]:rounded-lg [&_pre]:bg-muted/70 [&_pre]:p-3 [&_pre]:text-xs [&_code]:text-xs [&_ul]:list-disc [&_ol]:list-decimal [&_li]:ml-4 [&_h1]:my-3 [&_h2]:my-2 [&_h3]:my-2 [&_h1]:text-base [&_h2]:text-[15px] [&_h3]:text-sm [&_h1]:font-semibold [&_h2]:font-semibold [&_h3]:font-medium [&_p]:my-1.5 [&_blockquote]:border-l-2 [&_blockquote]:border-primary/30 [&_blockquote]:pl-3 [&_blockquote]:text-muted-foreground [&_a]:text-primary [&_a]:underline [&_table]:w-full [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1'

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
  <!-- 用户消息：右侧浅主色气泡（无头像、右下小尾巴） -->
  <div v-if="role === 'user'" class="group flex justify-end">
    <div class="flex max-w-[82%] min-w-0 flex-col items-end">
      <div class="whitespace-pre-wrap break-words rounded-[18px] rounded-br-[6px] bg-primary/10 px-4 py-2.5 text-sm leading-relaxed text-foreground [&_p]:my-0">
        {{ content }}
      </div>
      <button
        v-if="content && !pending"
        type="button"
        class="mt-0.5 cursor-pointer rounded p-1 text-muted-foreground/60 opacity-60 transition-all md:opacity-0 md:group-hover:opacity-100 hover:text-foreground"
        :title="$t('common.copy')"
        @click="copyText"
      >
        <FaIcon name="i-lucide:copy" class="text-xs" />
      </button>
    </div>
  </div>

  <!-- 助手消息：头像 + 透明排版流 -->
  <div v-else class="flex gap-2.5">
    <div class="mt-0.5 flex size-[30px] shrink-0 items-center justify-center rounded-[9px] bg-primary/12 text-primary">
      <FaIcon name="i-ri:sparkling-2-line" class="text-[15px]" />
    </div>
    <div class="flex min-w-0 flex-1 flex-col items-start">
      <!-- 深度思考折叠块 -->
      <YdAiReasoning
        v-if="reasoning || pending"
        class="w-full"
        :reasoning="reasoning"
        :streaming="pending && !content"
      />
      <!-- ZCode 式分段渲染：文本与工具块按时间序交错 -->
      <div v-if="segments.length" class="flex w-full flex-col gap-2">
        <template v-for="(seg, si) in segments" :key="si">
          <div
            v-if="seg.type === 'text' && seg.text"
            class="w-full text-sm leading-relaxed ai-md"
            :class="MD_CLS"
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
          v-if="steps.length"
          class="w-full"
          :steps="steps"
          :streaming="pending && !content"
        />
        <!-- 正文：透明排版流 -->
        <div
          v-if="content"
          class="w-full text-sm leading-relaxed ai-md"
          :class="MD_CLS"
        >
          <!-- eslint-disable-next-line vue/no-v-html -->
          <div v-html="html" />
          <span v-if="pending" class="ml-0.5 inline-block h-4 w-2 animate-pulse rounded-sm bg-primary align-middle" />
        </div>
      <!-- 思考占位：三点跳动动画 -->
      <div v-else-if="pending" class="flex items-center gap-1.5 py-1 text-[13px] text-muted-foreground">
        <span class="yd-dot" />
        <span class="yd-dot [animation-delay:0.2s]" />
        <span class="yd-dot [animation-delay:0.4s]" />
        <span class="ml-1">{{ $t('components.ydAiChat.thinking') }}…</span>
      </div>
      </template>
      <!-- 知识库引用来源（回答正文下方，可展开正文） -->
      <YdAiCitations
        v-if="knowledge.length"
        class="w-full"
        :knowledge="knowledge"
      />
      <!-- 操作栏：icon+文字 ghost，悬停主色；重新生成仅最后一条助手消息 -->
      <div v-if="showActions && !loading && content" class="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
        <button
          type="button"
          class="flex cursor-pointer items-center gap-1 rounded-md px-1.5 py-1 transition-colors hover:bg-accent/60 hover:text-primary"
          @click="copyText"
        >
          <FaIcon name="i-lucide:copy" class="text-xs" /> {{ $t('common.copy') }}
        </button>
        <button
          v-if="canRegenerate"
          type="button"
          class="flex cursor-pointer items-center gap-1 rounded-md px-1.5 py-1 transition-colors hover:bg-accent/60 hover:text-primary"
          @click="emit('regenerate')"
        >
          <FaIcon name="i-lucide:refresh-cw" class="text-xs" /> {{ $t('components.ydAiChat.regenerate') }}
        </button>
      </div>
      <!-- 流异常中断提示（M60）：服务端未正常收尾即断开 -->
      <div
        v-if="interrupted && !pending"
        class="mt-0.5 flex items-center gap-1 text-[11px] text-amber-600"
      >
        <FaIcon name="i-lucide:wifi-off" class="text-[10px]" />
        {{ $t('components.ydAiChat.interrupted') }}
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 思考占位三点动画（参考 YuDream-Admin yd-dot blink） */
.yd-dot {
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: oklch(var(--primary));
  animation: yd-dot-blink 1.2s infinite;
}

@keyframes yd-dot-blink {
  0%, 60%, 100% { opacity: 0.25; }
  30% { opacity: 1; }
}
</style>
