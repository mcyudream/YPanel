<script setup lang="ts">
// AI 对话（智能/对话，M60.1 参考 YuDream-Admin 沉浸式布局）：
// 全高双栏（会话侧栏可折叠 / 窄屏抽屉化）+ 欢迎页建议卡片网格 + 权限四档 +
// 供应商模型单选 + 流式对话；会话支持搜索 / 行内重命名 / 删除。
import { i18n } from '@/locales'
import type { AIProvider } from '@/api/modules/ai'
import aiApi, { conversationApi } from '@/api/modules/ai'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import SessionList from '@/components/YdAiChat/SessionList.vue'
import type { AiSessionItem } from '@/components/YdAiChat/SessionList.vue'
import ModeSelect from '@/components/YdAiChat/ModeSelect.vue'
import ModelSelect from '@/components/YdAiChat/ModelSelect.vue'
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

// ---- 会话侧栏折叠（M60.1）：桌面 toggle 收起/展开；≤760px 隐藏改抽屉 ----
const sidebarOpen = ref(true)
const drawerVisible = ref(false)
const isNarrow = ref(false)
let mq: MediaQueryList | null = null
function onMqChange(e: MediaQueryListEvent | MediaQueryList) {
  isNarrow.value = e.matches
  if (isNarrow.value) {
    sidebarOpen.value = false
  }
}
onMounted(() => {
  mq = window.matchMedia('(max-width: 760px)')
  onMqChange(mq)
  mq.addEventListener('change', onMqChange)
})
onBeforeUnmount(() => {
  mq?.removeEventListener('change', onMqChange)
})

function toggleSidebar() {
  if (isNarrow.value) {
    drawerVisible.value = true
  }
  else {
    sidebarOpen.value = !sidebarOpen.value
  }
}

// ---- 供应商/模型（合并单选，选中同时回写供应商与模型） ----
const providerId = ref<number>(0)
const providers = ref<AIProvider[]>([])
const model = ref('')

const currentProvider = computed(() => providers.value.find(p => p.id === providerId.value))

// 权限档位（四档，与后端 aichat.go 枚举对应）：默认变更前确认
const mode = ref<'read_only' | 'confirm' | 'auto_edit' | 'full'>('confirm')

const askStore = useAiAskStore()
const questionStore = useAiQuestionStore()

const { messages, streaming, send, stop, regenerate, clear, attachLive, conversationId, contextUsage } = useAiChat({
  scenePath: () => (embed ? scenePathOfApp(lastFocusedWin.value?.appId ?? '') : '/'),
  sceneFocus: () => (embed && lastFocusedWin.value ? focusHintOf(lastFocusedWin.value) : ''),
  providerId: () => providerId.value || undefined,
  model: () => model.value || undefined,
  mode: () => mode.value,
  onSaved: () => loadConversations(),
})

// 重新生成只出现在最后一条 assistant 消息上
const lastAssistantId = computed(() => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    if (messages.value[i].role === 'assistant') {
      return messages.value[i].id
    }
  }
  return ''
})

function firstModelOf(p?: AIProvider) {
  if (!p) {
    return ''
  }
  const list = (p.models || '').split(',').map(m => m.trim()).filter(Boolean)
  return p.model || list[0] || ''
}

function sendChat(t: string) {
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
    if (def && providerId.value !== def.id) {
      providerId.value = def.id
      model.value = firstModelOf(def)
    }
  }
  catch {}
}

// ---- 会话 ----
const conversations = ref<AiSessionItem[]>([])
const sessionsLoading = ref(false)

async function loadConversations() {
  try {
    sessionsLoading.value = true
    conversations.value = await conversationApi.list()
  }
  catch {}
  finally {
    sessionsLoading.value = false
  }
}

async function openConversation(id: number) {
  drawerVisible.value = false
  try {
    const conv = await conversationApi.get(id)
    const base = (conv.messages || []).map((m, i) => ({
      id: `c-${id}-${i}`,
      role: m.role as 'user' | 'assistant',
      content: m.content,
      reasoning: (m as any).reasoning || '',
      steps: (m as any).steps || [],
      knowledge: (m as any).knowledge || [],
    }))
    // M62：优先探测进行中的任务——命中则回放重建并无感接入（跨窗口/强刷新），否则静态历史
    const runStatus = await attachLive(id, base)
    if (runStatus === null) {
      messages.value = base
      conversationId.value = id
    }
    else if (runStatus === 'running') {
      toast.success(i18n.global.t('ai.chat.attachedRunning'))
    }
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
  if (isNarrow.value) {
    drawerVisible.value = false
  }
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

// 行内重命名：取原消息体后以新标题整体保存（save 为 upsert，空标题沿用旧值）
async function renameConversation(s: AiSessionItem, title: string) {
  try {
    const conv = await conversationApi.get(s.id)
    await conversationApi.save(s.id, title, conv.messages || [])
    await loadConversations()
  }
  catch (e: any) {
    toast.error(i18n.global.t('ai.chat.renameFailed'), { description: e?.message })
  }
}

// 欢迎页建议（卡片网格）
const suggestions = computed(() => [
  i18n.global.t('ai.chat.suggestStatus'),
  i18n.global.t('ai.chat.suggestRestart'),
  i18n.global.t('ai.chat.suggestScript'),
  i18n.global.t('ai.chat.suggestDeploy'),
])

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
  <FaPageMain :main-class="embed ? 'min-h-0 flex-1 p-0' : 'p-0'">
    <div class="chat-page" :class="embed ? 'chat-page--embed' : ''">
      <!-- 会话侧栏（桌面可折叠；窄屏隐藏改抽屉） -->
      <aside v-if="sidebarOpen && !isNarrow" class="chat-side">
        <SessionList
          :sessions="conversations"
          :active-id="conversationId"
          :loading="sessionsLoading"
          @select="openConversation"
          @create="newConversation"
          @rename="renameConversation"
          @remove="removeConversation"
        />
      </aside>
      <!-- 主列：顶栏 / 消息 / 发送器 -->
      <div class="chat-main">
        <header class="chat-top">
          <button
            type="button"
            class="chat-icon-btn"
            :title="isNarrow ? $t('ai.chat.sessionHistory') : (sidebarOpen ? $t('ai.chat.collapseSidebar') : $t('ai.chat.expandSidebar'))"
            @click="toggleSidebar"
          >
            <FaIcon name="i-lucide:panel-left" class="text-base" />
          </button>
          <span class="truncate text-sm font-medium">{{ $t('ai.chat.title') }}</span>
          <span
            v-if="currentProvider && model"
            class="hidden truncate font-mono text-[11px] text-muted-foreground sm:inline"
            :title="`${currentProvider.name} · ${model}`"
          >
            {{ currentProvider.name }} · {{ model }}
          </span>
          <button type="button" class="chat-top-link" @click="router.push('/ai/providers')">
            {{ $t('ai.chat.providerSettings') }}
          </button>
          <button type="button" class="chat-top-link ml-auto hover:text-red-500" @click="clear()">
            {{ $t('ai.chat.clearConversation') }}
          </button>
        </header>
        <MessageList :messages="messages" class="min-h-0 flex-1">
          <!-- 欢迎页：品牌标 + 大标题 + 建议卡片网格 -->
          <div v-if="!messages.length" class="chat-welcome">
            <div class="chat-welcome__mark">
              <FaIcon name="i-ri:sparkling-2-fill" class="text-3xl" />
            </div>
            <h1 class="chat-welcome__title">
              {{ $t('ai.chat.welcomeTitle') }}
            </h1>
            <p class="chat-welcome__desc">
              {{ $t('ai.chat.welcomeDesc') }}
            </p>
            <div class="chat-welcome__grid">
              <button
                v-for="s in suggestions"
                :key="s"
                type="button"
                class="chat-welcome__prompt group"
                @click="send(s)"
              >
                <FaIcon name="i-lucide:sparkles" class="shrink-0 text-primary" />
                <span class="min-w-0 flex-1 truncate text-left">{{ s }}</span>
                <FaIcon name="i-lucide:arrow-right-up" class="shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
              </button>
            </div>
          </div>
          <template v-else>
            <div v-for="m in messages" :key="m.id">
              <YdAiBubble
                :role="m.role"
                :content="m.content"
                :pending="m.pending"
                :reasoning="m.reasoning || ''"
                :steps="m.steps || []"
                :segments="m.segments || []"
                :knowledge="m.knowledge || []"
                :interrupted="m.interrupted === true"
                :show-actions="m.role === 'assistant'"
                :can-regenerate="m.role === 'assistant' && m.id === lastAssistantId && !streaming"
                @regenerate="regenerate"
              />
            </div>
          </template>
        </MessageList>
        <!-- 挂起的确认/提问面板替换输入框位置（ZCode 式） -->
        <div v-if="askStore.payload || questionStore.id" class="mx-auto w-full max-w-[920px] px-3 sm:px-4">
          <AskConfirmPanel v-if="askStore.payload" class="mb-1" />
          <QuestionPanel v-else-if="questionStore.id" class="mb-1" />
        </div>
        <Sender
          v-else
          :loading="streaming"
          :placeholder="$t('ai.chat.inputPlaceholder')"
          @send="sendChat"
          @stop="stop"
        >
          <template #toolbar>
            <div class="flex min-w-0 items-center gap-0.5 rounded-xl border bg-muted/40 p-1">
              <ModeSelect v-model="mode" bare />
              <ModelSelect
                v-model:provider-id="providerId"
                v-model:model="model"
                :providers="providers"
                bare
              />
              <span
                class="hidden items-center gap-1.5 px-1.5 text-[11px] text-muted-foreground md:flex"
                :title="$t('ai.chat.contextTip', { used: contextUsage.tokens.toLocaleString(), limit: Math.round(contextUsage.limit / 1000) })"
              >
                <span class="h-1 w-12 overflow-hidden rounded-full bg-muted">
                  <span
                    class="block h-full rounded-full transition-all"
                    :class="contextUsage.pct > 85 ? 'bg-red-500' : contextUsage.pct > 60 ? 'bg-amber-500' : 'bg-emerald-500'"
                    :style="{ width: `${contextUsage.pct}%` }"
                  />
                </span>
                {{ contextUsage.pct }}%
              </span>
            </div>
          </template>
          <template #actions>
            <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="saveConversation">
              {{ $t('common.save') }}
            </button>
          </template>
        </Sender>
      </div>
    </div>
    <!-- 窄屏会话抽屉 -->
    <FaDrawer v-model="drawerVisible" :title="$t('ai.chat.sessionHistory')" :footer="false">
      <div class="-m-4 h-full min-h-0">
        <SessionList
          :sessions="conversations"
          :active-id="conversationId"
          :loading="sessionsLoading"
          @select="openConversation"
          @create="newConversation"
          @rename="renameConversation"
          @remove="removeConversation"
        />
      </div>
    </FaDrawer>
  </FaPageMain>
</template>

<style scoped>
/* 沉浸式全高双栏（高度链参考 YuDream-Admin：随 fa 布局变量自适应） */
.chat-page {
  display: flex;
  height: calc(100dvh - var(--g-slots-layout-top-height, 0px) - var(--g-header-actual-height, 60px) - var(--g-topbar-actual-height, 0px) - var(--g-main-container-padding-top, 0px) - var(--g-slots-layout-bottom-height, 0px) - var(--g-main-container-padding-bottom, 0px) - 20px);
  min-height: 0;
  overflow: hidden;
  color: oklch(var(--foreground));
}

.chat-page--embed {
  height: 100%;
}

.chat-side {
  display: flex;
  width: 248px;
  flex-shrink: 0;
  min-height: 0;
  flex-direction: column;
  border-right: 1px solid oklch(var(--border));
  background: oklch(var(--muted) / 30%);
}

.chat-main {
  display: grid;
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow: hidden;
  /* 列宽钳到容器：无列定义时 auto 轨道按内容 min-content 撑宽（窄屏被 Sender 工具条撑到 ~522px 裁切） */
  grid-template-columns: minmax(0, 1fr);
  grid-template-rows: auto minmax(0, 1fr) auto;
}

.chat-top {
  display: flex;
  min-height: 46px;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid oklch(var(--border));
}

.chat-icon-btn {
  display: grid;
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: oklch(var(--muted-foreground));
  cursor: pointer;
  place-items: center;
}

.chat-icon-btn:hover {
  background: oklch(var(--accent));
  color: oklch(var(--foreground));
}

.chat-top-link {
  flex-shrink: 0;
  padding: 4px 6px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: oklch(var(--muted-foreground));
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.chat-top-link:hover {
  color: oklch(var(--foreground));
}

/* 欢迎页：品牌标 + 大标题 + 建议卡片网格（参考 YuDream-Admin YdWelcome） */
.chat-welcome {
  display: grid;
  height: 100%;
  justify-items: center;
  align-content: center;
  padding: 24px;
  text-align: center;
}

.chat-welcome__mark {
  display: grid;
  width: 64px;
  height: 64px;
  margin-bottom: 20px;
  place-items: center;
  border-radius: 20px;
  background: linear-gradient(135deg, oklch(var(--primary) / 16%), oklch(var(--primary) / 6%));
  color: oklch(var(--primary));
  box-shadow: 0 8px 24px oklch(var(--primary) / 12%);
}

.chat-welcome__title {
  margin: 0;
  color: oklch(var(--foreground));
  font-size: 26px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.chat-welcome__desc {
  max-width: 560px;
  margin: 10px 0 0;
  color: oklch(var(--muted-foreground));
  font-size: 13.5px;
  line-height: 1.7;
}

.chat-welcome__grid {
  display: grid;
  width: 100%;
  max-width: 640px;
  gap: 10px;
  margin-top: 32px;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.chat-welcome__prompt {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 13px 16px;
  border: 1px solid oklch(var(--border));
  border-radius: 14px;
  background: oklch(var(--background));
  color: oklch(var(--foreground) / 80%);
  cursor: pointer;
  font: inherit;
  font-size: 13.5px;
  text-align: left;
  transition: border-color 0.15s, background 0.15s, color 0.15s, transform 0.15s;
}

.chat-welcome__prompt:hover {
  border-color: oklch(var(--primary) / 40%);
  background: oklch(var(--primary) / 4%);
  color: oklch(var(--foreground));
  transform: translateY(-1px);
}

/* 窄屏自适应（≤760px）：紧凑顶栏；侧栏由 isNarrow 逻辑切抽屉；高度沿用变量链公式 */
@media (max-width: 760px) {
  .chat-top {
    min-height: 42px;
    padding: 5px 8px;
  }

  .chat-welcome {
    padding: 16px;
  }

  /* 建议卡单列：minmax(240px,1fr) 双列的 min-content(490px) 会沿 min-width:auto 链顶穿窄容器 */
  .chat-welcome__grid {
    grid-template-columns: 1fr;
  }
}
</style>
