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
  placeholder: '尽管问，或让我帮你做点什么…',
  suggestions: () => [],
  disclaimer: '内容由 YPanel AI 生成，请核对重要信息',
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
        :placeholder="placeholder"
        class="w-full resize-none bg-transparent text-sm outline-none placeholder:text-muted-foreground/70"
        @keydown="onKeydown"
      />
      <div class="mt-1 flex items-center justify-between">
        <span class="flex items-center gap-3">
          <span class="text-[11px] text-muted-foreground/70">{{ disclaimer }}</span>
          <!-- 左侧扩展操作（如清空对话） -->
          <slot name="actions" />
        </span>
        <FaButton v-if="loading" size="sm" variant="outline" @click="emit('stop')">
          停止生成
        </FaButton>
        <FaButton v-else size="sm" :disabled="!text.trim() || disabled" @click="doSend">
          发送
        </FaButton>
      </div>
    </div>
  </div>
</template>
