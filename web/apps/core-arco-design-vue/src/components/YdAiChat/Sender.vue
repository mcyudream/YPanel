<script setup lang="ts">
// YdAiSender：Kimi 风格输入发送器（Enter 发送 / Shift+Enter 换行 / loading 停止态 / 建议 chips / 免责声明）。
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
</script>

<template>
  <div class="border-t p-3">
    <!-- 建议 chips -->
    <div v-if="suggestions.length && !loading" class="mb-2 flex flex-wrap gap-1.5">
      <button
        v-for="s in suggestions"
        :key="s"
        type="button"
        class="cursor-pointer rounded-full border px-2.5 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-accent/50"
        @click="emit('suggestionClick', s)"
      >
        {{ s }}
      </button>
    </div>
    <div class="rounded-xl border bg-background p-2 transition-colors focus-within:border-primary">
      <textarea
        v-model="text"
        rows="3"
        :disabled="disabled"
        :placeholder="placeholder ?? $t('components.ydAiChat.inputPlaceholder')"
        class="w-full resize-none bg-transparent text-sm outline-none placeholder:text-muted-foreground/70"
        @keydown="onKeydown"
      />
      <div class="mt-1 flex items-center justify-between gap-2">
        <span class="flex min-w-0 items-center gap-2">
          <!-- ZCode 式发送框控件条：权限模式 / 模型选择 / 上下文用量（由调用方注入） -->
          <slot name="toolbar" />
          <span class="hidden truncate text-[11px] text-muted-foreground/70 sm:inline">{{ disclaimer ?? $t('components.ydAiChat.disclaimer') }}</span>
          <!-- 左侧扩展操作（如清空对话） -->
          <slot name="actions" />
        </span>
        <FaButton v-if="loading" size="sm" variant="outline" @click="emit('stop')">
          {{ $t('components.ydAiChat.stopGenerating') }}
        </FaButton>
        <FaButton v-else size="sm" :disabled="!text.trim() || disabled" @click="doSend">
          {{ $t('components.ydAiChat.send') }}
        </FaButton>
      </div>
    </div>
  </div>
</template>
