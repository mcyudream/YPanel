<script setup lang="ts">
// AI 对话（智能/对话）：会话列表 + 供应商选择 + 流式对话。
import { i18n } from '@/locales'
import type { AIProvider } from '@/api/modules/ai'
import aiApi, { conversationApi } from '@/api/modules/ai'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'
import { useAiAskStore } from '@/store/modules/aiAsk'
import { useAiQuestionStore } from '@/store/modules/aiQuestion'
import AskConfirmPanel from '@/components/YdAiChat/AskConfirmPanel.vue'
import QuestionPanel from '@/components/YdAiChat/QuestionPanel.vue'
import { useYwEmbed } from '@/views/desktop/embed'
import { focusHintOf, lastFocusedWin, scenePathOfApp } from '@/views/desktop/focus-scene'

defineOptions({
  name: 'aiChat',
})

const toast = useFaToast()
const router = useRouter()

// 桌面工作台承载时：场景感知改由焦点窗口派生（窗口不换路由，经典的路由感知失效）
const embed = useYwEmbed()

const providerId = ref<number>(0)
const providers = ref<AIProvider[]>([])

// ---- 对话 ----
const model = ref('')

const mode = ref<'read_only' | 'standard' | 'auto'>('standard')
const MODE_META = [
  { value: 'read_only', label: '只读' },
  { value: 'standard', label: '标准确认' },
  { value: 'auto', label: '自动执行' },
]

const askStore = useAiAskStore()
const questionStore = useAiQuestionStore()

const { messages, streaming, send, clear, conversationId, contextUsage } = useAiChat({
  scenePath: () => (embed ? scenePathOfApp(lastFocusedWin.value?.appId ?? '') : '/'),
  sceneFocus: () => (embed && lastFocusedWin.value ? focusHintOf(lastFocusedWin.value) : ''),
  providerId: () => providerId.value || undefined,
  model: () => model.value || undefined,
  mode: () => mode.value,
  onSaved: () => loadConversations(),
})

// 当前供应商可用模型（未配置列表则只有默认 model 一个）
const modelOptions = computed(() => {
  const p = providers.value.find(x => x.id === providerId.value)
  if (!p) {
    return []
  }
  const list = (p.models || '').split(',').map(m => m.trim()).filter(Boolean)
  if (!list.includes(p.model)) {
    list.unshift(p.model)
  }
  return list
})

watch(providerId, () => {
  model.value = modelOptions.value[0] || ''
})
watch(modelOptions, (list) => {
  if (!list.includes(model.value)) {
    model.value = list[0] || ''
  }
})

// 同名多实例（同一供应商建多个）：下拉文本补模型/ID 区分
const dupProviderNames = computed(() => {
  const count = new Map<string, number>()
  for (const p of providers.value) {
    count.set(p.name, (count.get(p.name) || 0) + 1)
  }
  return new Set([...count.entries()].filter(([, n]) => n > 1).map(([n]) => n))
})

function providerLabel(p: AIProvider) {
  if (!dupProviderNames.value.has(p.name)) {
    return p.name
  }
  return `${p.name} · ${modelOptions.value[0] || p.model} (#${p.id})`
}

function sendChat(t: string) {
  // Sender 组件经 emit('send', text) 传入文本（其内部 v-model 管理输入）
  const text = (t || '').trim()
  if (!text || streaming.value) {
    return
  }
  send(text)
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

const conversations = ref<{ id: number, title: string, updatedAt: string }[]>([])

async function loadConversations() {
  try {
    conversations.value = await conversationApi.list()
  }
  catch {}
}

async function openConversation(id: number) {
  try {
    const conv = await conversationApi.get(id)
    messages.value = (conv.messages || []).map((m, i) => ({
      id: `c-${id}-${i}`,
      role: m.role as 'user' | 'assistant',
      content: m.content,
      reasoning: (m as any).reasoning || '',
      steps: (m as any).steps || [],
      knowledge: (m as any).knowledge || [],
    }))
    conversationId.value = id
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.chat.loadConvFailed'), { description: e?.message })
  }
}

async function saveConversation() {
  if (!messages.value.length) {
    return
  }
  const msgs = messages.value.filter(m => m.content).map(m => ({ role: m.role, content: m.content }))
  try {
    const res = await conversationApi.save(conversationId.value, '', msgs as any)
    conversationId.value = (res.data as any)?.id || conversationId.value
    await loadConversations()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.chat.saveConvFailed'), { description: e?.message })
  }
}

async function newConversation() {
  await saveConversation()
  clear()
}

async function removeConversation(id: number) {
  try {
    await conversationApi.remove(id)
    await loadConversations()
    if (conversationId.value === id) {
      clear()
    }
  }
  catch {}
}

onMounted(() => {
  loadProviders()
  loadConversations()
})

// fa 多标签 KeepAlive：从供应商页返回时刷新列表
onActivated(() => {
  loadProviders()
  loadConversations()
})
</script>

<template>
  <div :class="embed ? 'flex h-full min-h-0 flex-col' : ''">
    <FaPageMain :class="embed ? 'min-h-0 flex-1!' : ''" :main-class="embed ? 'min-h-0 flex-1 p-0' : ''">
      <div :class="embed ? 'flex h-full min-h-0 flex-1 gap-3' : 'flex h-[calc(100vh-320px)] gap-3'">
        <div class="w-52 shrink-0 overflow-y-auto">
          <FaButton size="sm" class="mb-2 w-full" @click="newConversation">
            <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('ai.chat.newConversation') }}
          </FaButton>
          <div
            v-for="c in conversations"
            :key="c.id"
            class="group mt-1 flex items-center justify-between rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent/50"
            :class="conversationId === c.id ? 'bg-accent' : ''"
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
            <span class="text-sm font-medium">{{ $t('ai.chat.title') }}</span>
            <span class="font-mono text-[11px] text-muted-foreground">{{ model || $t('ai.chat.noModel') }}</span>
            <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="router.push('/ai/providers')">
              {{ $t('ai.chat.providerSettings') }}
            </button>
            <button type="button" class="ml-auto text-xs text-muted-foreground hover:text-red-500" @click="clear()">
              {{ $t('ai.chat.clearConversation') }}
            </button>
          </div>
          <MessageList :messages="messages" class="min-h-0 flex-1">
            <div v-if="!messages.length" class="space-y-3 py-10 text-center">
              <div class="text-4xl">
                ✨
              </div>
              <div class="text-sm font-medium">
                {{ $t('ai.chat.assistant') }}
              </div>
              <div class="flex flex-wrap justify-center gap-1.5 pt-2">
                <button
                  v-for="s in [$t('ai.chat.suggestStatus'), $t('ai.chat.suggestRestart'), $t('ai.chat.suggestScript')]"
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
                  :segments="m.segments || []"
                  :knowledge="m.knowledge || []"
                  show-actions
                />
              </div>
            </template>
          </MessageList>
          <!-- 挂起的确认/提问面板替换输入框位置（ZCode 式） -->
          <AskConfirmPanel v-if="askStore.payload" class="mx-0.5 mb-0.5" />
          <QuestionPanel v-else-if="questionStore.id" class="mx-0.5 mb-0.5" />
          <Sender
            v-else
            :loading="streaming"
            :placeholder="$t('ai.chat.inputPlaceholder')"
            @send="sendChat"
            @stop="() => {}"
          >
            <template #toolbar>
              <YdSelect
                v-model="mode"
                :options="MODE_META.map(m => ({ value: m.value, label: $t(`ai.chat.mode.${m.value}`) }))"
                size="sm"
              />
              <YdSelect
                v-model="providerId"
                :options="providers.map(p => ({ value: p.id, label: providerLabel(p) }))"
                size="sm"
                button-class="max-w-36"
              />
              <YdSelect
                v-if="modelOptions.length > 1"
                v-model="model"
                :options="modelOptions"
                size="sm"
                button-class="max-w-36 font-mono text-[11px]"
              />
              <span
                class="flex items-center gap-1.5 text-[11px] text-muted-foreground"
                :title="$t('ai.chat.contextTip', { used: contextUsage.tokens.toLocaleString(), limit: Math.round(contextUsage.limit / 1000) })"
              >
                <span class="h-1 w-14 overflow-hidden rounded-full bg-muted">
                  <span
                    class="block h-full rounded-full transition-all"
                    :class="contextUsage.pct > 85 ? 'bg-red-500' : contextUsage.pct > 60 ? 'bg-amber-500' : 'bg-emerald-500'"
                    :style="{ width: `${contextUsage.pct}%` }"
                  />
                </span>
                {{ contextUsage.pct }}%
              </span>
            </template>
            <template #actions>
              <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="clear()">
                {{ $t('common.clear') }}
              </button>
              <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="saveConversation">
                {{ $t('common.save') }}
              </button>
            </template>
          </Sender>
        </div>
      </div>
    </FaPageMain>
  </div>
</template>
