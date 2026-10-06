<script setup lang="ts">
import type { AIProvider, ChatMsg } from '@/api/modules/ai'
import aiApi from '@/api/modules/ai'
import { useAppAccountStore } from '@/store/modules/app/account'
import { useAppSettingsStore } from '@/store/modules/app/settings'

defineOptions({
  name: 'AIIndex',
})

const settingsStore = useAppSettingsStore()
const accountStore = useAppAccountStore()
void settingsStore
void accountStore

const toast = useFaToast()

const providers = ref<AIProvider[]>([])
const presets = ref<AIProvider[]>([])
const activeProviderId = ref<number>(0)
const messages = ref<ChatMsg[]>([])
const input = ref('')
const streaming = ref(false)
const streamContent = ref('')
const settingVisible = ref(false)
const editForm = ref<AIProvider>({ id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', isDefault: true })

async function loadProviders() {
  try {
    providers.value = await aiApi.providers()
    const def = providers.value.find(p => p.isDefault) || providers.value[0]
    if (def) {
      activeProviderId.value = def.id
    }
  }
  catch {}
}

async function loadPresets() {
  try {
    presets.value = await aiApi.presets()
  }
  catch {}
}

function openSetting(p?: AIProvider) {
  editForm.value = p
    ? { ...p, apiKey: '' }
    : { id: 0, name: '', apiType: 'openai', baseURL: '', apiKey: '', model: '', isDefault: providers.value.length === 0 }
  settingVisible.value = true
}

function applyPreset(preset: AIProvider) {
  editForm.value = { ...editForm.value, name: preset.name, apiType: preset.apiType, baseURL: preset.baseURL, model: preset.model }
}

async function saveSetting() {
  try {
    providers.value = ((await aiApi.saveProvider(editForm.value)).data as any) || []
    settingVisible.value = false
    toast.success('供应商已保存')
  }
  catch (e: any) {
    toast.error('保存失败', { description: e?.message })
  }
}

async function removeSetting(id: number) {
  try {
    await aiApi.removeProvider(id)
    providers.value = providers.value.filter(p => p.id !== id)
    toast.success('已删除')
  }
  catch (e: any) {
    toast.error('删除失败', { description: e?.message })
  }
}

const activeProvider = computed(() => providers.value.find(p => p.id === activeProviderId.value))

async function send() {
  const content = input.value.trim()
  if (!content || streaming.value) {
    return
  }
  if (!providers.value.length) {
    toast.error('请先在设置中添加 AI 供应商')
    openSetting()
    return
  }
  input.value = ''
  messages.value.push({ role: 'user', content })
  messages.value.push({ role: 'assistant', content: '' })
  streaming.value = true
  streamContent.value = ''
  const history = messages.value.slice(0, -1).map(m => ({ role: m.role, content: m.content }))
  try {
    await aiApi.chatStream(activeProviderId.value || undefined, history, (delta) => {
      streamContent.value += delta
      messages.value[messages.value.length - 1].content = streamContent.value
    })
  }
  catch (e: any) {
    messages.value[messages.value.length - 1].content = `出错了：${e?.message || e}`
  }
  finally {
    streaming.value = false
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

onMounted(() => {
  loadProviders()
  loadPresets()
})
</script>

<template>
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="sparkles" :size="24" />
          <span>AI 助手</span>
        </div>
      </template>
      <template #description>
        <span>多供应商大模型对话，自动附带当前主机实时状态上下文</span>
      </template>
      <FaButton variant="outline" size="sm" @click="openSetting()">
        <FaIcon name="i-lucide:settings-2" class="mr-1" /> 供应商设置
      </FaButton>
    </FaPageHeader>

    <FaPageMain>
      <div class="mx-auto flex h-[calc(100vh-260px)] max-w-4xl flex-col rounded-lg border">
        <!-- 供应商选择 -->
        <div class="flex items-center gap-2 border-b px-4 py-2 text-sm">
          <span class="text-muted-foreground">供应商</span>
          <select v-model="activeProviderId" class="h-8 rounded-md border bg-background px-2 outline-none">
            <option v-for="p in providers" :key="p.id" :value="p.id">
              {{ p.name }}（{{ p.model }}）
            </option>
          </select>
          <span v-if="activeProvider" class="text-xs text-muted-foreground">{{ activeProvider.baseURL }}</span>
          <button
            v-for="p in providers"
            :key="`e-${p.id}`"
            type="button"
            class="ml-auto text-xs text-muted-foreground hover:text-red-500"
            @click="removeSetting(p.id)"
          >
            删除 {{ p.name }}
          </button>
        </div>

        <!-- 消息流 -->
        <div class="flex-1 space-y-3 overflow-auto p-4">
          <div v-if="!messages.length" class="py-10 text-center text-sm text-muted-foreground">
            向 AI 描述你的运维问题，例如「内存占用偏高怎么排查」「帮我写一个清理 Docker 日志的脚本」
          </div>
          <div
            v-for="(m, idx) in messages"
            :key="idx"
            class="flex"
            :class="m.role === 'user' ? 'justify-end' : 'justify-start'"
          >
            <div
              class="max-w-85% rounded-lg px-3 py-2 text-sm whitespace-pre-wrap"
              :class="m.role === 'user' ? 'bg-primary/10' : 'bg-muted/60'"
            >
              {{ m.content }}
            </div>
          </div>
          <div v-if="streaming" class="text-xs text-muted-foreground">
            正在思考…
          </div>
        </div>

        <!-- 输入区 -->
        <div class="border-t p-3">
          <textarea
            v-model="input"
            class="h-20 w-full resize-none rounded-md border bg-background p-2 text-sm outline-none focus:ring-1 focus:ring-primary"
            placeholder="输入问题，Enter 发送，Shift+Enter 换行"
            @keydown="onKeydown"
          />
          <div class="mt-2 flex justify-end">
            <FaButton :loading="streaming" :disabled="!input.trim()" @click="send">
              发送
            </FaButton>
          </div>
        </div>
      </div>
    </FaPageMain>

    <!-- 供应商设置 -->
    <FaModal v-model="settingVisible" title="AI 供应商" class="max-w-2xl!" :destroy-on-close="true">
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
          <FaInput v-model="editForm.name" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">API 类型</span>
          <select v-model="editForm.apiType" class="h-9 flex-1 rounded-md border bg-background px-2 outline-none">
            <option value="openai">OpenAI Chat Completions</option>
            <option value="anthropic">Anthropic Messages</option>
            <option value="response">OpenAI Responses</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">Base URL</span>
          <FaInput v-model="editForm.baseURL" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">模型</span>
          <FaInput v-model="editForm.model" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-muted-foreground">API Key</span>
          <FaInput v-model="editForm.apiKey" type="password" :placeholder="editForm.id ? '留空不修改' : ''" class="flex-1" />
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <input v-model="editForm.isDefault" type="checkbox"> 设为默认
        </label>
      </div>
      <template #footer>
        <FaButton variant="outline" @click="settingVisible = false">
          取消
        </FaButton>
        <FaButton @click="saveSetting">
          保存
        </FaButton>
      </template>
    </FaModal>
  </div>
</template>
