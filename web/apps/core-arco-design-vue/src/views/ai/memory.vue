<script setup lang="ts">
// AI 长期记忆（智能/记忆）：对话中自动沉淀的运维经验列表。
import { i18n } from '@/locales'
import { memoryApi } from '@/api/modules/ai'

defineOptions({
  name: 'aiMemory',
})

const toast = useFaToast()

const memories = ref<{ id: number, content: string }[]>([])

async function loadMemories() {
  try {
    memories.value = await memoryApi.list()
  }
  catch {}
}

async function clearMemories() {
  try {
    await memoryApi.clear()
    memories.value = []
    toast.success(i18n.global.t('ai.memory.cleared'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.memory.clearFailed'), { description: e?.message })
  }
}

onMounted(loadMemories)
onActivated(loadMemories)
</script>

<template>
  <div>
    <FaPageMain>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="text-sm font-medium">
            {{ $t('common.total', { n: memories.length }) }}
          </div>
          <FaButton variant="outline" size="sm" class="text-red-500!" @click="clearMemories">
            {{ $t('ai.memory.clearMemories') }}
          </FaButton>
        </div>
        <div class="rounded-lg border">
          <div v-if="!memories.length" class="p-8 text-center text-sm text-muted-foreground">
            {{ $t('ai.memory.empty') }}
          </div>
          <div v-for="m in memories" :key="m.id" class="border-b px-4 py-2 text-sm last:border-b-0">
            {{ m.content }}
          </div>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
