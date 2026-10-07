<script setup lang="ts">
// AI 长期记忆（智能/记忆）：对话中自动沉淀的运维经验列表。
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
    toast.success('记忆已清空')
  }
  catch (e: any) {
    toast.error('清空失败', { description: e?.message })
  }
}

onMounted(loadMemories)
onActivated(loadMemories)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        AI 长期记忆
      </template>
      <template #description>
        AI 在对话中自动沉淀的运维经验与服务器特性，下次对话自动可用；也可让 AI「记住这件事」
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="text-sm font-medium">
            共 {{ memories.length }} 条
          </div>
          <FaButton variant="outline" size="sm" class="text-red-500!" @click="clearMemories">
            清空记忆
          </FaButton>
        </div>
        <div class="rounded-lg border">
          <div v-if="!memories.length" class="p-8 text-center text-sm text-muted-foreground">
            暂无记忆
          </div>
          <div v-for="m in memories" :key="m.id" class="border-b px-4 py-2 text-sm last:border-b-0">
            {{ m.content }}
          </div>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
