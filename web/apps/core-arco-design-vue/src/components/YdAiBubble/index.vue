<script setup lang="ts">
// YdAiBubble：AI 对话气泡（M60.1 参考 YuDream-Admin 去气泡化）——
// 助手消息透明排版流（Markdown 正文/深度思考折叠/工具块分段渲染/内联交互），
// 用户消息浅主色气泡（右下小尾巴、无头像）；操作栏 icon+文字 ghost 悬停主色。
import { computed } from 'vue'
import YdAiCitations from '@/components/YdAiChat/Citations.vue'
import YdAiMarkdown from '@/components/YdAiChat/Markdown.vue'
import YdAiProcess from '@/components/YdAiChat/Process.vue'
import YdAiReasoning from '@/components/YdAiChat/Reasoning.vue'
import YdAiToolBlock from '@/components/YdAiChat/ToolBlock.vue'

export interface AiBubbleSegment {
  type: 'text' | 'tool' | 'question'
  text?: string
  name?: string
  args?: string
  summary?: string
  outputId?: string
  output?: string
  status?: string
  risk?: string
  askId?: string
  done?: boolean
  answer?: string
  questions?: Array<{
    question: string
    header?: string
    multiSelect?: boolean
    options?: Array<{
      label: string
      description?: string
    }>
  }>
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
  steps?: Array<{
    type: string
    name?: string
    detail?: string
  }>
  knowledge?: Array<{
    title: string
    body?: string
  }>
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
const emit = defineEmits<{
  regenerate: [
  ]
}>()
const loading = computed(() => props.pending && !props.content)
async function copyText() {
  try {
    await navigator.clipboard.writeText(props.content)
  }
  catch {

  }
}
defineExpose({ copyText })
</script>

<template>
  <!-- 用户消息：右侧浅主色气泡（无头像、右下小尾巴） -->
  <div v-if="role === 'user'" class="group flex justify-end">
    <div class="flex flex-col max-w-[82%] min-w-0 items-end">
      <div class="text-sm text-foreground leading-relaxed px-4 py-2.5 rounded-[18px] rounded-br-[6px] bg-primary/10 whitespace-pre-wrap break-words [&_p]:my-0">
        {{ content }}
      </div>
      <button
        v-if="content && !pending"
        type="button"
        class="text-muted-foreground/60 mt-0.5 p-1 rounded opacity-60 cursor-pointer transition-all hover:text-foreground md:opacity-0 md:group-hover:opacity-100"
        :title="$t('common.copy')"
        @click="copyText"
      >
        <FaIcon name="i-lucide:copy" class="text-xs" />
      </button>
    </div>
  </div>

  <!-- 助手消息：头像 + 透明排版流 -->
  <div v-else class="flex gap-2.5">
    <div class="text-primary mt-0.5 rounded-[9px] bg-primary/12 flex shrink-0 size-[30px] items-center justify-center">
      <FaIcon name="i-ri:sparkling-2-line" class="text-[15px]" />
    </div>
    <div class="flex flex-1 flex-col min-w-0 items-start">
      <!-- 深度思考折叠块 -->
      <YdAiReasoning
        v-if="reasoning || (pending && !content)"
        class="w-full"
        :reasoning="reasoning"
        :streaming="pending && !content"
      />
      <!-- ZCode 式分段渲染：文本与工具块按时间序交错 -->
      <div v-if="segments.length" class="flex flex-col gap-2 w-full">
        <template v-for="(seg, si) in segments" :key="si">
          <div
            v-if="seg.type === 'text' && seg.text"
            class="w-full"
          >
            <YdAiMarkdown :text="seg.text" :streaming="pending" />
          </div>
          <YdAiToolBlock
            v-else-if="seg.type === 'tool'"
            :seg="{ name: seg.name || '', args: seg.args || '', summary: seg.summary || '', output: seg.output || '', outputId: seg.outputId, status: seg.status || '', risk: seg.risk, askId: seg.askId, done: seg.done }"
          />
          <div
            v-else-if="seg.type === 'question'"
            class="text-xs px-3 py-2 border border-primary/30 rounded-lg bg-primary/5 w-full"
          >
            <div class="font-medium flex gap-1.5 items-center">
              <FaIcon name="i-lucide:help-circle" class="text-primary" />
              {{ $t('components.ydAiChat.askUserWithCount', { n: (seg.questions || []).length }) }}
            </div>
            <div class="text-muted-foreground mt-1 space-y-0.5">
              <div v-for="(q, qi) in seg.questions || []" :key="qi">
                {{ qi + 1 }}. {{ q.question }}
              </div>
            </div>
            <div v-if="seg.answer || seg.summary" class="text-muted-foreground mt-1 pt-1 border-t border-primary/20 whitespace-pre-wrap">
              {{ seg.answer || seg.summary }}
            </div>
            <div v-else-if="!seg.done" class="text-primary mt-1">
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
          class="w-full"
        >
          <YdAiMarkdown :text="content" :streaming="pending" />
          <span v-if="pending" class="ml-0.5 align-middle rounded-sm bg-primary h-4 w-2 inline-block animate-pulse" />
        </div>
        <!-- 思考占位：三点跳动动画 -->
        <div v-else-if="pending" class="text-[13px] text-muted-foreground py-1 flex gap-1.5 items-center">
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
      <div v-if="showActions && !loading && content" class="text-xs text-muted-foreground mt-1 flex gap-1 items-center">
        <button
          type="button"
          class="px-1.5 py-1 rounded-md flex gap-1 cursor-pointer transition-colors items-center hover:text-primary hover:bg-accent/60"
          @click="copyText"
        >
          <FaIcon name="i-lucide:copy" class="text-xs" /> {{ $t('common.copy') }}
        </button>
        <button
          v-if="canRegenerate"
          type="button"
          class="px-1.5 py-1 rounded-md flex gap-1 cursor-pointer transition-colors items-center hover:text-primary hover:bg-accent/60"
          @click="emit('regenerate')"
        >
          <FaIcon name="i-lucide:refresh-cw" class="text-xs" /> {{ $t('components.ydAiChat.regenerate') }}
        </button>
      </div>
      <!-- 流异常中断提示（M60）：服务端未正常收尾即断开 -->
      <div
        v-if="interrupted && !pending"
        class="text-[11px] text-amber-600 mt-0.5 flex gap-1 items-center"
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
  background: oklch(var(--primary));
  border-radius: 9999px;
  animation: yd-dot-blink 1.2s infinite;
}

@keyframes yd-dot-blink {
  0%,
  60%,
  100% { opacity: 0.25; }
  30% { opacity: 1; }
}
</style>
