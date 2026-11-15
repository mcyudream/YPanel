<script setup lang="ts">
import type { DraftAttachment } from '@/store/modules/aiChat'

const props = defineProps<{
  loading?: boolean
  sending?: boolean
  disabled?: boolean
  uploads?: DraftAttachment[]
  vision?: boolean
}>()
const emit = defineEmits<{
  send: [
  ]
  stop: [
  ]
  files: [
    File[],
  ]
  remove: [
    string,
  ]
  retry: [
    string,
  ]
}>()
const text = defineModel<string>({ default: '' })
const fileInput = ref<HTMLInputElement>()
const textarea = ref<HTMLTextAreaElement>()
const dragging = ref(false)
const composing = ref(false)
const ready = computed(() => !props.loading && !props.sending && !props.disabled && (text.value.trim() || props.uploads?.length) && !props.uploads?.some(u => !u.attachment || u.error) && (props.vision || !props.uploads?.some(u => u.attachment?.kind === 'image')))
function send() {
  if (ready.value) {
    emit('send')
  }
}
function keydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing && !composing.value && e.keyCode !== 229) {
    e.preventDefault()
    send()
  }
}
function files(list: FileList | null) {
  if (list) {
    emit('files', Array.from(list))
  }
}
function drop(e: DragEvent) {
  dragging.value = false
  if (!props.loading && !props.sending) {
    files(e.dataTransfer?.files || null)
  }
}
function paste(e: ClipboardEvent) {
  const images = Array.from(e.clipboardData?.files || []).filter(f => f.type.startsWith('image/'))
  if (images.length) {
    e.preventDefault()
    emit('files', images)
  }
}
watch(text, () => nextTick(() => {
  if (textarea.value) {
    textarea.value.style.height = 'auto'
    textarea.value.style.height = `${Math.min(220, textarea.value.scrollHeight)}px`
  }
}))
</script>

<template>
  <div class="px-3 pb-3 pt-2 border-t bg-background/80 shrink-0 sm:px-4" @dragover.prevent="dragging = true" @dragleave.prevent="dragging = false" @drop.prevent="drop">
    <div class="mx-auto border rounded-2xl bg-background max-w-[920px] w-full shadow-sm transition-colors focus-within:border-primary/50" :class="dragging ? 'border-primary ring-2 ring-primary/20' : ''">
      <div v-if="dragging" class="text-sm text-primary p-4 text-center">
        {{ $t('aiAssistant.drop') }}
      </div>
      <div v-if="uploads?.length" class="p-3 pb-0 flex gap-2 overflow-x-auto">
        <div v-for="u in uploads" :key="u.key" class="text-xs p-2 border rounded-xl flex shrink-0 gap-2 w-52 items-center relative" :class="u.error ? 'border-red-500/50' : ''">
          <img v-if="u.preview" :src="u.preview" alt="" class="rounded-lg size-10 object-cover">
          <FaIcon v-else name="i-lucide:file-text" class="text-muted-foreground shrink-0 size-8" />
          <div class="flex-1 min-w-0">
            <div class="truncate" :title="u.file?.name || u.attachment?.name">
              {{ u.file?.name || u.attachment?.name }}
            </div><div v-if="u.error" class="text-red-500 mt-1" :title="u.error">
              {{ u.error }}
            </div><div v-else-if="u.progress < 100" class="text-muted-foreground mt-1">
              {{ $t('aiAssistant.parsing') }} {{ u.progress }}%
            </div><div v-else class="text-muted-foreground mt-1">
              {{ Math.ceil((u.file?.size || u.attachment?.size || 0) / 1024) }} KB
            </div>
          </div>
          <FaButton v-if="u.error && u.file" size="icon-sm" variant="ghost" :title="$t('aiAssistant.retry')" @click="emit('retry', u.key)">
            <FaIcon name="i-lucide:refresh-cw" />
          </FaButton>
          <FaButton size="icon-sm" variant="ghost" :disabled="sending" :title="$t('common.delete')" @click="emit('remove', u.key)">
            <FaIcon name="i-lucide:x" />
          </FaButton>
        </div>
      </div>
      <div v-if="!vision && uploads?.some(u => u.attachment?.kind === 'image')" class="text-xs text-amber-600 px-3 pt-2">
        {{ $t('aiAssistant.visionOff') }}
      </div>
      <textarea ref="textarea" v-model="text" :disabled="disabled || sending" rows="2" :placeholder="$t('ai.chat.inputPlaceholder')" class="text-sm leading-6 px-4 py-3 outline-none bg-transparent max-h-56 min-h-20 w-full block resize-none placeholder:text-muted-foreground/60" @keydown="keydown" @compositionstart="composing = true" @compositionend="composing = false" @paste="paste" />
      <div class="px-2 pb-2 flex gap-1 min-w-0 items-end">
        <input
          ref="fileInput" type="file" multiple class="hidden" accept=".png,.jpg,.jpeg,.webp,.txt,.md,.log,.json,.yaml,.yml,.toml,.csv,.go,.ts,.tsx,.js,.jsx,.vue,.py,.sh,.sql,.html,.css,.scss,.xml,.conf,.ini,.rs,.java,.c,.cpp,.h,.rb,.php,.pdf,.docx" @change="files(($event.target as HTMLInputElement).files);
                                                                                                                                                                                                                                                                  ($event.target as HTMLInputElement).value = ''"
        >
        <FaButton size="icon-sm" variant="ghost" :disabled="loading || sending || disabled" :title="$t('aiAssistant.uploadHint')" :aria-label="$t('aiAssistant.upload')" @click="fileInput?.click()">
          <FaIcon name="i-lucide:paperclip" />
        </FaButton>
        <div class="flex flex-1 flex-wrap gap-1 min-w-0 items-center">
          <slot name="toolbar" />
        </div>
        <FaButton v-if="loading" size="icon-sm" variant="outline" class="text-red-500 rounded-full shrink-0" :title="$t('components.ydAiChat.stopGenerating')" @click="emit('stop')">
          <FaIcon name="i-lucide:square" />
        </FaButton>
        <FaButton v-else size="icon-sm" class="rounded-full shrink-0" :disabled="!ready" :loading="sending" :title="$t('aiAssistant.send')" @click="send">
          <FaIcon name="i-lucide:arrow-up" />
        </FaButton>
      </div>
    </div>
    <p class="text-[10px] text-muted-foreground mx-auto mt-1.5 text-center max-w-[920px]">
      {{ $t('aiAssistant.background') }}
    </p>
  </div>
</template>
