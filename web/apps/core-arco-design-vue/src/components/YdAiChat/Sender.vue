<script setup lang="ts">
// YdAiSender：Kimi/ZCode 风格输入发送器（M60.1 参考 YuDream-Admin 形态）——
// 20px 大圆角输入区 + 框内分段工具条（权限档位/模型/用量）+ 右下角圆形墨色发送键（↑）/停止（■）；
// Enter 发送 / Shift+Enter 换行 / 建议 chips / 免责声明；内容宽度约束 920px 居中，窄屏自适应。
const props = withDefaults(defineProps<{
  loading?: boolean
  disabled?: boolean
  placeholder?: string
  suggestions?: string[]
  disclaimer?: string
}>(), {
  loading: false,
  disabled: false,
  placeholder: undefined,
  suggestions: () => [],
  disclaimer: undefined,
})

const emit = defineEmits<{
  send: [text: string]
  stop: []
  suggestionClick: [text: string]
}>()

const text = ref('')
const focused = ref(false)

const canSend = computed(() => !!text.value.trim() && !props.loading && !props.disabled)

function doSend() {
  const t = text.value.trim()
  if (!t || props.loading || props.disabled) {
    return
  }
  text.value = ''
  emit('send', t)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    doSend()
  }
}

// 工具条自适应：底栏实测放不下（<520px）时根挂 yd-sender-compact，YdSelect 图标形态随之生效。
// 容器宽 ≠ 视口宽（桌面悬浮窗 520px 同样放不下文字标签），媒体查询管不到，必须实测容器。
const barRef = useTemplateRef<HTMLDivElement>('bar')
const compactBar = ref(false)
let barRo: ResizeObserver | null = null
onMounted(() => {
  barRo = new ResizeObserver((entries) => {
    compactBar.value = (entries[0]?.contentRect.width ?? 9999) < 520
  })
  if (barRef.value) {
    barRo.observe(barRef.value)
  }
})
onBeforeUnmount(() => {
  barRo?.disconnect()
  barRo = null
})
</script>

<template>
  <div class="shrink-0 border-t px-3 pb-3 pt-2 sm:px-4">
    <!-- 建议 chips -->
    <div v-if="suggestions.length && !loading" class="mx-auto mb-2 flex w-full max-w-[920px] flex-wrap gap-1.5">
      <button
        v-for="s in suggestions"
        :key="s"
        type="button"
        class="cursor-pointer rounded-full border bg-background/60 px-3 py-1 text-xs text-muted-foreground transition-colors hover:bg-accent/50 hover:text-foreground"
        @click="emit('suggestionClick', s)"
      >
        {{ s }}
      </button>
    </div>
    <!-- Kimi 风发送框：20px 大圆角 + 柔和阴影，聚焦阴影加深；窄容器挂 compact（YdSelect 图标形态） -->
    <div
      ref="bar"
      class="mx-auto w-full max-w-[920px] rounded-[20px] border bg-background shadow-sm transition-all"
      :class="[focused ? 'border-primary/30 shadow-md' : '', compactBar ? 'yd-sender-compact' : '']"
    >
      <textarea
        v-model="text"
        rows="3"
        :disabled="disabled"
        :placeholder="placeholder ?? $t('components.ydAiChat.inputPlaceholder')"
        class="block w-full resize-none bg-transparent px-4 pb-1 pt-3.5 text-[15px] leading-7 outline-none placeholder:text-muted-foreground/60 max-sm:min-h-[72px] max-sm:px-3.5 max-sm:text-sm"
        @keydown="onKeydown"
        @focus="focused = true"
        @blur="focused = false"
      />
      <!-- 底栏：外层禁 wrap（发送/停止键恒居右下），工具条在自身弹性空间内换行 -->
      <div class="flex items-end justify-between gap-2 px-2.5 pb-2.5">
        <span class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
          <!-- 分段工具条：权限档位 / 模型选择 / 上下文用量（由调用方注入） -->
          <slot name="toolbar" />
          <span class="hidden truncate text-[11px] text-muted-foreground/60 lg:inline">{{ disclaimer ?? $t('components.ydAiChat.disclaimer') }}</span>
          <!-- 左侧扩展操作（如清空对话） -->
          <slot name="actions" />
        </span>
        <!-- 圆形按钮：生成中 = 停止（■），否则 = 发送（↑，墨色实底） -->
        <FaButton
          v-if="loading"
          size="icon-sm"
          variant="outline"
          class="shrink-0 rounded-full transition-colors hover:border-red-500/50 hover:bg-red-500/10 hover:text-red-500"
          :title="$t('components.ydAiChat.stopGenerating')"
          @click="emit('stop')"
        >
          <FaIcon name="i-lucide:square" class="fill-current text-[10px]!" />
        </FaButton>
        <FaButton
          v-else
          size="icon-sm"
          class="shrink-0 rounded-full bg-foreground text-background transition-colors hover:bg-foreground/80"
          :disabled="!canSend"
          :title="$t('components.ydAiChat.send')"
          @click="doSend"
        >
          <FaIcon name="i-lucide:arrow-up" class="text-base!" />
        </FaButton>
      </div>
    </div>
  </div>
</template>
