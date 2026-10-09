import io

# ---- chat.vue：模式/模型/用量集成进发送框控件条，顶部条精简 ----
p = 'views/ai/chat.vue'
s = io.open(p, encoding='utf-8').read()

old = """const { messages, streaming, send, clear, conversationId } = useAiChat({
  scenePath: () => '/',
  providerId: () => providerId.value || undefined,
  model: () => model.value || undefined,
  onSaved: () => loadConversations(),
})"""
new = """const mode = ref<'read_only' | 'standard' | 'auto'>('standard')
const MODE_META = [
  { value: 'read_only', label: '只读' },
  { value: 'standard', label: '标准确认' },
  { value: 'auto', label: '自动执行' },
]

const { messages, streaming, send, clear, conversationId, contextUsage } = useAiChat({
  scenePath: () => '/',
  providerId: () => providerId.value || undefined,
  model: () => model.value || undefined,
  mode: () => mode.value,
  onSaved: () => loadConversations(),
})"""
assert old in s, 'chat useAiChat'
s = s.replace(old, new, 1)

old = """          <div class="flex items-center gap-2 border-b px-4 py-2 text-sm">
            <span class="text-muted-foreground">供应商</span>
            <YdSelect
              v-model="providerId"
              :options="providers.map(p => ({ value: p.id, label: providerLabel(p) }))"
              button-class="max-w-44"
            />
            <YdSelect
              v-if="modelOptions.length > 1"
              v-model="model"
              :options="modelOptions"
              button-class="font-mono text-xs"
            />
            <span v-else class="font-mono text-xs text-muted-foreground">
              {{ model }}
            </span>
            <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="router.push('/ai/providers')">
              + 新增
            </button>
            <button type="button" class="ml-auto text-xs text-muted-foreground hover:text-red-500" @click="clear()">
              清空对话
            </button>
          </div>"""
new = """          <div class="flex items-center gap-2 border-b px-4 py-2 text-sm">
            <span class="text-sm font-medium">AI 对话</span>
            <span class="font-mono text-[11px] text-muted-foreground">{{ model || '未配置模型' }}</span>
            <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="router.push('/ai/providers')">
              供应商设置
            </button>
          </div>"""
assert old in s, 'topbar'
s = s.replace(old, new, 1)

old = """          <Sender
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
          </Sender>"""
new = """          <Sender
            :loading="streaming"
            placeholder="输入问题，Enter 发送…"
            @send="sendChat"
            @stop="() => {}"
          >
            <template #toolbar>
              <YdSelect
                v-model="mode"
                :options="MODE_META.map(m => ({ value: m.value, label: m.label }))"
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
                :title="`上下文约 ${contextUsage.tokens.toLocaleString()} / ${Math.round(contextUsage.limit / 1000)}k tokens`"
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
                清空
              </button>
              <button type="button" class="text-xs text-muted-foreground hover:text-foreground" @click="saveConversation">
                保存
              </button>
            </template>
          </Sender>"""
assert old in s, 'sender tpl'
s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('chat.vue ok')

# ---- Layout：移除全局 AiAskModal ----
p = 'layouts/index.vue'
s = io.open(p, encoding='utf-8').read()
s = s.replace("import AiAskModal from '@/components/YdAiAskModal/index.vue'\n", '', 1)
old = """    <Hotkeys />
    <!-- AI 危险操作确认全局弹窗（M31：aiAsk store 驱动，任何对话面共用） -->
    <AiAskModal />"""
new = """    <Hotkeys />"""
assert old in s, 'layout'
s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('layout ok')
