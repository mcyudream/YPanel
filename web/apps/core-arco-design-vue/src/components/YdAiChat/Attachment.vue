<script setup lang="ts">
import type { ChatAttachment } from '@/api/modules/aiChat'
import { aiChatApi } from '@/api/modules/aiChat'

const props = defineProps<{
  attachment: ChatAttachment
}>()
const src = ref('')
const error = ref('')
const loading = ref(false)
async function load(download = false) {
  loading.value = true
  try {
    if (!src.value) {
      src.value = URL.createObjectURL(await aiChatApi.attachment(props.attachment.id))
    }
    if (download || props.attachment.kind !== 'image') {
      const a = document.createElement('a')
      a.href = src.value
      a.download = props.attachment.name
      a.click()
    }
  }
  catch (e: any) {
    error.value = String(e?.message || e)
  }
  finally {
    loading.value = false
  }
}
onBeforeUnmount(() => {
  if (src.value) {
    URL.revokeObjectURL(src.value)
  }
})
</script>

<template>
  <div class="border rounded-xl bg-background max-w-64 overflow-hidden">
    <img v-if="src && attachment.kind === 'image'" :src="src" :alt="attachment.name" class="max-h-64 w-full object-contain">
    <FaButton variant="ghost" size="sm" class="text-xs max-w-full justify-start" :loading="loading" :title="attachment.name" @click="load(!!src)">
      <FaIcon :name="attachment.kind === 'image' ? 'i-lucide:image' : 'i-lucide:file-text'" class="mr-2 shrink-0" /><span class="truncate">{{ attachment.name }}</span><span class="text-muted-foreground ml-2">{{ Math.ceil(attachment.size / 1024) }}K</span>
    </FaButton>
    <p v-if="error" class="text-xs text-red-500 p-2">
      {{ error }}
    </p>
  </div>
</template>
