<script setup lang="ts">
// 知识库（智能/知识库）：条目 CRUD；对话时按关键词自动检索注入并展示引用来源。
import api from '@/api'

defineOptions({
  name: 'aiKnowledge',
})

const toast = useFaToast()

interface KnowledgeItem { id: number, title: string, body: string }
const knowledge = ref<KnowledgeItem[]>([])
const knowVisible = ref(false)
const knowSaving = ref(false)
const knowForm = ref<KnowledgeItem>({ id: 0, title: '', body: '' })

async function loadKnowledge() {
  try {
    const res = await api.get('api/v1/ai/knowledge', { silent: true })
    knowledge.value = res.data as KnowledgeItem[]
  }
  catch {}
}

function openKnow(k?: KnowledgeItem) {
  knowForm.value = k ? { ...k } : { id: 0, title: '', body: '' }
  knowVisible.value = true
}

async function saveKnow() {
  knowSaving.value = true
  try {
    await api.post('api/v1/ai/knowledge', knowForm.value)
    knowVisible.value = false
    toast.success('知识条目已保存')
    await loadKnowledge()
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    knowSaving.value = false
  }
}

async function removeKnow(id: number) {
  try {
    await api.delete(`api/v1/ai/knowledge/${id}`)
    knowledge.value = knowledge.value.filter(k => k.id !== id)
    toast.success('已删除')
  }
  catch (e: any) {
    toast.error('删除失败', { description: e?.message })
  }
}

onMounted(loadKnowledge)
onActivated(loadKnowledge)
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        知识库
      </template>
      <template #description>
        条目在对话时按关键词自动检索注入上下文，回答下方展示引用来源
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="space-y-4">
        <div class="flex justify-end">
          <FaButton size="sm" @click="openKnow()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 新增知识条目
          </FaButton>
        </div>
        <div v-if="!knowledge.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          暂无知识条目。知识条目会在对话时按关键词自动检索并注入上下文。
        </div>
        <div v-for="k in knowledge" :key="k.id" class="rounded-lg border p-4">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium">
              {{ k.title }}
            </div>
            <div class="flex gap-1">
              <FaButton variant="ghost" size="sm" @click="openKnow(k)">
                编辑
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeKnow(k.id)">
                删除
              </FaButton>
            </div>
          </div>
          <div class="mt-1 line-clamp-3 text-xs whitespace-pre-wrap text-muted-foreground">
            {{ k.body }}
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 知识库弹窗 -->
    <FaModal v-model="knowVisible" :title="knowForm.id ? '编辑知识条目' : '新增知识条目'" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-muted-foreground">标题</span>
          <FaInput v-model="knowForm.title" class="flex-1" />
        </div>
        <textarea
          v-model="knowForm.body"
          rows="8"
          class="w-full resize-y rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
          placeholder="知识内容（运维规范、脚本说明、常见问题处理方式等）"
        />
      </div>
      <template #footer>
        <FaButton variant="outline" @click="knowVisible = false">
          取消
        </FaButton>
        <FaButton :loading="knowSaving" @click="saveKnow">
          保存
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
