<script setup lang="ts">
// AI 网关（B18 v2）：统一入口——对话 / 供应商配置 / 知识库 / 工作空间。
import type { AIProvider } from '@/api/modules/ai'
import aiApi from '@/api/modules/ai'
import api from '@/api'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'

defineOptions({
  name: 'AIIndex',
})

const toast = useFaToast()

const tab = ref<'chat' | 'providers' | 'knowledge' | 'workspace'>('chat')
const providerId = ref<number>(0)
const providers = ref<AIProvider[]>([])

// ---- 对话 ----
const { messages, streaming, send, clear } = useAiChat({
  scenePath: () => '/',
  providerId: () => providerId.value || undefined,
})
const chatInput = ref('')

function sendChat() {
  const t = chatInput.value.trim()
  if (!t || streaming.value) {
    return
  }
  chatInput.value = ''
  send(t)
}

async function loadProviders() {
  try {
    providers.value = await aiApi.providers()
    const def = providers.value.find(p => p.isDefault) || providers.value[0]
    if (def) {
      providerId.value = def.id
    }
  }
  catch {}
}

// ---- 供应商 ----
const presets = ref<AIProvider[]>([])
const provVisible = ref(false)
const provSaving = ref(false)
const provForm = ref<AIProvider>({ id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', isDefault: false })

function openProv(p?: AIProvider) {
  provForm.value = p ? { ...p, apiKey: '' } : { id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', isDefault: false }
  provVisible.value = true
}

function applyPreset(preset: AIProvider) {
  provForm.value = { ...provForm.value, name: preset.name, apiType: preset.apiType, baseURL: preset.baseURL, model: preset.model }
}

async function saveProv() {
  provSaving.value = true
  try {
    providers.value = ((await aiApi.saveProvider(provForm.value)).data as any) || []
    const def = providers.value.find(p => p.isDefault) || providers.value[0]
    if (def) {
      providerId.value = def.id
    }
    provVisible.value = false
    toast.success('供应商已保存')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    provSaving.value = false
  }
}

async function removeProv(id: number) {
  try {
    await aiApi.removeProvider(id)
    providers.value = providers.value.filter(p => p.id !== id)
    toast.success('已删除')
  }
  catch (e: any) {
    toast.error('删除失败', { description: e?.message })
  }
}

// ---- 知识库 ----
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

// ---- 工作空间 ----
interface WorkspaceFile { name: string, size: number }
const wsFiles = ref<WorkspaceFile[]>([])
const wsCommand = ref('')
const wsOutput = ref('')
const wsRunning = ref(false)

async function loadWorkspace() {
  try {
    const res = await api.get('api/v1/ai/workspace', { silent: true })
    const entries = (res.data as any)?.entries || []
    wsFiles.value = entries.filter((e: any) => !e.isDir).map((e: any) => ({ name: e.name, size: e.size }))
  }
  catch {}
}

async function wsRun() {
  if (!wsCommand.value.trim()) {
    return
  }
  wsRunning.value = true
  wsOutput.value = ''
  try {
    const res = await api.post('api/v1/ai/workspace/run', { command: wsCommand.value })
    wsOutput.value = (res.data as any)?.output || '(无输出)'
    await loadWorkspace()
  }
  catch (e: any) {
    wsOutput.value = '执行失败：' + (e?.message || '')
  }
  finally {
    wsRunning.value = false
  }
}

const tabs = [
  { key: 'chat', label: '对话', icon: 'i-lucide:message-circle' },
  { key: 'providers', label: '供应商', icon: 'i-lucide:plug' },
  { key: 'knowledge', label: '知识库', icon: 'i-lucide:book-open' },
  { key: 'workspace', label: '工作空间', icon: 'i-lucide:folder-code' },
] as const

onMounted(() => {
  loadProviders()
  loadKnowledge()
  loadWorkspace()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="sparkles" :size="24" />
          <span>AI 网关</span>
        </div>
      </template>
      <template #description>
        <span>统一管理 AI 供应商、知识库与工作空间；任何页面可通过右下角浮层直接对话</span>
      </template>
    </FaPageHeader>

    <FaPageMain>
      <!-- 页内 tabs -->
      <div class="mb-4 flex flex-wrap gap-1.5">
        <button
          v-for="t in tabs"
          :key="t.key"
          type="button"
          class="flex cursor-pointer items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors"
          :class="tab === t.key ? 'bg-primary/10 font-medium text-foreground' : 'text-muted-foreground hover:bg-accent/50'"
          @click="tab = t.key"
        >
          <FaIcon :name="t.icon" class="text-base" />
          {{ t.label }}
        </button>
      </div>

      <!-- 对话 -->
      <div v-show="tab === 'chat'" class="flex h-[calc(100vh-320px)] flex-col rounded-lg border">
        <div class="flex items-center gap-2 border-b px-4 py-2 text-sm">
          <span class="text-muted-foreground">供应商</span>
          <select v-model="providerId" class="h-8 rounded-md border bg-background px-2 outline-none">
            <option v-for="p in providers" :key="p.id" :value="p.id">
              {{ p.name }}（{{ p.model }}）
            </option>
          </select>
          <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="openProv()">
            + 新增
          </button>
          <button type="button" class="ml-auto text-xs text-muted-foreground hover:text-red-500" @click="clear()">
            清空对话
          </button>
        </div>
        <MessageList :messages="messages">
          <div v-if="!messages.length" class="space-y-3 py-10 text-center">
            <div class="text-4xl">
              ✨
            </div>
            <div class="text-sm font-medium">
              YPanel AI 助手
            </div>
            <div class="flex flex-wrap justify-center gap-1.5 pt-2">
              <button
                v-for="s in ['当前服务器状态如何？', '帮我重启 Nginx', '写一个磁盘清理脚本']"
                :key="s"
                type="button"
                class="cursor-pointer rounded-full border px-3 py-1 text-xs text-muted-foreground hover:bg-accent/50"
                @click="send(s)"
              >
                {{ s }}
              </button>
            </div>
          </div>
          <template v-else>
            <div v-for="m in messages" :key="m.id" class="space-y-1">
              <div v-if="m.steps?.length" class="ml-9 space-y-0.5 text-xs text-muted-foreground">
                <div v-for="(st, si) in m.steps" :key="si" class="flex items-center gap-1">
                  <FaIcon name="i-lucide:wrench" class="text-[10px]" />
                  {{ st }}
                </div>
              </div>
              <YdAiBubble :role="m.role" :content="m.content" :pending="m.pending" />
            </div>
          </template>
        </MessageList>
        <Sender
          :loading="streaming"
          :placeholder="'输入问题，Enter 发送…'"
          @send="sendChat"
          @stop="() => {}"
        />
      </div>

      <!-- 供应商 -->
      <div v-show="tab === 'providers'" class="space-y-4">
        <div class="flex justify-end">
          <FaButton size="sm" @click="openProv()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 新增供应商
          </FaButton>
        </div>
        <div v-if="!providers.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          暂无供应商，点击右上角新增（可从内置预设快速填充）
        </div>
        <div v-for="p in providers" :key="p.id" class="flex items-center justify-between rounded-lg border p-4">
          <div>
            <div class="flex items-center gap-2 text-sm font-medium">
              {{ p.name }}
              <span v-if="p.isDefault" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">默认</span>
            </div>
            <div class="mt-1 font-mono text-xs text-muted-foreground">
              {{ p.baseURL }} · {{ p.model }} · {{ p.apiType }}
            </div>
          </div>
          <div class="flex gap-1">
            <FaButton variant="ghost" size="sm" @click="openProv(p)">
              编辑
            </FaButton>
            <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeProv(p.id)">
              删除
            </FaButton>
          </div>
        </div>
      </div>

      <!-- 知识库 -->
      <div v-show="tab === 'knowledge'" class="space-y-4">
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

      <!-- 工作空间 -->
      <div v-show="tab === 'workspace'" class="space-y-4">
        <div class="rounded-lg border p-4">
          <div class="mb-1 text-sm font-medium">
            沙箱目录：/opt/ypanel/ai-workspace
          </div>
          <p class="text-xs text-muted-foreground">
            AI 通过工具在此目录执行脚本与代码；也可在此手动运行命令查看结果。
          </p>
        </div>
        <div class="rounded-lg border p-4">
          <div class="mb-2 text-sm font-medium">
            文件列表
          </div>
          <div v-if="!wsFiles.length" class="py-3 text-center text-xs text-muted-foreground">
            目录为空
          </div>
          <div v-for="f in wsFiles" :key="f.name" class="border-b py-1.5 font-mono text-xs last:border-b-0">
            {{ f.name }} <span class="text-muted-foreground">({{ f.size }} B)</span>
          </div>
        </div>
        <div class="rounded-lg border p-4">
          <div class="mb-2 text-sm font-medium">
            在沙箱中执行命令
          </div>
          <div class="flex gap-2">
            <FaInput v-model="wsCommand" placeholder="如：echo hello > test.txt && cat test.txt" class="flex-1" @keyup.enter="wsRun" />
            <FaButton :loading="wsRunning" @click="wsRun">
              执行
            </FaButton>
          </div>
          <pre v-if="wsOutput" class="mt-3 max-h-60 overflow-auto rounded-md bg-muted/60 p-3 font-mono text-xs">{{ wsOutput }}</pre>
        </div>
      </div>
    </FaPageMain>

    <!-- 供应商弹窗 -->
    <FaModal v-model="provVisible" title="AI 供应商" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="mr-1 text-xs text-muted-foreground">内置预设</span>
          <button
            v-for="p in presets"
            :key="p.name"
            type="button"
            class="cursor-pointer rounded-full border px-2.5 py-0.5 text-xs text-muted-foreground hover:bg-accent/50"
            @click="applyPreset(p)"
          >
            {{ p.name }}
          </button>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">名称</span>
          <FaInput v-model="provForm.name" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">API 类型</span>
          <select v-model="provForm.apiType" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="openai">OpenAI Chat Completions（兼容智谱/DeepSeek/Ollama 等）</option>
            <option value="anthropic">Anthropic Messages</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">Base URL</span>
          <FaInput v-model="provForm.baseURL" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">模型</span>
          <FaInput v-model="provForm.model" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">API Key</span>
          <FaInput v-model="provForm.apiKey" type="password" :placeholder="provForm.id ? '留空不修改' : ''" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="provForm.isDefault" type="checkbox"> 设为默认
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="provVisible = false">
          取消
        </FaButton>
        <FaButton :loading="provSaving" @click="saveProv">
          保存
        </FaButton>
      </template>
    </FaModal>

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
