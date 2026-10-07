<script setup lang="ts">
// AI 网关（B18 v2）：统一入口——对话 / 供应商配置 / 知识库 / 工作空间。
import type { AIProvider } from '@/api/modules/ai'
import aiApi, { conversationApi, memoryApi, skillApi, mcpApi } from '@/api/modules/ai'
import type { AISkill, MCPServer } from '@/api/modules/ai'
import { providerPresets } from '@/api/modules/ai'
import api from '@/api'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'

defineOptions({
  name: 'AIIndex',
})

const toast = useFaToast()

const tab = ref<'chat' | 'providers' | 'knowledge' | 'workspace' | 'memory' | 'skills' | 'mcp'>('chat')
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

function iconFor(name: string): string | undefined {
  return providerPresets.find(p => p.name === name || name.includes(p.name.split(' ')[0]))?.icon
}

function openProvFromPreset(preset: AIProvider) {
  const existing = providers.value.find(x => x.name === preset.name)
  if (existing) {
    openProv(existing)
    return
  }
  provForm.value = { id: 0, name: preset.name, apiType: preset.apiType, baseURL: preset.baseURL, apiKey: '', model: preset.model, isDefault: providers.value.length === 0 }
  provVisible.value = true
}

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

const conversations = ref<{ id: number, title: string, updatedAt: string }[]>([])
const memories = ref<{ id: number, content: string }[]>([])
const activeConvId = ref<number>(0)

async function loadConversations() {
  try {
    conversations.value = await conversationApi.list()
  }
  catch {}
}

async function openConversation(id: number) {
  try {
    const conv = await conversationApi.get(id)
    messages.value = (conv.messages || []).map((m, i) => ({ id: `c-${id}-${i}`, role: m.role as 'user' | 'assistant', content: m.content, steps: [] }))
    activeConvId.value = id
  }
  catch (e: any) {
    toast.error('读取会话失败', { description: e?.message })
  }
}

async function saveConversation() {
  if (!messages.value.length) {
    return
  }
  const msgs = messages.value.filter(m => m.content).map(m => ({ role: m.role, content: m.content }))
  try {
    const res = await conversationApi.save(activeConvId.value, '', msgs as any)
    activeConvId.value = (res.data as any)?.id || activeConvId.value
    await loadConversations()
  }
  catch (e: any) {
    toast.error('会话保存失败', { description: e?.message })
  }
}

async function newConversation() {
  await saveConversation()
  clear()
  activeConvId.value = 0
}

async function removeConversation(id: number) {
  try {
    await conversationApi.remove(id)
    await loadConversations()
  }
  catch {}
}

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

const tabs = [
  { key: 'chat', label: '对话', icon: 'i-lucide:message-circle' },
  { key: 'providers', label: '供应商', icon: 'i-lucide:plug' },
  { key: 'knowledge', label: '知识库', icon: 'i-lucide:book-open' },
  { key: 'workspace', label: '工作空间', icon: 'i-lucide:folder-code' },
  { key: 'memory', label: '记忆', icon: 'i-lucide:brain' },
  { key: 'skills', label: '技能', icon: 'i-lucide:puzzle' },
  { key: 'mcp', label: 'MCP', icon: 'i-lucide:plug-zap' },
] as const

// ---- 技能 ----
const skills = ref<AISkill[]>([])
const skillVisible = ref(false)
const skillSaving = ref(false)
const skillForm = ref<AISkill>({ name: '', description: '', body: '', enabled: false })

async function loadSkills() {
  try {
    skills.value = await skillApi.list()
  }
  catch {}
}

function openSkill(k?: AISkill) {
  skillForm.value = k ? { ...k } : {
    name: '',
    description: '',
    body: '---\nname: my-skill\ndescription: 技能描述\n---\n\n技能正文（提示词与执行步骤）',
    enabled: false,
  }
  skillVisible.value = true
}

async function saveSkill() {
  skillSaving.value = true
  try {
    await skillApi.save({
      name: skillForm.value.name,
      description: skillForm.value.description,
      body: skillForm.value.body,
    })
    if (skillForm.value.enabled) {
      await skillApi.setEnabled(skillForm.value.name, true)
    }
    skillVisible.value = false
    toast.success('技能已保存')
    await loadSkills()
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    skillSaving.value = false
  }
}

async function toggleSkill(k: AISkill) {
  try {
    await skillApi.setEnabled(k.name, !k.enabled)
    await loadSkills()
  }
  catch (e: any) {
    toast.error('操作失败', { description: e?.message })
  }
}

async function removeSkill(k: AISkill) {
  try {
    await skillApi.remove(k.name)
    await loadSkills()
    toast.success('已删除')
  }
  catch (e: any) {
    toast.error('删除失败', { description: e?.message })
  }
}

// ---- MCP ----
const mcpServers = ref<MCPServer[]>([])
const mcpVisible = ref(false)
const mcpSaving = ref(false)
const mcpTesting = ref('')
const mcpForm = ref<MCPServer>({ name: '', transport: 'stdio', command: '', args: [], url: '', enabled: true })
const mcpTools = ref<string[]>([])
const mcpTestName = ref('')

async function loadMCP() {
  try {
    mcpServers.value = await mcpApi.list()
  }
  catch {}
}

const mcpArgsStr = computed({
  get: () => (mcpForm.value.args || []).join(' '),
  set: (v: string) => {
    mcpForm.value.args = v.split(/\s+/).filter(Boolean)
  },
})

function openMCP(s?: MCPServer) {
  mcpForm.value = s ? JSON.parse(JSON.stringify(s)) : { name: '', transport: 'stdio', command: '', args: [], url: '', enabled: true }
  mcpVisible.value = true
}

async function saveMCP() {
  mcpSaving.value = true
  try {
    mcpServers.value = ((await mcpApi.saveAll(mcpServers.value)).data as any) || []
    mcpVisible.value = false
    toast.success('MCP 服务器配置已保存')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
  finally {
    mcpSaving.value = false
  }
}

async function testMCP(s: MCPServer) {
  mcpTesting.value = s.name
  try {
    const res = await mcpApi.test(s)
    mcpTools.value = res.tools || []
    toast.success(`连接成功，发现 ${res.tools.length} 个工具`)
  }
  catch (e: any) {
    toast.error('连接失败', { description: e?.message })
  }
  finally {
    mcpTesting.value = ''
  }
}

onMounted(() => {
  loadProviders()
  loadKnowledge()
  loadWorkspace()
  loadConversations()
  loadMemories()
  loadSkills()
  loadMCP()
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

      <!-- 对话（左会话列表 + 右对话区） -->
      <div v-show="tab === 'chat'" class="flex h-[calc(100vh-320px)] gap-3">
        <div class="w-52 shrink-0 space-y-1 overflow-y-auto">
          <FaButton size="sm" class="w-full" @click="newConversation">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 新会话
          </FaButton>
          <div
            v-for="c in conversations"
            :key="c.id"
            class="group flex items-center justify-between rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent/50"
            :class="activeConvId === c.id ? 'bg-accent' : ''"
          >
            <button type="button" class="min-w-0 flex-1 cursor-pointer truncate text-left" @click="openConversation(c.id)">
              {{ c.title }}
            </button>
            <FaIcon
              name="i-lucide:trash-2"
              class="ml-1 hidden size-3.5 shrink-0 cursor-pointer text-muted-foreground group-hover:block hover:text-red-500"
              @click="removeConversation(c.id)"
            />
          </div>
        </div>
        <div class="flex min-w-0 flex-1 flex-col rounded-lg border">
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
              <YdAiBubble
                :role="m.role"
                :content="m.content"
                :pending="m.pending"
                :reasoning="m.reasoning || ''"
                :steps="m.steps || []"
                show-actions
              />
            </div>
          </template>
        </MessageList>
        <Sender
          :loading="streaming"
          :placeholder="'输入问题，Enter 发送…；可让我直接查询数据库'"
          @send="sendChat"
          @stop="() => {}"
        >
          <template #actions>
            <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="saveConversation">
              保存会话
            </button>
          </template>
        </Sender>
      </div>
      </div>

      <!-- 供应商（LobeHub 风格：预设网格 + 已配置列表） -->
      <div v-show="tab === 'providers'" class="space-y-5">
        <!-- 已配置 -->
        <div v-if="providers.length" class="space-y-2">
          <div class="text-sm font-medium">
            已配置（{{ providers.length }}）
          </div>
          <div v-for="p in providers" :key="p.id" class="flex items-center justify-between rounded-lg border p-3">
            <div class="flex items-center gap-3">
              <img v-if="iconFor(p.name)" :src="iconFor(p.name)" class="size-6">
              <FaIcon v-else name="i-lucide:box" class="size-5 text-muted-foreground" />
              <div>
                <div class="flex items-center gap-2 text-sm font-medium">
                  {{ p.name }}
                  <span v-if="p.isDefault" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">默认</span>
                </div>
                <div class="font-mono text-xs text-muted-foreground">
                  {{ p.baseURL }} · {{ p.model }}
                </div>
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

        <!-- 可添加的供应商（LobeHub 风格网格卡片） -->
        <div class="space-y-2">
          <div class="text-sm font-medium">
            添加供应商
          </div>
          <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <button
              v-for="preset in providerPresets"
              :key="preset.name"
              type="button"
              class="flex cursor-pointer items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:border-primary hover:bg-accent/30"
              :class="providers.some(x => x.name === preset.name) ? 'op-50' : ''"
              @click="openProvFromPreset(preset)"
            >
              <img v-if="preset.icon" :src="preset.icon" class="size-7 shrink-0">
              <span v-else class="flex size-7 items-center justify-center rounded bg-muted">
                <FaIcon name="i-lucide:box" class="text-sm" />
              </span>
              <span class="min-w-0">
                <span class="block truncate text-sm">{{ preset.name }}</span>
                <span class="block truncate font-mono text-[11px] text-muted-foreground">{{ preset.model }}</span>
              </span>
              <span v-if="providers.some(x => x.name === preset.name)" class="ml-auto shrink-0 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] text-emerald-600">
                已配置
              </span>
            </button>
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

      <!-- 记忆 -->
      <div v-show="tab === 'memory'" class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm font-medium">
              AI 长期记忆（{{ memories.length }}）
            </div>
            <p class="mt-0.5 text-xs text-muted-foreground">
              AI 在对话中自动沉淀的运维经验与服务器特性，下次对话自动可用；也可让 AI「记住这件事」
            </p>
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
          <!-- 技能 -->
      <div v-show="tab === 'skills'" class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm font-medium">
              技能包（SKILL.md 格式，目录 /opt/ypanel/ai/skills）
            </div>
            <p class="mt-0.5 text-xs text-muted-foreground">
              启用的技能会在对话时注入 AI 上下文；也可直接 scp 编辑 SKILL.md 文件
            </p>
          </div>
          <FaButton size="sm" @click="openSkill()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 新增技能
          </FaButton>
        </div>
        <div v-if="!skills.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          暂无技能包
        </div>
        <div v-for="k in skills" :key="k.name" class="rounded-lg border p-4">
          <div class="flex items-center justify-between">
            <div class="min-w-0">
              <div class="flex items-center gap-2 text-sm font-medium">
                {{ k.name }}
                <span
                  class="rounded-full px-2 py-0.5 text-xs"
                  :class="k.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'"
                >
                  {{ k.enabled ? '已启用' : '未启用' }}
                </span>
              </div>
              <div class="mt-1 line-clamp-2 text-xs text-muted-foreground">
                {{ k.description || k.body.slice(0, 100) }}
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <label class="flex cursor-pointer items-center gap-1.5 text-xs">
                <input type="checkbox" :checked="k.enabled" @change="toggleSkill(k)"> 启用
              </label>
              <FaButton variant="ghost" size="sm" @click="openSkill(k)">
                编辑
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="removeSkill(k)">
                删除
              </FaButton>
            </div>
          </div>
        </div>
      </div>

      <!-- MCP -->
      <div v-show="tab === 'mcp'" class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm font-medium">
              MCP 服务器（Model Context Protocol）
            </div>
            <p class="mt-0.5 text-xs text-muted-foreground">
              接入外部工具服务器（stdio 常驻进程 / streamable-http 端点），其工具自动并入对话
            </p>
          </div>
          <FaButton size="sm" @click="openMCP()">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 添加服务器
          </FaButton>
        </div>
        <div v-if="!mcpServers.length" class="rounded-lg border p-8 text-center text-sm text-muted-foreground">
          暂未接入 MCP 服务器
        </div>
        <div v-for="sv in mcpServers" :key="sv.name" class="rounded-lg border p-4">
          <div class="flex items-center justify-between">
            <div class="min-w-0">
              <div class="flex items-center gap-2 text-sm font-medium">
                {{ sv.name }}
                <span class="rounded-full bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">{{ sv.transport }}</span>
                <span v-if="sv.enabled" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600">启用</span>
              </div>
              <div class="mt-1 truncate font-mono text-xs text-muted-foreground">
                {{ sv.transport === 'stdio' ? `${sv.command} ${sv.args?.join(' ') || ''}` : sv.url }}
              </div>
            </div>
            <div class="flex shrink-0 gap-1">
              <FaButton variant="ghost" size="sm" :loading="mcpTesting === sv.name" @click="testMCP(sv)">
                测试
              </FaButton>
              <FaButton variant="ghost" size="sm" @click="openMCP(sv)">
                编辑
              </FaButton>
              <FaButton variant="ghost" size="sm" class="text-red-500!" @click="() => { mcpServers = mcpServers.filter(x => x.name !== sv.name) }">
                移除
              </FaButton>
            </div>
          </div>
          <div v-if="mcpTestName === sv.name && mcpTools.length" class="mt-2 flex flex-wrap gap-1">
            <span v-for="t in mcpTools" :key="t" class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">
              {{ t }}
            </span>
          </div>
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

    <!-- 技能编辑 -->
    <FaModal v-model="skillVisible" :title="skillForm.name ? `编辑技能：${skillForm.name}` : '新增技能'" class="max-w-3xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-muted-foreground">技能名</span>
          <FaInput v-model="skillForm.name" placeholder="如 deploy-site" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-20 shrink-0 text-muted-foreground">描述</span>
          <FaInput v-model="skillForm.description" placeholder="一句话说明技能用途（AI 按此决定是否使用）" class="flex-1" />
        </div>
        <textarea
          v-model="skillForm.body"
          rows="10"
          class="w-full resize-y rounded-md border bg-background p-2 font-mono text-xs outline-none focus:border-primary"
          placeholder="技能正文（YAML frontmatter + 提示词/步骤）"
        />
        <label class="flex cursor-pointer items-center gap-2 text-xs">
          <input v-model="skillForm.enabled" type="checkbox"> 保存后立即启用
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="skillVisible = false">
          取消
        </FaButton>
        <FaButton :loading="skillSaving" @click="saveSkill">
          保存
        </FaButton>
      </template>
    </FaModal>

    <!-- MCP 编辑 -->
    <FaModal v-model="mcpVisible" :title="mcpForm.name ? `编辑 MCP：${mcpForm.name}` : '添加 MCP 服务器'" class="max-w-2xl!" :destroy-on-close="true">
      <div class="space-y-3 text-sm">
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">名称</span>
          <FaInput v-model="mcpForm.name" placeholder="如 filesystem" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">传输类型</span>
          <select v-model="mcpForm.transport" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="stdio">stdio（本地子进程）</option>
            <option value="streamable-http">streamable-http（远程端点）</option>
          </select>
        </div>
        <div v-if="mcpForm.transport === 'stdio'" class="space-y-3">
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-muted-foreground">启动命令</span>
            <FaInput v-model="mcpForm.command" placeholder="npx" class="flex-1" />
          </div>
          <div class="flex items-center gap-3">
            <span class="w-24 shrink-0 text-muted-foreground">参数</span>
            <FaInput v-model="mcpArgsStr" placeholder="-y @modelcontextprotocol/server-filesystem /tmp" class="flex-1" />
          </div>
        </div>
        <div v-else class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">端点 URL</span>
          <FaInput v-model="mcpForm.url" placeholder="http://127.0.0.1:3001/mcp" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">环境变量</span>
          <FaInput v-model="mcpArgsStr" placeholder="KEY=VALUE（暂不支持）" disabled class="flex-1" />
        </div>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="mcpVisible = false">
          取消
        </FaButton>
        <FaButton :loading="mcpSaving" @click="saveMCP">
          保存
        </FaButton>
      </template>
    </FaModal>

</template>
