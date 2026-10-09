<script setup lang="ts">
// YdAiToolBlock：ZCode 式工具块——对话流内一等块（参数折叠 + 可读摘要 + 原始输出滚动区默认可见），
// 危险操作挂起时内联批准/拒绝按钮（aiAsk store 驱动）。
import { computed, ref } from 'vue'
import { useAiAskStore } from '@/store/modules/aiAsk'
import { i18n } from '@/locales'

const props = defineProps<{
  seg: {
    name: string
    args: string
    summary: string
    output: string
    status: string
    risk?: string
    askId?: string
    done?: boolean
  }
}>()

const askStore = useAiAskStore()
const collapsed = ref(false)

const askPending = computed(() => !props.seg.done && askStore.payload?.id === props.seg.askId)

const stateText = computed(() => {
  if (askPending.value) {
    return i18n.global.t('components.ydAiChat.pendingConfirmShort')
  }
  if (!props.seg.done) {
    return i18n.global.t('components.ydAiChat.stateRunning')
  }
  return props.seg.status || i18n.global.t('components.ydAiChat.stateDone')
})

const stateClass = computed(() => {
  if (askPending.value) {
    return 'border-amber-500/50 bg-amber-500/10 text-amber-600'
  }
  if (!props.seg.done) {
    return 'border-primary/40 bg-primary/10 text-primary'
  }
  if (props.seg.status === '失败') {
    return 'border-red-500/40 bg-red-500/10 text-red-500'
  }
  return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600'
})

const icon = computed(() => {
  if (!props.seg.done) {
    return 'i-lucide:loader-circle'
  }
  if (props.seg.status === '失败') {
    return 'i-lucide:x'
  }
  return 'i-lucide:check'
})

function prettyArgs(raw: string) {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  }
  catch {
    return raw || ''
  }
}
</script>

<template>
  <div
    class="w-full overflow-hidden rounded-lg border bg-background/60 transition-colors"
    :class="askPending ? 'border-amber-500/50' : 'border-border/60'"
  >
    <!-- 头部：工具名 + 摘要 + 状态（点击折叠/展开） -->
    <button
      type="button"
      class="flex w-full cursor-pointer items-center gap-2 px-2.5 py-2 text-left text-xs"
      @click="collapsed = !collapsed"
    >
      <FaIcon
        :name="icon"
        class="shrink-0 text-[11px]"
        :class="!seg.done ? 'animate-spin text-primary' : seg.status === '失败' ? 'text-red-500' : 'text-emerald-600'"
      />
      <span class="shrink-0 font-medium">{{ seg.name }}</span>
      <span v-if="seg.summary" class="min-w-0 flex-1 truncate text-muted-foreground" :title="seg.summary">
        {{ seg.summary }}
      </span>
      <span class="shrink-0 rounded-full px-2 py-0.5 text-[10px]" :class="stateClass">{{ stateText }}</span>
      <FaIcon :name="collapsed ? 'i-lucide:chevron-down' : 'i-lucide:chevron-up'" class="shrink-0 text-[10px] text-muted-foreground" />
    </button>
    <!-- 展开区：可读摘要 + 原始输出（默认展开可见，输出区可滚动） -->
    <div v-if="!collapsed" class="space-y-1.5 border-t border-border/50 px-2.5 py-2">
      <div v-if="seg.summary" class="text-[11px] leading-relaxed text-muted-foreground">{{ seg.summary }}</div>
      <div v-if="seg.output">
        <div class="mb-0.5 text-[10px] font-medium text-muted-foreground/70">{{ $t('components.ydAiChat.rawOutput') }}</div>
        <pre class="max-h-52 overflow-y-auto whitespace-pre-wrap break-words rounded bg-muted/50 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-muted-foreground">{{ seg.output }}</pre>
      </div>
      <div v-if="seg.args && seg.args !== seg.output">
        <div class="mb-0.5 text-[10px] font-medium text-muted-foreground/70">{{ $t('components.ydAiChat.args') }}</div>
        <pre class="max-h-28 overflow-y-auto whitespace-pre-wrap break-words rounded bg-muted/50 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-muted-foreground">{{ prettyArgs(seg.args) }}</pre>
      </div>
      <div v-if="askPending" class="pt-1 text-[11px] text-amber-600">
        <FaIcon name="i-lucide:arrow-down" class="mr-0.5" />{{ $t('components.ydAiChat.approveHint') }}
      </div>
    </div>
  </div>
</template>
