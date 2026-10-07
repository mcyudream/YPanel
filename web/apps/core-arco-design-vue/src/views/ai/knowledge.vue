<script setup lang="ts">
// 知识库（智能/知识库）：条目 CRUD；对话时按关键词自动检索注入并展示引用来源。
import api from '@/api'
import { knowledgeDocApi } from '@/api/modules/ai'
import type { AIKnowledgeDocMeta } from '@/api/modules/ai'

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

// ---- 知识文档 ----
const docs = ref<AIKnowledgeDocMeta[]>([])
const docInput = ref<HTMLInputElement>()
const docUploading = ref(false)
const docViewVisible = ref(false)
const docView = ref<{ title: string, filename: string, content: string } | null>(null)

async function loadDocs() {
  try {
    docs.value = await knowledgeDocApi.list()
  }
  catch {}
}

async function onDocFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) {
    return
  }
  if (file.size > 2 << 20) {
    toast.error('文档过大（上限 2MB）')
    return
  }
  docUploading.value = true
  try {
    const content = await file.text()
    await knowledgeDocApi.save({ filename: file.name, content })
    toast.success(`文档已注入：${file.name}`)
    await loadDocs()
  }
  catch (e: any) {
    toast.error('注入失败', { description: e?.message })
  }
  finally {
    docUploading.value = false
  }
}

async function viewDoc(id: number) {
  try {
    docView.value = await knowledgeDocApi.get(id)
    docViewVisible.value = true
  }
  catch (e: any) {
    toast.error('读取失败', { description: e?.message })
  }
}

async function removeDoc(id: number) {
  try {
    await knowledgeDocApi.remove(id)
    await loadDocs()
    toast.success('文档已删除')
  }
  catch (e: any) {
    toast.error('删除失败', { description: e?.message })
  }
}

onMounted(() => { loadKnowledge(); loadDocs() })
onActivated(() => { loadKnowledge(); loadDocs() })
</script>

<template>
  <div>
    <FaPageMain>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm font-medium">
              知识文档（md / txt，保存后自动分块）
            </div>
            <p class="mt-0.5 text-xs text-muted-foreground">
              对话按关键词检索命中文档片段并展示引用来源；AI 可用 read_knowledge 工具按需读取全文
            </p>
          </div>
          <FaButton size="sm" :loading="docUploading" @click="docInput?.click()">
            <FaIcon name="i-lucide:file-up" class="mr-1" /> 注入文档
          </FaButton>
        </div>
        <div v-if="docs.length" class="rounded-lg border">
          <div v-for="d in docs" :key="d.id" class="flex items-center justify-between border-b px-4 py-2 text-sm last:border-b-0">
            <button type="button" class="min-w-0 flex-1 cursor-pointer truncate text-left hover:text-primary" @click="viewDoc(d.id)">
              {{ d.title }}
              <span class="ml-2 font-mono text-[11px] text-muted-foreground">{{ d.filename }} · {{ d.chunks }} 块</span>
            </button>
            <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeDoc(d.id)">
              删除
            </FaButton>
          </div>
        </div>

        <div class="flex justify-end pt-2">
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

    <FaModal v-model="docViewVisible" :title="docView ? `文档：${docView.title}` : '文档'" class="max-w-3xl!" :destroy-on-close="true">
      <pre class="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/40 p-3 font-mono text-xs">{{ docView?.content }}</pre>
    </FaModal>

    <input ref="docInput" type="file" accept=".md,.txt,.markdown" class="hidden" @change="onDocFile">
  </div>
</template>
