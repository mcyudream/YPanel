<script setup lang="ts">
// AiFloatLayer：全局 AI 浮层（B18）——任何页面右下角悬浮球，点开对话窗口；
// 场景感知：自动携带当前页面路径，后端注入该页实时数据摘要。
import { useRoute } from 'vue-router'
import ChatWindow from '@/components/YdAiChat/ChatWindow.vue'
import MessageList from '@/components/YdAiChat/MessageList.vue'
import Sender from '@/components/YdAiChat/Sender.vue'
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
// 会话控件状态（M32 仿 ZCode 发送框控件条）：权限模式 / 供应商模型 / 上下文用量
// labelKey 展示层词条化；value 为发后端的枚举值，保持不变
const mode = ref<'read_only' | 'standard' | 'auto'>('standard')
const MODE_META = [
  { value: 'read_only', labelKey: 'components.aiFloat.modeReadOnly' },
  { value: 'standard', labelKey: 'components.aiFloat.modeStandard' },
  { value: 'auto', labelKey: 'components.aiFloat.modeAuto' },
]
const providers = ref<AIProvider[]>([])
const providerId = ref(0)
const model = ref('')
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

const { messages, streaming, send, stop, clear, contextUsage } = useAiChat({
  scenePath: () => scenePath.value,
  providerId: () => providerId.value || undefined,
  model: () => model.value || undefined,
  mode: () => mode.value,
  contextLimit: () => contextLimitOf(model.value),
})

watch(open, (v) => {
  if (v && !providers.value.length) {
    aiApi.providers().then((ps) => {
      providers.value = ps
      const def = ps.find(x => x.isDefault) || ps[0]
      if (def) {
        providerId.value = def.id
        model.value = modelOptions.value[0] || def.model
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
    <!-- 悬浮球 -->
    <button
      v-if="!open"
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
          <div class="text-4xl">
            ✨
          </div>
          <div class="text-sm font-medium">
            {{ $t('components.aiFloat.welcomeTitle') }}
          </div>
          <div class="text-xs text-muted-foreground">
            {{ $t('components.aiFloat.sceneAware', { page: route.meta.title || route.path }) }}
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

        <!-- 消息气泡（YDBubble 风格） -->
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
              @regenerate="() => {}"
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
        :suggestions="!messages.length ? suggestions : []"
        :placeholder="$t('components.aiFloat.placeholder', { page: route.meta.title || route.path })"
        @send="send"
        @stop="stop"
        @suggestion-click="send"
      >
        <template #toolbar>
          <YdSelect
            v-model="mode"
            :options="MODE_META.map(m => ({ value: m.value, label: $t(m.labelKey) }))"
            size="sm"
          />
          <YdSelect
            v-if="providers.length"
            v-model="providerId"
            :options="providers.map(p => ({ value: p.id, label: p.name }))"
            size="sm"
            button-class="max-w-28"
          />
          <YdSelect
            v-if="modelOptions.length > 1"
            v-model="model"
            :options="modelOptions"
            size="sm"
            button-class="max-w-32 font-mono text-[11px]"
          />
          <span
            class="flex items-center gap-1 text-[11px] text-muted-foreground"
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
