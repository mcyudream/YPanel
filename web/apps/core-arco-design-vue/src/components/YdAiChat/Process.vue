<script setup lang="ts">
// YdAiProcess：工具/步骤流程时间线（竖向连线 + 状态节点 + 工具卡片）。
// 每张卡片三态：执行中（spinner，蓝）/ 完成（✓，绿）/ 失败（✗，红）；工具按名称单独设计图标与文案。
// 点击卡片展开完整入参与结果详情。
import { computed, ref } from 'vue'
import type { AiChatStep } from '@/composables/useAiChat'
import { i18n } from '@/locales'
import { toolMetaOf } from '@/components/YdAiChat/toolMeta'

const props = withDefaults(defineProps<{
  steps: AiChatStep[]
  streaming?: boolean
}>(), {
  streaming: false,
})

// 展开详情的卡片（索引集合）
const expanded = ref(new Set<number>())


function toggle(i: number) {
  const next = new Set(expanded.value)
  if (next.has(i)) {
    next.delete(i)
  }
  else {
    next.add(i)
  }
  expanded.value = next
}

// 内置工具元数据由 toolMeta.ts 共享（系统工具管理页同源）
function meta(s: AiChatStep) {
  if (s.type === 'scene') {
    return { label: i18n.global.t('components.ydAiChat.stepReadPage'), icon: 'i-lucide:eye' }
  }
  if (s.type === 'ask') {
    return { label: i18n.global.t('components.ydAiChat.pendingConfirm', { tool: toolMetaOf(s.name || '').label }), icon: 'i-lucide:shield-alert' }
  }
  if (s.type === 'question') {
    return { label: i18n.global.t('components.ydAiChat.stepAskUser'), icon: 'i-lucide:help-circle' }
  }
  if (s.type === 'tool' || s.type === 'tool_result' || s.type === 'action') {
    return toolMetaOf(s.name || '')
  }
  return { label: s.type, icon: 'i-lucide:wrench' }
}

type StepState = 'running' | 'done' | 'error' | 'stopped'

function state(s: AiChatStep): StepState {
  if (s.status === '失败' || s.status === '已拒绝' || s.status === '已超时' || s.status === '已取消') {
    return 'error'
  }
  if (s.done || s.status === '完成') {
    return 'done'
  }
  return props.streaming ? 'running' : 'stopped'
}

// 展示态文案（数据里的 status 中文枚举是后端/前端写入的数据值，不在此翻译）
const stateLabel = computed<Record<StepState, string>>(() => ({
  running: i18n.global.t('components.ydAiChat.stateRunning'),
  done: i18n.global.t('components.ydAiChat.stateDone'),
  error: i18n.global.t('components.ydAiChat.stateError'),
  stopped: i18n.global.t('components.ydAiChat.stateStopped'),
}))

const stateClass: Record<StepState, string> = {
  running: 'border-primary/40 bg-primary/10 text-primary',
  done: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600',
  error: 'border-red-500/40 bg-red-500/10 text-red-500',
  stopped: 'border-border bg-muted text-muted-foreground',
}

const stateTextClass: Record<StepState, string> = {
  running: 'text-primary',
  done: 'text-emerald-600',
  error: 'text-red-500',
  stopped: 'text-muted-foreground',
}
</script>

<template>
  <div class="mb-2 rounded-md border border-border/60 bg-muted/30 px-2.5 py-2">
    <div
      v-for="(s, i) in steps"
      :key="i"
      class="relative flex gap-2.5 pb-2 last:pb-0"
    >
      <!-- 连接线 -->
      <span
        v-if="i < steps.length - 1"
        class="absolute bottom-0 left-[11px] top-7 w-px bg-border"
      />
      <!-- 状态节点 -->
      <span
        class="z-1 mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border text-[11px]"
        :class="stateClass[state(s)]"
      >
        <FaIcon v-if="state(s) === 'running'" name="i-lucide:loader-circle" class="animate-spin" />
        <FaIcon v-else-if="state(s) === 'error'" name="i-lucide:x" />
        <FaIcon v-else-if="state(s) === 'done'" name="i-lucide:check" />
        <FaIcon v-else name="i-lucide:minus" />
      </span>
      <!-- 工具卡片（点击展开详情） -->
      <div
        class="min-w-0 flex-1 cursor-pointer rounded-md transition-colors hover:bg-accent/30"
        @click="toggle(i)"
      >
        <div class="flex items-center gap-1.5 text-xs">
          <FaIcon :name="meta(s).icon" class="text-[11px] text-muted-foreground" />
          <span class="font-medium">{{ meta(s).label }}</span>
          <span class="ml-auto shrink-0 text-[10px]" :class="stateTextClass[state(s)]">
            {{ stateLabel[state(s)] }}
          </span>
          <FaIcon
            :name="expanded.has(i) ? 'i-lucide:chevron-up' : 'i-lucide:chevron-down'"
            class="shrink-0 text-[10px] text-muted-foreground"
          />
        </div>
        <div v-if="s.detail && !expanded.has(i)" class="mt-0.5 truncate text-[11px] text-muted-foreground" :title="s.detail">
          {{ s.detail }}
        </div>
        <div v-if="s.summary && !expanded.has(i)" class="truncate text-[11px] text-muted-foreground/80" :class="s.detail ? '' : 'mt-0.5'" :title="s.summary">
          {{ s.summary }}
        </div>
        <!-- 展开态：完整入参与结果 -->
        <div v-if="expanded.has(i)" class="mt-1 space-y-1.5">
          <div v-if="s.detail">
            <div class="text-[10px] font-medium text-muted-foreground/70">{{ $t('components.ydAiChat.args') }}</div>
            <div class="mt-0.5 max-h-40 overflow-y-auto whitespace-pre-wrap break-words rounded bg-background/60 px-2 py-1.5 text-[11px] leading-relaxed text-muted-foreground">{{ s.detail }}</div>
          </div>
          <div v-if="s.output">
            <div class="text-[10px] font-medium" :class="s.status === '失败' ? 'text-red-500' : 'text-emerald-600'">{{ $t('components.ydAiChat.rawOutputWithStatus', { status: s.status || $t('components.ydAiChat.stateEnded') }) }}</div>
            <div class="mt-0.5 max-h-60 overflow-y-auto whitespace-pre-wrap break-words rounded bg-background/60 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-muted-foreground">{{ s.output }}</div>
          </div>
          <div v-else-if="s.summary">
            <div class="text-[10px] font-medium" :class="s.status === '失败' ? 'text-red-500' : 'text-emerald-600'">{{ $t('components.ydAiChat.resultWithStatus', { status: s.status || $t('components.ydAiChat.stateEnded') }) }}</div>
            <div class="mt-0.5 max-h-60 overflow-y-auto whitespace-pre-wrap break-words rounded bg-background/60 px-2 py-1.5 text-[11px] leading-relaxed text-muted-foreground">{{ s.summary }}</div>
          </div>
          <div v-if="!s.detail && !s.summary" class="text-[11px] text-muted-foreground/60">
            {{ $t('components.ydAiChat.noDetail') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
