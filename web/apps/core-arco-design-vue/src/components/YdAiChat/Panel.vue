<script setup lang="ts">
import type { ChatMessage, ConversationMeta } from '@/api/modules/aiChat'
import { useI18n } from 'vue-i18n'
import { aiChatApi } from '@/api/modules/aiChat'
import YdAiBubble from '@/components/YdAiBubble/index.vue'
import { useAiChat } from '@/composables/useAiChat'
import Attachment from './Attachment.vue'
import Interaction from './Interaction.vue'
import MessageList from './MessageList.vue'
import ModelSelect from './ModelSelect.vue'
import ModeSelect from './ModeSelect.vue'
import Sender from './Sender.vue'
import SessionList from './SessionList.vue'
import Settings from './Settings.vue'

const props = withDefaults(defineProps<{
  compact?: boolean
  scene?: string
  focus?: string
  initialId?: number
}>(), { scene: '/', focus: '' })
const { store, current: e, selected, open, create, send } = useAiChat()
const { t } = useI18n()
const toast = useFaToast()
const host = ref<HTMLElement>()
const list = ref<InstanceType<typeof MessageList>>()
const wide = ref(false)
const sidebar = ref(!props.compact)
const drawer = ref(false)
const settingsVisible = ref(false)
const sessions = ref<ConversationMeta[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const search = ref('')
const archived = ref(false)
const deleteID = ref(0)
const editMessage = ref<ChatMessage>()
const editText = ref('')
const compacting = ref(false)
let requestGeneration = 0
let active = true
let resize: ResizeObserver | undefined
let poll: ReturnType<typeof setInterval> | undefined
const running = computed(() => e.value.status === 'running')
const interaction = computed(() => running.value ? e.value.messages.at(-1)?.segments?.find(s => !s.done && s.askId && s.status === 'waiting') : undefined)
const capabilities = computed(() => {
  try {
    return JSON.parse(store.providers.find(p => p.id === e.value.settings.providerId)?.capabilities || '{}')[e.value.settings.model] || {}
  }
  catch {
    return {}
  }
})
const usage = computed(() => {
  const c = e.value.context
  return c?.tokens ? Math.round(c.tokens / Math.max(1, c.limit - c.outputReserve) * 100) : 0
})
const suggestions = computed(() => ['suggestStatus', 'suggestScript', 'suggestDeploy'].map(k => t(`ai.chat.${k}`)))
async function load(more = false, silent = false) {
  const generation = ++requestGeneration
  if (!silent) {
    loading.value = true
  }
  try {
    const p = await aiChatApi.list({ search: search.value, archived: archived.value, page: more ? page.value + 1 : 1, pageSize: 40 })
    if (generation !== requestGeneration) {
      return
    }
    sessions.value = more ? [...sessions.value, ...p.items] : p.items
    total.value = p.total
    page.value = p.page
  }
  catch (err: any) {
    if (!silent) {
      toast.error(String(err?.message || err))
    }
  }
  finally {
    if (generation === requestGeneration) {
      loading.value = false
    }
  }
}
function select(id: number) {
  open(id)
  drawer.value = false
}
function newChat() {
  create()
  drawer.value = false
}
async function submit() {
  try {
    await send(props.scene, props.focus)
  }
  catch (err: any) {
    e.value.error = String(err?.message || err)
  }
}
async function patch(id: number, data: {
  title?: string
  pinned?: boolean
  archived?: boolean
}) {
  try {
    await aiChatApi.update(id, data)
    if (store.entries[id]) {
      await store.refresh(store.entries[id])
    }
    if (data.title && store.entries[id]) {
      store.entries[id].title = data.title
    }
    await load()
  }
  catch {

  }
}
async function remove() {
  const id = deleteID.value
  try {
    await aiChatApi.remove(id)
    deleteID.value = 0
    if (e.value.id === id) {
      newChat()
    }
    await load()
  }
  catch {

  }
}
function setModel(value: string) {
  e.value.settings.model = value
  const c = capabilities.value
  if (c.contextLimit) {
    e.value.settings.contextLimit = c.contextLimit
  }
  if (c.maxOutputTokens) {
    e.value.settings.maxOutputTokens = Math.min(c.maxOutputTokens, e.value.settings.contextLimit - 128)
  }
  if (!c.temperature) {
    delete e.value.settings.temperature
  }
}
async function branch(message: ChatMessage, include = true, text?: string) {
  if (running.value) {
    return
  }
  try {
    const { id } = await aiChatApi.fork(e.value.id, message.id, include)
    open(id)
    await store.refresh(store.entry(id))
    if (!include) {
      e.value.draft = text ?? message.content
      e.value.uploads = (message.attachments || []).map(a => ({ key: crypto.randomUUID(), attachment: a, progress: 100 }))
    }
    await load()
  }
  catch (err: any) {
    toast.error(String(err?.message || err))
  }
}
async function regenerate() {
  const m = [...e.value.messages].reverse().find(m => m.role === 'user')
  if (m) {
    await branch(m, false)
    await submit()
  }
}
async function compact() {
  if (compacting.value || running.value) {
    return
  }
  compacting.value = true
  const target = e.value
  try {
    store.applySnapshot(target, await aiChatApi.compact(target.id))
    target.notice = ''
  }
  catch (err: any) {
    target.notice = String(err?.message || err)
  }
  finally {
    compacting.value = false
  }
}
async function exportChat() {
  try {
    let messages = e.value.messages.slice()
    while (messages.length < e.value.total) {
      const s = await aiChatApi.snapshot(e.value.id, e.value.total - messages.length)
      if (!s.messages.length) {
        break
      }
      messages = [...s.messages, ...messages]
    }
    const text = `# ${e.value.title || t('ai.chat.title')}\n\n${messages.map(m => `## ${m.role}\n\n${m.content}\n\n${(m.attachments || []).map(a => `- ${a.name} (${a.kind})`).join('\n')}\n${(m.segments || []).filter(s => s.type === 'tool').map(s => `\n### ${s.name}\n\n${s.summary || ''}\n\n\`\`\`text\n${s.output || ''}\n\`\`\``).join('\n')}`).join('\n\n')}`
    const url = URL.createObjectURL(new Blob([text], { type: 'text/markdown;charset=utf-8' }))
    const a = document.createElement('a')
    a.href = url
    a.download = 'conversation.md'
    a.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }
  catch (err: any) {
    toast.error(String(err?.message || err))
  }
}
const sessionEvents = {
  select,
  create: newChat,
  search: (v: string) => {
    search.value = v
    void load()
  },
  more: () => load(true),
  archiveView: () => {
    archived.value = !archived.value
    void load()
  },
  rename: (s: ConversationMeta, title: string) => patch(s.id, { title }),
  remove: (id: number) => {
    deleteID.value = id
  },
  patch,
}
watch(() => store.listRevision, () => {
  if (active) {
    void load(false, true)
  }
})
watch(() => props.initialId, (id) => {
  if (id && id !== selected.value) {
    open(id)
  }
})
function startPoll() {
  if (poll) {
    return
  }
  poll = setInterval(() => {
    if (document.visibilityState === 'visible') {
      void load(false, true)
    }
  }, 10000)
}
onMounted(() => {
  if (props.initialId) {
    open(props.initialId)
  }
  void store.initialize().catch((err) => {
    e.value.error = String(err?.message || err)
  })
  void load()
  resize = new ResizeObserver((entries) => {
    wide.value = entries[0].contentRect.width >= 820
  })
  if (host.value) {
    resize.observe(host.value)
  }
  startPoll()
})
onActivated(() => {
  active = true
  startPoll()
  void store.reloadProviders().catch(() => {})
  void store.refresh(e.value)
})
onDeactivated(() => {
  active = false
  clearInterval(poll)
  poll = undefined
})
onBeforeUnmount(() => {
  requestGeneration++
  resize?.disconnect()
  clearInterval(poll)
})
defineExpose({ conversationId: computed(() => e.value.id) })
</script>

<template>
  <div ref="host" class="yd-ai-panel bg-background flex h-full min-h-0 min-w-0 overflow-hidden">
    <aside v-if="sidebar && wide" class="border-r bg-muted/20 shrink-0 w-60">
      <SessionList :sessions="sessions" :active-id="e.id" :loading="loading" :more="sessions.length < total" :archived="archived" v-on="sessionEvents" />
    </aside>
    <div class="flex flex-1 flex-col min-h-0 min-w-0">
      <header class="px-3 border-b flex shrink-0 gap-2 h-13 items-center">
        <FaButton size="icon-sm" variant="ghost" :title="$t('ai.chat.sessionHistory')" @click="wide ? sidebar = !sidebar : drawer = true">
          <FaIcon name="i-lucide:panel-left" />
        </FaButton>
        <div class="flex-1 min-w-0">
          <div class="text-sm font-medium truncate">
            {{ e.title || $t('ai.chat.title') }}
          </div><div class="text-[10px] text-muted-foreground mt-0.5">
            <span v-if="running" class="text-primary inline-flex gap-1 items-center"><span class="rounded-full bg-primary size-1.5 animate-pulse" />{{ $t('aiAssistant.running') }}</span><span v-else>{{ e.id > 0 ? $t('aiAssistant.saved') : 'YPanel · YuDream' }}</span>
          </div>
        </div>
        <FaButton v-if="!wide || !sidebar" size="icon-sm" variant="ghost" :title="$t('aiAssistant.new')" @click="newChat">
          <FaIcon name="i-lucide:square-pen" />
        </FaButton>
        <FaButton v-if="e.id > 0" size="icon-sm" variant="ghost" :title="$t('aiAssistant.export')" @click="exportChat">
          <FaIcon name="i-lucide:download" />
        </FaButton>
        <FaButton size="icon-sm" variant="ghost" :title="$t('aiAssistant.settings')" @click="settingsVisible = true">
          <FaIcon name="i-lucide:sliders-horizontal" />
        </FaButton>
      </header>
      <MessageList ref="list" :revision="e.revision" :identity="selected">
        <div v-if="e.loading && !e.messages.length" class="text-sm text-muted-foreground p-8 text-center">
          {{ $t('common.loading') }}
        </div>
        <div v-else-if="!e.messages.length" class="mx-auto px-2 py-12 text-center flex flex-col max-w-xl items-center">
          <div class="text-primary mb-6 rounded-3xl bg-primary/10 grid size-16 place-items-center">
            <FaIcon name="i-ri:sparkling-2-line" class="text-3xl" />
          </div>
          <h1 class="text-2xl tracking-tight font-semibold">
            {{ $t('aiAssistant.emptyTitle') }}
          </h1>
          <p class="text-sm text-muted-foreground leading-7 mt-3">
            {{ $t('aiAssistant.emptyDesc') }}
          </p>
          <div class="mt-8 gap-3 grid w-full">
            <FaButton v-for="s in suggestions" :key="s" variant="outline" class="px-4 py-3 text-left h-auto whitespace-normal justify-start" @click="e.draft = s">
              <FaIcon name="i-lucide:sparkles" class="text-primary mr-3 shrink-0" />{{ s }}
            </FaButton>
          </div>
          <RouterLink v-if="!store.providers.length" to="/ai/providers" class="text-sm text-primary mt-6 underline">
            {{ $t('aiAssistant.noProvider') }}
          </RouterLink>
        </div>
        <template v-else>
          <div v-if="e.hasMore" class="text-center">
            <FaButton size="sm" variant="ghost" :loading="e.loading" @click="list?.preserve(() => store.older(e))">
              {{ $t('aiAssistant.older') }}
            </FaButton>
          </div>
          <article v-for="(m, i) in e.messages" :key="m.id" v-memo="[m.id, i === e.messages.length - 1 ? e.revision : m.pending]" class="yd-ai-message group">
            <div v-if="m.attachments?.length" class="mb-2 flex flex-wrap gap-2 justify-end">
              <Attachment v-for="a in m.attachments" :key="a.id" :attachment="a" />
            </div>
            <YdAiBubble :role="m.role" :content="m.content" :pending="m.pending" :reasoning="m.reasoning || ''" :steps="m.steps" :segments="m.segments" :knowledge="m.knowledge" :interrupted="m.interrupted" :show-actions="m.role === 'assistant'" :can-regenerate="m.role === 'assistant' && i === e.messages.length - 1 && !running" @regenerate="regenerate" />
            <div v-if="!running" class="mt-1 opacity-60 flex gap-1 transition-opacity items-center group-hover:opacity-100" :class="m.role === 'user' ? 'justify-end' : 'pl-10'">
              <FaButton
                v-if="m.role === 'user'" size="sm" variant="ghost" :title="$t('aiAssistant.edit')" @click="editMessage = m;
                                                                                                           editText = m.content"
              >
                <FaIcon name="i-lucide:pencil" />
              </FaButton>
              <FaButton size="sm" variant="ghost" :title="$t('aiAssistant.branch')" @click="branch(m)">
                <FaIcon name="i-lucide:git-branch" />
              </FaButton>
              <span v-if="m.createdAt" class="text-[10px] text-muted-foreground">{{ new Date(m.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}<span v-if="m.durationMs" class="ml-2">{{ (m.durationMs / 1000).toFixed(1) }}s</span></span>
            </div>
          </article>
        </template>
      </MessageList>
      <div v-if="e.error || e.notice || e.status === 'interrupted'" class="text-xs text-amber-600 px-4 py-2 border-t bg-muted/20 flex gap-2 items-center">
        <span class="flex-1 min-w-0 break-words">{{ e.error || e.notice || $t('aiAssistant.interrupted') }}</span><FaButton v-if="e.error && e.id > 0" size="sm" variant="ghost" @click="store.refresh(e)">
          {{ $t('aiAssistant.reconnect') }}
        </FaButton>
      </div>
      <div v-if="interaction" class="p-3 shrink-0 max-h-[50%] overflow-y-auto">
        <Interaction :key="interaction.askId" :segment="interaction" @resolved="store.refresh(e)" /><FaButton size="sm" variant="ghost" @click="store.stop(e)">
          {{ $t('components.ydAiChat.stopGenerating') }}
        </FaButton>
      </div>
      <Sender v-else v-model="e.draft" :loading="running" :sending="e.sending" :disabled="!store.providers.length" :uploads="e.uploads" :vision="capabilities.vision === true" @send="submit" @stop="store.stop(e)" @files="store.upload(e, $event)" @remove="store.removeUpload(e, $event)" @retry="store.retryUpload(e, $event)">
        <template #toolbar>
          <ModeSelect v-model="e.settings.mode" bare :disabled="running || e.sending" />
          <ModelSelect :provider-id="e.settings.providerId" :model="e.settings.model" :providers="store.providers" :disabled="running || e.sending" bare @update:provider-id="v => e.settings.providerId = v" @update:model="setModel" />
          <FaButton v-if="e.context?.tokens" size="sm" variant="ghost" class="text-[10px] text-muted-foreground ml-auto" :class="usage > 85 ? 'text-amber-600' : ''" :title="`${$t('aiAssistant.estimated')}: ${e.context.tokens} / ${e.context.limit - e.context.outputReserve}`" @click="settingsVisible = true">
            <FaIcon name="i-lucide:chart-pie" class="mr-1" />{{ usage }}%
          </FaButton>
        </template>
      </Sender>
    </div>
    <FaDrawer v-model="drawer" :title="$t('ai.chat.sessionHistory')" :footer="false">
      <div class="h-full -m-4">
        <SessionList :sessions="sessions" :active-id="e.id" :loading="loading" :more="sessions.length < total" :archived="archived" v-on="sessionEvents" />
      </div>
    </FaDrawer>
    <Settings
      v-model="settingsVisible" :entry="e" :providers="store.providers" @compact="compact" @saved="(v, def) => {
        e.settings = v;
        if (def) store.defaults = v;
        store.refresh(e)
      }"
    />
    <FaModal :model-value="!!deleteID" :title="$t('aiAssistant.deleteTitle')" :show-cancel-button="true" @update:model-value="v => { if (!v) deleteID = 0 }" @confirm="remove">
      <p>{{ $t('aiAssistant.deleteDesc') }}</p>
    </FaModal>
    <FaModal :model-value="!!editMessage" :title="$t('aiAssistant.editTitle')" :footer="false" @update:model-value="v => { if (!v) editMessage = undefined }">
      <textarea v-model="editText" rows="6" class="p-3 outline-none border rounded-xl bg-background w-full" /><div class="mt-3 flex justify-end">
        <FaButton
          @click="editMessage && branch(editMessage, false, editText).then(submit);
                  editMessage = undefined"
        >
          {{ $t('aiAssistant.send') }}
        </FaButton>
      </div>
    </FaModal>
    <div v-if="compacting" class="text-xs text-primary bottom-1 right-4 absolute">
      {{ $t('aiAssistant.compact') }}…
    </div>
  </div>
</template>

<style scoped>
.yd-ai-panel {
  position: relative;
}

.yd-ai-message {
  contain-intrinsic-size: auto 140px;
  content-visibility: auto;
}
</style>
