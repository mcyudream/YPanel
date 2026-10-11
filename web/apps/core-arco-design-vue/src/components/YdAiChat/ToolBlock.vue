<script setup lang="ts">
// YdAiToolBlock：工具调用块（M60.1 参考 YuDream-Admin 胶囊化）——头部为 pill 胶囊
// （状态色 icon + 工具名 + 摘要 + 状态徽标），点击展开参数/原始输出；危险操作挂起时
// 内联批准/拒绝提示（aiAsk store 驱动）。
import { computed, ref, watch } from 'vue'
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
const collapsed = ref(true)

// 仅真正挂了 askId 的工具块参与「待确认」判定——否则 undefined === undefined
// 会把运行中的普通工具块（无 ask，如 full 档任务）误标成待确认（M62 修复）
const askPending = computed(() =>
  !props.seg.done && !!props.seg.askId && askStore.payload?.id === props.seg.askId,
)

// 危险操作挂起时自动展开参数区，方便核对后到下方批准
watch(askPending, (v) => {
  if (v) {
    collapsed.value = false
  }
})

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
  // M60：RBAC 无权限单列（区别于普通失败，去角色管理开通即可解决）
  if (props.seg.status === '无权限') {
    return 'border-orange-500/50 bg-orange-500/10 text-orange-600'
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
  if (props.seg.status === '无权限') {
    return 'i-lucide:lock'
  }
  return 'i-lucide:check'
})

const iconClass = computed(() => {
  if (!props.seg.done) {
    return 'animate-spin text-primary'
  }
  if (props.seg.status === '失败') {
    return 'text-red-500'
  }
  if (props.seg.status === '无权限') {
    return 'text-orange-500'
  }
  return 'text-emerald-600'
})

// 胶囊上的可读摘要：挂起/运行/失败优先展示，完成态优先展示结果摘要
const pillText = computed(() => {
  if (askPending.value || !props.seg.done) {
    return props.seg.summary || props.seg.name
  }
  return props.seg.summary || ''
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
  <div class="w-full">
    <!-- 头部胶囊：状态 icon + 工具名 + 摘要（长结果限三行换行，不横向撑页） + 状态徽标（点击折叠/展开） -->
    <button
      type="button"
      class="flex max-w-full cursor-pointer items-start gap-2 rounded-2xl border border-border/60 bg-background/80 py-1.5 pl-2.5 pr-2 text-left text-xs transition-colors hover:border-border"
      :class="askPending ? 'border-amber-500/50' : ''"
      @click="collapsed = !collapsed"
    >
      <FaIcon :name="icon" class="mt-0.5 shrink-0 text-[11px]" :class="iconClass" />
      <span class="shrink-0 font-medium text-foreground">{{ seg.name }}</span>
      <span v-if="pillText" class="min-w-0 flex-1 line-clamp-3 break-words text-muted-foreground" :title="pillText">
        {{ pillText }}
      </span>
      <span class="mt-0.5 shrink-0 rounded-full px-2 py-0.5 text-[10px]" :class="stateClass">{{ stateText }}</span>
      <FaIcon :name="collapsed ? 'i-lucide:chevron-down' : 'i-lucide:chevron-up'" class="mt-0.5 shrink-0 text-[10px] text-muted-foreground" />
    </button>
    <!-- 展开区：可读摘要 + 原始输出（默认折叠为胶囊，点开可见） -->
    <div v-if="!collapsed" class="mt-1.5 space-y-1.5 rounded-lg border border-border/50 bg-muted/20 px-2.5 py-2">
      <div v-if="seg.summary" class="text-[11px] leading-relaxed text-muted-foreground">
        {{ seg.summary }}
      </div>
      <div v-if="seg.output">
        <div class="mb-0.5 text-[10px] font-medium text-muted-foreground/70">
          {{ $t('components.ydAiChat.rawOutput') }}
        </div>
        <pre class="max-h-52 overflow-y-auto whitespace-pre-wrap break-words rounded bg-muted/50 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-muted-foreground">{{ seg.output }}</pre>
      </div>
      <div v-if="seg.args && seg.args !== seg.output">
        <div class="mb-0.5 text-[10px] font-medium text-muted-foreground/70">
          {{ $t('components.ydAiChat.args') }}
        </div>
        <pre class="max-h-28 overflow-y-auto whitespace-pre-wrap break-words rounded bg-muted/50 px-2 py-1.5 font-mono text-[11px] leading-relaxed text-muted-foreground">{{ prettyArgs(seg.args) }}</pre>
      </div>
      <div v-if="askPending" class="pt-1 text-[11px] text-amber-600">
        <FaIcon name="i-lucide:arrow-down" class="mr-0.5" />{{ $t('components.ydAiChat.approveHint') }}
      </div>
    </div>
  </div>
</template>
