<script setup lang="ts">
import type { AIKnowledgeDocMeta } from '@/api/modules/ai'
import api from '@/api'
import { knowledgeDocApi } from '@/api/modules/ai'
// 知识库（智能/知识库）：条目 CRUD；对话时按关键词自动检索注入并展示引用来源。
import { i18n } from '@/locales'

defineOptions({
  name: 'AiKnowledge',
})
const toast = useFaToast()
const admin = computed(() => useAppAccountStore().permissions.includes('*'))
interface KnowledgeItem {
  shared?: boolean
  id: number
  title: string
  body: string
}
const knowledge = ref<KnowledgeItem[]>([])
const knowVisible = ref(false)
const knowSaving = ref(false)
const knowForm = ref<KnowledgeItem>({ id: 0, title: '', body: '', shared: false })
async function loadKnowledge() {
  try {
    const res = await api.get('api/v1/ai/knowledge', { silent: true })
    knowledge.value = res.data as KnowledgeItem[]
  }
  catch {

  }
}
function openKnow(k?: KnowledgeItem) {
  knowForm.value = k ? { ...k } : { id: 0, title: '', body: '', shared: false }
  knowVisible.value = true
}
async function saveKnow() {
  knowSaving.value = true
  try {
    await api.post('api/v1/ai/knowledge', knowForm.value)
    knowVisible.value = false
    toast.success(i18n.global.t('ai.knowledge.entrySaved'))
    await loadKnowledge()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.knowledge.saveFailed'), { description: e?.message })
  }
  finally {
    knowSaving.value = false
  }
}
async function removeKnow(id: number) {
  try {
    await api.delete(`api/v1/ai/knowledge/${id}`)
    knowledge.value = knowledge.value.filter(k => k.id !== id)
    toast.success(i18n.global.t('ai.knowledge.deleted'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.knowledge.deleteFailed'), { description: e?.message })
  }
}
// ---- 知识文档 ----
const docs = ref<AIKnowledgeDocMeta[]>([])
const docInput = ref<HTMLInputElement>()
const docUploading = ref(false)
const docViewVisible = ref(false)
const docView = ref<{
  title: string
  filename: string
  content: string
} | null>(null)
async function loadDocs() {
  try {
    docs.value = await knowledgeDocApi.list()
  }
  catch {

  }
}
async function onDocFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) {
    return
  }
  if (file.size > 2 << 20) {
    toast.error(i18n.global.t('ai.knowledge.docTooLarge'))
    return
  }
  docUploading.value = true
  try {
    const content = await file.text()
    await knowledgeDocApi.save({ filename: file.name, content })
    toast.success(i18n.global.t('ai.knowledge.docInjected', { name: file.name }))
    await loadDocs()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.knowledge.injectFailed'), { description: e?.message })
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
    toast.error(i18n.global.t('ai.knowledge.readFailed'), { description: e?.message })
  }
}
async function removeDoc(id: number) {
  try {
    await knowledgeDocApi.remove(id)
    await loadDocs()
    toast.success(i18n.global.t('ai.knowledge.docDeleted'))
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.knowledge.deleteFailed'), { description: e?.message })
  }
}
onMounted(() => {
  loadKnowledge()
  loadDocs()
})
onActivated(() => {
  loadKnowledge()
  loadDocs()
})
</script>

<template>
  <div>
    <FaPageMain>
      <div class="space-y-4">
        <p class="text-xs text-muted-foreground">
          {{ $t('aiAssistant.scopeHelp') }}
        </p>
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm font-medium">
              {{ $t('ai.knowledge.docTitle') }}
            </div>
            <p class="text-xs text-muted-foreground mt-0.5">
              {{ $t('ai.knowledge.docDesc') }}
            </p>
          </div>
          <FaButton size="sm" :loading="docUploading" @click="docInput?.click()">
            <FaIcon name="i-lucide:file-up" class="mr-1" /> {{ $t('ai.knowledge.injectDoc') }}
          </FaButton>
        </div>
        <div v-if="docs.length" class="border rounded-lg">
          <div v-for="d in docs" :key="d.id" class="text-sm px-4 py-2 border-b flex items-center justify-between last:border-b-0">
            <button type="button" class="text-left flex-1 min-w-0 cursor-pointer truncate hover:text-primary" @click="viewDoc(d.id)">
              {{ d.title }}
              <span class="text-[11px] text-muted-foreground font-mono ml-2">{{ d.filename }} · {{ $t('ai.knowledge.chunks', { n: d.chunks }) }}</span>
            </button>
            <FaSwitch v-if="admin" :model-value="d.shared" :title="$t('aiAssistant.shared')" @update:model-value="v => knowledgeDocApi.share(d.id, !!v).then(loadDocs)" />
            <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeDoc(d.id)">
              {{ $t('common.delete') }}
            </FaButton>
          </div>
        </div>

        <div class="pt-2 flex justify-end">
          <FaButton size="sm" @click="openKnow()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('ai.knowledge.addEntry') }}
          </FaButton>
        </div>
        <div v-if="!knowledge.length" class="text-sm text-muted-foreground p-8 text-center border rounded-lg">
          {{ $t('ai.knowledge.empty') }}
        </div>
        <div v-for="k in knowledge" :key="k.id" class="p-4 border rounded-lg">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium">
              {{ k.title }}
            </div>
            <div class="flex gap-1">
              <FaButton variant="ghost" size="sm" @click="openKnow(k)">
                {{ $t('common.edit') }}
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeKnow(k.id)">
                {{ $t('common.delete') }}
              </FaButton>
            </div>
          </div>
          <div class="text-xs text-muted-foreground mt-1 whitespace-pre-wrap line-clamp-3">
            {{ k.body }}
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 知识库弹窗 -->
    <FaModal v-model="knowVisible" :title="knowForm.id ? $t('ai.knowledge.editEntry') : $t('ai.knowledge.addEntry')" class="max-w-2xl!" :destroy-on-close="true">
      <div class="text-sm space-y-3">
        <div class="flex gap-3 items-center">
          <span class="text-muted-foreground shrink-0 w-20">{{ $t('ai.knowledge.titleLabel') }}</span>
          <FaInput v-model="knowForm.title" class="flex-1" />
        </div>
        <textarea
          v-model="knowForm.body"
          rows="8"
          class="text-xs font-mono p-2 outline-none border rounded-md bg-background w-full resize-y focus:border-primary"
          :placeholder="$t('ai.knowledge.bodyPlaceholder')"
        />
      </div>
      <label v-if="admin" class="text-sm my-3 flex items-center justify-between"><span>{{ $t('aiAssistant.shared') }}</span><FaSwitch v-model="knowForm.shared" /></label>
      <template #footer>
        <FaButton variant="outline" @click="knowVisible = false">
          {{ $t('common.cancel') }}
        </FaButton>
        <FaButton :loading="knowSaving" @click="saveKnow">
          {{ $t('common.save') }}
        </FaButton>
      </template>
    </FaModal>

    <FaModal v-model="docViewVisible" :title="docView ? $t('ai.knowledge.docViewTitle', { name: docView.title }) : $t('ai.knowledge.doc')" class="max-w-3xl!" :destroy-on-close="true">
      <pre class="text-xs font-mono p-3 rounded-md bg-muted/40 max-h-96 whitespace-pre-wrap break-words overflow-auto">{{ docView?.content }}</pre>
    </FaModal>

    <input ref="docInput" type="file" accept=".md,.txt,.markdown" class="hidden" @change="onDocFile">
  </div>
</template>
