<script setup lang="ts">
// YdAiSessionList：会话列表（M60.1 参考 YuDream-Admin）——搜索过滤 + 悬停浮现操作
// （行内重命名 / 删除）+ 空态；桌面侧栏与窄屏抽屉共用同一实例形态。
import { computed, nextTick, ref } from 'vue'

export interface AiSessionItem {
  id: number
  title: string
  updatedAt?: string
  /** 运行态标注（M62）：running / waiting_confirm / waiting_answer；空 = 无任务 */
  runStatus?: string
}

const props = withDefaults(defineProps<{
  sessions: AiSessionItem[]
  activeId?: number
  loading?: boolean
}>(), {
  activeId: 0,
  loading: false,
})

const emit = defineEmits<{
  select: [id: number]
  create: []
  rename: [session: AiSessionItem, title: string]
  remove: [id: number]
}>()

const keyword = ref('')
const editingId = ref(0)
const editingTitle = ref('')
const editInput = ref<HTMLInputElement | HTMLInputElement[]>()

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) {
    return props.sessions
  }
  return props.sessions.filter(s => s.title.toLowerCase().includes(k))
})

async function startRename(s: AiSessionItem) {
  editingId.value = s.id
  editingTitle.value = s.title
  await nextTick()
  // v-for 内模板 ref 收集为数组（同一时刻仅一个编辑行，取首个即可）
  const el = Array.isArray(editInput.value) ? editInput.value[0] : editInput.value
  el?.focus()
  el?.select()
}

function commitRename(s: AiSessionItem) {
  const title = editingTitle.value.trim()
  editingId.value = 0
  if (title && title !== s.title) {
    emit('rename', s, title)
  }
}

const RUN_BADGE: Record<string, { cls: string, key: string }> = {
  running: { cls: 'bg-emerald-500/10 text-emerald-600', key: 'ai.chat.runRunning' },
  waiting_confirm: { cls: 'bg-amber-500/10 text-amber-600', key: 'ai.chat.runWaitingConfirm' },
  waiting_answer: { cls: 'bg-sky-500/10 text-sky-600', key: 'ai.chat.runWaitingAnswer' },
}

function onEditKeydown(e: KeyboardEvent, s: AiSessionItem) {
  if (e.key === 'Enter') {
    e.preventDefault()
    commitRename(s)
  }
  else if (e.key === 'Escape') {
    editingId.value = 0
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="grid gap-2 px-3 pb-2 pt-3">
      <FaButton size="sm" class="w-full!" @click="emit('create')">
        <FaIcon name="i-lucide:plus" class="mr-1" /> {{ $t('ai.chat.newConversation') }}
      </FaButton>
      <div class="flex items-center gap-1.5 rounded-lg border bg-background px-2.5 focus-within:border-primary/50">
        <FaIcon name="i-lucide:search" class="shrink-0 text-xs text-muted-foreground" />
        <input
          v-model="keyword"
          type="text"
          :placeholder="$t('ai.chat.searchSessions')"
          class="w-full bg-transparent py-1.5 text-xs outline-none placeholder:text-muted-foreground/60"
        >
      </div>
    </div>
    <div class="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
      <div v-if="loading" class="px-3 py-6 text-center text-xs text-muted-foreground">
        {{ $t('common.loading') }}
      </div>
      <div v-else-if="!filtered.length" class="px-3 py-6 text-center text-xs text-muted-foreground">
        {{ sessions.length ? $t('ai.chat.noMatchSession') : $t('ai.chat.noSessions') }}
      </div>
      <div
        v-for="s in filtered"
        :key="s.id"
        class="group relative mt-0.5 flex items-center gap-2 rounded-lg px-2.5 py-2 text-[13px] transition-colors"
        :class="s.id === activeId ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'"
      >
        <FaIcon
          name="i-lucide:message-square"
          class="shrink-0 text-sm"
          :class="s.id === activeId ? 'text-primary' : 'text-muted-foreground/60'"
        />
        <template v-if="editingId === s.id">
          <input
            ref="editInput"
            v-model="editingTitle"
            type="text"
            class="min-w-0 flex-1 rounded border border-primary/40 bg-background px-1.5 py-0.5 text-[13px] outline-none"
            @keydown="onEditKeydown($event, s)"
            @blur="commitRename(s)"
            @click.stop
          >
        </template>
        <template v-else>
          <button type="button" class="min-w-0 flex-1 cursor-pointer truncate text-left" @click="emit('select', s.id)">
            {{ s.title }}
          </button>
          <!-- 运行态徽标（悬停时让位给操作按钮） -->
          <span
            v-if="s.runStatus && RUN_BADGE[s.runStatus] && editingId !== s.id"
            class="hidden shrink-0 rounded-full px-1.5 py-0.5 text-[10px] group-hover:hidden sm:inline-flex"
            :class="RUN_BADGE[s.runStatus].cls"
          >
            {{ $t(RUN_BADGE[s.runStatus].key) }}
          </span>
          <span class="hidden shrink-0 items-center gap-0.5 group-hover:flex max-md:flex">
            <button
              type="button"
              class="cursor-pointer rounded p-1 text-muted-foreground/70 transition-colors hover:bg-background hover:text-foreground"
              :title="$t('common.rename')"
              @click.stop="startRename(s)"
            >
              <FaIcon name="i-lucide:pencil" class="text-[11px]" />
            </button>
            <button
              type="button"
              class="cursor-pointer rounded p-1 text-muted-foreground/70 transition-colors hover:bg-background hover:text-red-500"
              :title="$t('common.delete')"
              @click.stop="emit('remove', s.id)"
            >
              <FaIcon name="i-lucide:trash-2" class="text-[11px]" />
            </button>
          </span>
        </template>
      </div>
    </div>
  </div>
</template>
