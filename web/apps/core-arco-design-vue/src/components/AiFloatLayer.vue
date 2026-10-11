<script setup lang="ts">
// AiFloatLayer：全局 AI 浮层（B18）——任何页面右下角悬浮球，点开对话窗口；
// 场景感知：自动携带当前页面路径，后端注入该页实时数据摘要。
// M60：权限四档 + 供应商模型合并单选 + stop/regenerate 实接 + 气泡操作栏分角色。
import { useRoute } from 'vue-router'
import ChatWindow from '@/components/YdAiChat/ChatWindow.vue'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
import ModeSelect from '@/components/YdAiChat/ModeSelect.vue'
import ModelSelect from '@/components/YdAiChat/ModelSelect.vue'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'
import { useAiAskStore } from '@/store/modules/aiAsk'
import { useAiQuestionStore } from '@/store/modules/aiQuestion'
import AskConfirmPanel from '@/components/YdAiChat/AskConfirmPanel.vue'
import QuestionPanel from '@/components/YdAiChat/QuestionPanel.vue'
import aiApi, { type AIProvider } from '@/api/modules/ai'
import { i18n } from '@/locales'

defineOptions({ name: 'AiFloatLayer' })

const route = useRoute()
const open = ref(false)

const scenePath = ref(route.fullPath)
watch(() => route.fullPath, (v) => { scenePath.value = v })

// 场景页名：路由标题经 i18n 解析；未命中词条（如 404 路由的 menu.notFound）裸 key 不上屏，回退路径段
const { generateTitle } = useAppMenu()
const scenePage = computed(() => {
  const key = String(route.meta.title ?? '')
  const label = key ? String(generateTitle(key)) : ''
  return label && label !== key ? label : (route.path.replace(/^\//, '') || route.path)
})

// 权限档位（M60 四档，value 为发后端的枚举值）；默认变更前确认
const mode = ref<'read_only' | 'confirm' | 'auto_edit' | 'full'>('confirm')

const providers = ref<AIProvider[]>([])
const providerId = ref(0)
const model = ref('')

function contextLimitOf(m: string) {
  if (/glm-4\.[6-9]|claude|gemini-1\.5|gemini-2/.test(m)) {
    return 200000
  }
  if (/32k|128k|4o|qwen|deepseek|grok/.test(m)) {
    return 128000
  }
  return 64000
}

const askStore = useAiAskStore()
const questionStore = useAiQuestionStore()

const { messages, streaming, send, stop, regenerate, clear, contextUsage } = useAiChat({
  scenePath: () => scenePath.value,
  providerId: () => providerId.value || undefined,
  model: () => model.value || undefined,
  mode: () => mode.value,
  contextLimit: () => contextLimitOf(model.value),
})

// 重新生成只出现在最后一条 assistant 消息上（M60）
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

watch(open, (v) => {
  if (v && !providers.value.length) {
    aiApi.providers().then((ps) => {
      providers.value = ps
      const def = ps.find(x => x.isDefault) || ps[0]
      if (def) {
        providerId.value = def.id
        model.value = firstModelOf(def)
      }
    }).catch(() => {})
  }
})

const suggestions = computed(() => {
  switch (true) {
    case route.path.startsWith('/container'):
      return [i18n.global.t('components.aiFloat.suggestContainer1'), i18n.global.t('components.aiFloat.suggestContainer2')]
    case route.path.startsWith('/sites'):
      return [i18n.global.t('components.aiFloat.suggestSites1'), i18n.global.t('components.aiFloat.suggestSites2')]
    case route.path.startsWith('/database'):
      return [i18n.global.t('components.aiFloat.suggestDb1'), i18n.global.t('components.aiFloat.suggestDb2')]
    default:
      return [i18n.global.t('components.aiFloat.suggestDefault1'), i18n.global.t('components.aiFloat.suggestDefault2'), i18n.global.t('components.aiFloat.suggestDefault3')]
  }
})

</script>

<template>
  <div>
    <!-- 悬浮球（M60.1：对话页已有完整入口，隐藏悬浮球避免压住贴底发送键） -->
    <button
      v-if="!open && !route.path.startsWith('/ai/chat')"
      type="button"
      class="fixed bottom-6 right-6 z-1600 flex size-12 cursor-pointer items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105"
      :title="$t('components.ydAiChat.assistantTitle')"
      @click="open = true"
    >
      <FaIcon name="i-ri:sparkling-2-line" class="text-lg" />
    </button>

    <!-- 对话窗口 -->
    <ChatWindow
      v-if="open"
      :title="$t('components.ydAiChat.assistantTitle')"
      :width="520"
      :height="720"
      :expand-only="false"
      @close="open = false"
      @expand="() => {}"
    >
      <MessageList :messages="messages">
        <!-- 欢迎页 -->
        <div v-if="!messages.length" class="space-y-3 py-8 text-center">
          <div class="flex justify-center">
            <span class="flex size-12 items-center justify-center rounded-2xl bg-primary/10">
              <FaIcon name="i-ri:sparkling-2-line" class="text-xl text-primary" />
            </span>
          </div>
          <div class="text-sm font-medium">
            {{ $t('components.aiFloat.welcomeTitle') }}
          </div>
          <div class="text-xs text-muted-foreground">
            {{ $t('components.aiFloat.sceneAware', { page: scenePage }) }}
          </div>
          <div class="flex flex-wrap justify-center gap-1.5 px-6 pt-2">
            <button
              v-for="s in suggestions"
              :key="s"
              type="button"
              class="cursor-pointer rounded-full border px-3 py-1 text-xs text-muted-foreground hover:bg-accent/50"
              @click="send(s)"
            >
              {{ s }}
            </button>
          </div>
        </div>

        <!-- 消息气泡：操作栏仅助手消息（重新生成仅最后一条），用户消息悬停复制 -->
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
              :interrupted="m.interrupted === true"
              :show-actions="m.role === 'assistant'"
              :can-regenerate="m.role === 'assistant' && m.id === lastAssistantId && !streaming"
              @regenerate="regenerate"
            />
          </div>
        </template>
      </MessageList>

      <!-- 挂起的确认/提问面板替换输入框位置（ZCode 式），处理完自动恢复 -->
      <AskConfirmPanel v-if="askStore.payload" />
      <QuestionPanel v-else-if="questionStore.id" />
      <Sender
        v-else
        :loading="streaming"
        :suggestions="[]"
        :placeholder="$t('components.aiFloat.placeholder', { page: scenePage })"
        @send="send"
        @stop="stop"
        @suggestion-click="send"
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
              class="hidden items-center gap-1 px-1.5 text-[11px] text-muted-foreground sm:flex"
              :title="$t('components.aiFloat.contextUsage', { n: contextUsage.tokens.toLocaleString() })"
            >
              <span class="h-1 w-10 overflow-hidden rounded-full bg-muted">
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
          <button
            type="button"
            class="text-xs text-muted-foreground hover:text-red-500"
            @click="clear()"
          >
            {{ $t('common.clear') }}
          </button>
        </template>
      </Sender>
    </ChatWindow>
  </div>
</template>
