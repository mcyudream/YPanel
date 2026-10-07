<script setup lang="ts">
// AI 对话（智能/对话）：会话列表 + 供应商选择 + 流式对话。
import type { AIProvider } from '@/api/modules/ai'
import aiApi, { conversationApi } from '@/api/modules/ai'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'

defineOptions({
  name: 'aiChat',
})

const toast = useFaToast()
const router = useRouter()

const providerId = ref<number>(0)
const providers = ref<AIProvider[]>([])

// ---- 对话 ----
const { messages, streaming, send, clear, conversationId } = useAiChat({
  scenePath: () => '/',
  providerId: () => providerId.value || undefined,
  onSaved: () => loadConversations(),
})

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
    toast.error('读取会话失败', { description: e?.message })
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
    toast.error('会话保存失败', { description: e?.message })
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
  <div>
    <FaPageHeader>
      <template #title>
        <div class="flex items-center gap-2">
          <YdMorphIcon name="sparkles" :size="24" />
          <span>AI 对话</span>
        </div>
      </template>
      <template #description>
        选择供应商开始对话；任何页面也可通过右下角浮层直接对话
      </template>
    </FaPageHeader>

    <FaPageMain>
      <div class="flex h-[calc(100vh-320px)] gap-3">
        <div class="w-52 shrink-0 overflow-y-auto">
          <FaButton size="sm" class="mb-2 w-full" @click="newConversation">
            <FaIcon name="i-lucide:plus" class="mr-1" /> 新会话
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
            <span class="text-muted-foreground">供应商</span>
            <select v-model="providerId" class="h-8 rounded-md border bg-background px-2 outline-none">
              <option v-for="p in providers" :key="p.id" :value="p.id">
                {{ p.name }}（{{ p.model }}）
              </option>
            </select>
            <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="router.push('/ai/providers')">
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
                  :knowledge="m.knowledge || []"
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
    </FaPageMain>
  </div>
</template>
